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
