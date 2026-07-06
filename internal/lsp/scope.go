package lsp

import "github.com/jiejie-dev/funny/v2/internal/ast"

// localSym describes a local binding (function parameter, `let`, or `for`
// loop variable) visible at some cursor position.
type localSym struct {
	Name    string
	TypeStr string // declared type annotation, "" if inferred/unknown
	Pos     ast.Pos
	Kind    string // "param" | "let" | "for"
}

// localsAt returns the local symbols in scope at target, in outer-to-inner
// declaration order (later entries shadow earlier ones with the same name).
// This is a best-effort static approximation: since the AST does not track
// block end positions, a block is treated as "possibly containing target"
// whenever it starts before target, without an upper bound check. This can
// over-include bindings from an earlier sibling block that has already
// closed, which is an acceptable trade-off for editor tooling (worst case:
// an occasional stale suggestion), not a soundness requirement.
func localsAt(prog *ast.Program, target ast.Pos) []localSym {
	var acc []localSym
	scanStmts(prog.Stmts, target, &acc)
	return acc
}

func before(p, target ast.Pos) bool {
	if p.Line != target.Line {
		return p.Line < target.Line
	}
	return p.Col <= target.Col
}

func scanStmts(stmts []ast.Statement, target ast.Pos, acc *[]localSym) {
	for _, s := range stmts {
		if !before(s.Pos(), target) {
			return
		}
		switch n := s.(type) {
		case *ast.LetStmt:
			*acc = append(*acc, localSym{Name: n.Name, TypeStr: n.TypeAnn, Pos: n.NodePos, Kind: "let"})
		case *ast.FnDecl:
			sub := append([]localSym{}, (*acc)...)
			for _, p := range n.Params {
				sub = append(sub, localSym{Name: p.Name, TypeStr: p.TypeAnn, Pos: n.NodePos, Kind: "param"})
			}
			if n.Body != nil {
				scanStmts(n.Body.Statements, target, &sub)
			}
			*acc = sub
		case *ast.IfStmt:
			scanIf(n, target, acc)
		case *ast.ForStmt:
			sub := append([]localSym{}, (*acc)...)
			sub = append(sub, localSym{Name: n.Name, Pos: n.NodePos, Kind: "for"})
			if n.Body != nil {
				scanStmts(n.Body.Statements, target, &sub)
			}
			*acc = sub
		case *ast.WhileStmt:
			sub := append([]localSym{}, (*acc)...)
			if n.Body != nil {
				scanStmts(n.Body.Statements, target, &sub)
			}
			*acc = sub
		case *ast.MatchStmt:
			for _, arm := range n.Arms {
				if arm.Body == nil {
					continue
				}
				sub := append([]localSym{}, (*acc)...)
				scanStmts(arm.Body.Statements, target, &sub)
				*acc = sub
			}
		}
	}
}

func scanIf(n *ast.IfStmt, target ast.Pos, acc *[]localSym) {
	if n.Then != nil {
		sub := append([]localSym{}, (*acc)...)
		scanStmts(n.Then.Statements, target, &sub)
		*acc = sub
	}
	if n.ElseIf != nil {
		scanIf(n.ElseIf, target, acc)
	}
	if n.ElseBlock != nil {
		sub := append([]localSym{}, (*acc)...)
		scanStmts(n.ElseBlock.Statements, target, &sub)
		*acc = sub
	}
}

// lookupLocal finds the innermost (last-declared) local with the given
// name, or ok=false if none is in scope.
func lookupLocal(locals []localSym, name string) (localSym, bool) {
	for i := len(locals) - 1; i >= 0; i-- {
		if locals[i].Name == name {
			return locals[i], true
		}
	}
	return localSym{}, false
}
