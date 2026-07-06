// Package module implements real (disk-backed) resolution of `import`
// statements: reading the imported file, recursively resolving its own
// imports, and splicing its `fn`/`struct` declarations into the importing
// program so that they can be type-checked, evaluated, and compiled like
// any other top-level declaration.
//
// Design summary (see docs/language-manual.md for user-facing behavior):
//
//   - Import paths are resolved relative to the *importing file's*
//     directory (or used as-is if already absolute).
//   - Only top-level `fn` and `struct` declarations are extracted from an
//     imported file; any other top-level statement in a dependency file
//     (let, expr, meta, plan, ...) is ignored. Dependency files are treated
//     purely as function/struct libraries.
//   - `import "path"` (no alias) merges the module's `pub` functions into
//     the importer's flat namespace under their original bare names, so
//     they're called directly (`add(1, 2)`).
//   - `import "path" as m` does NOT rename the module's declarations; it
//     only teaches the *call sites in the importing file* that `m.add(...)`
//     means "call the pub function `add` from that module". This mirrors
//     Python's `import numpy as np` semantics: `np` is a local nickname,
//     not a rename of numpy's internals.
//   - Struct types (pub or not) are always merged under their original bare
//     name, regardless of alias, since there is no `m.Point(...)` literal
//     syntax; only functions support alias-qualified calls.
//   - A module's own private (non-`pub`) functions are hygienically
//     renamed (e.g. `helper#3`) so they can never collide with, or be
//     called directly by, code outside that module, while still being
//     reachable from that module's own `pub` functions.
//   - Every dependency file is resolved and merged at most once per run,
//     even if reached via multiple import paths (diamond dependencies).
//   - Circular imports and duplicate top-level symbol names (across
//     distinct files) are compile errors.
package module

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jiejie-dev/funny/v2/internal/ast"
	"github.com/jiejie-dev/funny/v2/internal/errs"
	"github.com/jiejie-dev/funny/v2/internal/parser"
)

// resolvedModule is the fully-processed result of resolving one dependency
// file. ownDecls holds only *this file's* fn/struct declarations (already
// privacy-renamed and alias-rewritten); directDeps lists the modules it
// itself imports. Flattening the transitive graph into a single decl list
// happens once, globally, in flatten() - never eagerly per file - so that a
// module reached via two different import paths (a diamond dependency)
// still ends up in the final program exactly once.
type resolvedModule struct {
	path       string
	ownDecls   []ast.Statement
	directDeps []*resolvedModule
	pubFuncs   map[string]bool
	pubStructs map[string]bool
}

type resolver struct {
	cache    map[string]*resolvedModule
	inFlight map[string]bool
	stack    []string
	counter  int
}

// Resolve expands every top-level `import` statement reachable from prog
// (whose source lives at mainPath) into real declarations loaded from disk,
// returning a new Program with those declarations spliced in ahead of
// prog's own statements. mainPath is used only to resolve relative import
// paths; prog is not re-read from it.
func Resolve(prog *ast.Program, mainPath string) (*ast.Program, error) {
	if !hasImports(prog) {
		return prog, nil
	}
	absMain := mainAbsPath(mainPath)
	r := &resolver{
		cache:    map[string]*resolvedModule{},
		inFlight: map[string]bool{absMain: true},
		stack:    []string{absMain},
	}

	directDeps, aliases, err := r.resolveImportsOf(prog, absMain)
	if err != nil {
		return nil, err
	}

	ctx := &rewriteCtx{aliases: aliases}
	for _, s := range prog.Stmts {
		if err := rewriteStmtRefs(s, ctx); err != nil {
			return nil, err
		}
	}

	global := map[string]string{}
	depDecls, err := flatten(directDeps, global)
	if err != nil {
		return nil, err
	}
	for _, s := range prog.Stmts {
		name, ok := declName(s)
		if !ok {
			continue
		}
		if owner, dup := global[name]; dup {
			return nil, errs.New("E1104",
				fmt.Sprintf("duplicate symbol %q: already declared by imported module %s", name, owner),
				toErrsPos(s.Pos()), "rename one of the two, or import the module with `as` and remove the unaliased one")
		}
		global[name] = absMain
	}

	out := &ast.Program{
		NodePos: prog.NodePos,
		Stmts:   append(append([]ast.Statement{}, depDecls...), prog.Stmts...),
	}
	return out, nil
}

