package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiejie-dev/funny/v2/internal/pkgman"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPkgInstall_AndRunWithPkgImport(t *testing.T) {
	root := t.TempDir()
	vendor := filepath.Join(root, "vendor")
	require.NoError(t, os.MkdirAll(vendor, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(vendor, "math.fn"), []byte(`pub fn add(a: int, b: int) -> int:
    return a + b
`), 0o644))
	require.NoError(t, pkgman.SaveManifest(root, &pkgman.Manifest{
		Dependencies: map[string]pkgman.Dependency{
			"math": {Source: "path:vendor/math.fn"},
		},
	}))

	require.NoError(t, PkgInstall(root, nil))

	mainPath := filepath.Join(root, "main.fn")
	require.NoError(t, os.WriteFile(mainPath, []byte(`import "pkg:math"
println(add(2, 3))
`), 0o644))

	out := captureStdout(t, func() {
		data, err := os.ReadFile(mainPath)
		require.NoError(t, err)
		require.NoError(t, Run(data, mainPath))
	})
	assert.Equal(t, "5\n", out)
}
