// v2/internal/agent/engine.go
package agent

import (
	"fmt"

	"github.com/jerloo/funny/v2/internal/ast"
	"github.com/jerloo/funny/v2/internal/evaluator"
)

// Engine executes plan blocks step-by-step.
type Engine struct {
	eval *evaluator.Evaluator
}

func New() *Engine {
	return &Engine{eval: evaluator.New(nil)}
}

// RunPlan executes a plan block. Steps are processed in order.
func (e *Engine) RunPlan(plan *ast.PlanBlock, file string) error {
	return e.execBlock(plan.Body)
}

func (e *Engine) execBlock(b *ast.Block) error {
	if b == nil {
		return nil
	}
	for _, stmt := range b.Statements {
		if err := e.execStmt(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) execStmt(s ast.Statement) error {
	switch n := s.(type) {
	case *ast.Step:
		return e.execStep(n)
	case *ast.LetStmt, *ast.AssignStmt, *ast.IfStmt, *ast.WhileStmt, *ast.ExprStmt:
		return e.eval.Exec(toProgram(n))
	case *ast.ReturnStmt:
		return fmt.Errorf("return outside function in plan step")
	}
	return fmt.Errorf("agent: unsupported statement type %T", s)
}

func (e *Engine) execStep(s *ast.Step) error {
	e.eval.Scope().Set("__step_name", s.Name)
	return e.execBlock(s.Body)
}

// toProgram wraps a statement in a Program for evaluator.Exec.
func toProgram(s ast.Statement) *ast.Program {
	return &ast.Program{Stmts: []ast.Statement{s}}
}
