package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun_SimpleScript(t *testing.T) {
	src := `let x = 1 + 2
println("x =", x)
`
	err := Run([]byte(src), "test.fn")
	assert.NoError(t, err)
}

func TestAst_OutputsJSON(t *testing.T) {
	src := `let x = 1`
	out, err := Ast([]byte(src), "test.fn")
	require.NoError(t, err)
	assert.Contains(t, string(out), `"NodePos"`)
	assert.Contains(t, string(out), `"Stmts"`)
}
