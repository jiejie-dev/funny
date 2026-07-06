// v2/internal/agent/engine.go
package agent

import (
	"fmt"
	"sync"
	"time"

	"github.com/jiejie-dev/funny/v2/internal/ast"
	"github.com/jiejie-dev/funny/v2/internal/evaluator"
)

// retryBackoffBase is the unit delay `backoff` strategies scale from. It's
// intentionally small (and not yet user-configurable — there's no grammar
// for it) so retry-heavy plans/tests stay fast; only the *shape* of the
// delay (constant/linear/exponential) is under the author's control today.
const retryBackoffBase = 10 * time.Millisecond

// Engine executes plan blocks step-by-step.
type Engine struct {
	eval *evaluator.Evaluator
}

func New() *Engine {
	return &Engine{eval: evaluator.New(nil)}
}

// RunPlan executes a plan block. Steps are processed in order. Each step's
// body is evaluated; the value of the body's final bare-expression
// statement (if any) is stored in scope as __result, so later steps can
// read what the previous one produced (e.g. `println(__result)`).
func (e *Engine) RunPlan(plan *ast.PlanBlock, file string) error {
	_, _, err := e.execBlock(plan.Body)
	return err
}

// execBlock runs every statement in b in order and returns the value of
// the LAST one (only), plus whether that last statement actually produced
// a meaningful value at all. A bare expression, `return <value>`, or a
// taken if/else branch that itself ends that way counts as "produced a
// value"; `let`/`assign`/`while`/an empty else-less if all reset it to
// false — a step ending in `let z = 0` has nothing to say about
// __result or a `guard`'s pass/fail, and must not be confused with an
// explicit falsy signal.
func (e *Engine) execBlock(b *ast.Block) (v any, has bool, err error) {
	if b == nil {
		return nil, false, nil
	}
	for _, stmt := range b.Statements {
		v, has, err = e.execStmt(stmt)
		if err != nil {
			return nil, false, err
		}
	}
	return v, has, nil
}

func (e *Engine) execStmt(s ast.Statement) (any, bool, error) {
	switch n := s.(type) {
	case *ast.Step:
		return nil, false, e.execStep(n)
	case *ast.LetStmt, *ast.AssignStmt:
		return nil, false, e.eval.Exec(toProgram(n))
	case *ast.ExprStmt:
		v, err := e.eval.Eval(n.X)
		if err != nil {
			return nil, false, err
		}
		return v, true, nil
	case *ast.IfStmt:
		return e.execIf(n)
	case *ast.WhileStmt:
		return nil, false, e.execWhile(n)
	case *ast.ReturnStmt:
		return e.execReturn(n)
	case *ast.CommentStmt:
		// A plan body interleaves `# ...` comments between step
		// declarations at the same indent level (that's exactly how a
		// well-documented, readable plan is written), so these show up
		// as real statements in plan.Body.Statements right alongside the
		// *ast.Step nodes - not just at file scope. Before this, any
		// plan with such a comment failed outright with "unsupported
		// statement type *ast.CommentStmt" the moment RunPlan reached
		// it, even though the compiler and evaluator have always treated
		// comments as no-ops everywhere else.
		return nil, false, nil
	}
	return nil, false, fmt.Errorf("agent: unsupported statement type %T", s)
}

// execIf executes an if-statement within a plan step body so that
// returns in nested blocks are caught by the engine, propagating the
// taken branch's value (see execBlock).
func (e *Engine) execIf(n *ast.IfStmt) (any, bool, error) {
	cond, err := e.eval.Eval(n.Cond)
	if err != nil {
		return nil, false, err
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
	return nil, false, nil
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
		if _, _, err := e.execBlock(n.Body); err != nil {
			return err
		}
	}
}

// execReturn treats a return statement as a step-level signal.
// A bare return is treated as step success with no value; `return <value>`
// is a step success carrying that value; `return err(...)` (a Result
// tagged "err") is treated as a step error so retry logic can catch it.
func (e *Engine) execReturn(n *ast.ReturnStmt) (any, bool, error) {
	if n.Value == nil {
		return nil, false, nil
	}
	v, err := e.eval.Eval(n.Value)
	if err != nil {
		return nil, false, err
	}
	if m, ok := v.(map[string]any); ok {
		if tag, _ := m["tag"].(string); tag == "err" {
			return nil, false, fmt.Errorf("%v", m["val"])
		}
	}
	return v, true, nil
}

func (e *Engine) execStep(s *ast.Step) error {
	e.eval.Scope().Set("__step_name", s.Name)
	if s.Kind == ast.StepParallel {
		return e.execParallel(s)
	}
	if s.Kind == ast.StepDelay {
		d, err := stepTimeout(s)
		if err != nil {
			return fmt.Errorf("step %q: %w", s.Name, err)
		}
		if d == 0 {
			return fmt.Errorf("step %q: a `delay` step needs `with timeout=\"<duration>\"` to know how long to wait", s.Name)
		}
		time.Sleep(d)
	}
	return e.execBlockRetry(s)
}

