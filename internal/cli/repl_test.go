package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jiejie-dev/funny/v2/internal/repl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepl_LetAndPrintExpr(t *testing.T) {
	in := strings.NewReader("let x = 10\nx + 5\n:quit\n")
	var buf bytes.Buffer
	require.NoError(t, repl.Run(".", in, &buf))
	out := buf.String()
	assert.Contains(t, out, "15")
}

func TestRepl_MultiLineBlock(t *testing.T) {
	in := strings.NewReader("if true:\n    99\n:quit\n")
	var buf bytes.Buffer
	require.NoError(t, repl.Run(".", in, &buf))
	assert.Contains(t, buf.String(), "99")
}
