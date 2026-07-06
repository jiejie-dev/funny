package bytecode

import "fmt"

// SourceLoc maps a bytecode instruction index to a source position.
// Line and Col are 0-based, matching ast.Pos.
type SourceLoc struct {
	File string
	Line int
	Col  int
}

// IsZero reports whether no source location was recorded.
func (l SourceLoc) IsZero() bool {
	return l.File == "" && l.Line == 0 && l.Col == 0
}

// Display returns a human-readable 1-based position (file:line:col).
func (l SourceLoc) Display() string {
	if l.IsZero() {
		return "<unknown>"
	}
	return fmt.Sprintf("%s:%d:%d", l.File, l.Line+1, l.Col+1)
}
