package parser

import (
	"testing"

	"github.com/jiejie-dev/funny/v2/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFString_LiteralOnly(t *testing.T) {
	p := New(`f"hello world"`, "t")
	e, err := p.parseExpression()
	require.NoError(t, err)
	f := e.(*ast.FStringExpr)
	require.Len(t, f.Parts, 1)
	assert.Equal(t, "hello world", f.Parts[0].Text)
	assert.Nil(t, f.Parts[0].Expr)
}

func TestParseFString_SimpleInterpolation(t *testing.T) {
	p := New(`f"hi {name}!"`, "t")
	e, err := p.parseExpression()
	require.NoError(t, err)
	f := e.(*ast.FStringExpr)
	require.Len(t, f.Parts, 3)
	assert.Equal(t, "hi ", f.Parts[0].Text)
	v, ok := f.Parts[1].Expr.(*ast.VariableExpr)
	require.True(t, ok)
	assert.Equal(t, "name", v.Name)
	assert.Equal(t, "!", f.Parts[2].Text)
}

func TestParseFString_WithFormatSpec(t *testing.T) {
	p := New(`f"{price:.2f}"`, "t")
	e, err := p.parseExpression()
	require.NoError(t, err)
	f := e.(*ast.FStringExpr)
	require.Len(t, f.Parts, 1)
	assert.Equal(t, ".2f", f.Parts[0].Spec)
}

func TestParseFString_NamedArgColonNotMistakenForSpec(t *testing.T) {
	p := New(`f"{Point(x: 1, y: 2)}"`, "t")
	e, err := p.parseExpression()
	require.NoError(t, err)
	f := e.(*ast.FStringExpr)
	require.Len(t, f.Parts, 1)
	assert.Equal(t, "", f.Parts[0].Spec)
	_, ok := f.Parts[0].Expr.(*ast.StructLiteralExpr)
	assert.True(t, ok)
}

func TestParseFString_DoubleBraceEscape(t *testing.T) {
	p := New(`f"{{not interpolated}}"`, "t")
	e, err := p.parseExpression()
	require.NoError(t, err)
	f := e.(*ast.FStringExpr)
	require.Len(t, f.Parts, 1)
	assert.Equal(t, "{not interpolated}", f.Parts[0].Text)
}

func TestParseFString_EmptyInterpolationErrors(t *testing.T) {
	p := New(`f"{}"`, "t")
	_, err := p.parseExpression()
	assert.Error(t, err)
}

func TestParseFString_UnmatchedCloseBraceErrors(t *testing.T) {
	p := New(`f"oops }"`, "t")
	_, err := p.parseExpression()
	assert.Error(t, err)
}

func TestParseFString_IndexExpression(t *testing.T) {
	p := New(`f"{items[0]}"`, "t")
	e, err := p.parseExpression()
	require.NoError(t, err)
	f := e.(*ast.FStringExpr)
	require.Len(t, f.Parts, 1)
	_, ok := f.Parts[0].Expr.(*ast.IndexExpr)
	assert.True(t, ok)
}

func TestParseFString_NestedStringLiteralInExpr(t *testing.T) {
	p := New(`f"{greet('x')}"`, "t")
	e, err := p.parseExpression()
	require.NoError(t, err)
	f := e.(*ast.FStringExpr)
	require.Len(t, f.Parts, 1)
	_, ok := f.Parts[0].Expr.(*ast.CallExpr)
	assert.True(t, ok)
}

func TestParseFString_MultipleInterpolationsWithSpecs(t *testing.T) {
	p := New(`f"{a:>5} and {b:.1f}"`, "t")
	e, err := p.parseExpression()
	require.NoError(t, err)
	f := e.(*ast.FStringExpr)
	require.Len(t, f.Parts, 3)
	assert.Equal(t, ">5", f.Parts[0].Spec)
	assert.Equal(t, " and ", f.Parts[1].Text)
	assert.Equal(t, ".1f", f.Parts[2].Spec)
}
