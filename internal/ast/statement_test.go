package ast

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBlock_String(t *testing.T) {
	b := &Block{Statements: []Statement{
		&LetStmt{Name: "x", Value: &LiteralExpr{Value: 1}},
		&ExprStmt{X: &VariableExpr{Name: "x"}},
	}}
	s := b.String()
	assert.Contains(t, s, "let x = 1")
	assert.Contains(t, s, "x")
}

func TestLetStmt_String(t *testing.T) {
	s := &LetStmt{Name: "x", Value: &LiteralExpr{Value: 1}}
	assert.Equal(t, "let x = 1", s.String())
}

func TestLetStmt_StringWithType(t *testing.T) {
	s := &LetStmt{Name: "x", TypeAnn: "int", Value: &LiteralExpr{Value: 1}}
	assert.Equal(t, "let x: int = 1", s.String())
}

func TestAssignStmt_String(t *testing.T) {
	s := &AssignStmt{Target: &VariableExpr{Name: "x"}, Value: &LiteralExpr{Value: 2}}
	assert.Equal(t, "x = 2", s.String())
}

func TestIfStmt_String(t *testing.T) {
	s := &IfStmt{
		Cond: &VariableExpr{Name: "x"},
		Then: &Block{Statements: []Statement{
			&LetStmt{Name: "y", Value: &LiteralExpr{Value: 1}},
		}},
	}
	out := s.String()
	assert.Contains(t, out, "if x:")
	assert.Contains(t, out, "let y = 1")
}

func TestIfStmt_StringWithElse(t *testing.T) {
	s := &IfStmt{
		Cond:      &VariableExpr{Name: "x"},
		Then:      &Block{Statements: []Statement{&LetStmt{Name: "a", Value: &LiteralExpr{Value: 1}}}},
		ElseBlock: &Block{Statements: []Statement{&LetStmt{Name: "b", Value: &LiteralExpr{Value: 2}}}},
	}
	out := s.String()
	assert.Contains(t, out, "if x:")
	assert.Contains(t, out, "else:")
	assert.Contains(t, out, "let a = 1")
	assert.Contains(t, out, "let b = 2")
}

func TestForStmt_String(t *testing.T) {
	s := &ForStmt{
		Name:     "i",
		Iterable: &VariableExpr{Name: "items"},
		Body:     &Block{Statements: []Statement{&ExprStmt{X: &VariableExpr{Name: "i"}}}},
	}
	out := s.String()
	assert.Contains(t, out, "for i in items:")
}

func TestWhileStmt_String(t *testing.T) {
	s := &WhileStmt{
		Cond: &VariableExpr{Name: "x"},
		Body: &Block{Statements: []Statement{&ExprStmt{X: &VariableExpr{Name: "x"}}}},
	}
	out := s.String()
	assert.Contains(t, out, "while x:")
}

func TestReturnStmt_String(t *testing.T) {
	s := &ReturnStmt{Value: &LiteralExpr{Value: 1}}
	assert.Equal(t, "return 1", s.String())
}

func TestReturnStmt_NoValue(t *testing.T) {
	s := &ReturnStmt{}
	assert.Equal(t, "return", s.String())
}

func TestBreakStmt_String(t *testing.T) {
	s := &BreakStmt{}
	assert.Equal(t, "break", s.String())
}

func TestContinueStmt_String(t *testing.T) {
	s := &ContinueStmt{}
	assert.Equal(t, "continue", s.String())
}

func TestExprStmt_String(t *testing.T) {
	s := &ExprStmt{X: &VariableExpr{Name: "x"}}
	assert.Equal(t, "x", s.String())
}

func TestCommentStmt_String(t *testing.T) {
	s := &CommentStmt{Text: " hello"}
	assert.Equal(t, "# hello", s.String())
}

func TestCommentStmt_String_Doc(t *testing.T) {
	s := &CommentStmt{Text: " doc", Doc: true}
	assert.Equal(t, "## doc", s.String())
}