// flatten walks the dependency graph rooted at deps in DFS post-order
// (dependencies before dependents), visiting each unique module exactly
// once across the *entire* call - this is what makes diamond dependencies
// safe. seenNames accumulates symbol -> owning-module-path for duplicate
// detection and is also useful to the caller for checking the importer's
// own top-level names against it afterwards.
func flatten(deps []*resolvedModule, seenNames map[string]string) ([]ast.Statement, error) {
	visited := map[string]bool{}
	var out []ast.Statement
	var visit func(m *resolvedModule) error
	visit = func(m *resolvedModule) error {
		if visited[m.path] {
			return nil
		}
		visited[m.path] = true
		for _, dep := range m.directDeps {
			if err := visit(dep); err != nil {
				return err
			}
		}
		for _, d := range m.ownDecls {
			if name, ok := declName(d); ok {
				if owner, dup := seenNames[name]; dup && owner != m.path {
					return errs.New("E1104",
						fmt.Sprintf("duplicate symbol %q: declared in both %s and %s", name, owner, m.path),
						toErrsPos(d.Pos()), "")
				}
				seenNames[name] = m.path
			}
		}
		out = append(out, m.ownDecls...)
		return nil
	}
	for _, m := range deps {
		if err := visit(m); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func hasImports(prog *ast.Program) bool {
	for _, s := range prog.Stmts {
		if _, ok := s.(*ast.ImportDecl); ok {
			return true
		}
	}
	return false
}

func mainAbsPath(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	abs = filepath.Clean(abs)
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved
	}
	return abs
}

// resolveImportsOf resolves every top-level ImportDecl found directly in
// prog (belonging to the file at ownerAbsPath), returning the list of
// directly-imported modules (deduplicated, in import order) and a map of
// alias -> module for later call-site rewriting. Transitive flattening is
// deferred to flatten(), so this never eagerly expands nested imports.
func (r *resolver) resolveImportsOf(prog *ast.Program, ownerAbsPath string) ([]*resolvedModule, map[string]*resolvedModule, error) {
	aliases := map[string]*resolvedModule{}
	seen := map[string]bool{}
	var deps []*resolvedModule

	for _, s := range prog.Stmts {
		imp, ok := s.(*ast.ImportDecl)
		if !ok {
			continue
		}
		depPath, err := resolveImportPath(filepath.Dir(ownerAbsPath), imp.Path)
		if err != nil {
			return nil, nil, wrapImportErr(imp, err)
		}
		mod, err := r.resolveFile(depPath)
		if err != nil {
			return nil, nil, wrapImportErr(imp, err)
		}
		if imp.Alias != "" {
			aliases[imp.Alias] = mod
		}
		if seen[mod.path] {
			continue
		}
		seen[mod.path] = true
		deps = append(deps, mod)
	}
	return deps, aliases, nil
}

// resolveFile parses and fully resolves the dependency file at path
// (absolute, cleaned), memoizing the result so diamond dependencies are
// only processed once.
func (r *resolver) resolveFile(path string) (*resolvedModule, error) {
	if mod, ok := r.cache[path]; ok {
		return mod, nil
	}
	if r.inFlight[path] {
		chain := append(append([]string{}, r.stack...), path)
		return nil, errs.New("E1101",
			fmt.Sprintf("circular import: %s", strings.Join(chain, " -> ")),
			errs.Position{File: path}, "")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errs.New("E1102", fmt.Sprintf("cannot read imported module %q: %v", path, err), errs.Position{File: path}, "")
	}
	p := parser.New(string(data), path)
	prog, err := p.Parse()
	if err != nil {
		return nil, err
	}

	r.inFlight[path] = true
	r.stack = append(r.stack, path)
	defer func() {
		delete(r.inFlight, path)
		r.stack = r.stack[:len(r.stack)-1]
	}()

	directDeps, aliases, err := r.resolveImportsOf(prog, path)
	if err != nil {
		return nil, err
	}

	r.counter++
	modID := r.counter

	var ownDecls []ast.Statement
	privateRename := map[string]string{}
	pubFuncs := map[string]bool{}
	pubStructs := map[string]bool{}
	for _, s := range prog.Stmts {
		switch n := s.(type) {
		case *ast.FnDecl:
			ownDecls = append(ownDecls, n)
			if n.Pub {
				pubFuncs[n.Name] = true
			} else {
				privateRename[n.Name] = fmt.Sprintf("%s#%d", n.Name, modID)
			}
		case *ast.StructDecl:
			ownDecls = append(ownDecls, n)
			if n.Pub {
				pubStructs[n.Name] = true
			}
		}
	}

	ctx := &rewriteCtx{aliases: aliases, privateRename: privateRename}
	for _, s := range ownDecls {
		fn, ok := s.(*ast.FnDecl)
		if !ok {
			continue
		}
		if err := rewriteBlockRefs(fn.Body, ctx); err != nil {
			return nil, err
		}
	}
	for _, s := range ownDecls {
		if fn, ok := s.(*ast.FnDecl); ok {
			if newName, ok := privateRename[fn.Name]; ok {
				fn.Name = newName
			}
		}
	}

	mod := &resolvedModule{
		path:       path,
		ownDecls:   ownDecls,
		directDeps: directDeps,
		pubFuncs:   pubFuncs,
		pubStructs: pubStructs,
	}
	r.cache[path] = mod
	return mod, nil
}

func resolveImportPath(baseDir, importPath string) (string, error) {
	if strings.TrimSpace(importPath) == "" {
		return "", fmt.Errorf("empty import path")
	}
	p := importPath
	if !filepath.IsAbs(p) {
		p = filepath.Join(baseDir, p)
	}
	p = filepath.Clean(p)
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		p = resolved
	}
	return p, nil
}