// execBlockRetry runs the step body with retry support, an optional
// per-attempt wall-clock timeout, and — only for StepGuard — treats a
// falsy/err(...) final expression as a failed assertion the same way a
// `return err(...)` already is. If the body ends in a bare
// expression/return value, that value is published to scope as __result
// on success.
func (e *Engine) execBlockRetry(s *ast.Step) error {
	maxAttempts := 1
	if s.Retry != nil && s.Retry.Max > 0 {
		maxAttempts = s.Retry.Max
	}
	timeout, err := stepTimeout(s)
	if err != nil {
		return fmt.Errorf("step %q: %w", s.Name, err)
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		result, has, err := e.runStepBodyOnce(s, timeout)
		if err == nil {
			if has {
				e.eval.Scope().Set("__result", result)
			}
			return nil
		}
		lastErr = err
		if attempt < maxAttempts {
			if d := backoffDelay(s.Retry, attempt); d > 0 {
				time.Sleep(d)
			}
		}
	}
	if maxAttempts == 1 {
		return fmt.Errorf("step %q failed: %w", s.Name, lastErr)
	}
	return fmt.Errorf("step %q failed after %d attempts: %w", s.Name, maxAttempts, lastErr)
}

// runStepBodyOnce executes s.Body once (bounded by timeout, if set) and,
// for a `guard` step whose body ends in a bare expression/return value
// (has == true — see execBlock), turns a falsy/err(...) final value into
// an error. A guard body that ends in `let`/`assign`/a value-less branch
// (has == false) has made no explicit assertion and always passes.
func (e *Engine) runStepBodyOnce(s *ast.Step, timeout time.Duration) (any, bool, error) {
	run := func() (any, bool, error) { return e.execBlock(s.Body) }
	var v any
	var has bool
	var err error
	if timeout > 0 {
		v, has, err = e.execWithTimeout(run, timeout)
	} else {
		v, has, err = run()
	}
	if err != nil {
		return nil, false, err
	}
	if s.Kind == ast.StepGuard && has {
		if reason := guardFailureReason(v); reason != "" {
			return nil, false, fmt.Errorf("guard failed: %s", reason)
		}
	}
	return v, has, nil
}

// execWithTimeout runs fn on its own goroutine and returns a timeout error
// if it doesn't finish within d.
//
// Caveat (documented, not silently glossed over): the tree-walking
// evaluator has no preemption point, so a fn that's genuinely stuck (e.g.
// `while true: let x = 1`) keeps its goroutine running in the background
// after this returns the timeout error — it is not killed. Since that
// goroutine shares e.eval's scope with whatever step runs next, a timed-out
// step that later "wakes up" and keeps mutating scope races with
// subsequent steps. Treat `timeout` as a plan-control-flow SLA signal for
// well-behaved (eventually-terminating, e.g. blocked-on-I/O) steps, not as
// a hard isolation guarantee for adversarial/infinite-looping ones.
func (e *Engine) execWithTimeout(fn func() (any, bool, error), d time.Duration) (any, bool, error) {
	type outcome struct {
		v   any
		has bool
		err error
	}
	ch := make(chan outcome, 1)
	go func() {
		v, has, err := fn()
		ch <- outcome{v, has, err}
	}()
	select {
	case o := <-ch:
		return o.v, o.has, o.err
	case <-time.After(d):
		return nil, false, fmt.Errorf("timed out after %s", d)
	}
}

// backoffDelay returns how long to wait after a failed attempt before the
// next one. Immediate retry (0 delay) unless the step opted into a
// `backoff` strategy — this keeps `with retry max=N` (no backoff)
// behaving exactly as it did before backoff support existed.
func backoffDelay(r *ast.Retry, attempt int) time.Duration {
	if r == nil || r.Backoff == "" {
		return 0
	}
	switch r.Backoff {
	case "constant":
		return retryBackoffBase
	case "linear":
		return retryBackoffBase * time.Duration(attempt)
	case "exp":
		return retryBackoffBase * time.Duration(uint(1)<<uint(attempt-1))
	default:
		return 0
	}
}

// guardFailureReason reports why v should fail a `guard` step's
// assertion, or "" if it passes. An err(...) Result fails regardless of
// its .val's truthiness (mirroring execReturn's `return err(...)`
// handling); an ok(...) Result always passes; anything else falls back to
// plain truthy().
func guardFailureReason(v any) string {
	if m, ok := v.(map[string]any); ok {
		tag, _ := m["tag"].(string)
		switch tag {
		case "err":
			return fmt.Sprintf("%v", m["val"])
		case "ok":
			return ""
		}
	}
	if !truthy(v) {
		return "condition was false"
	}
	return ""
}

// stepTimeout parses s.Timeout (already validated at parse time by
// time.ParseDuration, see parseStep) or returns 0 if unset.
func stepTimeout(s *ast.Step) (time.Duration, error) {
	if s.Timeout == "" {
		return 0, nil
	}
	d, err := time.ParseDuration(s.Timeout)
	if err != nil {
		return 0, fmt.Errorf("invalid timeout %q: %w", s.Timeout, err)
	}
	return d, nil
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
