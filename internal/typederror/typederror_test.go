package typederror

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTypeOf_StringIsStr(t *testing.T) {
	assert.Equal(t, "str", TypeOf("boom"))
}

func TestTypeOf_StructUsesTypeField(t *testing.T) {
	m := TagStruct("NetworkError", map[string]any{"message": "timeout"})
	assert.Equal(t, "NetworkError", TypeOf(m))
}

func TestMatchesOn_EmptyOnMatchesAll(t *testing.T) {
	err := &Error{Type: "str", Message: "x"}
	assert.True(t, MatchesOn(nil, err))
}

func TestMatchesOn_FiltersByType(t *testing.T) {
	err := &Error{Type: "NetworkError", Message: "timeout"}
	assert.True(t, MatchesOn([]string{"NetworkError"}, err))
	assert.False(t, MatchesOn([]string{"FatalError"}, err))
}

func TestFromValue_UnwrapsErrResult(t *testing.T) {
	val := map[string]any{
		"tag": "err",
		"val": TagStruct("NetworkError", map[string]any{"message": "x"}),
	}
	err := FromValue(val)
	assert.Equal(t, "NetworkError", err.Type)
}
