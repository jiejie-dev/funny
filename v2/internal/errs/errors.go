package errs

import "fmt"

type Position struct {
	File string
	Line int
	Col  int
}

type Error struct {
	Code    string
	Message string
	Pos     Position
	Hint    string
}

func New(code, message string, pos Position, hint string) *Error {
	return &Error{Code: code, Message: message, Pos: pos, Hint: hint}
}

func (e *Error) Error() string {
	return e.Format()
}

func (e *Error) Format() string {
	s := fmt.Sprintf("error[%s]: %s\n --> %s:%d:%d\n",
		e.Code, e.Message, e.Pos.File, e.Pos.Line, e.Pos.Col)
	if e.Hint != "" {
		s += fmt.Sprintf("\nhelp: %s", e.Hint)
	}
	return s
}
