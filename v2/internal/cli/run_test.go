package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun_SimpleScript(t *testing.T) {
	src := `let x = 1 + 2
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

func TestRun_BytecodeVM_Basic(t *testing.T) {
	src := `let x = 1 + 2`
	err := Run([]byte(src), "test.fn")
	assert.NoError(t, err)
}

func TestRun_BytecodeVM_If(t *testing.T) {
	src := `let x = 10
if x > 5:
    x = 1
else:
    x = 2
`
	err := Run([]byte(src), "test.fn")
	assert.NoError(t, err)
}

func TestRun_BytecodeVM_While(t *testing.T) {
	src := `let sum = 0
let i = 0
while i < 5:
    sum = sum + i
    i = i + 1
`
	err := Run([]byte(src), "test.fn")
	assert.NoError(t, err)
}

func TestRun_TypeError_Still_Caught(t *testing.T) {
	src := `let x: int = "hello"`
	err := Run([]byte(src), "test.fn")
	assert.Error(t, err)
}

func TestDisasm_Outputs(t *testing.T) {
	src := `let x = 1`
	out, err := Disasm([]byte(src), "test.fn")
	assert.NoError(t, err)
	assert.Contains(t, out, "module test.fn")
	assert.Contains(t, out, "PUSH_INT")
}
