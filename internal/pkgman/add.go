package pkgman

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// AddOptions adds one dependency to funny.pkg and installs it.
type AddOptions struct {
	ProjectRoot string
	Name        string
	Source      string
	Version     string // constraint (optional)
	Entry       string
}

// Add declares a dependency, writes funny.pkg, and installs it.
func Add(opts AddOptions) (*Lockfile, error) {
	root, err := filepath.Abs(opts.ProjectRoot)
	if err != nil {
		return nil, err
	}
	if opts.Name == "" {
		return nil, fmt.Errorf("package name is required")
	}
	if opts.Source == "" {
		return nil, fmt.Errorf("source is required (path:, https://, or git+)")
	}
	manifest, created, err := LoadOrCreateManifest(root)
	if err != nil {
		return nil, err
	}
	if created && manifest.Name == "" {
		manifest.Name = filepath.Base(root)
	}
	if manifest.Dependencies == nil {
		manifest.Dependencies = map[string]Dependency{}
	}
	manifest.Dependencies[opts.Name] = Dependency{
		Source:  opts.Source,
		Version: opts.Version,
		Entry:   opts.Entry,
	}
	if err := SaveManifest(root, manifest); err != nil {
		return nil, err
	}
	return Install(InstallOptions{ProjectRoot: root, Names: []string{opts.Name}})
}

// UpdateOptions refreshes installed packages from funny.pkg sources.
type UpdateOptions struct {
	ProjectRoot string
	Names       []string // empty = all manifest dependencies
}

// Update re-fetches packages and refreshes funny.lock checksums/versions.
func Update(opts UpdateOptions) (*Lockfile, []string, error) {
	root, err := filepath.Abs(opts.ProjectRoot)
	if err != nil {
		return nil, nil, err
	}
	manifest, err := LoadManifest(root)
	if err != nil {
		return nil, nil, err
	}
	oldLock, err := LoadLock(root)
	if err != nil {
		return nil, nil, err
	}
	names := opts.Names
	if len(names) == 0 {
		for name := range manifest.Dependencies {
			names = append(names, name)
		}
	}
	var changed []string
	for _, name := range names {
		if _, ok := manifest.Dependencies[name]; !ok {
			return nil, nil, fmt.Errorf("dependency %q not declared in %s", name, ManifestFile)
		}
		oldSum := ""
		if prev, ok := oldLock.Packages[name]; ok {
			oldSum = prev.Checksum
		}
		lock, err := Install(InstallOptions{ProjectRoot: root, Names: []string{name}})
		if err != nil {
			return nil, nil, err
		}
		if pkg, ok := lock.Packages[name]; ok && pkg.Checksum != oldSum {
			changed = append(changed, name)
		}
		oldLock = lock
	}
	return oldLock, changed, nil
}

// LoadOrCreateManifest reads funny.pkg or returns an empty manifest if missing.
func LoadOrCreateManifest(projectRoot string) (*Manifest, bool, error) {
	path := filepath.Join(projectRoot, ManifestFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Manifest{Dependencies: map[string]Dependency{}}, true, nil
		}
		return nil, false, fmt.Errorf("read %s: %w", ManifestFile, err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, false, fmt.Errorf("parse %s: %w", ManifestFile, err)
	}
	if m.Dependencies == nil {
		m.Dependencies = map[string]Dependency{}
	}
	return &m, false, nil
}

// ListDeclared returns sorted dependency names from funny.pkg (installed or not).
func ListDeclared(projectRoot string) ([]string, error) {
	m, err := LoadManifestAllowEmpty(projectRoot)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(m.Dependencies))
	for name := range m.Dependencies {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

// LoadManifestAllowEmpty reads funny.pkg; missing file yields empty manifest.
func LoadManifestAllowEmpty(projectRoot string) (*Manifest, error) {
	m, _, err := LoadOrCreateManifest(projectRoot)
	return m, err
}
