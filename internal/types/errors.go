package types

import (
	"fmt"

	"github.com/jiejie-dev/funny/internal/ast"
)

// Error is a type-checking error.
type Error struct {
	Code     string
	Message  string
	Pos      ast.Pos
	Expected Type
	Actual   Type
	Hint     string
}

// New creates a new Error with the given code and message.
func New(code, msg string, pos ast.Pos) *Error {
	return &Error{Code: code, Message: msg, Pos: pos}
}

// NewMismatch creates a type-mismatch error with both types annotated.
func NewMismatch(pos ast.Pos, expected, actual Type) *Error {
	return &Error{
		Code:     "E2010",
		Message:  fmt.Sprintf("type mismatch: expected %s, got %s", expected, actual),
		Pos:      pos,
		Expected: expected,
		Actual:   actual,
		Hint:     fmt.Sprintf("expected %s here", expected),
	}
}

// Error implements the error interface.
func (e *Error) Error() string { return e.Format() }

// Format produces the unified error format:
//
//	error[E2010]: type mismatch: expected int, got str
//	 --> <file>:<line>:<col>
//	help: expected int here
func (e *Error) Format() string {
	msg := e.Message
	if e.Expected != nil && e.Actual != nil {
		msg = fmt.Sprintf("%s: expected %s, got %s", e.Message, e.Expected, e.Actual)
	}
	s := fmt.Sprintf("error[%s]: %s\n --> %s:%d:%d\n",
		e.Code, msg, e.Pos.File, e.Pos.Line+1, e.Pos.Col+1)
	if e.Hint != "" {
		s += fmt.Sprintf("\nhelp: %s", e.Hint)
	}
	return s
}
