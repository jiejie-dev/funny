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

func TestParser_Let(t *testing.T) {
	p := New("let x = 1", "")
	prog, err := p.Parse()
	assert.NoError(t, err)
	let := prog.Stmts[0].(*ast.LetStmt)
	assert.Equal(t, "x", let.Name)
}

func TestParser_LetWithType(t *testing.T) {
	p := New("let x: int = 1", "")
	prog, err := p.Parse()
	assert.NoError(t, err)
	let := prog.Stmts[0].(*ast.LetStmt)
	assert.Equal(t, "int", let.TypeAnn)
}

func TestParser_Assign(t *testing.T) {
	p := New("x = 2", "")
	prog, err := p.Parse()
	assert.NoError(t, err)
	assign := prog.Stmts[0].(*ast.AssignStmt)
	assert.Equal(t, "2", assign.Value.String())
}

func TestParser_ExprStatement(t *testing.T) {
	p := New("foo()", "")
	prog, err := p.Parse()
	assert.NoError(t, err)
	exprStmt := prog.Stmts[0].(*ast.ExprStmt)
	call := exprStmt.X.(*ast.CallExpr)
	assert.Equal(t, "foo", call.Func.String())
}
