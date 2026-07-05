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

// TestRun_ExtendedStdlibBuiltins is an end-to-end regression test for a
// previously severe gap: regex_match/regex_replace/env_get/file_read/
// file_exists/http_get/md5/sha256/b64_encode/b64_decode/jwt_encode/
// jwt_decode/sql_open were implemented in internal/vm/builtins.go and
// documented in docs/language-manual.md, but were missing from both
// internal/types.builtinTypeNames and internal/compiler.builtinNames, so
// every call to them failed to even compile ("undefined function") - only
// Go-level VM tests that hand-built bytecode (bypassing the type checker
// and compiler entirely) ever exercised the implementations.
func TestRun_ExtendedStdlibBuiltins(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	require.NoError(t, os.WriteFile(target, []byte("hello"), 0o644))

	src := `println(regex_match("a.c", "abc"))
println(regex_replace("[0-9]+", "id-42", "#"))
println(len(env_get("HOME")) >= 0)

let fr = file_read("` + filepath.ToSlash(target) + `")
if fr.tag == "err":
    println("unexpected read error:", fr.val)
else:
    println("read ok:", fr.val)

println(file_exists("` + filepath.ToSlash(target) + `"))
println(md5("hello"))
println(sha256("hello"))
let enc = b64_encode("hello")
let dec = b64_decode(enc)
if dec.tag == "ok":
    println("b64 roundtrip:", dec.val)

let tok = jwt_encode("{}", "{}", "secret")
let claims = jwt_decode(tok, "secret")
println("jwt tag:", claims.tag)

println(sql_open(":memory:"))
`
	require.NoError(t, Run([]byte(src), filepath.Join(dir, "main.fn")))
}

// TestRun_FloatComparisonsAndBooleanLogic is an end-to-end regression test
// for a set of pre-existing bytecode-compiler gaps found alongside the
// builtin registration one above: `!=` had no opcode for any type, `<`
// `>` `<=` `>=` `==` only worked for int (not float), and `and`/`or` had
// no opcode at all — every one of these compiled fine under
// FUNNY_INTERPRET=1 (the tree-walking evaluator has no static value-type
// tracking to trip over) but failed with "compile: pickBinaryOp:
// unsupported op ..." under the default bytecode VM.
func TestRun_FloatComparisonsAndBooleanLogic(t *testing.T) {
	src := `println(1.0 == 1.0)
println(1.0 != 2.0)
println(1 != 2)
println(1.5 < 2.5)
println(2.5 > 1.5)
println(1.0 <= 1.0)
println(1.0 >= 1.0)
println(true and false)
println(true or false)
println(sqrt(4.0) < 3.0)
println(len("hello") > 0)
`
	require.NoError(t, Run([]byte(src), "test.fn"))
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
