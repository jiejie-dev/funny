package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParser_Empty(t *testing.T) {
	p := New("", "")
	prog, err := p.Parse()
	assert.NoError(t, err)
	assert.Empty(t, prog.Stmts)
}

func TestParser_Stubs(t *testing.T) {
	p := New("let x = 1", "")
	_, err := p.Parse()
	assert.Error(t, err)
}
