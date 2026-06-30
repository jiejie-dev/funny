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

func TestRun_TypeCheckPasses(t *testing.T) {
	src := `let x: int = 42
let y: int = x + 1
`
	err := Run([]byte(src), "test.fn")
	assert.NoError(t, err)
}

func TestRun_TypeCheckFails(t *testing.T) {
	src := `let x: int = "hello"`
	err := Run([]byte(src), "test.fn")
	assert.Error(t, err)
}
