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
func Equal(a, b Type) bool {
	if a == nil || b == nil {
		return false
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
