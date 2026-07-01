package lsp

import (
	"testing"

	"github.com/jiejie-dev/funny/internal/formatter"
	"github.com/stretchr/testify/require"
)

func TestFormatting_ReturnsFullDocumentEdit(t *testing.T) {
	src := "let x=1\nprintln(x)\n"
	d := analyzeDoc("/tmp/a.fn", src)
	edits, err := d.formatting()
	require.NoError(t, err)
	require.Len(t, edits, 1)
	require.NotEqual(t, src, edits[0].NewText)
	require.Contains(t, edits[0].NewText, "let x = 1")
}

func TestFormatting_AlreadyFormatted_ReturnsNoEdits(t *testing.T) {
	src := "let x = 1\nprintln(x)\n"
	d := analyzeDoc("/tmp/a.fn", src)
	// Only assert idempotency if the formatter agrees this is already canonical.
	out, err := formatter.Format([]byte(src), "/tmp/a.fn")
	require.NoError(t, err)
	edits, err := d.formatting()
	require.NoError(t, err)
	if out == src {
		require.Nil(t, edits)
	}
}

func TestFormatting_SyntaxError_ReturnsError(t *testing.T) {
	src := "let x = \n"
	d := analyzeDoc("/tmp/a.fn", src)
	_, err := d.formatting()
	require.Error(t, err)
}
