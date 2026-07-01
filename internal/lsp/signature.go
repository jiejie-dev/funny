package lsp

import (
	"strings"

	"github.com/jiejie-dev/funny/internal/lexer"
	"github.com/jiejie-dev/funny/internal/types"
)

// signatureHelp finds the call expression enclosing pos (via bracket
// matching over the token stream, since the AST has no end positions) and
// returns its signature with the active parameter highlighted.
func (d *document) signatureHelp(pos Position) *SignatureHelp {
	toks := tokenize(d.text, d.path)
	name, argIndex, ok := enclosingCall(toks, pos)
	if !ok {
		return nil
	}
	var params []types.Type
	var ret types.Type
	switch {
	case d.env != nil && funcIsKnown(d.env, name):
		fn, _ := d.env.LookupFunc(name)
		params, ret = fn.Params, fn.Return
	case isBuiltin(name):
		return &SignatureHelp{
			Signatures:      []SignatureInformation{{Label: name + "(...)"}},
			ActiveParameter: argIndex,
		}
	default:
		return nil
	}
	label := name + "(" + joinTypes(params) + ")"
	if ret != nil {
		label += " -> " + ret.String()
	}
	paramInfos := make([]ParameterInformation, len(params))
	for i, p := range params {
		paramInfos[i] = ParameterInformation{Label: p.String()}
	}
	if argIndex >= len(paramInfos) && len(paramInfos) > 0 {
		argIndex = len(paramInfos) - 1
	}
	return &SignatureHelp{
		Signatures:      []SignatureInformation{{Label: label, Parameters: paramInfos}},
		ActiveParameter: argIndex,
	}
}

func funcIsKnown(env *types.Env, name string) bool {
	_, ok := env.LookupFunc(name)
	return ok
}

func isBuiltin(name string) bool {
	for _, b := range types.BuiltinNames() {
		if b == name {
			return true
		}
	}
	return false
}

func joinTypes(ts []types.Type) string {
	parts := make([]string, len(ts))
	for i, t := range ts {
		parts[i] = t.String()
	}
	return strings.Join(parts, ", ")
}

type bracketFrame struct {
	open          lexer.Kind
	commaCount    int
	precedingName string
}

// enclosingCall walks toks up to pos, tracking bracket nesting, and returns
// the name and active-argument index of the innermost `(...)` group whose
// opening paren was immediately preceded by an identifier (i.e. an actual
// call, as opposed to a parenthesized grouping expression).
func enclosingCall(toks []lexer.Token, pos Position) (string, int, bool) {
	var stack []bracketFrame
	lastName := ""
	for _, t := range toks {
		if t.Pos.Line > pos.Line || (t.Pos.Line == pos.Line && t.Pos.Col >= pos.Character) {
			break
		}
		switch t.Kind {
		case lexer.LPAREN:
			stack = append(stack, bracketFrame{open: lexer.LPAREN, precedingName: lastName})
		case lexer.LBRACK:
			stack = append(stack, bracketFrame{open: lexer.LBRACK})
		case lexer.LBRACE:
			stack = append(stack, bracketFrame{open: lexer.LBRACE})
		case lexer.RPAREN, lexer.RBRACK, lexer.RBRACE:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case lexer.COMMA:
			if len(stack) > 0 && stack[len(stack)-1].open == lexer.LPAREN {
				stack[len(stack)-1].commaCount++
			}
		}
		if t.Kind == lexer.NAME {
			lastName = t.Data
		} else if t.Kind != lexer.LPAREN {
			lastName = ""
		}
	}
	for i := len(stack) - 1; i >= 0; i-- {
		if stack[i].open == lexer.LPAREN && stack[i].precedingName != "" {
			return stack[i].precedingName, stack[i].commaCount, true
		}
	}
	return "", 0, false
}
