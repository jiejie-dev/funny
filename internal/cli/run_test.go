package cli

import (
	"os"
	"path/filepath"
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

func TestRun_ImportUnaliased_CallsPubFuncDirectly(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "math.fn"),
		[]byte("pub fn add(a: int, b: int) -> int:\n    return a + b\n"), 0o644))
	mainPath := filepath.Join(dir, "main.fn")
	src := `import "math.fn"
let r = add(1, 2)
println(r)
`
	require.NoError(t, os.WriteFile(mainPath, []byte(src), 0o644))

	data, err := os.ReadFile(mainPath)
	require.NoError(t, err)
	assert.NoError(t, Run(data, mainPath))
}

func TestRun_ImportAliased_CallsViaNamespace(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "math.fn"),
		[]byte("pub fn add(a: int, b: int) -> int:\n    return a + b\n"), 0o644))
	mainPath := filepath.Join(dir, "main.fn")
	src := `import "math.fn" as m
let r = m.add(1, 2)
println(r)
`
	require.NoError(t, os.WriteFile(mainPath, []byte(src), 0o644))

	data, err := os.ReadFile(mainPath)
	require.NoError(t, err)
	assert.NoError(t, Run(data, mainPath))
}

func TestRun_ImportMissingFile_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.fn")
	src := `import "nope.fn"
println(1)
`
	require.NoError(t, os.WriteFile(mainPath, []byte(src), 0o644))

	data, err := os.ReadFile(mainPath)
	require.NoError(t, err)
	err = Run(data, mainPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "E1102")
}

func TestDisasm_WithImport_IncludesImportedFunction(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "math.fn"),
		[]byte("pub fn add(a: int, b: int) -> int:\n    return a + b\n"), 0o644))
	mainPath := filepath.Join(dir, "main.fn")
	src := `import "math.fn"
let r = add(1, 2)
`
	require.NoError(t, os.WriteFile(mainPath, []byte(src), 0o644))

	data, err := os.ReadFile(mainPath)
	require.NoError(t, err)
	out, err := Disasm(data, mainPath)
	require.NoError(t, err)
	assert.Contains(t, out, "add")
}

func TestDescribe_Plan(t *testing.T) {
	src := `meta:
    name: "demo"
    version: "1.0"

plan "demo":
    step "s1":
        pass
    step "s2":
        pass
`
	out, err := Describe([]byte(src), "test.fn")
	assert.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, "demo")
	assert.Contains(t, s, "s1")
	assert.Contains(t, s, "s2")
}
