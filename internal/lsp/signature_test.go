package lsp

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSignatureHelp_FirstArg(t *testing.T) {
	src := "fn add(a: int, b: int) -> int:\n    return a + b\nlet r = add(1, 2)\n"
	d := analyzeDoc("/tmp/a.fn", src)
	// cursor right after "add(" on line 2
	help := d.signatureHelp(Position{Line: 2, Character: 12})
	require.NotNil(t, help)
	require.Len(t, help.Signatures, 1)
	require.Contains(t, help.Signatures[0].Label, "add")
	require.Equal(t, 0, help.ActiveParameter)
}

func TestSignatureHelp_SecondArg(t *testing.T) {
	src := "fn add(a: int, b: int) -> int:\n    return a + b\nlet r = add(1, 2)\n"
	d := analyzeDoc("/tmp/a.fn", src)
	// cursor right after the comma, before "2"
	help := d.signatureHelp(Position{Line: 2, Character: 16})
	require.NotNil(t, help)
	require.Equal(t, 1, help.ActiveParameter)
}

func TestSignatureHelp_Builtin(t *testing.T) {
	src := "println(1, 2)\n"
	d := analyzeDoc("/tmp/a.fn", src)
	help := d.signatureHelp(Position{Line: 0, Character: 9})
	require.NotNil(t, help)
	require.Contains(t, help.Signatures[0].Label, "println")
}

func TestSignatureHelp_NotInsideCall_ReturnsNil(t *testing.T) {
	src := "let x = 1\n"
	d := analyzeDoc("/tmp/a.fn", src)
	help := d.signatureHelp(Position{Line: 0, Character: 5})
	require.Nil(t, help)
}

func TestEnclosingCall_IgnoresPlainGrouping(t *testing.T) {
	toks := tokenize("let x = (1 + 2)\n", "t.fn")
	_, _, ok := enclosingCall(toks, Position{Line: 0, Character: 11})
	require.False(t, ok, "a parenthesized grouping is not a call")
}