func declName(s ast.Statement) (string, bool) {
	switch n := s.(type) {
	case *ast.FnDecl:
		return n.Name, true
	case *ast.StructDecl:
		return n.Name, true
	}
	return "", false
}

func wrapImportErr(imp *ast.ImportDecl, err error) error {
	return fmt.Errorf("import %q at %s: %w", imp.Path, imp.NodePos.String(), err)
}

func toErrsPos(p ast.Pos) errs.Position {
	return errs.Position{File: p.File, Line: p.Line, Col: p.Col}
}

// rewriteCtx carries the two kinds of call-site rewrites that must be
// applied while walking a file's function bodies:
//   - aliases: `alias.field(...)` -> plain call of the module's pub `field`
//   - privateRename: bare call of one of *this file's own* private
//     functions -> its hygienic name
type rewriteCtx struct {
	aliases       map[string]*resolvedModule
	privateRename map[string]string
}

func rewriteBlockRefs(b *ast.Block, ctx *rewriteCtx) error {
	if b == nil {
		return nil
	}
	for _, s := range b.Statements {
		if err := rewriteStmtRefs(s, ctx); err != nil {
			return err
		}
	}
	return nil
}

func rewriteStmtRefs(s ast.Statement, ctx *rewriteCtx) error {
	switch n := s.(type) {
	case *ast.LetStmt:
		return rewriteExprRefs(n.Value, ctx)
	case *ast.AssignStmt:
		if err := rewriteExprRefs(n.Target, ctx); err != nil {
			return err
		}
		return rewriteExprRefs(n.Value, ctx)
	case *ast.ExprStmt:
		return rewriteExprRefs(n.X, ctx)
	case *ast.ReturnStmt:
		if n.Value == nil {
			return nil
		}
		return rewriteExprRefs(n.Value, ctx)
	case *ast.IfStmt:
		if err := rewriteExprRefs(n.Cond, ctx); err != nil {
			return err
		}
		if err := rewriteBlockRefs(n.Then, ctx); err != nil {
			return err
		}
		if n.ElseIf != nil {
			if err := rewriteStmtRefs(n.ElseIf, ctx); err != nil {
				return err
			}
		}
		return rewriteBlockRefs(n.ElseBlock, ctx)
	case *ast.ForStmt:
		if err := rewriteExprRefs(n.Iterable, ctx); err != nil {
			return err
		}
		return rewriteBlockRefs(n.Body, ctx)
	case *ast.WhileStmt:
		if err := rewriteExprRefs(n.Cond, ctx); err != nil {
			return err
		}
		return rewriteBlockRefs(n.Body, ctx)
	case *ast.MatchStmt:
		if err := rewriteExprRefs(n.Expr, ctx); err != nil {
			return err
		}
		for _, arm := range n.Arms {
			if err := rewriteExprRefs(arm.Pattern, ctx); err != nil {
				return err
			}
			if err := rewriteBlockRefs(arm.Body, ctx); err != nil {
				return err
			}
		}
		return nil
	case *ast.FnDecl:
		return rewriteBlockRefs(n.Body, ctx)
	case *ast.Block:
		return rewriteBlockRefs(n, ctx)
	}
	return nil
}

