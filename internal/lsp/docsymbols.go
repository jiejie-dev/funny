package lsp

import (
	"fmt"
	"strings"

	"github.com/jiejie-dev/funny/internal/ast"
	"github.com/jiejie-dev/funny/internal/lexer"
)

// documentSymbols builds the outline for this document: top-level `fn` and
// `struct` declarations, plus `plan` blocks with their `step`s nested
// underneath (a lightweight stand-in for the plan-as-graph visualization
// called for in the language design doc — a generic LSP client can already
// render this as a tree from documentSymbol; a bespoke step-graph view
// would need a custom protocol extension, which is out of scope here).
// Declarations spliced in from imported files are excluded: they belong to
// the outline of their own file, not this one.
func (d *document) documentSymbols() []DocumentSymbol {
	if d.prog == nil {
		return nil
	}
	toks := tokenize(d.text, d.path)
	lastLine := len(strings.Split(d.text, "\n")) - 1

	var out []DocumentSymbol
	stmts := d.prog.Stmts
	for i, s := range stmts {
		if s.Pos().File != d.path {
			continue // declaration merged in from an imported file
		}
		endLine := lastLine
		for j := i + 1; j < len(stmts); j++ {
			if stmts[j].Pos().File == d.path {
				endLine = max(stmts[j].Pos().Line-1, s.Pos().Line)
				break
			}
		}
		switch n := s.(type) {
		case *ast.FnDecl:
			out = append(out, DocumentSymbol{
				Name:           n.Name,
				Detail:         fnSignature(n),
				Kind:           SKFunction,
				Range:          lineRange(n.Pos().Line, endLine),
				SelectionRange: nameTokenRange(toks, n.Pos().Line, n.Name),
			})
		case *ast.StructDecl:
			out = append(out, DocumentSymbol{
				Name:           n.Name,
				Kind:           SKStruct,
				Range:          lineRange(n.Pos().Line, endLine),
				SelectionRange: nameTokenRange(toks, n.Pos().Line, n.Name),
				Children:       structFieldSymbols(n, toks),
			})
		case *ast.PlanBlock:
			out = append(out, DocumentSymbol{
				Name:           n.Name,
				Detail:         "plan",
				Kind:           SKModule,
				Range:          lineRange(n.Pos().Line, endLine),
				SelectionRange: lineRange(n.Pos().Line, n.Pos().Line),
				Children:       planStepSymbols(n, endLine),
			})
		}
	}
	return out
}

func fnSignature(n *ast.FnDecl) string {
	parts := make([]string, len(n.Params))
	for i, p := range n.Params {
		parts[i] = p.String()
	}
	sig := fmt.Sprintf("(%s)", strings.Join(parts, ", "))
	if n.RetType != "" {
		sig += " -> " + n.RetType
	}
	return sig
}

func structFieldSymbols(n *ast.StructDecl, toks []lexer.Token) []DocumentSymbol {
	out := make([]DocumentSymbol, 0, len(n.Fields))
	for _, f := range n.Fields {
		rng := nameTokenRange(toks, n.Pos().Line, f.Name)
		out = append(out, DocumentSymbol{Name: f.Name, Detail: f.TypeAnn, Kind: SKField, Range: rng, SelectionRange: rng})
	}
	return out
}

func planStepSymbols(n *ast.PlanBlock, endLine int) []DocumentSymbol {
	if n.Body == nil {
		return nil
	}
	var out []DocumentSymbol
	stmts := n.Body.Statements
	for i, s := range stmts {
		step, ok := s.(*ast.Step)
		if !ok {
			continue
		}
		stepEnd := endLine
		for j := i + 1; j < len(stmts); j++ {
			stepEnd = max(stmts[j].Pos().Line-1, step.Pos().Line)
			break
		}
		out = append(out, DocumentSymbol{
			Name:           step.Name,
			Detail:         string(step.Kind),
			Kind:           SKEvent,
			Range:          lineRange(step.Pos().Line, stepEnd),
			SelectionRange: lineRange(step.Pos().Line, step.Pos().Line),
		})
	}
	return out
}

func lineRange(startLine, endLine int) Range {
	if endLine < startLine {
		endLine = startLine
	}
	return Range{Start: Position{Line: startLine}, End: Position{Line: endLine}}
}

// nameTokenRange finds the NAME token matching name on the given source
// line and returns its exact span, falling back to a zero-width range at
// the start of the line if not found (e.g. a param name reused across
// multiple lines in a multi-line signature).
func nameTokenRange(toks []lexer.Token, line int, name string) Range {
	for _, t := range toks {
		if t.Pos.Line == line && t.Kind == lexer.NAME && t.Data == name {
			return nameRange(Position{Line: t.Pos.Line, Character: t.Pos.Col}, len(name))
		}
	}
	return lineRange(line, line)
}
