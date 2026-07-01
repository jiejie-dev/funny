package lsp

import (
	"github.com/jiejie-dev/funny/internal/ast"
	"github.com/jiejie-dev/funny/internal/lexer"
)

// definition resolves the identifier at pos to its declaration site. For
// functions and structs, this also resolves across `import`ed files: since
// module.Resolve splices imported declarations into the program while
// preserving their original ast.Pos.File, a plain top-level scan already
// finds cross-file declarations without any extra bookkeeping.
func (d *document) definition(pos Position) *Location {
	tok, _, ok := tokenAt(d.text, d.path, pos)
	if !ok || tok.Kind != lexer.NAME || d.prog == nil {
		return nil
	}
	name := tok.Data
	target := ast.Pos{File: d.path, Line: tok.Pos.Line, Col: tok.Pos.Col}

	if sym, ok := lookupLocal(localsAt(d.prog, target), name); ok {
		return &Location{URI: pathToURI(sym.Pos.File), Range: pointRange(astPosToLSP(sym.Pos))}
	}
	if decl := findTopLevelDecl(d.prog, name); decl != nil {
		return &Location{URI: pathToURI(decl.Pos().File), Range: pointRange(astPosToLSP(decl.Pos()))}
	}
	return nil
}

// findTopLevelDecl looks for a top-level fn/struct declaration named name.
func findTopLevelDecl(prog *ast.Program, name string) ast.Node {
	for _, s := range prog.Stmts {
		switch n := s.(type) {
		case *ast.FnDecl:
			if n.Name == name {
				return n
			}
		case *ast.StructDecl:
			if n.Name == name {
				return n
			}
		}
	}
	return nil
}
