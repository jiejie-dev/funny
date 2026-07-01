package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnv_DeclareLookup(t *testing.T) {
	e := NewEnv(nil)
	e.DeclareVar("x", Primitive("int"))
	ty, ok := e.LookupVar("x")
	assert.True(t, ok)
	assert.Equal(t, Primitive("int"), ty)
}

func TestEnv_NestedLookup(t *testing.T) {
	outer := NewEnv(nil)
	outer.DeclareVar("a", Primitive("int"))
	inner := NewEnv(outer)
	inner.DeclareVar("b", Primitive("str"))
	ty, _ := inner.LookupVar("a")
	assert.Equal(t, Primitive("int"), ty)
	ty, _ = inner.LookupVar("b")
	assert.Equal(t, Primitive("str"), ty)
}

func TestEnv_FuncLookup(t *testing.T) {
	e := NewEnv(nil)
	f := Func{Params: []Type{Primitive("int")}, Return: Primitive("int")}
	e.DeclareFunc("add", f)
	got, ok := e.LookupFunc("add")
	assert.True(t, ok)
	assert.True(t, got.Equal(f))
}

func TestEnv_StructLookup(t *testing.T) {
	e := NewEnv(nil)
	s := Struct{Name: "User", Fields: map[string]Type{"name": Primitive("str")}}
	e.DeclareStruct("User", s)
	got, ok := e.LookupStruct("User")
	assert.True(t, ok)
	assert.True(t, got.Equal(s))
}

func TestEnv_Shadowing(t *testing.T) {
	outer := NewEnv(nil)
	outer.DeclareVar("x", Primitive("int"))
	inner := NewEnv(outer)
	inner.DeclareVar("x", Primitive("str"))
	ty, _ := inner.LookupVar("x")
	assert.Equal(t, Primitive("str"), ty)
}
