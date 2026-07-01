package ast

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPos_String(t *testing.T) {
	p := Pos{File: "a.fn", Line: 2, Col: 3}
	assert.Equal(t, "a.fn:3:4", p.String())
}

func TestPos_String_ZeroIndex(t *testing.T) {
	p := Pos{File: "a.fn"}
	assert.Equal(t, "a.fn:1:1", p.String())
}
