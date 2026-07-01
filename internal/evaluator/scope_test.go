package evaluator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScope_SetGet(t *testing.T) {
	s := NewScope(nil)
	s.Set("x", 42)
	v, ok := s.Get("x")
	assert.True(t, ok)
	assert.Equal(t, 42, v)
}

func TestScope_NestedLookup(t *testing.T) {
	outer := NewScope(nil)
	outer.Set("a", 1)
	inner := NewScope(outer)
	inner.Set("b", 2)
	v, _ := inner.Get("a")
	assert.Equal(t, 1, v)
	v, _ = inner.Get("b")
	assert.Equal(t, 2, v)
}

func TestScope_Shadowing(t *testing.T) {
	outer := NewScope(nil)
	outer.Set("x", 1)
	inner := NewScope(outer)
	inner.Set("x", 2)
	v, _ := inner.Get("x")
	assert.Equal(t, 2, v)
	v, _ = outer.Get("x")
	assert.Equal(t, 1, v)
}

func TestScope_Assign(t *testing.T) {
	outer := NewScope(nil)
	outer.Set("x", 1)
	inner := NewScope(outer)
	assert.True(t, inner.Assign("x", 99))
	v, _ := outer.Get("x")
	assert.Equal(t, 99, v)
}
