package pkgman

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const packagesDir = ".funny/packages"

// InstallOptions configures dependency installation.
type InstallOptions struct {
	ProjectRoot string
	// Names limits installation to these dependency keys (empty = all).
	Names []string
}

// Install reads funny.pkg and installs dependencies into .funny/packages/,
// updating funny.lock.
func Install(opts InstallOptions) (*Lockfile, error) {
	root, err := filepath.Abs(opts.ProjectRoot)
	if err != nil {
		return nil, err
	}
	manifest, err := LoadManifest(root)
	if err != nil {
		return nil, err
	}
	lock, err := LoadLock(root)
	if err != nil {
		return nil, err
	}

	names := opts.Names
	if len(names) == 0 {
		for name := range manifest.Dependencies {
			names = append(names, name)
		}
	}

	for _, name := range names {
		dep, ok := manifest.Dependencies[name]
		if !ok {
			return nil, fmt.Errorf("dependency %q not declared in %s", name, ManifestFile)
		}
		entry := dep.Entry
		if entry == "" {
			entry = name + ".fn"
		}
		installRel := filepath.Join(packagesDir, name)
		installAbs := filepath.Join(root, installRel)
		if err := os.RemoveAll(installAbs); err != nil {
			return nil, fmt.Errorf("remove %s: %w", installAbs, err)
		}
		if err := os.MkdirAll(filepath.Dir(installAbs), 0o755); err != nil {
			return nil, err
		}
		if err := fetchSource(root, dep.Source, installAbs, entry); err != nil {
			return nil, fmt.Errorf("install %q: %w", name, err)
		}
		entryPath := filepath.Join(installAbs, entry)
		if _, err := os.Stat(entryPath); err != nil {
			return nil, fmt.Errorf("install %q: entry file %q missing after fetch", name, entry)
		}
		sum, err := fileSHA256(entryPath)
		if err != nil {
			return nil, err
		}
		lock.Packages[name] = LockedPackage{
			Source:     dep.Source,
			InstallDir: installRel,
			Entry:      entry,
			Checksum:   "sha256:" + sum,
		}
	}
	if err := SaveLock(root, lock); err != nil {
		return nil, err
	}
	return lock, nil
}

func fetchSource(projectRoot, source, destDir, entry string) error {
	switch {
	case strings.HasPrefix(source, "path:"):
		src := strings.TrimPrefix(source, "path:")
		if !filepath.IsAbs(src) {
			src = filepath.Join(projectRoot, src)
		}
		src = filepath.Clean(src)
		info, err := os.Stat(src)
		if err != nil {
			return err
		}
		if info.IsDir() {
			return copyTree(src, destDir)
		}
		if err := os.MkdirAll(destDir, 0o755); err != nil {
			return err
		}
		return copyFile(src, filepath.Join(destDir, entry))
	case strings.HasPrefix(source, "git+"):
		return fetchGit(strings.TrimPrefix(source, "git+"), destDir)
	case strings.HasPrefix(source, "http://"), strings.HasPrefix(source, "https://"):
		if err := os.MkdirAll(destDir, 0o755); err != nil {
			return err
		}
		return downloadURL(source, filepath.Join(destDir, entry))
	default:
		return fmt.Errorf("unsupported source %q (use path:, git+, or https://)", source)
	}
}

func fetchGit(spec, destDir string) error {
	url, ref := splitGitURL(spec)
	args := []string{"clone", "--depth", "1"}
	if ref != "" {
		args = append(args, "--branch", ref)
	}
	args = append(args, url, destDir)
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone: %w", err)
	}
	return nil
}

// splitGitURL separates git+<url>@ref into clone URL and optional ref.
func splitGitURL(spec string) (url, ref string) {
	if i := strings.LastIndex(spec, "@"); i > 0 {
		candidate := spec[i+1:]
		if candidate != "" && !strings.ContainsAny(candidate, "/:") {
			return spec[:i], candidate
		}
	}
	return spec, ""
}

func downloadURL(url, dest string) error {
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(path, target)
	})
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
