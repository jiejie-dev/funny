package cli

import (
	"fmt"
	"os"
	"path/filepath"

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
		fmt.Fprintf(os.Stdout, "installed %s -> %s/%s (%s)\n", name, pkg.InstallDir, pkg.Entry, pkg.Checksum)
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
		fmt.Printf("%s  %s/%s  %s\n", name, pkg.InstallDir, pkg.Entry, pkg.Checksum)
	}
	return nil
}
