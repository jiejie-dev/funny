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
	String() string
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

// ----- Expressions -----

type LiteralExpr struct {
	NodePos Pos
	Value   any
}

func (e *LiteralExpr) Pos() Pos    { return e.NodePos }
func (e *LiteralExpr) exprMarker() {}
func (e *LiteralExpr) nodeMarker() {}
func (e *LiteralExpr) String() string {
	if s, ok := e.Value.(string); ok {
		return fmt.Sprintf("%q", s)
	}
	return fmt.Sprintf("%v", e.Value)
}

type VariableExpr struct {
	NodePos Pos
	Name    string
}

func (e *VariableExpr) Pos() Pos    { return e.NodePos }
func (e *VariableExpr) exprMarker() {}
func (e *VariableExpr) nodeMarker() {}
func (e *VariableExpr) String() string {
	return e.Name
}

type BinaryExpr struct {
	NodePos Pos
	Left    Expression
	Op      string
	Right   Expression
}

func (e *BinaryExpr) Pos() Pos    { return e.NodePos }
func (e *BinaryExpr) exprMarker() {}
func (e *BinaryExpr) nodeMarker() {}
func (e *BinaryExpr) String() string {
	return fmt.Sprintf("%s %s %s", e.Left.String(), e.Op, e.Right.String())
}

type UnaryExpr struct {
	NodePos Pos
	Op      string
	Expr    Expression
}

func (e *UnaryExpr) Pos() Pos    { return e.NodePos }
func (e *UnaryExpr) exprMarker() {}
func (e *UnaryExpr) nodeMarker() {}
func (e *UnaryExpr) String() string {
	return fmt.Sprintf("%s %s", e.Op, e.Expr.String())
}

type SubExpr struct {
	NodePos Pos
	Inner   Expression
}

func (e *SubExpr) Pos() Pos    { return e.NodePos }
func (e *SubExpr) exprMarker() {}
func (e *SubExpr) nodeMarker() {}
func (e *SubExpr) String() string {
	return fmt.Sprintf("(%s)", e.Inner.String())
}

type ListExpr struct {
	NodePos  Pos
	Elements []Expression
}

func (e *ListExpr) Pos() Pos    { return e.NodePos }
func (e *ListExpr) exprMarker() {}
func (e *ListExpr) nodeMarker() {}
func (e *ListExpr) String() string {
	parts := make([]string, len(e.Elements))
	for i, el := range e.Elements {
		parts[i] = el.String()
	}
	return "[" + joinComma(parts) + "]"
}

type IndexExpr struct {
	NodePos Pos
	Object  Expression
	Index   Expression
}

func (e *IndexExpr) Pos() Pos    { return e.NodePos }
func (e *IndexExpr) exprMarker() {}
func (e *IndexExpr) nodeMarker() {}
func (e *IndexExpr) String() string {
	return fmt.Sprintf("%s[%s]", e.Object.String(), e.Index.String())
}

type FieldExpr struct {
	NodePos Pos
	Object  Expression
	Field   string
}

func (e *FieldExpr) Pos() Pos    { return e.NodePos }
func (e *FieldExpr) exprMarker() {}
func (e *FieldExpr) nodeMarker() {}
func (e *FieldExpr) String() string {
	return fmt.Sprintf("%s.%s", e.Object.String(), e.Field)
}

type CallExpr struct {
	NodePos Pos
	Func    Expression
	Args    []Expression
}

func (e *CallExpr) Pos() Pos    { return e.NodePos }
func (e *CallExpr) exprMarker() {}
func (e *CallExpr) nodeMarker() {}
func (e *CallExpr) String() string {
	parts := make([]string, len(e.Args))
	for i, a := range e.Args {
		parts[i] = a.String()
	}
	return fmt.Sprintf("%s(%s)", e.Func.String(), joinComma(parts))
}

type FStringExpr struct {
	NodePos Pos
	Raw     string
}

func (e *FStringExpr) Pos() Pos    { return e.NodePos }
func (e *FStringExpr) exprMarker() {}
func (e *FStringExpr) nodeMarker() {}
func (e *FStringExpr) String() string {
	return fmt.Sprintf("f%q", e.Raw)
}

func joinComma(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += ", "
		}
		out += p
	}
	return out
}
