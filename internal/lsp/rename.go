package lsp

import (
	"fmt"

	"github.com/jiejie-dev/funny/v2/internal/ast"
	"github.com/jiejie-dev/funny/v2/internal/lexer"
)

// renameTarget resolves the identifier at pos and reports whether it can be
// renamed, its scope, and the token/range that would be replaced.
func (d *document) renameTarget(pos Position) (name string, rng Range, kind symbolKind, declPos ast.Pos, scope *ast.FnDecl, err error) {
	tok, _, ok := tokenAt(d.text, d.path, pos)
	if !ok || tok.Kind != lexer.NAME {
		return "", Range{}, symUnknown, ast.Pos{}, nil, fmt.Errorf("no renameable symbol at this position")
	}
	name = tok.Data
	if isBuiltin(name) {
		return "", Range{}, symUnknown, ast.Pos{}, nil, fmt.Errorf("%q is a builtin and cannot be renamed", name)
	}
	target := ast.Pos{File: d.path, Line: tok.Pos.Line, Col: tok.Pos.Col}
	kind, declPos, scope = d.resolveSymbol(name, target)
	if kind == symUnknown {
		return "", Range{}, symUnknown, ast.Pos{}, nil, fmt.Errorf("cannot resolve %q to a renameable declaration", name)
	}
	rng = nameRange(Position{Line: tok.Pos.Line, Character: tok.Pos.Col}, len(name))
	return name, rng, kind, declPos, scope, nil
}

// prepareRename validates that pos points at a renameable identifier and
// returns its current range/text, letting the client seed its rename UI.
func (d *document) prepareRename(pos Position) (*PrepareRenameResult, error) {
	name, rng, _, _, _, err := d.renameTarget(pos)
	if err != nil {
		return nil, err
	}
	return &PrepareRenameResult{Range: rng, Placeholder: name}, nil
}

// rename computes a WorkspaceEdit that renames every occurrence of the
// identifier at pos (as found by referencesTo — see its doc comment for
// the accepted over-approximation and the same-document-only scope) to
// newName.
//
// Per the language design doc's "auto-update `meta` references" note: meta
// blocks only hold free-form string fields (there's no grammar construct
// tying a meta field's value to a `fn`/`struct` name), so there's nothing
// structurally identifiable to update there; this intentionally does not
// attempt to pattern-match meta field values as symbol references.
func (d *document) rename(pos Position, newName string) (*WorkspaceEdit, error) {
	if !isValidIdentifier(newName) {
		return nil, fmt.Errorf("%q is not a valid identifier", newName)
	}
	name, _, kind, _, scope, err := d.renameTarget(pos)
	if err != nil {
		return nil, err
	}
	positions := referencesTo(d, name, kind, scope)
	if len(positions) == 0 {
		return nil, fmt.Errorf("no occurrences of %q found", name)
	}
	edits := make([]TextEdit, 0, len(positions))
	for _, p := range positions {
		edits = append(edits, TextEdit{Range: nameRange(astPosToLSP(p), len(name)), NewText: newName})
	}
	return &WorkspaceEdit{Changes: map[string][]TextEdit{d.uri: edits}}, nil
}

func isValidIdentifier(s string) bool {
	if s == "" || isBuiltin(s) {
		return false
	}
	for _, kw := range keywordCompletions {
		if s == kw {
			return false
		}
	}
	for i, r := range s {
		if i == 0 && !(r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')) {
			return false
		}
		if i > 0 && !isIdentRune(r) {
			return false
		}
	}
	return true
}
