package types

// Env is a type environment that tracks variables, functions, and structs.
type Env struct {
	parent  *Env
	vars    map[string]Type
	funcs   map[string]Func
	structs map[string]Struct
}

// NewEnv creates a new Env, optionally nested inside parent.
func NewEnv(parent *Env) *Env {
	return &Env{
		parent:  parent,
		vars:    map[string]Type{},
		funcs:   map[string]Func{},
		structs: map[string]Struct{},
	}
}

// DeclareVar defines a variable in this scope (no parent traversal).
func (e *Env) DeclareVar(name string, t Type) {
	e.vars[name] = t
}

// LookupVar finds a variable, walking up parent scopes.
func (e *Env) LookupVar(name string) (Type, bool) {
	if t, ok := e.vars[name]; ok {
		return t, true
	}
	if e.parent != nil {
		return e.parent.LookupVar(name)
	}
	return nil, false
}

// DeclareFunc registers a function in this scope.
func (e *Env) DeclareFunc(name string, f Func) {
	e.funcs[name] = f
}

// LookupFunc finds a function by name.
func (e *Env) LookupFunc(name string) (Func, bool) {
	if f, ok := e.funcs[name]; ok {
		return f, true
	}
	if e.parent != nil {
		return e.parent.LookupFunc(name)
	}
	return Func{}, false
}

// DeclareStruct registers a struct type in this scope.
func (e *Env) DeclareStruct(name string, s Struct) {
	e.structs[name] = s
}

// LookupStruct finds a struct type by name.
func (e *Env) LookupStruct(name string) (Struct, bool) {
	if s, ok := e.structs[name]; ok {
		return s, true
	}
	if e.parent != nil {
		return e.parent.LookupStruct(name)
	}
	return Struct{}, false
}

// Funcs returns the functions declared directly in this scope (not
// including parent scopes). Used by tooling (e.g. the LSP server) that
// needs to enumerate available symbols; not used by the type checker
// itself.
func (e *Env) Funcs() map[string]Func {
	return e.funcs
}

// Structs returns the struct types declared directly in this scope (not
// including parent scopes). See Funcs for usage notes.
func (e *Env) Structs() map[string]Struct {
	return e.structs
}

// Vars returns the variables declared directly in this scope (not
// including parent scopes). See Funcs for usage notes.
func (e *Env) Vars() map[string]Type {
	return e.vars
}

// Has reports whether any binding with this name exists in this scope chain.
func (e *Env) Has(name string) bool {
	if _, ok := e.vars[name]; ok {
		return true
	}
	if _, ok := e.funcs[name]; ok {
		return true
	}
	if _, ok := e.structs[name]; ok {
		return true
	}
	if e.parent != nil {
		return e.parent.Has(name)
	}
	return false
}
