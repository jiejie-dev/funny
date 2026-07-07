package repl

import (
	"strings"

	"github.com/jiejie-dev/funny/v2/internal/lexer"
	"github.com/jiejie-dev/funny/v2/internal/parser"
)

// InputStatus reports whether src is a complete Funny cell and any parse error.
// Incomplete cells (open blocks/brackets) return complete=false with err=nil.
func InputStatus(src string) (complete bool, err error) {
	if strings.TrimSpace(src) == "" {
		return false, nil
	}
	lx := lexer.New(src, "<repl>")
	for {
		if lx.Next().Kind == lexer.EOF {
			break
		}
	}
	st := lx.Snapshot()
	if st.ParenDepth > 0 || len(st.IndentStack) > 1 {
		return false, nil
	}
	prog, err := parser.New(src, "<repl>").Parse()
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
