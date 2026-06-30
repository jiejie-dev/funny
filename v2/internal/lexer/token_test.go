package lexer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokenKind_IsKeyword(t *testing.T) {
	assert.True(t, IF.IsKeyword())
	assert.True(t, FN.IsKeyword())
	assert.True(t, LET.IsKeyword())
	assert.False(t, NAME.IsKeyword())
	assert.False(t, INT.IsKeyword())
	assert.False(t, PLUS.IsKeyword())
}

func TestTokenKind_String(t *testing.T) {
	assert.Equal(t, "if", string(IF))
	assert.Equal(t, "fn", string(FN))
	assert.Equal(t, "+", string(PLUS))
}
