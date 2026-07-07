package repl

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jiejie-dev/funny/v2/internal/pkgman"
)

// LoadFile reads and evaluates a .funny/.fn file into the session.
func (s *Session) LoadFile(path string) error {
	abs, err := resolvePath(s.workDir, path)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return err
	}
	src := strings.TrimSpace(string(data))
	if src == "" {
		return fmt.Errorf("empty file")
	}
	_, _, err = s.EvalCell(src)
	return err
}

// InstallPackages runs funny.pkg install for the session project root.
func (s *Session) InstallPackages(names []string) (string, error) {
	lock, err := pkgman.Install(pkgman.InstallOptions{
		ProjectRoot: s.workDir,
		Names:       names,
	})
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for name, pkg := range lock.Packages {
		if len(names) > 0 {
			found := false
			for _, n := range names {
				if n == name {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		fmt.Fprintf(&b, "installed %s -> %s/%s\n", name, pkg.InstallDir, pkg.Entry)
	}
	return strings.TrimSpace(b.String()), nil
}

func resolvePath(workDir, path string) (string, error) {
	if filepath.IsAbs(path) {
		return path, nil
	}
	return filepath.Join(workDir, path), nil
}

func defaultLessonsDir(workDir string) string {
	candidates := []string{
		filepath.Join(workDir, "docs"),
		"docs",
		filepath.Join("..", "docs"),
	}
	for _, dir := range candidates {
		if st, err := os.Stat(dir); err == nil && st.IsDir() {
			if entries, _ := os.ReadDir(dir); len(entries) > 0 {
				for _, e := range entries {
					if strings.HasPrefix(e.Name(), "tutorial-") {
						abs, _ := filepath.Abs(dir)
						return abs
					}
				}
			}
		}
	}
	abs, _ := filepath.Abs(filepath.Join(workDir, "docs"))
	return abs
}
