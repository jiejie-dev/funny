package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestType_Primitive_String(t *testing.T) {
	p := Primitive("int")
	assert.Equal(t, "int", p.String())
}

func TestType_Primitive_Equal(t *testing.T) {
	a := Primitive("int")
	b := Primitive("int")
	c := Primitive("str")
	assert.True(t, a.Equal(b))
	assert.False(t, a.Equal(c))
}

func TestType_Equal_Nil(t *testing.T) {
	var nilType Type
	p := Primitive("int")
	assert.False(t, p.Equal(nilType))
	assert.False(t, Equal(nilType, p))
}

func TestType_List_String(t *testing.T) {
	lt := List{Primitive("int")}
	assert.Equal(t, "list[int]", lt.String())
}

func TestType_List_Equal(t *testing.T) {
	a := List{Primitive("int")}
	b := List{Primitive("int")}
	c := List{Primitive("str")}
	assert.True(t, a.Equal(b))
	assert.False(t, a.Equal(c))
}

func TestType_Map_String(t *testing.T) {
	m := Map{Primitive("str"), Primitive("int")}
	assert.Equal(t, "map[str, int]", m.String())
}

func TestType_Map_Equal(t *testing.T) {
	a := Map{Primitive("str"), Primitive("int")}
	b := Map{Primitive("str"), Primitive("int")}
	c := Map{Primitive("str"), Primitive("str")}
	assert.True(t, a.Equal(b))
	assert.False(t, a.Equal(c))
}

func TestType_Struct_String(t *testing.T) {
	s := Struct{
		Name: "User",
		Fields: map[string]Type{
			"name": Primitive("str"),
			"age":  Primitive("int"),
		},
	}
	out := s.String()
	assert.Contains(t, out, "User")
	assert.Contains(t, out, "name: str")
	assert.Contains(t, out, "age: int")
}

func TestType_Struct_Equal(t *testing.T) {
	a := Struct{
		Name: "User",
		Fields: map[string]Type{
			"name": Primitive("str"),
			"age":  Primitive("int"),
		},
	}
	b := Struct{
		Name: "User",
		Fields: map[string]Type{
			"name": Primitive("str"),
			"age":  Primitive("int"),
		},
	}
	c := Struct{
		Name:   "User",
		Fields: map[string]Type{"name": Primitive("str")},
	}
	assert.True(t, a.Equal(b))
	assert.False(t, a.Equal(c))
}

func TestType_Struct_Field(t *testing.T) {
	s := Struct{
		Name:   "User",
		Fields: map[string]Type{"name": Primitive("str")},
	}
	f, ok := s.Field("name")
	assert.True(t, ok)
	assert.Equal(t, Primitive("str"), f)
	_, ok = s.Field("missing")
	assert.False(t, ok)
}

func TestType_Func_String(t *testing.T) {
	f := Func{
		Params: []Type{Primitive("int"), Primitive("int")},
		Return: Primitive("int"),
	}
	assert.Equal(t, "(int, int) -> int", f.String())
}

func TestType_Func_StringNoParams(t *testing.T) {
	f := Func{Return: Primitive("str")}
	assert.Equal(t, "() -> str", f.String())
}

func TestType_Func_Equal(t *testing.T) {
	a := Func{Params: []Type{Primitive("int")}, Return: Primitive("str")}
	b := Func{Params: []Type{Primitive("int")}, Return: Primitive("str")}
	c := Func{Params: []Type{Primitive("str")}, Return: Primitive("str")}
	d := Func{Params: []Type{Primitive("int")}, Return: Primitive("int")}
	assert.True(t, a.Equal(b))
	assert.False(t, a.Equal(c))
	assert.False(t, a.Equal(d))
}

func TestType_Func_Arity(t *testing.T) {
	f := Func{Params: []Type{Primitive("int"), Primitive("str")}, Return: Primitive("bool")}
	assert.Equal(t, 2, f.Arity())
}

func TestType_Result_String(t *testing.T) {
	r := Result{Ok: Primitive("int"), Err: Primitive("str")}
	assert.Equal(t, "Result[int, str]", r.String())
}

func TestType_Result_Equal(t *testing.T) {
	a := Result{Ok: Primitive("int"), Err: Primitive("str")}
	b := Result{Ok: Primitive("int"), Err: Primitive("str")}
	c := Result{Ok: Primitive("str"), Err: Primitive("str")}
	assert.True(t, a.Equal(b))
	assert.False(t, a.Equal(c))
}

func TestType_Optional_String(t *testing.T) {
	o := Optional{Primitive("int")}
	assert.Equal(t, "int?", o.String())
}

func TestType_Optional_Equal(t *testing.T) {
	a := Optional{Primitive("int")}
	b := Optional{Primitive("int")}
	c := Optional{Primitive("str")}
	assert.True(t, a.Equal(b))
	assert.False(t, a.Equal(c))
}
