// Package typederror identifies runtime error values for plan retry.on.
package typederror

import "fmt"

// StructTypeField tags struct instances created from struct literals.
const StructTypeField = "__type"

// TypeOf reports the logical error/type name for a runtime value.
// String payloads are "str"; struct maps with __type use that name.
func TypeOf(val any) string {
	switch v := val.(type) {
	case string:
		return "str"
	case map[string]any:
		if t, ok := v[StructTypeField].(string); ok && t != "" {
			return t
		}
	}
	return ""
}

// TagStruct records a struct's type name on its runtime map value.
func TagStruct(typeName string, fields map[string]any) map[string]any {
	if typeName != "" {
		fields[StructTypeField] = typeName
	}
	return fields
}

// Error is a failure carrying an optional logical type name for retry.on.
type Error struct {
	Type    string
	Message string
	Value   any
}

func (e *Error) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("%v", e.Value)
}

// TypeName returns the logical type of err, or "" if unknown.
func TypeName(err error) string {
	if te, ok := err.(*Error); ok {
		return te.Type
	}
	return ""
}

// FromValue builds a typed error from a Result err payload or other value.
func FromValue(val any) *Error {
	if m, ok := val.(map[string]any); ok {
		if tag, _ := m["tag"].(string); tag == "err" {
			return FromValue(m["val"])
		}
	}
	return &Error{
		Type:    TypeOf(val),
		Message: fmt.Sprintf("%v", val),
		Value:   val,
	}
}

// MatchesOn reports whether err's type is listed in on.
// An empty on list matches every error (backward compatible).
func MatchesOn(on []string, err error) bool {
	if len(on) == 0 {
		return true
	}
	t := TypeName(err)
	for _, allowed := range on {
		if t == allowed {
			return true
		}
	}
	return false
}
