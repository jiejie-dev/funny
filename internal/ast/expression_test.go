package ast

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLiteralExpr_String(t *testing.T) {
	e := &LiteralExpr{Value: 42}
	assert.Equal(t, "42", e.String())
}

func TestVariableExpr_String(t *testing.T) {
	e := &VariableExpr{Name: "foo"}
	assert.Equal(t, "foo", e.String())
}

func TestBinaryExpr_String(t *testing.T) {
	e := &BinaryExpr{Left: &VariableExpr{Name: "a"}, Op: "+", Right: &LiteralExpr{Value: 1}}
	assert.Equal(t, "a + 1", e.String())
}

func TestUnaryExpr_String(t *testing.T) {
	e := &UnaryExpr{Op: "not", Expr: &VariableExpr{Name: "x"}}
	assert.Equal(t, "not x", e.String())
}

func TestSubExpr_String(t *testing.T) {
	e := &SubExpr{Inner: &LiteralExpr{Value: 1}}
	assert.Equal(t, "(1)", e.String())
}

func TestListExpr_String(t *testing.T) {
	e := &ListExpr{Elements: []Expression{&LiteralExpr{Value: 1}, &LiteralExpr{Value: 2}}}
	assert.Equal(t, "[1, 2]", e.String())
}

func TestMapLiteralExpr_String(t *testing.T) {
	e := &MapLiteralExpr{
		Keys:   []Expression{&LiteralExpr{Value: "a"}},
		Values: []Expression{&LiteralExpr{Value: 1}},
	}
	assert.Equal(t, `{"a": 1}`, e.String())
}

func TestIndexExpr_String(t *testing.T) {
	e := &IndexExpr{Object: &VariableExpr{Name: "a"}, Index: &LiteralExpr{Value: 0}}
	assert.Equal(t, "a[0]", e.String())
}

func TestFieldExpr_String(t *testing.T) {
	e := &FieldExpr{Object: &VariableExpr{Name: "p"}, Field: "name"}
	assert.Equal(t, "p.name", e.String())
}

func TestCallExpr_String(t *testing.T) {
	e := &CallExpr{Func: &VariableExpr{Name: "f"}, Args: []Expression{&LiteralExpr{Value: 1}}}
	assert.Equal(t, "f(1)", e.String())
}

func TestFStringExpr_String(t *testing.T) {
	e := &FStringExpr{Raw: "hello {name}"}
	assert.Equal(t, `f"hello {name}"`, e.String())
}

func TestLiteralExpr_String_StringLiteral(t *testing.T) {
	e := &LiteralExpr{Value: "hi"}
	assert.Equal(t, `"hi"`, e.String())
}
