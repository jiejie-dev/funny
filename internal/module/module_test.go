package module

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jiejie-dev/funny/internal/ast"
	"github.com/jiejie-dev/funny/internal/parser"
)

// writeFiles creates each file (path -> contents) under a fresh temp dir and
// returns the absolute path to dir.
func writeFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, contents := range files {
		full := filepath.Join(dir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(contents), 0o644))
	}
	return dir
}

func parseFile(t *testing.T, path string) *ast.Program {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	p := parser.New(string(data), path)
	prog, err := p.Parse()
	require.NoError(t, err)
	return prog
}

func fnNames(prog *ast.Program) []string {
	var names []string
	for _, s := range prog.Stmts {
		if fn, ok := s.(*ast.FnDecl); ok {
			names = append(names, fn.Name)
		}
	}
	return names
}

func TestResolve_NoImports_ReturnsSameProgram(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"main.fn": "let x = 1\nprintln(x)\n",
	})
	mainPath := filepath.Join(dir, "main.fn")
	prog := parseFile(t, mainPath)

	out, err := Resolve(prog, mainPath)
	require.NoError(t, err)
	assert.Same(t, prog, out)
}

func TestResolve_UnaliasedImport_MergesPubFuncUnderBareName(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"math.fn": "pub fn add(a: int, b: int) -> int:\n    return a + b\n",
		"main.fn": "import \"math.fn\"\nlet r = add(1, 2)\nprintln(r)\n",
	})
	mainPath := filepath.Join(dir, "main.fn")
	prog := parseFile(t, mainPath)

	out, err := Resolve(prog, mainPath)
	require.NoError(t, err)
	assert.Contains(t, fnNames(out), "add")
}

func TestResolve_UnaliasedImport_DoesNotExposePrivateHelper(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"math.fn": "fn helper(x: int) -> int:\n    return x * 2\n\npub fn add(a: int, b: int) -> int:\n    return helper(a) + b\n",
		"main.fn": "import \"math.fn\"\nlet r = add(1, 2)\nprintln(r)\n",
	})
	mainPath := filepath.Join(dir, "main.fn")
	prog := parseFile(t, mainPath)

	out, err := Resolve(prog, mainPath)
	require.NoError(t, err)
	names := fnNames(out)
	assert.Contains(t, names, "add")
	assert.NotContains(t, names, "helper")
	found := false
	for _, n := range names {
		if n != "add" {
			found = true
		}
	}
	assert.True(t, found, "expected a hygienically-renamed private helper to still be present")
}

func TestResolve_AliasedImport_RewritesFieldCallToBareName(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"math.fn": "pub fn add(a: int, b: int) -> int:\n    return a + b\n",
		"main.fn": "import \"math.fn\" as m\nlet r = m.add(1, 2)\nprintln(r)\n",
	})
	mainPath := filepath.Join(dir, "main.fn")
	prog := parseFile(t, mainPath)

	out, err := Resolve(prog, mainPath)
	require.NoError(t, err)

	// The last statement (println) must now call a bare `add`, not `m.add`.
	var call *ast.CallExpr
	for _, s := range out.Stmts {
		if let, ok := s.(*ast.LetStmt); ok {
			if c, ok := let.Value.(*ast.CallExpr); ok {
				call = c
			}
		}
	}
	require.NotNil(t, call)
	varExpr, ok := call.Func.(*ast.VariableExpr)
	require.True(t, ok, "expected call target to be rewritten to a bare VariableExpr, got %T", call.Func)
	assert.Equal(t, "add", varExpr.Name)
}

func TestResolve_AliasedImport_PrivateFunctionNotAccessible(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"math.fn": "fn secret() -> int:\n    return 42\n\npub fn ok() -> int:\n    return secret()\n",
		"main.fn": "import \"math.fn\" as m\nprintln(m.secret())\n",
	})
	mainPath := filepath.Join(dir, "main.fn")
	prog := parseFile(t, mainPath)

	_, err := Resolve(prog, mainPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "E1105")
}

func TestResolve_AliasedImport_UnknownFunctionErrors(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"math.fn": "pub fn add(a: int, b: int) -> int:\n    return a + b\n",
		"main.fn": "import \"math.fn\" as m\nprintln(m.missing())\n",
	})
	mainPath := filepath.Join(dir, "main.fn")
	prog := parseFile(t, mainPath)

	_, err := Resolve(prog, mainPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "E1105")
}

