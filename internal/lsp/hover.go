package lsp

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jiejie-dev/funny/v2/internal/ast"
	"github.com/jiejie-dev/funny/v2/internal/lexer"
	"github.com/jiejie-dev/funny/v2/internal/types"
)

var keywordDocs = map[string]string{
	"fn": "Declares a function.", "struct": "Declares a struct type.",
	"let": "Declares a local variable.", "if": "Conditional branch.",
	"elif": "Else-if branch.", "else": "Else branch.",
	"for": "Iterates over a list.", "in": "Used in `for x in xs:`.",
	"while": "Loop while a condition holds.", "match": "Pattern-matches an expression.",
	"return": "Returns from the current function.", "break": "Exits the nearest loop.",
	"continue": "Skips to the next loop iteration.", "import": "Imports declarations from another file.",
	"as": "Aliases an import.", "pub": "Marks a declaration as importable from other files.",
	"meta": "Declares agent metadata for this skill file.", "plan": "Declares an executable step plan.",
	"step": "Declares one step within a `plan` block.", "test": "Declares a unit test (run with `funny test`).", "true": "Boolean literal.",
	"false": "Boolean literal.", "nil": "The absence of a value.",
	"and": "Logical AND.", "or": "Logical OR.", "not": "Logical NOT.",
}

// hover computes the hover contents for the identifier at pos, or nil if
// there is nothing to show.
func (d *document) hover(pos Position) *Hover {
	tok, _, ok := tokenAt(d.text, d.path, pos)
	if !ok || tok.Kind != lexer.NAME {
		if ok && tok.Kind.IsKeyword() {
			if doc, has := keywordDocs[string(tok.Kind)]; has {
				rng := nameRange(Position{Line: tok.Pos.Line, Character: tok.Pos.Col}, len(tok.Data))
				return &Hover{Contents: MarkupContent{Kind: "markdown", Value: fmt.Sprintf("**%s** (keyword)\n\n%s", tok.Data, doc)}, Range: &rng}
			}
		}
		return nil
	}
	name := tok.Data
	rng := nameRange(Position{Line: tok.Pos.Line, Character: tok.Pos.Col}, len(name))
	target := ast.Pos{File: d.path, Line: tok.Pos.Line, Col: tok.Pos.Col}

	if d.prog != nil {
		locals := localsAt(d.prog, target)
		if sym, ok := lookupLocal(locals, name); ok {
			kindLabel := map[string]string{"param": "parameter", "let": "local variable", "for": "loop variable"}[sym.Kind]
			typ := sym.TypeStr
			if typ == "" {
				typ = "(inferred)"
			}
			md := fmt.Sprintf("```funny\n%s: %s\n```\n%s", name, typ, kindLabel)
			return &Hover{Contents: MarkupContent{Kind: "markdown", Value: md}, Range: &rng}
		}
	}
	if d.env != nil {
		if fn, ok := d.env.LookupFunc(name); ok {
			md := fmt.Sprintf("```funny\nfn %s%s\n```\nfunction", name, fn.String())
			return &Hover{Contents: MarkupContent{Kind: "markdown", Value: md}, Range: &rng}
		}
		if s, ok := d.env.LookupStruct(name); ok {
			md := fmt.Sprintf("```funny\nstruct %s:\n%s```", name, structFieldsBlock(s))
			return &Hover{Contents: MarkupContent{Kind: "markdown", Value: md}, Range: &rng}
		}
		if t, ok := d.env.LookupVar(name); ok {
			md := fmt.Sprintf("```funny\n%s: %s\n```\nvariable", name, t.String())
			return &Hover{Contents: MarkupContent{Kind: "markdown", Value: md}, Range: &rng}
		}
	}
	for _, b := range types.BuiltinNames() {
		if b == name {
			return &Hover{Contents: MarkupContent{Kind: "markdown", Value: fmt.Sprintf("```funny\n%s(...)\n```\nbuiltin function", name)}, Range: &rng}
		}
	}
	return nil
}

func structFieldsBlock(s types.Struct) string {
	names := s.FieldNames()
	sort.Strings(names)
	var sb strings.Builder
	for _, name := range names {
		t, _ := s.Field(name)
		sb.WriteString("    " + name + ": " + t.String() + "\n")
	}
	return sb.String()
}
