package docgen

import (
	"strings"

	"github.com/jiejie-dev/funny/v2/internal/ast"
	"github.com/jiejie-dev/funny/v2/internal/types"
)

// CollectSymbols extracts documented symbols from an already type-checked program.
func CollectSymbols(prog *ast.Program, env *types.Env) []SymbolDoc {
	if prog == nil {
		return nil
	}
	var symbols []SymbolDoc
	var pending []string
	flushPending := func() []string {
		if len(pending) == 0 {
			return nil
		}
		lines := append([]string(nil), pending...)
		pending = nil
		return lines
	}
	for _, s := range prog.Stmts {
		switch n := s.(type) {
		case *ast.CommentStmt:
			if n.Doc {
				line := strings.TrimSpace(n.Text)
				if line != "" {
					pending = append(pending, line)
				}
			}
		case *ast.MetaBlock:
			pending = nil
		case *ast.FnDecl:
			lines := flushPending()
			symbols = append(symbols, fnSymbol(n, lines, env))
		case *ast.StructDecl:
			lines := flushPending()
			symbols = append(symbols, structSymbol(n, lines))
		default:
			pending = nil
		}
	}
	return symbols
}

// SymbolIndex maps symbol name to its documentation.
func SymbolIndex(prog *ast.Program, env *types.Env) map[string]SymbolDoc {
	symbols := CollectSymbols(prog, env)
	idx := make(map[string]SymbolDoc, len(symbols))
	for _, sym := range symbols {
		idx[sym.Name] = sym
	}
	return idx
}
