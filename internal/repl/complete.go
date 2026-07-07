package repl

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jiejie-dev/funny/v2/internal/ast"
	"github.com/jiejie-dev/funny/v2/internal/lexer"
	"github.com/jiejie-dev/funny/v2/internal/parser"
	"github.com/jiejie-dev/funny/v2/internal/types"
)

// InputStatus reports whether src is a complete Funny cell and any parse error.
// Incomplete cells (open blocks/brackets) return complete=false with err=nil.
func InputStatus(src string) (complete bool, err error) {
	if strings.TrimSpace(src) == "" {
		return false, nil
	}
	lx := lexer.New(src, replFile)
	for {
		if lx.Next().Kind == lexer.EOF {
			break
		}
	}
	st := lx.Snapshot()
	if st.ParenDepth > 0 || len(st.IndentStack) > 1 {
		return false, nil
	}
	prog, err := parser.New(src, replFile).Parse()
	if err != nil {
		if isIncompleteError(err) {
			return false, nil
		}
		return true, err
	}
	_ = prog
	return true, nil
}

func isIncompleteError(err error) bool {
	s := err.Error()
	return strings.Contains(s, "INDENT") ||
		strings.Contains(s, "DEDENT") ||
		strings.Contains(s, "expected `)`") ||
		strings.Contains(s, "expected `]`") ||
		strings.Contains(s, "expected `}`")
}

var replKeywords = []string{
	"let", "if", "elif", "else", "for", "while", "match", "fn", "struct",
	"return", "break", "continue", "import", "pub", "plan", "step", "meta",
	"guard", "parallel", "branch", "delay", "not", "in", "true", "false", "nil",
}

// Completions returns identifier completions for the last token on the line.
func Completions(sess *Session, line string) []string {
	prefix := lastTokenPrefix(line)
	if prefix == "" {
		return nil
	}
	seen := map[string]struct{}{}
	var out []string
	add := func(name string) {
		if !strings.HasPrefix(name, prefix) {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	for _, kw := range replKeywords {
		add(kw)
	}
	for _, name := range types.BuiltinNames() {
		add(name)
	}
	if sess != nil {
		for name := range sess.env.Vars() {
			add(name)
		}
		for name := range sess.env.Funcs() {
			add(name)
		}
		for name := range sess.env.Structs() {
			add(name)
		}
		for name := range sess.sessionBindings() {
			add(name)
		}
	}
	sort.Strings(out)
	return out
}

func lastTokenPrefix(line string) string {
	line = strings.TrimSpace(line)
	if line == "" {
		return ""
	}
	i := len(line) - 1
	for i >= 0 {
		c := line[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			i--
			continue
		}
		break
	}
	return line[i+1:]
}

// TypeOfExpr type-checks a single expression in the current session env.
func (s *Session) TypeOfExpr(src string) (string, error) {
	expr, err := parseExpressionSource(src)
	if err != nil {
		return "", err
	}
	t, err := types.CheckExpr(expr, s.env)
	if err != nil {
		return "", err
	}
	return t.String(), nil
}

// DescribeName returns a type summary for a binding in the session.
func (s *Session) DescribeName(name string) (string, error) {
	if t, ok := s.env.LookupVar(name); ok {
		return fmt.Sprintf("let %s: %s", name, t), nil
	}
	if fn, ok := s.env.LookupFunc(name); ok {
		return fn.String(), nil
	}
	if st, ok := s.env.LookupStruct(name); ok {
		return st.String(), nil
	}
	if v, ok := s.sessionBindings()[name]; ok {
		switch x := v.(type) {
		case *ast.FnDecl:
			return fmt.Sprintf("fn %s(...)", x.Name), nil
		default:
			return fmt.Sprintf("%s = %s", name, FormatValue(v)), nil
		}
	}
	return "", fmt.Errorf("unknown name %q", name)
}

func parseExpressionSource(src string) (ast.Expression, error) {
	prog, err := parser.New(strings.TrimSpace(src), replFile).Parse()
	if err != nil {
		return nil, err
	}
	if len(prog.Stmts) != 1 {
		return nil, fmt.Errorf("expected a single expression")
	}
	es, ok := prog.Stmts[0].(*ast.ExprStmt)
	if !ok {
		return nil, fmt.Errorf("expected an expression")
	}
	return es.X, nil
}
