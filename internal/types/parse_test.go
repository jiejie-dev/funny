package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseType_Primitive(t *testing.T) {
	cases := []struct {
		in   string
		want Type
	}{
		{"int", Primitive("int")},
		{"str", Primitive("str")},
		{"bool", Primitive("bool")},
		{"float", Primitive("float")},
	}
	for _, c := range cases {
		got, err := ParseType(c.in)
		assert.NoError(t, err, c.in)
		assert.True(t, got.Equal(c.want), c.in)
	}
}

func TestParseType_List(t *testing.T) {
	got, err := ParseType("list[int]")
	assert.NoError(t, err)
	want := List{Primitive("int")}
	assert.True(t, got.Equal(want))
}

func TestParseType_Map(t *testing.T) {
	got, err := ParseType("map[str, int]")
	assert.NoError(t, err)
	want := Map{Primitive("str"), Primitive("int")}
	assert.True(t, got.Equal(want))
}

func TestParseType_Optional(t *testing.T) {
	got, err := ParseType("int?")
	assert.NoError(t, err)
	want := Optional{Primitive("int")}
	assert.True(t, got.Equal(want))
}

func TestParseType_Result(t *testing.T) {
	got, err := ParseType("Result[int, str]")
	assert.NoError(t, err)
	want := Result{Ok: Primitive("int"), Err: Primitive("str")}
	assert.True(t, got.Equal(want))
}

func TestParseType_Nested(t *testing.T) {
	got, err := ParseType("list[map[str, int?]]")
	assert.NoError(t, err)
	want := List{Map{Primitive("str"), Optional{Primitive("int")}}}
	assert.True(t, got.Equal(want))
}

func TestParseType_Func(t *testing.T) {
	got, err := ParseType("(int, str) -> bool")
	assert.NoError(t, err)
	want := Func{Params: []Type{Primitive("int"), Primitive("str")}, Return: Primitive("bool")}
	assert.True(t, got.Equal(want))
}

func TestParseType_Invalid(t *testing.T) {
	_, err := ParseType("list[")
	assert.Error(t, err)
}
