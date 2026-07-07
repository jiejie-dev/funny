package repl

import (
	"fmt"

	"github.com/jiejie-dev/funny/v2/internal/ast"
	"github.com/jiejie-dev/funny/v2/internal/compiler"
	"github.com/jiejie-dev/funny/v2/internal/vm"
)

func (s *Session) evalCellVM(prog *ast.Program) (result string, showed bool, err error) {
	s.stmts = append(s.stmts, prog.Stmts...)
	full := &ast.Program{Stmts: append([]ast.Statement(nil), s.stmts...)}
	mod, err := compiler.Compile(full, s.replPath)
	if err != nil {
		return "", false, fmt.Errorf("compile: %w", err)
	}
	m := vm.New(mod)
	val, err := m.Run()
	if err != nil {
		return "", false, err
	}
	s.bindings = mergeBindings(m.MainBindings(), declBindings(s.stmts))
	if shouldShowVMResult(prog, val) {
		return FormatValue(val), true, nil
	}
	return "", false, nil
}
