// Tests for `{key: value, ...}` map literal parsing, including the
// multi-line form where the braces span several lines and each pair ends
// with a trailing comma (bracket line-continuation, see internal/lexer).
package parser

import (
	"testing"

	"github.com/jiejie-dev/funny/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMapLiteral_SingleLine(t *testing.T) {
	p := New(`{"a": 1, "b": 2}`, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	m := prog.Stmts[0].(*ast.ExprStmt).X.(*ast.MapLiteralExpr)
	require.Len(t, m.Keys, 2)
	require.Len(t, m.Values, 2)
	assert.Equal(t, "a", m.Keys[0].(*ast.LiteralExpr).Value)
	assert.Equal(t, 1, m.Values[0].(*ast.LiteralExpr).Value)
	assert.Equal(t, "b", m.Keys[1].(*ast.LiteralExpr).Value)
	assert.Equal(t, 2, m.Values[1].(*ast.LiteralExpr).Value)
}

func TestParseMapLiteral_Empty(t *testing.T) {
	p := New(`{}`, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	m := prog.Stmts[0].(*ast.ExprStmt).X.(*ast.MapLiteralExpr)
	assert.Empty(t, m.Keys)
	assert.Empty(t, m.Values)
}

func TestParseMapLiteral_TrailingCommaSingleLine(t *testing.T) {
	p := New(`{"a": 1,}`, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	m := prog.Stmts[0].(*ast.ExprStmt).X.(*ast.MapLiteralExpr)
	require.Len(t, m.Keys, 1)
}

// TestParseMapLiteral_MultiLine is the exact style requested: braces span
// multiple lines, and every key/value pair is on its own line ending with a
// trailing comma.
func TestParseMapLiteral_MultiLine(t *testing.T) {
	src := "{\n    \"a\": 1,\n    \"b\": 2,\n}\n"
	p := New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	m := prog.Stmts[0].(*ast.ExprStmt).X.(*ast.MapLiteralExpr)
	require.Len(t, m.Keys, 2)
	assert.Equal(t, "a", m.Keys[0].(*ast.LiteralExpr).Value)
	assert.Equal(t, 1, m.Values[0].(*ast.LiteralExpr).Value)
	assert.Equal(t, "b", m.Keys[1].(*ast.LiteralExpr).Value)
	assert.Equal(t, 2, m.Values[1].(*ast.LiteralExpr).Value)
}

// TestParseMapLiteral_MultiLineInLetStmt matches the RELEASE_NOTES-promised
// syntax `let m: map[str, int] = {...}`, spanning multiple lines.
func TestParseMapLiteral_MultiLineInLetStmt(t *testing.T) {
	src := "let m: map[str, int] = {\n    \"a\": 1,\n    \"b\": 2,\n}\n"
	p := New(src, "t")
	prog, err := p.Parse()
	require.NoError(t, err)
	let := prog.Stmts[0].(*ast.LetStmt)
	assert.Equal(t, "map[str, int]", let.TypeAnn)
	m := let.Value.(*ast.MapLiteralExpr)
	require.Len(t, m.Keys, 2)
}

func TestParseMapLiteral_NestedInList(t *testing.T) {
	src := "[\n    {\"a\": 1},\n    {\"b\": 2},\n]\n"
	p := New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	list := prog.Stmts[0].(*ast.ExprStmt).X.(*ast.ListExpr)
	require.Len(t, list.Elements, 2)
	_, ok := list.Elements[0].(*ast.MapLiteralExpr)
	assert.True(t, ok)
}

func TestParseMapLiteral_MissingColonIsError(t *testing.T) {
	p := New(`{"a" 1}`, "")
	_, err := p.Parse()
	assert.Error(t, err)
}
