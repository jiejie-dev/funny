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