func TestResolve_CircularImport_Errors(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"a.fn":    "import \"b.fn\"\npub fn from_a() -> int:\n    return 1\n",
		"b.fn":    "import \"a.fn\"\npub fn from_b() -> int:\n    return 2\n",
		"main.fn": "import \"a.fn\"\nprintln(from_a())\n",
	})
	mainPath := filepath.Join(dir, "main.fn")
	prog := parseFile(t, mainPath)

	_, err := Resolve(prog, mainPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "E1101")
	assert.Contains(t, err.Error(), "circular import")
}

func TestResolve_DuplicateSymbolAcrossModules_Errors(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"x.fn":    "pub fn add(a: int, b: int) -> int:\n    return a + b\n",
		"y.fn":    "pub fn add(a: int, b: int) -> int:\n    return a - b\n",
		"main.fn": "import \"x.fn\"\nimport \"y.fn\"\nprintln(add(1, 2))\n",
	})
	mainPath := filepath.Join(dir, "main.fn")
	prog := parseFile(t, mainPath)

	_, err := Resolve(prog, mainPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "E1104")
}

func TestResolve_DuplicateSymbolBetweenMainAndImport_Errors(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"x.fn":    "pub fn add(a: int, b: int) -> int:\n    return a + b\n",
		"main.fn": "import \"x.fn\"\nfn add(a: int, b: int) -> int:\n    return a * b\nprintln(add(1, 2))\n",
	})
	mainPath := filepath.Join(dir, "main.fn")
	prog := parseFile(t, mainPath)

	_, err := Resolve(prog, mainPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "E1104")
}

func TestResolve_DiamondDependency_MergedOnce(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"base.fn": "pub fn base_val() -> int:\n    return 1\n",
		"a.fn":    "import \"base.fn\"\npub fn from_a() -> int:\n    return base_val() + 1\n",
		"b.fn":    "import \"base.fn\"\npub fn from_b() -> int:\n    return base_val() + 2\n",
		"main.fn": "import \"a.fn\"\nimport \"b.fn\"\nprintln(from_a() + from_b())\n",
	})
	mainPath := filepath.Join(dir, "main.fn")
	prog := parseFile(t, mainPath)

	out, err := Resolve(prog, mainPath)
	require.NoError(t, err)
	count := 0
	for _, n := range fnNames(out) {
		if n == "base_val" {
			count++
		}
	}
	assert.Equal(t, 1, count, "base_val should be merged exactly once despite being imported via two paths")
}

func TestResolve_ImportedStruct_MergedUnderBareName(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"shapes.fn": "pub struct Point:\n    x: int\n    y: int\n",
		"main.fn":   "import \"shapes.fn\" as s\nlet p = Point(x: 1, y: 2)\nprintln(p.x)\n",
	})
	mainPath := filepath.Join(dir, "main.fn")
	prog := parseFile(t, mainPath)

	out, err := Resolve(prog, mainPath)
	require.NoError(t, err)
	found := false
	for _, s := range out.Stmts {
		if sd, ok := s.(*ast.StructDecl); ok && sd.Name == "Point" {
			found = true
		}
	}
	assert.True(t, found)
}

func TestResolve_MissingFile_Errors(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"main.fn": "import \"does_not_exist.fn\"\nprintln(1)\n",
	})
	mainPath := filepath.Join(dir, "main.fn")
	prog := parseFile(t, mainPath)

	_, err := Resolve(prog, mainPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "E1102")
}

func TestResolve_NestedImportAliasCall_Rewritten(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"trig.fn": "pub fn sin_ish(x: int) -> int:\n    return x\n",
		"math.fn": "import \"trig.fn\" as t\npub fn compute(x: int) -> int:\n    return t.sin_ish(x) + 1\n",
		"main.fn": "import \"math.fn\" as m\nprintln(m.compute(5))\n",
	})
	mainPath := filepath.Join(dir, "main.fn")
	prog := parseFile(t, mainPath)

	out, err := Resolve(prog, mainPath)
	require.NoError(t, err)
	names := fnNames(out)
	assert.Contains(t, names, "compute")
	assert.Contains(t, names, "sin_ish")
}
