package lsp

import (
	"strings"

	"github.com/jiejie-dev/funny/internal/formatter"
)

// formatting delegates to the shared formatter and returns a single edit
// replacing the whole document (the formatter always re-prints the entire
// file, so a full-document TextEdit is simplest and always correct).
func (d *document) formatting() ([]TextEdit, error) {
	out, err := formatter.Format([]byte(d.text), d.path)
	if err != nil {
		return nil, err
	}
	if out == d.text {
		return nil, nil
	}
	lastLine := strings.Count(d.text, "\n")
	lastCol := len(lastLineOf(d.text))
	fullRange := Range{Start: Position{}, End: Position{Line: lastLine, Character: lastCol}}
	return []TextEdit{{Range: fullRange, NewText: out}}, nil
}

func lastLineOf(s string) string {
	lines := strings.Split(s, "\n")
	return lines[len(lines)-1]
}
