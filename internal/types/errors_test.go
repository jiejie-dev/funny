package types

import (
	"testing"

	"github.com/jiejie-dev/funny/internal/ast"
	"github.com/stretchr/testify/assert"
)

func TestError_Format(t *testing.T) {
	e := &Error{
		Code:     "E2010",
		Message:  "type mismatch",
		Pos:      ast.Pos{Line: 3, Col: 5},
		Expected: Primitive("int"),
		Actual:   Primitive("str"),
	}
	s := e.Format()
	assert.Contains(t, s, "E2010")
	assert.Contains(t, s, "type mismatch")
	assert.Contains(t, s, "int")
	assert.Contains(t, s, "str")
}

func TestError_Error(t *testing.T) {
	e := &Error{Code: "E2001", Message: "undefined variable: x", Pos: ast.Pos{}}
	assert.Contains(t, e.Error(), "undefined variable")
}

func TestNewMismatch(t *testing.T) {
	e := NewMismatch(ast.Pos{Line: 1, Col: 2}, Primitive("int"), Primitive("str"))
	assert.Equal(t, "E2010", e.Code)
	assert.Contains(t, e.Message, "int")
	assert.Contains(t, e.Message, "str")
	assert.NotEmpty(t, e.Hint)
}
