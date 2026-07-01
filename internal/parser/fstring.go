// internal/parser/fstring.go
package parser

import (
	"fmt"
	"strings"

	"github.com/jiejie-dev/funny/internal/ast"
	"github.com/jiejie-dev/funny/internal/errs"
	"github.com/jiejie-dev/funny/internal/lexer"
)

// parseFStringParts splits the raw text captured by lexFString into literal
// and interpolation parts, recursively parsing each `{expr[:spec]}` with a
// fresh sub-parser. tokPos is the FSTR token's position, used (approximately)
// for error reporting inside interpolations.
func (p *Parser) parseFStringParts(raw string, tokPos lexer.Position) ([]ast.FStringPart, error) {
	var parts []ast.FStringPart
	var lit strings.Builder
	flush := func() {
		if lit.Len() > 0 {
			parts = append(parts, ast.FStringPart{Text: lit.String()})
			lit.Reset()
		}
	}
	i, n := 0, len(raw)
	for i < n {
		switch {
		case raw[i] == '\\' && i+1 < n:
			lit.WriteByte(unescapeFStringChar(raw[i+1]))
			i += 2
		case raw[i] == '{' && i+1 < n && raw[i+1] == '{':
			lit.WriteByte('{')
			i += 2
		case raw[i] == '}' && i+1 < n && raw[i+1] == '}':
			lit.WriteByte('}')
			i += 2
		case raw[i] == '{':
			flush()
			exprSrc, spec, next, serr := splitInterpolation(raw, i+1)
			if serr != nil {
				return nil, errs.New("E1050", serr.Error(), errPosFromPos(tokPos), "")
			}
			if strings.TrimSpace(exprSrc) == "" {
				return nil, errs.New("E1051", "empty expression in f-string interpolation `{}`", errPosFromPos(tokPos), "")
			}
			expr, perr := p.parseSubExpr(exprSrc, tokPos.File)
			if perr != nil {
				return nil, perr
			}
			parts = append(parts, ast.FStringPart{Expr: expr, Spec: spec})
			i = next
		case raw[i] == '}':
			return nil, errs.New("E1052", "unmatched `}` in f-string (use `}}` for a literal `}`)", errPosFromPos(tokPos), "")
		default:
			lit.WriteByte(raw[i])
			i++
		}
	}
	flush()
	return parts, nil
}

func unescapeFStringChar(c byte) byte {
	switch c {
	case 'n':
		return '\n'
	case 't':
		return '\t'
	case 'r':
		return '\r'
	default:
		return c
	}
}

// splitInterpolation scans raw[start:] for the matching top-level `}` and the
// (optional) top-level `:` that separates the expression from a format spec.
// Depth tracking covers (), [], {} and skips over quoted-string content so
// that named-arg colons (e.g. `Point(x: 1)`) and nested strings don't confuse
// the spec-separator search.
func splitInterpolation(raw string, start int) (exprSrc, spec string, next int, err error) {
	depth := 0
	specStart := -1
	i, n := start, len(raw)
	for i < n {
		ch := raw[i]
		switch {
		case ch == '\'' || ch == '"':
			quote := ch
			i++
			for i < n && raw[i] != quote {
				if raw[i] == '\\' && i+1 < n {
					i += 2
					continue
				}
				i++
			}
			if i < n {
				i++
			}
		case ch == '(' || ch == '[' || ch == '{':
			depth++
			i++
		case ch == ')' || ch == ']':
			depth--
			i++
		case ch == '}':
			if depth == 0 {
				exprEnd := i
				if specStart >= 0 {
					exprEnd = specStart
				}
				spec = ""
				if specStart >= 0 {
					spec = raw[specStart+1 : i]
				}
				return strings.TrimSpace(raw[start:exprEnd]), spec, i + 1, nil
			}
			depth--
			i++
		case ch == ':' && depth == 0 && specStart < 0:
			specStart = i
			i++
		default:
			i++
		}
	}
	return "", "", 0, fmt.Errorf("unterminated f-string interpolation (missing `}`)")
}

// parseSubExpr parses a standalone expression fragment (from inside an
// f-string interpolation) using a fresh Parser, and requires it to consume
// the fragment entirely.
func (p *Parser) parseSubExpr(src, file string) (ast.Expression, error) {
	sub := New(src, file)
	expr, err := sub.parseExpression()
	if err != nil {
		return nil, err
	}
	if sub.cur.Kind != lexer.EOF {
		return nil, errs.New("E1053",
			fmt.Sprintf("unexpected token %s after expression in f-string interpolation", sub.cur.Kind),
			errPos(sub.cur.Pos), "")
	}
	return expr, nil
}

func errPosFromPos(p lexer.Position) errs.Position {
	return errs.Position{File: p.File, Line: p.Line, Col: p.Col}
}
