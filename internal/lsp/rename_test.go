package lsp

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrepareRename_LocalVariable_ReturnsRangeAndPlaceholder(t *testing.T) {
	src := "let x: int = 1\nprintln(x)\n"
	d := analyzeDoc("/tmp/a.fn", src)
	res, err := d.prepareRename(Position{Line: 1, Character: 9})
	require.NoError(t, err)
	require.Equal(t, "x", res.Placeholder)
}

func TestPrepareRename_Builtin_Errors(t *testing.T) {
	src := "println(1)\n"
	d := analyzeDoc("/tmp/a.fn", src)
	_, err := d.prepareRename(Position{Line: 0, Character: 2})
	require.Error(t, err)
}

func TestRename_LocalVariable_UpdatesAllOccurrences(t *testing.T) {
	src := "fn f() -> int:\n    let x: int = 1\n    return x + x\n"
	d := analyzeDoc("/tmp/a.fn", src)
	edit, err := d.rename(Position{Line: 1, Character: 8}, "count")
	require.NoError(t, err)
	edits := edit.Changes[d.uri]
	require.Len(t, edits, 3)
	for _, e := range edits {
		require.Equal(t, "count", e.NewText)
	}
}

func TestRename_Function_UpdatesDeclarationAndCallSites(t *testing.T) {
	src := "fn add(a: int, b: int) -> int:\n    return a + b\nlet x = add(1, 2)\nlet y = add(3, 4)\n"
	d := analyzeDoc("/tmp/a.fn", src)
	edit, err := d.rename(Position{Line: 0, Character: 4}, "sum")
	require.NoError(t, err)
	edits := edit.Changes[d.uri]
	require.Len(t, edits, 3)
}

func TestRename_InvalidNewName_Errors(t *testing.T) {
	src := "let x: int = 1\nprintln(x)\n"
	d := analyzeDoc("/tmp/a.fn", src)
	_, err := d.rename(Position{Line: 1, Character: 9}, "let")
	require.Error(t, err)
}

func TestRename_UnknownIdentifier_Errors(t *testing.T) {
	src := "println(mystery)\n"
	d := analyzeDoc("/tmp/a.fn", src)
	_, err := d.rename(Position{Line: 0, Character: 10}, "known")
	require.Error(t, err)
}
