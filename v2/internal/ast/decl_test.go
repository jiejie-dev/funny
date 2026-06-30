package ast

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParam_String(t *testing.T) {
	p := Param{Name: "x", TypeAnn: "int"}
	assert.Equal(t, "x: int", p.String())
}

func TestParam_StringNoType(t *testing.T) {
	p := Param{Name: "x"}
	assert.Equal(t, "x", p.String())
}

func TestFnDecl_String(t *testing.T) {
	f := &FnDecl{
		Name:    "add",
		Params:  []Param{{Name: "a", TypeAnn: "int"}, {Name: "b", TypeAnn: "int"}},
		RetType: "int",
		Body: &Block{Statements: []Statement{
			&ReturnStmt{Value: &BinaryExpr{Left: &VariableExpr{Name: "a"}, Op: "+", Right: &VariableExpr{Name: "b"}}},
		}},
	}
	out := f.String()
	assert.Contains(t, out, "fn add(a: int, b: int) -> int:")
	assert.Contains(t, out, "return a + b")
}

func TestFnDecl_StringNoReturn(t *testing.T) {
	f := &FnDecl{
		Name:   "noop",
		Params: []Param{},
		Body:   &Block{Statements: []Statement{}},
	}
	out := f.String()
	assert.Contains(t, out, "fn noop():")
}

func TestFnDecl_StringPub(t *testing.T) {
	f := &FnDecl{
		Pub:     true,
		Name:    "hello",
		Params:  []Param{},
		RetType: "str",
		Body:    &Block{Statements: []Statement{&ReturnStmt{Value: &LiteralExpr{Value: "hi"}}}},
	}
	out := f.String()
	assert.Contains(t, out, "pub fn hello() -> str:")
}

func TestStructDecl_String(t *testing.T) {
	s := &StructDecl{
		Name:   "User",
		Fields: []Param{{Name: "name", TypeAnn: "str"}, {Name: "age", TypeAnn: "int"}},
	}
	out := s.String()
	assert.Contains(t, out, "struct User:")
	assert.Contains(t, out, "name: str")
	assert.Contains(t, out, "age: int")
}

func TestStructDecl_StringPub(t *testing.T) {
	s := &StructDecl{
		Pub:    true,
		Name:   "Item",
		Fields: []Param{{Name: "id", TypeAnn: "int"}},
	}
	out := s.String()
	assert.Contains(t, out, "pub struct Item:")
}
