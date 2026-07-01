// v2/internal/evaluator/scope.go
package evaluator

type Scope struct {
	parent *Scope
	vars   map[string]any
}

func NewScope(parent *Scope) *Scope {
	return &Scope{parent: parent, vars: map[string]any{}}
}

func (s *Scope) Set(name string, value any) {
	s.vars[name] = value
}

func (s *Scope) Get(name string) (any, bool) {
	if v, ok := s.vars[name]; ok {
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
	if _, ok := s.vars[name]; ok {
		s.vars[name] = value
		return true
	}
	if s.parent != nil {
		return s.parent.Assign(name, value)
	}
	return false
}
