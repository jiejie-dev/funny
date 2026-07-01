package errs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestError_Format(t *testing.T) {
	pos := Position{File: "test.fn", Line: 3, Col: 5}
	e := New("E1001", "unexpected token", pos, "expected `:`")
	got := e.Format()
	assert.Contains(t, got, "error[E1001]")
	assert.Contains(t, got, "unexpected token")
	assert.Contains(t, got, "test.fn:3:5")
	assert.Contains(t, got, "expected `:`")
}

func TestError_Format_WithoutHint(t *testing.T) {
	pos := Position{File: "test.fn", Line: 0, Col: 0}
	e := New("E0001", "lexer error", pos, "")
	got := e.Format()
	assert.Contains(t, got, "error[E0001]")
	assert.NotContains(t, got, "help:")
}