func rewriteExprRefs(e ast.Expression, ctx *rewriteCtx) error {
	switch n := e.(type) {
	case nil, *ast.LiteralExpr, *ast.VariableExpr:
		return nil
	case *ast.BinaryExpr:
		if err := rewriteExprRefs(n.Left, ctx); err != nil {
			return err
		}
		return rewriteExprRefs(n.Right, ctx)
	case *ast.UnaryExpr:
		return rewriteExprRefs(n.Expr, ctx)
	case *ast.SubExpr:
		return rewriteExprRefs(n.Inner, ctx)
	case *ast.ListExpr:
		for _, el := range n.Elements {
			if err := rewriteExprRefs(el, ctx); err != nil {
				return err
			}
		}
		return nil
	case *ast.MapLiteralExpr:
		for i := range n.Keys {
			if err := rewriteExprRefs(n.Keys[i], ctx); err != nil {
				return err
			}
			if err := rewriteExprRefs(n.Values[i], ctx); err != nil {
				return err
			}
		}
		return nil
	case *ast.IndexExpr:
		if err := rewriteExprRefs(n.Object, ctx); err != nil {
			return err
		}
		return rewriteExprRefs(n.Index, ctx)
	case *ast.FieldExpr:
		return rewriteExprRefs(n.Object, ctx)
	case *ast.StructLiteralExpr:
		for _, v := range n.Fields {
			if err := rewriteExprRefs(v, ctx); err != nil {
				return err
			}
		}
		return nil
	case *ast.FStringExpr:
		for _, part := range n.Parts {
			if part.Expr != nil {
				if err := rewriteExprRefs(part.Expr, ctx); err != nil {
					return err
				}
			}
		}
		return nil
	case *ast.TryExpr:
		return rewriteExprRefs(n.Inner, ctx)
	case *ast.CallExpr:
		if err := rewriteExprRefs(n.Func, ctx); err != nil {
			return err
		}
		for _, a := range n.Args {
			if err := rewriteExprRefs(a, ctx); err != nil {
				return err
			}
		}
		return rewriteCallTarget(n, ctx)
	}
	return nil
}

func rewriteCallTarget(n *ast.CallExpr, ctx *rewriteCtx) error {
	switch fn := n.Func.(type) {
	case *ast.VariableExpr:
		if newName, ok := ctx.privateRename[fn.Name]; ok {
			fn.Name = newName
		}
		return nil
	case *ast.FieldExpr:
		obj, ok := fn.Object.(*ast.VariableExpr)
		if !ok {
			return nil
		}
		mod, ok := ctx.aliases[obj.Name]
		if !ok {
			return nil
		}
		if !mod.pubFuncs[fn.Field] {
			return errs.New("E1105",
				fmt.Sprintf("module %q has no public function %q", mod.path, fn.Field),
				toErrsPos(n.NodePos), fmt.Sprintf("only `pub fn %s` declarations are reachable via `%s.%s(...)`", fn.Field, obj.Name, fn.Field))
		}
		n.Func = &ast.VariableExpr{NodePos: fn.NodePos, Name: fn.Field}
		return nil
	}
	return nil
}
