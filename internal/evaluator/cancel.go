package evaluator

import (
	"context"
	"errors"
)

// ErrCancelled is returned when execution observes a cancelled context.
var ErrCancelled = errors.New("evaluator: cancelled")

// NewWithContext returns an evaluator that stops at preemption points when
// ctx is cancelled (used by the plan engine for step timeouts).
func NewWithContext(scope *Scope, ctx context.Context) *Evaluator {
	if scope == nil {
		scope = NewScope(nil)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return &Evaluator{scope: scope, ctx: ctx}
}

func (e *Evaluator) checkCancel() error {
	if e.ctx == nil {
		return nil
	}
	select {
	case <-e.ctx.Done():
		return ErrCancelled
	default:
		return nil
	}
}

// CheckCancel reports whether this evaluator's context has been cancelled.
func (e *Evaluator) CheckCancel() error {
	return e.checkCancel()
}
