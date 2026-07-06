package pkgman

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Manifest is the project dependency file (funny.pkg).
type Manifest struct {
	Name         string                  `json:"name,omitempty"`
	Dependencies map[string]Dependency   `json:"dependencies"`
}

// Dependency describes one package to install.
type Dependency struct {
	// Source is one of:
	//   path:<relative-or-absolute>  — copy a local file or directory tree
	//   https://...                  — download a single .fn file
	//   git+<url>[@ref]              — shallow clone (optional @tag/branch/commit)
	Source string `json:"source"`
	// Entry is the main .fn file inside the installed tree (default: <name>.fn).
	Entry string `json:"entry,omitempty"`
}

// LoadManifest reads funny.pkg from projectRoot.
func LoadManifest(projectRoot string) (*Manifest, error) {
	data, err := os.ReadFile(filepath.Join(projectRoot, ManifestFile))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", ManifestFile, err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse %s: %w", ManifestFile, err)
	}
	if len(m.Dependencies) == 0 {
		return nil, fmt.Errorf("%s: no dependencies declared", ManifestFile)
	}
	return &m, nil
}

// SaveManifest writes funny.pkg (used by tests and init helpers).
func SaveManifest(projectRoot string, m *Manifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(projectRoot, ManifestFile), data, 0o644)
}

const ManifestFile = "funny.pkg"
