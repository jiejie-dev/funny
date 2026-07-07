// v2/internal/evaluator/scope.go
package evaluator

import "sync"

// Scope's map access is synchronized because it's no longer guaranteed to
// only ever see one goroutine at a time: internal/agent's plan engine runs
// a `parallel` step's body statements concurrently (execParallel), and a
// step that exceeds its `timeout` leaves its goroutine running in the
// background (execWithTimeout) while later steps continue on the same
// Evaluator/Scope. The mutex only guards against data races (undefined
// behavior / map corruption); it does not make the *values* those
// goroutines see consistent — see execWithTimeout's doc comment in
// internal/agent/engine.go for that caveat.
type Scope struct {
	mu     sync.RWMutex
	parent *Scope
	vars   map[string]any
}

func NewScope(parent *Scope) *Scope {
	return &Scope{parent: parent, vars: map[string]any{}}
}

func (s *Scope) Set(name string, value any) {
	s.mu.Lock()
	s.vars[name] = value
	s.mu.Unlock()
}

func (s *Scope) Get(name string) (any, bool) {
	s.mu.RLock()
	v, ok := s.vars[name]
	s.mu.RUnlock()
	if ok {
		return v, true
	}
	if s.parent != nil {
		return s.parent.Get(name)
	}
	return nil, false
}

func (s *Scope) Has(name string) bool {
	_, ok := s.Get(name)
	return ok
}

func (s *Scope) Assign(name string, value any) bool {
	s.mu.Lock()
	_, ok := s.vars[name]
	if ok {
		s.vars[name] = value
	}
	s.mu.Unlock()
	if ok {
		return true
	}
	if s.parent != nil {
		return s.parent.Assign(name, value)
	}
	return false
}

// Bindings returns all names visible from this scope (locals shadow parents).
func (s *Scope) Bindings() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := map[string]any{}
	for k, v := range s.vars {
		out[k] = v
	}
	if s.parent != nil {
		for k, v := range s.parent.Bindings() {
			if _, ok := out[k]; !ok {
				out[k] = v
			}
		}
	}
	return out
}
