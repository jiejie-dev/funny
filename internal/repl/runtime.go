package repl

import (
	"os"

	"github.com/jiejie-dev/funny/v2/internal/ast"
)

func useInterpretBackend() bool {
	if os.Getenv("FUNNY_REPL_INTERPRET") != "" {
		return true
	}
	return os.Getenv("FUNNY_INTERPRET") != ""
}

func lastMeaningfulStmt(prog *ast.Program) ast.Statement {
	if prog == nil {
		return nil
	}
	for i := len(prog.Stmts) - 1; i >= 0; i-- {
		switch prog.Stmts[i].(type) {
		case *ast.CommentStmt:
			continue
		default:
			return prog.Stmts[i]
		}
	}
	return nil
}

// cellMayShowResult reports whether the cell's last statement can produce
// a displayed value (final decision for if/match is runtime).
func cellMayShowResult(prog *ast.Program) bool {
	last := lastMeaningfulStmt(prog)
	if last == nil {
		return false
	}
	switch last.(type) {
	case *ast.ExprStmt, *ast.IfStmt:
		return true
	default:
		return false
	}
}

// shouldShowVMResult decides whether to print the VM stack top for a cell.
func shouldShowVMResult(prog *ast.Program, val any) bool {
	if !cellMayShowResult(prog) {
		return false
	}
	last := lastMeaningfulStmt(prog)
	if es, ok := last.(*ast.ExprStmt); ok {
		if _, isCall := es.X.(*ast.CallExpr); isCall {
			return true
		}
	}
	return val != nil
}

func declBindings(stmts []ast.Statement) map[string]any {
	out := map[string]any{}
	for _, s := range stmts {
		switch n := s.(type) {
		case *ast.FnDecl:
			out[n.Name] = n
		case *ast.StructDecl:
			out[n.Name] = n
		}
	}
	return out
}

func mergeBindings(runtime map[string]any, decls map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range decls {
		out[k] = v
	}
	for k, v := range runtime {
		out[k] = v
	}
	return out
}
