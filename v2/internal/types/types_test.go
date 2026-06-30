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
