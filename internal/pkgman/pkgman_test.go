package pkgman

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstall_PathSource(t *testing.T) {
	root := t.TempDir()
	srcDir := filepath.Join(root, "vendor")
	require.NoError(t, os.MkdirAll(srcDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "math.fn"), []byte(`pub fn add(a: int, b: int) -> int:
    return a + b
`), 0o644))

	manifest := &Manifest{
		Name: "demo",
		Dependencies: map[string]Dependency{
			"math": {Source: "path:vendor/math.fn"},
		},
	}
	require.NoError(t, SaveManifest(root, manifest))

	lock, err := Install(InstallOptions{ProjectRoot: root})
	require.NoError(t, err)
	require.Contains(t, lock.Packages, "math")
	assert.True(t, len(lock.Packages["math"].Checksum) > 10)

	entry := filepath.Join(root, lock.Packages["math"].InstallDir, lock.Packages["math"].Entry)
	data, err := os.ReadFile(entry)
	require.NoError(t, err)
	assert.Contains(t, string(data), "pub fn add")
}

func TestResolvePkgImport(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "src")
	require.NoError(t, os.MkdirAll(sub, 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "vendor"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "vendor", "math.fn"), []byte("pub fn add(a: int, b: int) -> int:\n    return a + b\n"), 0o644))
	require.NoError(t, SaveManifest(root, &Manifest{
		Dependencies: map[string]Dependency{
			"math": {Source: "path:vendor/math.fn"},
		},
	}))
	_, err := Install(InstallOptions{ProjectRoot: root})
	require.NoError(t, err)

	path, err := ResolvePkgImport(sub, "pkg:math")
	require.NoError(t, err)
	assert.FileExists(t, path)
}

func TestSplitGitURL(t *testing.T) {
	url, ref := splitGitURL("https://github.com/org/repo@v1.2.0")
	assert.Equal(t, "https://github.com/org/repo", url)
	assert.Equal(t, "v1.2.0", ref)

	url, ref = splitGitURL("https://github.com/org/repo")
	assert.Equal(t, "https://github.com/org/repo", url)
	assert.Empty(t, ref)
}

func TestFindProjectRoot(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "a", "b")
	require.NoError(t, os.MkdirAll(nested, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, LockFile), []byte(`{"version":1,"packages":{}}`), 0o644))

	found, err := FindProjectRoot(nested)
	require.NoError(t, err)
	assert.Equal(t, root, found)
}

func TestAdd_CreatesManifestAndInstalls(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "vendor")
	require.NoError(t, os.MkdirAll(src, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(src, "math.fn"), []byte("pub fn add(a: int, b: int) -> int:\n    return a + b\n"), 0o644))

	lock, err := Add(AddOptions{
		ProjectRoot: root,
		Name:        "math",
		Source:      "path:vendor/math.fn",
		Version:     ">=0.0.0",
	})
	require.NoError(t, err)
	require.Contains(t, lock.Packages, "math")
	assert.Equal(t, "0.0.0", lock.Packages["math"].Version)

	m, err := LoadManifest(root)
	require.NoError(t, err)
	assert.Equal(t, "path:vendor/math.fn", m.Dependencies["math"].Source)
	assert.Equal(t, ">=0.0.0", m.Dependencies["math"].Version)
}

func TestInstall_VersionConstraintFails(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "vendor"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "vendor", "math.fn"), []byte("pub fn add(a: int, b: int) -> int:\n    return a + b\n"), 0o644))
	require.NoError(t, SaveManifest(root, &Manifest{
		Dependencies: map[string]Dependency{
			"math": {Source: "path:vendor/math.fn", Version: "9.9.9"},
		},
	}))

	_, err := Install(InstallOptions{ProjectRoot: root})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not satisfy")
}

func TestUpdate_DetectsChecksumChange(t *testing.T) {
	root := t.TempDir()
	vendor := filepath.Join(root, "vendor")
	require.NoError(t, os.MkdirAll(vendor, 0o755))
	entry := filepath.Join(vendor, "math.fn")
	require.NoError(t, os.WriteFile(entry, []byte("pub fn add(a: int, b: int) -> int:\n    return a + b\n"), 0o644))
	require.NoError(t, SaveManifest(root, &Manifest{
		Dependencies: map[string]Dependency{
			"math": {Source: "path:vendor/math.fn"},
		},
	}))
	_, err := Install(InstallOptions{ProjectRoot: root})
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(entry, []byte("pub fn add(a: int, b: int) -> int:\n    return a + b + 1\n"), 0o644))
	lock, changed, err := Update(UpdateOptions{ProjectRoot: root})
	require.NoError(t, err)
	assert.Equal(t, []string{"math"}, changed)
	assert.Contains(t, lock.Packages["math"].Checksum, "sha256:")
}

func TestListDeclared(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, SaveManifest(root, &Manifest{
		Dependencies: map[string]Dependency{
			"z": {Source: "path:a"},
			"a": {Source: "path:b"},
		},
	}))
	names, err := ListDeclared(root)
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "z"}, names)
}
