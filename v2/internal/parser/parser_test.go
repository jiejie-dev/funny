package parser

import (
	"testing"

	"github.com/jerloo/funny/v2/internal/ast"
	"github.com/stretchr/testify/assert"
)

func TestParser_Empty(t *testing.T) {
	p := New("", "")
	prog, err := p.Parse()
	assert.NoError(t, err)
	assert.Empty(t, prog.Stmts)
}

func TestParser_Stubs(t *testing.T) {
	p := New("let x = 1", "")
	_, err := p.Parse()
	assert.Error(t, err)
}

func TestParser_ExprLiteral(t *testing.T) {
	p := New("42", "")
	prog, err := p.Parse()
	assert.NoError(t, err)
	exprStmt := prog.Stmts[0].(*ast.ExprStmt)
	lit := exprStmt.X.(*ast.LiteralExpr)
	assert.Equal(t, 42, lit.Value)
}

func TestParser_ExprBinary(t *testing.T) {
	p := New("1 + 2 * 3", "")
	prog, err := p.Parse()
	assert.NoError(t, err)
	expr := prog.Stmts[0].(*ast.ExprStmt).X.(*ast.BinaryExpr)
	assert.Equal(t, "+", expr.Op)
	rhs := expr.Right.(*ast.BinaryExpr)
	assert.Equal(t, "*", rhs.Op)
}

func TestParser_ExprSub(t *testing.T) {
	p := New("(1 + 2) * 3", "")
	prog, err := p.Parse()
	assert.NoError(t, err)
	expr := prog.Stmts[0].(*ast.ExprStmt).X.(*ast.BinaryExpr)
	assert.Equal(t, "*", expr.Op)
}
