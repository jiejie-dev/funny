package lsp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiejie-dev/funny/v2/internal/pkgman"
	"github.com/stretchr/testify/require"
)

func TestPkgImportContext_DetectsPrefix(t *testing.T) {
	src := `import "pkg:ma`
	prefix, ok := pkgImportContext(src, Position{Line: 0, Character: len(src)})
	require.True(t, ok)
	require.Equal(t, "ma", prefix)

	_, ok = pkgImportContext(`import "local.fn"`, Position{Line: 0, Character: 18})
	require.False(t, ok)
}

func TestCompletion_PkgImport_SuggestsDeclaredPackages(t *testing.T) {
	root := t.TempDir()
	srcDir := filepath.Join(root, "src")
	require.NoError(t, os.MkdirAll(srcDir, 0o755))
	require.NoError(t, pkgman.SaveManifest(root, &pkgman.Manifest{
		Dependencies: map[string]pkgman.Dependency{
			"math": {Source: "path:vendor/math.fn", Version: "^1.0.0"},
			"util": {Source: "path:vendor/util.fn"},
		},
	}))
	docPath := filepath.Join(srcDir, "main.fn")
	src := `import "pkg:`
	d := analyzeDoc(docPath, src)
	items := d.completion(Position{Line: 0, Character: len(src)})
	require.True(t, hasLabel(items, "math"))
	require.True(t, hasLabel(items, "util"))
	for _, it := range items {
		if it.Label == "math" {
			require.Equal(t, CIKModule, it.Kind)
			require.Contains(t, it.Detail, "path:vendor/math.fn")
		}
	}
}
