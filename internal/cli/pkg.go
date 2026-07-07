package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jiejie-dev/funny/v2/internal/pkgman"
)

// PkgInstall installs dependencies from funny.pkg into .funny/packages/.
func PkgInstall(projectDir string, names []string) error {
	root, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}
	lock, err := pkgman.Install(pkgman.InstallOptions{
		ProjectRoot: root,
		Names:       names,
	})
	if err != nil {
		return err
	}
	printInstalled(lock, names)
	return nil
}

// PkgAdd declares a dependency and installs it.
func PkgAdd(projectDir, name, source, version, entry string) error {
	root, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}
	lock, err := pkgman.Add(pkgman.AddOptions{
		ProjectRoot: root,
		Name:        name,
		Source:      source,
		Version:     version,
		Entry:       entry,
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "added %s to %s\n", name, pkgman.ManifestFile)
	printInstalled(lock, []string{name})
	return nil
}

// PkgUpdate re-fetches dependencies and refreshes funny.lock.
func PkgUpdate(projectDir string, names []string) error {
	root, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}
	lock, changed, err := pkgman.Update(pkgman.UpdateOptions{
		ProjectRoot: root,
		Names:       names,
	})
	if err != nil {
		return err
	}
	if len(changed) == 0 {
		fmt.Println("all packages up to date")
		return nil
	}
	for _, name := range changed {
		pkg := lock.Packages[name]
		ver := pkg.Version
		if ver != "" {
			ver = " " + ver
		}
		fmt.Fprintf(os.Stdout, "updated %s%s -> %s/%s (%s)\n", name, ver, pkg.InstallDir, pkg.Entry, pkg.Checksum)
	}
	return nil
}

// PkgList prints installed packages from funny.lock.
func PkgList(projectDir string) error {
	root, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}
	lock, err := pkgman.LoadLock(root)
	if err != nil {
		return err
	}
	if len(lock.Packages) == 0 {
		fmt.Println("no packages installed")
		return nil
	}
	for name, pkg := range lock.Packages {
		ver := ""
		if pkg.Version != "" {
			ver = pkg.Version + "  "
		}
		fmt.Printf("%s  %s%s/%s  %s\n", name, ver, pkg.InstallDir, pkg.Entry, pkg.Checksum)
	}
	return nil
}

func printInstalled(lock *pkgman.Lockfile, names []string) {
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
		ver := ""
		if pkg.Version != "" {
			ver = " " + pkg.Version
		}
		fmt.Fprintf(os.Stdout, "installed %s%s -> %s/%s (%s)\n", name, ver, pkg.InstallDir, pkg.Entry, pkg.Checksum)
	}
}

// NormalizePkgSource ensures path:/git+/https prefixes.
func NormalizePkgSource(source string) string {
	source = strings.TrimSpace(source)
	if source == "" {
		return source
	}
	if strings.HasPrefix(source, "path:") ||
		strings.HasPrefix(source, "git+") ||
		strings.HasPrefix(source, "http://") ||
		strings.HasPrefix(source, "https://") {
		return source
	}
	return "path:" + source
}
