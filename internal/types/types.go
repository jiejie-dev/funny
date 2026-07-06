package types

// Type is the sealed interface for all type system types.
// Only types in this package can implement it (private marker).
type Type interface {
	String() string
	Equal(other Type) bool
	typeMarker()
}

// Primitive is a built-in type like "int", "str", "bool", "float", "nil".
type Primitive string

func (p Primitive) String() string { return string(p) }
func (p Primitive) Equal(other Type) bool {
	o, ok := other.(Primitive)
	return ok && p == o
}
func (p Primitive) typeMarker() {}

// Equal is a convenience for comparing two Types.
// Returns false if either is nil.
// Special case: bare `Result` (Primitive "Result") matches any Result[T, E],
// supporting Result as a top-level placeholder when concrete Ok/Err types
// don't matter (e.g., a function returning any Result).
func Equal(a, b Type) bool {
	if a == nil || b == nil {
		return false
	}
	if p, ok := a.(Primitive); ok && string(p) == "Result" {
		if _, isResult := b.(Result); isResult {
			return true
		}
	}
	if p, ok := b.(Primitive); ok && string(p) == "Result" {
		if _, isResult := a.(Result); isResult {
			return true
		}
	}
	return a.Equal(b)
}

// List is a homogeneous list type: list[T].
type List struct {
	Elem Type
}

func (l List) String() string {
	return "list[" + l.Elem.String() + "]"
}
func (l List) Equal(other Type) bool {
	o, ok := other.(List)
	return ok && Equal(l.Elem, o.Elem)
}
func (l List) typeMarker() {}

// Map is a key-value map type: map[K, V].
type Map struct {
	Key   Type
	Value Type
}

func (m Map) String() string {
	return "map[" + m.Key.String() + ", " + m.Value.String() + "]"
}
func (m Map) Equal(other Type) bool {
	o, ok := other.(Map)
	return ok && Equal(m.Key, o.Key) && Equal(m.Value, o.Value)
}
func (m Map) typeMarker() {}

// Struct is a user-defined struct type with named fields.
type Struct struct {
	Name    string
	Fields  map[string]Type
	Mutable map[string]bool // field name → declared with `mut`
}

func (s Struct) String() string {
	out := s.Name + ":\n"
	for k, v := range s.Fields {
		out += "    " + k + ": " + v.String() + "\n"
	}
	return out
}

func (s Struct) Equal(other Type) bool {
	o, ok := other.(Struct)
	if !ok || s.Name != o.Name || len(s.Fields) != len(o.Fields) {
		return false
	}
	for k, v := range s.Fields {
		ov, ok := o.Fields[k]
		if !ok || !Equal(v, ov) {
			return false
		}
		if s.Mutable[k] != o.Mutable[k] {
			return false
		}
	}
	return true
}

// IsMutable reports whether field name was declared with `mut`.
func (s Struct) IsMutable(name string) bool {
	return s.Mutable[name]
}

func (s Struct) typeMarker() {}

// Field looks up a field by name. Returns (nil, false) if not found.
func (s Struct) Field(name string) (Type, bool) {
	t, ok := s.Fields[name]
	return t, ok
}

func (s Struct) FieldNames() []string {
	out := make([]string, 0, len(s.Fields))
	for k := range s.Fields {
		out = append(out, k)
	}
	return out
}

// Func is a function type: (params) -> return.
type Func struct {
	Params []Type
	Return Type
}

func (f Func) String() string {
	out := "("
	for i, p := range f.Params {
		if i > 0 {
			out += ", "
		}
		out += p.String()
	}
	out += ") -> " + f.Return.String()
	return out
}

func (f Func) Equal(other Type) bool {
	o, ok := other.(Func)
	if !ok || len(f.Params) != len(o.Params) {
		return false
	}
	for i := range f.Params {
		if !Equal(f.Params[i], o.Params[i]) {
			return false
		}
	}
	return Equal(f.Return, o.Return)
}

func (f Func) typeMarker() {}

// Arity returns the number of parameters.
func (f Func) Arity() int { return len(f.Params) }

// Result is a fallible operation result: Result[T, E].
type Result struct {
	Ok  Type
	Err Type
}

func (r Result) String() string {
	return "Result[" + r.Ok.String() + ", " + r.Err.String() + "]"
}

func (r Result) Equal(other Type) bool {
	o, ok := other.(Result)
	return ok && Equal(r.Ok, o.Ok) && Equal(r.Err, o.Err)
}

func (r Result) typeMarker() {}

// Optional is a nullable type: T?.
type Optional struct {
	Inner Type
}

func (o Optional) String() string {
	return o.Inner.String() + "?"
}

func (o Optional) Equal(other Type) bool {
	inner, ok := other.(Optional)
	return ok && Equal(o.Inner, inner.Inner)
}

func (o Optional) typeMarker() {}
