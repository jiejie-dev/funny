package testrunner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun_MathTests(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "testrunner")
	report, err := Run(Options{Path: root})
	require.NoError(t, err)
	assert.Equal(t, 2, report.Passed)
	assert.Equal(t, 0, report.Failed)
}

func TestRun_FailingAssert(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad_test.fn")
	require.NoError(t, os.WriteFile(path, []byte(`test "fail":
    assert(false)
`), 0o644))
	report, err := Run(Options{Path: path})
	require.NoError(t, err)
	assert.Equal(t, 0, report.Passed)
	assert.Equal(t, 1, report.Failed)
	assert.Contains(t, report.Tests[0].Error, "assertion failed")
}
