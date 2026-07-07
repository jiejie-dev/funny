package lsp

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/jiejie-dev/funny/v2/internal/pkgman"
)

// pkgImportContext reports whether the cursor is completing a pkg: import path prefix.
func pkgImportContext(src string, pos Position) (prefix string, ok bool) {
	line := lineAt(src, pos.Line)
	runes := []rune(line)
	col := pos.Character
	if col < 0 {
		return "", false
	}
	if col > len(runes) {
		col = len(runes)
	}
	before := string(runes[:col])
	imp := strings.LastIndex(before, "import")
	if imp < 0 {
		return "", false
	}
	rest := strings.TrimSpace(before[imp+6:])
	if len(rest) == 0 {
		return "", false
	}
	quote := rest[0]
	if quote != '"' && quote != '\'' {
		return "", false
	}
	inner := rest[1:]
	idx := strings.LastIndex(inner, "pkg:")
	if idx < 0 {
		return "", false
	}
	prefix = inner[idx+4:]
	if strings.ContainsAny(prefix, `"'`) {
		return "", false
	}
	return prefix, true
}

func (d *document) pkgCompletions(prefix string) []CompletionItem {
	root, err := pkgman.FindProjectRoot(filepath.Dir(d.path))
	if err != nil {
		return nil
	}
	manifest, err := pkgman.LoadManifestAllowEmpty(root)
	if err != nil {
		return nil
	}
	type candidate struct {
		name   string
		detail string
	}
	var cands []candidate
	for name, dep := range manifest.Dependencies {
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		detail := dep.Source
		if dep.Version != "" {
			detail += " (" + dep.Version + ")"
		}
		cands = append(cands, candidate{name: name, detail: detail})
	}
	lock, _ := pkgman.LoadLock(root)
	for name, pkg := range lock.Packages {
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		found := false
		for _, c := range cands {
			if c.name == name {
				found = true
				break
			}
		}
		if found {
			continue
		}
		detail := pkg.Source
		if pkg.Version != "" {
			detail += " locked@" + pkg.Version
		}
		cands = append(cands, candidate{name: name, detail: detail})
	}
	sort.Slice(cands, func(i, j int) bool { return cands[i].name < cands[j].name })
	items := make([]CompletionItem, 0, len(cands))
	for _, c := range cands {
		items = append(items, CompletionItem{
			Label:  c.name,
			Kind:   CIKModule,
			Detail: c.detail,
		})
	}
	return items
}
