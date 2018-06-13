package langs

import (
	"fmt"
	"strings"
)

func intent(s string) string {
	ss := strings.Split(s, "\n")
	for index, item := range ss {
		ss[index] = fmt.Sprintf("  %s", item)
	}
	return strings.Join(ss, "\n")
}

type Statement interface {
	Position() Position
	String() string
}

// Variable
type Variable struct {
	pos  Position
	Name string
}

func (v *Variable) String() string {
	return fmt.Sprintf("%s", v.Name)
}

func (v *Variable) Position() Position {
	return v.pos
}

// Literal
type Literal struct {
	pos   Position
	Value interface{}
}

func (l *Literal) Position() Position {
	return l.pos
}

func (l *Literal) String() string {
	fmt.Print(Typing(l.Value))
	if Typing(l.Value) == "string"{
		return fmt.Sprintf("'%v'", l.Value)
	}
	return fmt.Sprintf("%v", l.Value)
}

// Expression
type Expresion interface {
	Position() Position
	String() string
}

// BinaryExpression
type BinaryExpression struct {
	pos      Position
	Left     Expresion
	Operator Token
	Right    Expresion
}

func (b *BinaryExpression) Position() Position {
	return b.pos
}

func (b *BinaryExpression) String() string {
	return fmt.Sprintf("%s %s %s", b.Left.String(), b.Operator.Data, b.Right.String())
}

// Assign
type Assign struct {
	pos    Position
	Target Expresion
	Value  Expresion
}

func (a *Assign) Position() Position {
	return a.pos
}

func (a *Assign) String() string {
	return fmt.Sprintf("%s = %s", a.Target.String(), a.Value.String())
}

// List
type List struct {
	pos    Position
	Values []Expresion
}

func (l *List) Position() Position {
	return l.pos
}

func (l *List) String() string {
	var s []string
	for _, item := range (l.Values) {
		s = append(s, item.String())
	}
	return fmt.Sprintf("%s", strings.Join(s, ", "))
}

// Block
type Block []Statement

func (b *Block) Position() Position {
	return Position{}
}

func (b *Block) String() string {
	var s []string
	for _, item := range (*b) {
		s = append(s, item.String())
	}
	return strings.Join(s, "\n")
}

// Function
type Function struct {
	pos        Position
	Name       string
	Parameters []Expresion
	Body       Block
}

func (f *Function) Position() Position {
	return f.pos
}

func (f *Function) String() string {
	var args []string
	for _, item := range (f.Parameters) {
		args = append(args, item.String())
	}
	s := block(f.Body)
	return fmt.Sprintf("%s(%s) {\n%s\n}", f.Name, strings.Join(args, ", "), s)
}

type FunctionCall struct {
	pos        Position
	Name       string
	Parameters []Expresion
}

func (c *FunctionCall) Position() Position {
	return c.pos
}

func (c *FunctionCall) String() string {
	var args []string
	for _, item := range (c.Parameters) {
		args = append(args, item.String())
	}
	return fmt.Sprintf("%s(%s)", c.Name, strings.Join(args, ", "))
}

func block(b Block) (s string) {
	var ss []string
	for _, item := range (b) {
		ss = append(ss, intent(item.String()))
	}
	return strings.Join(ss, "\n")
}

// Program
type Program struct {
	Statements Block
}

func (p *Program) String() string {
	return p.Statements.String()
}

// IFStatement
type IFStatement struct {
	pos       Position
	Condition Expresion
	Body      Block
	Else      Block
}

func (i *IFStatement) Position() Position {
	return i.pos
}

func (i *IFStatement) String() string {
	return fmt.Sprintf("if %s {\n%s\n} else {\n%s\n}\n", i.Condition.String(), block(i.Body), block(i.Else))
}

type FORStatement struct {
	pos      Position
	Iterable IterableExpression
	Block    Block

	CurrentIndex Variable
	CurrentItem  Expresion
}

func (f *FORStatement) Position() Position {
	return f.pos
}

func (f *FORStatement) String() string {
	return fmt.Sprintf("for %s, %s in %s {\n%s\n}",
		f.CurrentIndex.String(),
		f.CurrentItem.String(),
		f.Iterable.Name.String(),
		intent(f.Block.String()))
}

// IterableExpression
type IterableExpression struct {
	pos   Position
	Name  Variable
	Index int
	Items []Expresion
}

func (i *IterableExpression) Position() Position {
	return i.pos
}

func (i *IterableExpression) String() string {
	return fmt.Sprintf("", )
}

func (i *IterableExpression) Next() (int, Expresion) {
	if i.Index+1 >= len(i.Items) {
		return -1, nil
	}
	item := i.Items[i.Index]
	i.Index++
	return i.Index, item
}

type Break struct {
	pos Position
}

func (b *Break) Position() Position {
	return b.pos
}

func (b *Break) String() string {
	return "break"
}

type Continue struct {
	pos Position
}

func (b *Continue) Position() Position {
	return b.pos
}

func (b *Continue) String() string {
	return "continue"
}

type Return struct {
	pos   Position
	Value Expresion
}

func (r *Return) Position() Position {
	return r.pos
}

func (r *Return) String() string {
	return fmt.Sprintf("return %s", r.Value.String())
}

type Field struct {
	pos      Position
	Variable Variable
	Value    Expresion
}

func (f *Field) Position() Position {
	return f.pos
}

func (f *Field) String() string {
	return fmt.Sprintf("%s.%s", f.Variable.String(), f.Value.String())
}

type Boolen struct {
	pos   Position
	Value bool
}

func (b *Boolen) Position() Position {
	return b.pos
}

func (b *Boolen) String() string {
	if b.Value {
		return "true"
	}
	return "false"
}

type StringExpression struct {
	pos   Position
	Value string
}

func (s *StringExpression) Position() Position {
	return s.pos
}

func (s *StringExpression) String() string {
	return s.Value
}