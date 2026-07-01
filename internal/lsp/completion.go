package lsp

import (
	"sort"
	"strings"

	"github.com/jiejie-dev/funny/internal/ast"
	"github.com/jiejie-dev/funny/internal/types"
)

var keywordCompletions = []string{
	"and", "as", "break", "continue", "elif", "else", "false", "fn",
	"for", "if", "import", "in", "let", "match", "meta", "nil", "not", "or",
	"plan", "pub", "return", "step", "struct", "true", "while",
}

// completion computes completion items for pos. When the cursor immediately
// follows `<expr>.`, only that expression's fields (struct fields, or the
// `tag`/`val` accessors of a Result) are offered — matching the type-aware,
// same-type-only completion behavior called for in the language design
// (docs/superpowers/specs/2026-07-01-funny-v2-ai-native-language-design.md
// §5.7). Otherwise every symbol reachable at pos (locals, functions,
// structs, builtins, keywords) is offered.
func (d *document) completion(pos Position) []CompletionItem {
	if objName, ok := dotContext(d.text, pos); ok {
		if items := d.fieldCompletions(objName, pos); items != nil {
			return items
		}
		// The single most common moment a `.`-triggered completion fires is
		// right after typing the dot, before any field name character —
		// e.g. "let v = p." — which is a syntax error on its own and makes
		// the whole document fail to parse, leaving d.prog/d.env without
		// the information needed to resolve `p`'s type. Re-analyze a
		// throwaway patched copy with a placeholder field name spliced in
		// at the cursor so resolution can still succeed; this never
		// affects the document's real diagnostics.
		if patched, ok := patchTrailingDot(d.text, pos); ok {
			tmp := &document{path: d.path, text: patched}
			tmp.analyze()
			if tmp.prog != nil {
				savedProg, savedEnv := d.prog, d.env
				d.prog, d.env = tmp.prog, tmp.env
				items := d.fieldCompletions(objName, pos)
				d.prog, d.env = savedProg, savedEnv
				return items
			}
		}
		return nil
	}
	return d.generalCompletions(pos)
}

// patchTrailingDot inserts a placeholder identifier right at pos (which
// must immediately follow a `.`), turning e.g. "let v = p." into
// "let v = p.zzzzzz" so it parses.
func patchTrailingDot(src string, pos Position) (string, bool) {
	lines := strings.Split(src, "\n")
	if pos.Line < 0 || pos.Line >= len(lines) {
		return "", false
	}
	runes := []rune(lines[pos.Line])
	if pos.Character < 0 || pos.Character > len(runes) {
		return "", false
	}
	patchedLine := string(runes[:pos.Character]) + "zzzzzz" + string(runes[pos.Character:])
	lines[pos.Line] = patchedLine
	return strings.Join(lines, "\n"), true
}

// dotContext reports the identifier immediately before a trailing `.` at
// pos, e.g. for "point." with the cursor right after the dot it returns
// ("point", true).
func dotContext(src string, pos Position) (string, bool) {
	runes := []rune(lineAt(src, pos.Line))
	i := pos.Character - 1
	if i < 0 || i >= len(runes) || runes[i] != '.' {
		return "", false
	}
	j := i - 1
	end := j + 1
	for j >= 0 && isIdentRune(runes[j]) {
		j--
	}
	name := string(runes[j+1 : end])
	if name == "" {
		return "", false
	}
	return name, true
}

func isIdentRune(r rune) bool {
	return r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

func (d *document) resolveType(name string, pos Position) (types.Type, bool) {
	target := ast.Pos{File: d.path, Line: pos.Line, Col: pos.Character}
	if d.prog != nil {
		if sym, ok := lookupLocal(localsAt(d.prog, target), name); ok && sym.TypeStr != "" {
			if t, err := types.ParseType(sym.TypeStr); err == nil {
				return types.ResolveNamedType(t, d.env), true
			}
		}
	}
	if d.env != nil {
		if t, ok := d.env.LookupVar(name); ok {
			return t, true
		}
		if fn, ok := d.env.LookupFunc(name); ok {
			return fn.Return, true
		}
	}
	return nil, false
}

func (d *document) fieldCompletions(objName string, pos Position) []CompletionItem {
	t, ok := d.resolveType(objName, pos)
	if !ok {
		return nil
	}
	if d.env != nil {
		if s, isStruct := t.(types.Struct); isStruct {
			names := s.FieldNames()
			sort.Strings(names)
			items := make([]CompletionItem, 0, len(names))
			for _, name := range names {
				ft, _ := s.Field(name)
				items = append(items, CompletionItem{Label: name, Kind: CIKField, Detail: ft.String()})
			}
			return items
		}
	}
	if _, isResult := t.(types.Result); isResult {
		return []CompletionItem{
			{Label: "tag", Kind: CIKField, Detail: "str"},
			{Label: "val", Kind: CIKField, Detail: "the Ok value"},
		}
	}
	return nil
}

func (d *document) generalCompletions(pos Position) []CompletionItem {
	var items []CompletionItem
	for _, kw := range keywordCompletions {
		items = append(items, CompletionItem{Label: kw, Kind: CIKKeyword})
	}
	for _, b := range types.BuiltinNames() {
		items = append(items, CompletionItem{Label: b, Kind: CIKFunction, Detail: "builtin"})
	}
	if d.env != nil {
		for name, fn := range d.env.Funcs() {
			items = append(items, CompletionItem{Label: name, Kind: CIKFunction, Detail: fn.String()})
		}
		for name, s := range d.env.Structs() {
			items = append(items, CompletionItem{Label: name, Kind: CIKClass, Detail: s.Name})
		}
	}
	if d.prog != nil {
		target := ast.Pos{File: d.path, Line: pos.Line, Col: pos.Character}
		for _, sym := range localsAt(d.prog, target) {
			detail := sym.TypeStr
			if detail == "" {
				detail = "(inferred)"
			}
			items = append(items, CompletionItem{Label: sym.Name, Kind: CIKVariable, Detail: detail})
		}
	}
	return items
}
