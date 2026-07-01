package lsp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDocumentStore_OpenUpdateClose(t *testing.T) {
	s := newDocumentStore()
	d := s.open("file:///tmp/a.fn", "let x = 1\n", 1)
	require.NotNil(t, d)
	got, ok := s.get("file:///tmp/a.fn")
	require.True(t, ok)
	require.Equal(t, "let x = 1\n", got.text)

	s.update("file:///tmp/a.fn", "let x = 2\n", 2)
	got, ok = s.get("file:///tmp/a.fn")
	require.True(t, ok)
	require.Equal(t, "let x = 2\n", got.text)
	require.Equal(t, 2, got.version)

	s.close("file:///tmp/a.fn")
	_, ok = s.get("file:///tmp/a.fn")
	require.False(t, ok)
}

func TestAnalyze_ParseError_ProducesDiagnostic(t *testing.T) {
	d := analyzeDoc("/tmp/a.fn", "let x = \n")
	require.Len(t, d.diagnostics, 1)
	require.Equal(t, SeverityError, d.diagnostics[0].Severity)
}

func TestAnalyze_TypeError_ProducesDiagnosticWithCode(t *testing.T) {
	d := analyzeDoc("/tmp/a.fn", "let x: int = \"str\"\n")
	require.Len(t, d.diagnostics, 1)
	require.Equal(t, "E2010", d.diagnostics[0].Code)
}

func TestAnalyze_ValidProgram_NoDiagnostics(t *testing.T) {
	d := analyzeDoc("/tmp/a.fn", "fn add(a: int, b: int) -> int:\n    return a + b\nlet r = add(1, 2)\nprintln(r)\n")
	require.Empty(t, d.diagnostics)
	require.NotNil(t, d.prog)
	_, ok := d.env.LookupFunc("add")
	require.True(t, ok)
}

func TestAnalyze_MissingImport_ProducesDiagnostic(t *testing.T) {
	d := analyzeDoc("/tmp/a.fn", "import \"does_not_exist.fn\"\n")
	require.Len(t, d.diagnostics, 1)
	require.Equal(t, "E1102", d.diagnostics[0].Code)
	// The unresolved program is still retained for best-effort intelligence.
	require.NotNil(t, d.prog)
}

func TestAnalyze_ErrorInImportedFile_AnchorsDiagnosticInThisDocument(t *testing.T) {
	dir := t.TempDir()
	libPath := filepath.Join(dir, "lib.fn")
	mainPath := filepath.Join(dir, "main.fn")
	require.NoError(t, os.WriteFile(libPath, []byte("pub fn bad():\n    let x: int = \"oops\"\n"), 0o644))
	mainSrc := "import \"lib.fn\"\n"
	require.NoError(t, os.WriteFile(mainPath, []byte(mainSrc), 0o644))

	d := analyzeDoc(mainPath, mainSrc)
	require.Len(t, d.diagnostics, 1)
	require.Contains(t, d.diagnostics[0].Message, "imported module")
}
