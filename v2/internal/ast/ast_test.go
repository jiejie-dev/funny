package ast

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPos_String(t *testing.T) {
	p := Pos{File: "a.fn", Line: 2, Col: 3}
	s := p.String()
	assert.Contains(t, s, "a.fn")
}