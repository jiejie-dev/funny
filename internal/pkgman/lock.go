package pkgman

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const LockFile = "funny.lock"

// Lockfile records installed packages with version locks (checksums).
type Lockfile struct {
	Version  int                       `json:"version"`
	Packages map[string]LockedPackage  `json:"packages"`
}

// LockedPackage is one installed dependency entry.
type LockedPackage struct {
	Source     string `json:"source"`
	Version    string `json:"version,omitempty"` // resolved version at install time
	InstallDir string `json:"install_dir"`       // relative to project root
	Entry      string `json:"entry"`             // file name inside install_dir
	Checksum   string `json:"checksum"`          // sha256:...
}

// LoadLock reads funny.lock from projectRoot. Missing file returns empty lock.
func LoadLock(projectRoot string) (*Lockfile, error) {
	path := filepath.Join(projectRoot, LockFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Lockfile{Version: 1, Packages: map[string]LockedPackage{}}, nil
		}
		return nil, fmt.Errorf("read %s: %w", LockFile, err)
	}
	var lf Lockfile
	if err := json.Unmarshal(data, &lf); err != nil {
		return nil, fmt.Errorf("parse %s: %w", LockFile, err)
	}
	if lf.Packages == nil {
		lf.Packages = map[string]LockedPackage{}
	}
	if lf.Version == 0 {
		lf.Version = 1
	}
	return &lf, nil
}

// SaveLock writes funny.lock under projectRoot.
func SaveLock(projectRoot string, lf *Lockfile) error {
	if lf.Version == 0 {
		lf.Version = 1
	}
	if lf.Packages == nil {
		lf.Packages = map[string]LockedPackage{}
	}
	data, err := json.MarshalIndent(lf, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(projectRoot, LockFile), data, 0o644)
}

// EntryPath returns the absolute path to a locked package's entry .fn file.
func (lf *Lockfile) EntryPath(projectRoot, name string) (string, error) {
	pkg, ok := lf.Packages[name]
	if !ok {
		return "", fmt.Errorf("package %q not installed (run funny pkg install)", name)
	}
	return filepath.Join(projectRoot, pkg.InstallDir, pkg.Entry), nil
}
