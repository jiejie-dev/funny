package pkgman

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FindProjectRoot walks upward from startDir looking for funny.pkg or funny.lock.
func FindProjectRoot(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}
	for {
		if fileExists(filepath.Join(dir, ManifestFile)) || fileExists(filepath.Join(dir, LockFile)) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no %s or %s found above %s", ManifestFile, LockFile, startDir)
		}
		dir = parent
	}
}

// ResolvePkgImport maps `pkg:<name>` to the installed entry file path.
func ResolvePkgImport(importingFileDir, importPath string) (string, error) {
	if !strings.HasPrefix(importPath, "pkg:") {
		return "", fmt.Errorf("not a pkg import")
	}
	name := strings.TrimPrefix(importPath, "pkg:")
	if name == "" {
		return "", fmt.Errorf("empty package name in %q", importPath)
	}
	root, err := FindProjectRoot(importingFileDir)
	if err != nil {
		return "", err
	}
	lock, err := LoadLock(root)
	if err != nil {
		return "", err
	}
	return lock.EntryPath(root, name)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
