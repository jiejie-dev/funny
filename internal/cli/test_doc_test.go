package cli

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCLI_TestRunner(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "testrunner")
	out := captureStdout(t, func() {
		require.NoError(t, Test(root, false, false))
	})
	require.Contains(t, out, "2 passed")
}

func TestCLI_DocMarkdown(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "docgen")
	out := captureStdout(t, func() {
		require.NoError(t, Doc(root, "markdown", "", false))
	})
	require.Contains(t, out, "Add two integers")
}
