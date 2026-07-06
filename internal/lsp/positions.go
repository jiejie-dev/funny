package lsp

import (
	"strings"

	"github.com/jiejie-dev/funny/v2/internal/ast"
	"github.com/jiejie-dev/funny/v2/internal/errs"
	"github.com/jiejie-dev/funny/v2/internal/lexer"
)

// astPosToLSP converts an ast.Pos (0-indexed line/col, same convention the
// lexer/parser use throughout the compiler) directly into an LSP Position
// (also 0-indexed line/character). Column is treated as a rune count within
// the line; multi-byte non-ASCII content before the target column on the
// same line is not corrected to UTF-16 code units (a known simplification,
// acceptable since funny identifiers/keywords are ASCII).
func astPosToLSP(p ast.Pos) Position {
	return Position{Line: p.Line, Character: p.Col}
}

func errsPosToLSP(p errs.Position) Position {
	return Position{Line: p.Line, Character: p.Col}
}

// pointRange returns a zero-width range at pos, used when no better span is
// known.
func pointRange(pos Position) Range {
	return Range{Start: pos, End: pos}
}

// nameRange returns the range covering an identifier of length n starting
// at pos.
func nameRange(pos Position, n int) Range {
	return Range{Start: pos, End: Position{Line: pos.Line, Character: pos.Character + n}}
}

// lspPosToAST converts an LSP Position back into the ast.Pos convention
// (same File/Line/Col shape used by the lexer/parser).
func lspPosToAST(file string, pos Position) ast.Pos {
	return ast.Pos{File: file, Line: pos.Line, Col: pos.Character}
}

// tokenAt tokenizes src and returns the token whose span contains pos
// (inclusive of the position immediately after the token, so hovering right
// after an identifier still resolves it), along with the raw token list for
// callers that need surrounding context (e.g. signature help).
func tokenAt(src, file string, pos Position) (lexer.Token, []lexer.Token, bool) {
	toks := tokenize(src, file)
	for _, t := range toks {
		if t.Kind == lexer.EOF || t.Kind == lexer.NEWLINE || t.Kind == lexer.INDENT || t.Kind == lexer.DEDENT {
			continue
		}
		if t.Pos.Line != pos.Line {
			continue
		}
		start := t.Pos.Col
		end := t.Pos.Col + len([]rune(t.Data))
		if pos.Character >= start && pos.Character <= end {
			return t, toks, true
		}
	}
	return lexer.Token{}, toks, false
}

// tokenize runs the lexer to completion and returns every emitted token.
func tokenize(src, file string) []lexer.Token {
	lx := lexer.New(src, file)
	var out []lexer.Token
	for {
		t := lx.Next()
		out = append(out, t)
		if t.Kind == lexer.EOF {
			break
		}
	}
	return out
}

// lineAt returns the (0-indexed) line of text, or "" if out of range.
func lineAt(src string, line int) string {
	lines := strings.Split(src, "\n")
	if line < 0 || line >= len(lines) {
		return ""
	}
	return lines[line]
}
