package parser

import (
	"testing"

	"github.com/jiejie-dev/funny/v2/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_TestBlock(t *testing.T) {
	src := `test "hello":
    assert(true)
`
	p := New(src, "t.fn")
	prog, err := p.Parse()
	require.NoError(t, err)
	require.Len(t, prog.Stmts, 1)
	tb, ok := prog.Stmts[0].(*ast.TestBlock)
	require.True(t, ok)
	assert.Equal(t, "hello", tb.Name)
	require.NotNil(t, tb.Body)
}
