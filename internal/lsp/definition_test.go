package lsp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// realPath resolves symlinks the same way internal/module does internally,
// so tests comparing against locations produced by the resolver aren't
// tripped up by e.g. macOS's /tmp -> /private/tmp symlink.
func realPath(t *testing.T, p string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(p)
	require.NoError(t, err)
	return resolved
}

func TestDefinition_LocalVariable(t *testing.T) {
	src := "let x: int = 5\nprintln(x)\n"
	d := analyzeDoc("/tmp/a.fn", src)
	loc := d.definition(Position{Line: 1, Character: 9})
	require.NotNil(t, loc)
	require.Equal(t, pathToURI("/tmp/a.fn"), loc.URI)
	require.Equal(t, 0, loc.Range.Start.Line)
}

func TestDefinition_FunctionParam(t *testing.T) {
	src := "fn double(n: int) -> int:\n    return n * 2\n"
	d := analyzeDoc("/tmp/a.fn", src)
	loc := d.definition(Position{Line: 1, Character: 11})
	require.NotNil(t, loc)
	require.Equal(t, 0, loc.Range.Start.Line)
}

func TestDefinition_FunctionDecl(t *testing.T) {
	src := "fn add(a: int, b: int) -> int:\n    return a + b\nlet r = add(1, 2)\n"
	d := analyzeDoc("/tmp/a.fn", src)
	loc := d.definition(Position{Line: 2, Character: 9})
	require.NotNil(t, loc)
	require.Equal(t, 0, loc.Range.Start.Line)
}

func TestDefinition_UnknownIdentifier_ReturnsNil(t *testing.T) {
	src := "println(mystery)\n"
	d := analyzeDoc("/tmp/a.fn", src)
	loc := d.definition(Position{Line: 0, Character: 10})
	require.Nil(t, loc)
}

func TestDefinition_CrossFile_ImportedFunction(t *testing.T) {
	dir := t.TempDir()
	libPath := filepath.Join(dir, "lib.fn")
	mainPath := filepath.Join(dir, "main.fn")
	require.NoError(t, os.WriteFile(libPath, []byte("pub fn square(n: int) -> int:\n    return n * n\n"), 0o644))

	mainSrc := "import \"lib.fn\"\nlet r = square(3)\n"
	require.NoError(t, os.WriteFile(mainPath, []byte(mainSrc), 0o644))

	d := analyzeDoc(mainPath, mainSrc)
	require.Empty(t, d.diagnostics, "expected clean resolution, got: %v", d.diagnostics)

	// "square" call site on line 1
	loc := d.definition(Position{Line: 1, Character: 10})
	require.NotNil(t, loc)
	require.Equal(t, pathToURI(realPath(t, libPath)), loc.URI, "definition should jump into the imported file")
	require.Equal(t, 0, loc.Range.Start.Line)
}
