package lsp

import "github.com/jiejie-dev/funny/internal/ast"

// symbolKind classifies what a resolved identifier refers to, driving how
// far referencesTo searches for other occurrences of the same name.
type symbolKind int

const (
	symUnknown symbolKind = iota
	symLocal              // function parameter or a `let`/`for` binding local to one function
	symGlobal             // top-level `let`, or a `fn`/`struct` declaration
)

// resolveSymbol identifies what the identifier `name` at target refers to,
// mirroring the same lookup order hover/completion use, and reports which
// top-level ast.Node (a *ast.FnDecl, if any) lexically contains target — the
// search boundary used for symLocal.
func (d *document) resolveSymbol(name string, target ast.Pos) (kind symbolKind, declPos ast.Pos, scope *ast.FnDecl) {
	if d.prog == nil {
		return symUnknown, ast.Pos{}, nil
	}
	if sym, ok := lookupLocal(localsAt(d.prog, target), name); ok {
		// A top-level `let` also shows up via localsAt (it scans every
		// LetStmt regardless of nesting), but it is visible from inside
		// every function body too (Funny functions close over the file's
		// top-level scope), so treat it as global rather than scoping the
		// search to a single function.
		if fn := enclosingFn(d.prog, sym.Pos); fn != nil {
			return symLocal, sym.Pos, fn
		}
		return symGlobal, sym.Pos, nil
	}
	if decl := findTopLevelDecl(d.prog, name); decl != nil {
		return symGlobal, decl.Pos(), nil
	}
	return symUnknown, ast.Pos{}, nil
}

// enclosingFn returns the top-level *ast.FnDecl whose body lexically
// contains pos, or nil if pos is not inside any function (e.g. it's a
// top-level declaration).
func enclosingFn(prog *ast.Program, pos ast.Pos) *ast.FnDecl {
	for i, s := range prog.Stmts {
		fn, ok := s.(*ast.FnDecl)
		if !ok {
			continue
		}
		end := ast.Pos{Line: 1 << 30}
		if i+1 < len(prog.Stmts) {
			end = prog.Stmts[i+1].Pos()
		}
		if before(fn.Pos(), pos) && pos.Line < end.Line {
			return fn
		}
	}
	return nil
}

// referencesTo finds every occurrence of an identifier named `name`
// (declaration included) within root, which is either a single function's
// body (symLocal) or the whole document (symGlobal).
func referencesTo(d *document, name string, kind symbolKind, scope *ast.FnDecl) []ast.Pos {
	var out []ast.Pos
	switch kind {
	case symLocal:
		if scope == nil {
			return nil
		}
		for _, p := range scope.Params {
			if p.Name == name {
				out = append(out, scope.Pos()) // best-effort: exact param column isn't tracked on ast.Param
			}
		}
		if scope.Body != nil {
			walkBlockForName(scope.Body, name, &out)
		}
	case symGlobal:
		for _, s := range d.prog.Stmts {
			walkStmtForName(s, name, &out)
		}
	}
	return out
}

func walkBlockForName(b *ast.Block, name string, out *[]ast.Pos) {
	if b == nil {
		return
	}
	for _, s := range b.Statements {
		walkStmtForName(s, name, out)
	}
}

