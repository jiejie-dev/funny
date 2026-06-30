package ast

import "fmt"

type Pos struct {
	File string
	Line int
	Col  int
}

func (p Pos) String() string {
	return fmt.Sprintf("%s:%d:%d", p.File, p.Line+1, p.Col+1)
}

type Node interface {
	Pos() Pos
	nodeMarker()
}

type Statement interface {
	Node
	stmtMarker()
}

type Expression interface {
	Node
	exprMarker()
}