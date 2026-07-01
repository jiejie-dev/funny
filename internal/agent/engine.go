// v2/internal/agent/engine.go
package agent

import (
	"fmt"
	"sync"

	"github.com/jerloo/funny/internal/ast"
	"github.com/jerloo/funny/internal/evaluator"
)

// Engine executes plan blocks step-by-step.
type Engine struct {
	eval *evaluator.Evaluator
}

func New() *Engine {
	return &Engine{eval: evaluator.New(nil)}
}

// RunPlan executes a plan block. Steps are processed in order.
// Each step's body is evaluated; for "tool" steps the final expression's
// value is stored in the scope as __result.
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
	case *ast.LetStmt, *ast.AssignStmt, *ast.ExprStmt:
		return e.eval.Exec(toProgram(n))
	case *ast.IfStmt:
		return e.execIf(n)
	case *ast.WhileStmt:
		return e.execWhile(n)
	case *ast.ReturnStmt:
		return e.execReturn(n)
	}
	return fmt.Errorf("agent: unsupported statement type %T", s)
}

// execIf executes an if-statement within a plan step body so that
// returns in nested blocks are caught by the engine.
func (e *Engine) execIf(n *ast.IfStmt) error {
	cond, err := e.eval.Eval(n.Cond)
	if err != nil {
		return err
	}
	if truthy(cond) {
		return e.execBlock(n.Then)
	}
	if n.ElseIf != nil {
		return e.execStmt(n.ElseIf)
	}
	if n.ElseBlock != nil {
		return e.execBlock(n.ElseBlock)
	}
	return nil
}

// execWhile executes a while-loop within a plan step body.
func (e *Engine) execWhile(n *ast.WhileStmt) error {
	for {
		cond, err := e.eval.Eval(n.Cond)
		if err != nil {
			return err
		}
		if !truthy(cond) {
			return nil
		}
		if err := e.execBlock(n.Body); err != nil {
			return err
		}
	}
}

// execReturn treats a return statement as a step-level signal.
// A bare return or `return <value>` is treated as step success, but
// `return err(...)` (a Result tagged "err") is treated as a step error
// so retry logic can catch it.
func (e *Engine) execReturn(n *ast.ReturnStmt) error {
	if n.Value == nil {
		return nil
	}
	v, err := e.eval.Eval(n.Value)
	if err != nil {
		return err
	}
	if m, ok := v.(map[string]any); ok {
		if tag, _ := m["tag"].(string); tag == "err" {
			return fmt.Errorf("%v", m["val"])
		}
	}
	return nil
}

func (e *Engine) execStep(s *ast.Step) error {
	e.eval.Scope().Set("__step_name", s.Name)
	switch s.Kind {
	case ast.StepParallel:
		return e.execParallel(s)
	default:
		return e.execBlockRetry(s)
	}
}

// execBlockRetry runs the step body with retry support.
func (e *Engine) execBlockRetry(s *ast.Step) error {
	maxAttempts := 1
	if s.Retry != nil && s.Retry.Max > 0 {
		maxAttempts = s.Retry.Max
	}
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := e.execBlock(s.Body); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	return fmt.Errorf("step %q failed after %d attempts: %w", s.Name, maxAttempts, lastErr)
}

// execParallel runs each statement in the step body concurrently using goroutines.
func (e *Engine) execParallel(s *ast.Step) error {
	if s.Body == nil {
		return nil
	}
	var wg sync.WaitGroup
	errCh := make(chan error, len(s.Body.Statements))
	for _, stmt := range s.Body.Statements {
		wg.Add(1)
		stmt := stmt
		go func() {
			defer wg.Done()
			if err := e.eval.Exec(toProgram(stmt)); err != nil {
				errCh <- err
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			return err
		}
	}
	return nil
}

// truthy mirrors evaluator.truthy: only nil and false are falsy.
func truthy(v any) bool {
	if v == nil {
		return false
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return true
}

// toProgram wraps a statement in a Program for evaluator.Exec.
func toProgram(s ast.Statement) *ast.Program {
	return &ast.Program{Stmts: []ast.Statement{s}}
}