func walkStmtForName(s ast.Statement, name string, out *[]ast.Pos) {
	switch n := s.(type) {
	case *ast.LetStmt:
		if n.Name == name {
			*out = append(*out, n.NodePos)
		}
		walkExprForName(n.Value, name, out)
	case *ast.AssignStmt:
		walkExprForName(n.Target, name, out)
		walkExprForName(n.Value, name, out)
	case *ast.IfStmt:
		walkExprForName(n.Cond, name, out)
		walkBlockForName(n.Then, name, out)
		if n.ElseIf != nil {
			walkStmtForName(n.ElseIf, name, out)
		}
		walkBlockForName(n.ElseBlock, name, out)
	case *ast.ForStmt:
		if n.Name == name {
			*out = append(*out, n.NodePos)
		}
		walkExprForName(n.Iterable, name, out)
		walkBlockForName(n.Body, name, out)
	case *ast.WhileStmt:
		walkExprForName(n.Cond, name, out)
		walkBlockForName(n.Body, name, out)
	case *ast.MatchStmt:
		walkExprForName(n.Expr, name, out)
		for _, arm := range n.Arms {
			walkExprForName(arm.Pattern, name, out)
			walkBlockForName(arm.Body, name, out)
		}
	case *ast.ReturnStmt:
		if n.Value != nil {
			walkExprForName(n.Value, name, out)
		}
	case *ast.ExprStmt:
		walkExprForName(n.X, name, out)
	case *ast.FnDecl:
		if n.Name == name {
			*out = append(*out, n.NodePos)
		}
		for _, p := range n.Params {
			if p.Name == name {
				*out = append(*out, n.NodePos)
			}
		}
		walkBlockForName(n.Body, name, out)
	case *ast.StructDecl:
		if n.Name == name {
			*out = append(*out, n.NodePos)
		}
	case *ast.PlanBlock:
		walkBlockForName(n.Body, name, out)
	case *ast.Step:
		walkBlockForName(n.Body, name, out)
	}
}

func walkExprForName(e ast.Expression, name string, out *[]ast.Pos) {
	switch n := e.(type) {
	case nil, *ast.LiteralExpr:
		return
	case *ast.VariableExpr:
		if n.Name == name {
			*out = append(*out, n.NodePos)
		}
	case *ast.BinaryExpr:
		walkExprForName(n.Left, name, out)
		walkExprForName(n.Right, name, out)
	case *ast.UnaryExpr:
		walkExprForName(n.Expr, name, out)
	case *ast.SubExpr:
		walkExprForName(n.Inner, name, out)
	case *ast.ListExpr:
		for _, el := range n.Elements {
			walkExprForName(el, name, out)
		}
	case *ast.MapLiteralExpr:
		for i := range n.Keys {
			walkExprForName(n.Keys[i], name, out)
			walkExprForName(n.Values[i], name, out)
		}
	case *ast.IndexExpr:
		walkExprForName(n.Object, name, out)
		walkExprForName(n.Index, name, out)
	case *ast.FieldExpr:
		walkExprForName(n.Object, name, out)
	case *ast.CallExpr:
		walkExprForName(n.Func, name, out)
		for _, a := range n.Args {
			walkExprForName(a, name, out)
		}
	case *ast.StructLiteralExpr:
		if n.TypeName == name {
			*out = append(*out, n.NodePos)
		}
		for _, v := range n.Fields {
			walkExprForName(v, name, out)
		}
	case *ast.FStringExpr:
		for _, part := range n.Parts {
			if part.Expr != nil {
				walkExprForName(part.Expr, name, out)
			}
		}
	case *ast.TryExpr:
		walkExprForName(n.Inner, name, out)
	}
}

// references implements textDocument/references: every occurrence of the
// identifier at pos, scoped to this document (cross-file references are
// not tracked — only the buffers open in this server would be candidates,
// and this server does not currently index other open documents for
// symbol usage).
func (d *document) references(pos Position, includeDeclaration bool) []Location {
	tok, _, ok := tokenAt(d.text, d.path, pos)
	if !ok || d.prog == nil {
		return nil
	}
	name := tok.Data
	target := ast.Pos{File: d.path, Line: tok.Pos.Line, Col: tok.Pos.Col}
	kind, declPos, scope := d.resolveSymbol(name, target)
	if kind == symUnknown {
		return nil
	}
	positions := referencesTo(d, name, kind, scope)
	locs := make([]Location, 0, len(positions))
	for _, p := range positions {
		if !includeDeclaration && p == declPos {
			continue
		}
		locs = append(locs, Location{URI: d.uri, Range: nameRange(astPosToLSP(p), len(name))})
	}
	return locs
}
