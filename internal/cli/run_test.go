package cli

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureStdout redirects os.Stdout for the duration of fn (builtins like
// print/println write straight to os.Stdout, not an injectable writer) and
// returns everything written to it.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	orig := os.Stdout
	os.Stdout = w
	fn()
	os.Stdout = orig
	require.NoError(t, w.Close())
	data, err := io.ReadAll(r)
	require.NoError(t, err)
	return string(data)
}

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

// TestRun_ForLoop_VisitsFirstElement is an end-to-end regression test for a
// severe compileFor bug (see internal/compiler/control_test.go for the
// full explanation): the loop index used to be initialized from
// Constants[0] instead of the literal 0, so `for` silently skipped the
// first element of its iterable under the default VM path.
func TestRun_ForLoop_VisitsFirstElement(t *testing.T) {
	dir := t.TempDir()
	out := captureStdout(t, func() {
		src := `let sum = 0
for i in [10, 20, 30]:
    sum = sum + i
println(sum)
`
		require.NoError(t, Run([]byte(src), filepath.Join(dir, "main.fn")))
	})
	assert.Equal(t, "60\n", out)
}

// TestRun_StructFieldArithmeticAndListParams is an end-to-end regression
// test combining three related compiler-level type-tracking fixes found
// while building out the extended stdlib example: struct field access
// used to be unconditionally (and wrongly) typed as a string, `list[T]`
// parameter annotations weren't parsed at all, and builtin return types
// were all opaquely "any" - each broke real, natural code patterns
// (struct-field arithmetic, looping over a `list[T]` parameter with a
// typed comparison, returning a builtin's result from a typed function)
// under the default VM path. See internal/compiler and internal/types
// regression tests for the isolated repros of each.
func TestRun_StructFieldArithmeticAndListParams(t *testing.T) {
	src := `struct Point:
    x: int
    y: int

fn dist(p: Point) -> float:
    return sqrt(to_int(p.x * p.x + p.y * p.y))

println(dist(Point(x: 3, y: 4)))

fn count_positive(xs: list[int]) -> int:
    let c = 0
    for x in xs:
        if x > 0:
            c = c + 1
    return c

println(count_positive([1, -2, 3, -4, 5]))
`
	out := captureStdout(t, func() {
		require.NoError(t, Run([]byte(src), "test.fn"))
	})
	assert.Equal(t, "5\n3\n", out)
}

// TestRun_AppendBuildsListFromEmptyAnnotatedLet is an end-to-end regression
// test for two related gaps found while building the log-audit example:
// (1) `let xs: list[T] = []` always failed type-checking with E2011
// "cannot infer type of empty list", even with an explicit annotation,
// because checkLet type-checked the RHS before ever looking at the
// annotation; and (2) there was no `append` builtin at all, so there was
// no way to grow a list from within a loop (no `lst[i] = x` past the end,
// no `+` on lists) - meaning collecting per-iteration results into a
// list was entirely impossible. See internal/types.checkLet and
// internal/vm/builtins.go's "append" case for the fixes.
func TestRun_AppendBuildsListFromEmptyAnnotatedLet(t *testing.T) {
	src := `let evens: list[int] = []
for n in [1, 2, 3, 4, 5, 6]:
    if n % 2 == 0:
        evens = append(evens, n)

println(len(evens))
for e in evens:
    println(e)
`
	out := captureStdout(t, func() {
		require.NoError(t, Run([]byte(src), "test.fn"))
	})
	assert.Equal(t, "3\n2\n4\n6\n", out)
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
