package ast

import (
	"fmt"
	"strconv"
	"strings"
)

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
	switch v := e.Value.(type) {
	case nil:
		// %v (via the default branch below) prints a bare Go nil
		// interface as the literal text "<nil>", not funny's `nil`
		// keyword - so `return nil` round-tripped through the formatter
		// came back out as the syntactically invalid `return <nil>`.
		return "nil"
	case string:
		return fmt.Sprintf("%q", v)
	case float64:
		// %v (via the default branch below) prints a whole-number float
		// like 500.0 as just "500" - indistinguishable from the int
		// literal 500. Since this String() is what the formatter
		// (internal/formatter) re-emits as actual source code, that
		// silently turned a float literal into what re-parses as an int
		// token, changing the expression's static type on round-trip
		// (e.g. `x > 500.0` reformatted to `x > 500`, breaking a
		// float-typed comparison). Always keep at least one decimal
		// digit so it round-trips as a float.
		s := strconv.FormatFloat(v, 'g', -1, 64)
		if !strings.ContainsAny(s, ".eEnN") { // "n"/"N" catches NaN/Inf
			s += ".0"
		}
		return s
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

// MapLiteralExpr is a `{key: value, ...}` literal. Keys and Values are
// parallel slices (not a Go map) to preserve source order and to allow keys
// that are arbitrary expressions, not just compile-time-constant strings.
type MapLiteralExpr struct {
	NodePos Pos
	Keys    []Expression
	Values  []Expression
}

func (e *MapLiteralExpr) Pos() Pos    { return e.NodePos }
func (e *MapLiteralExpr) exprMarker() {}
func (e *MapLiteralExpr) nodeMarker() {}
func (e *MapLiteralExpr) String() string {
	parts := make([]string, len(e.Keys))
	for i, k := range e.Keys {
		parts[i] = fmt.Sprintf("%s: %s", k.String(), e.Values[i].String())
	}
	return "{" + joinComma(parts) + "}"
}

type StructLiteralExpr struct {
	NodePos  Pos
	TypeName string
	Fields   map[string]Expression
}

func (e *StructLiteralExpr) Pos() Pos    { return e.NodePos }
func (e *StructLiteralExpr) exprMarker() {}
func (e *StructLiteralExpr) nodeMarker() {}
func (e *StructLiteralExpr) String() string {
	parts := make([]string, 0, len(e.Fields))
	for k, v := range e.Fields {
		parts = append(parts, fmt.Sprintf("%s: %s", k, v.String()))
	}
	return fmt.Sprintf("%s(%s)", e.TypeName, joinComma(parts))
}

// FStringPart is one segment of an f-string: either literal text (Expr == nil)
// or an interpolated expression with an optional format spec.
type FStringPart struct {
	Text string     // literal text, already unescaped (used when Expr == nil)
	Expr Expression // interpolated expression (nil for literal parts)
	Spec string     // raw format spec after ':' (e.g. ".2f", ">10"); "" = default
}

type FStringExpr struct {
	NodePos Pos
	Raw     string // original raw text between the f-string quotes, as captured by the lexer
	Parts   []FStringPart
}

func (e *FStringExpr) Pos() Pos    { return e.NodePos }
func (e *FStringExpr) exprMarker() {}
func (e *FStringExpr) nodeMarker() {}
func (e *FStringExpr) String() string {
	return fmt.Sprintf("f%q", e.Raw)
}

// TryExpr is a postfix-? expression: `expr?` — propagates Err, unwraps Ok.
type TryExpr struct {
	NodePos Pos
	Inner   Expression
}

func (e *TryExpr) Pos() Pos    { return e.NodePos }
func (e *TryExpr) exprMarker() {}
func (e *TryExpr) nodeMarker() {}
func (e *TryExpr) String() string {
	return e.Inner.String() + "?"
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

// ----- Statements -----

type Block struct {
	NodePos    Pos
	Statements []Statement
}

func (s *Block) Pos() Pos    { return s.NodePos }
func (s *Block) stmtMarker() {}
func (s *Block) nodeMarker() {}
func (s *Block) String() string {
	out := ""
	for _, s := range s.Statements {
		out += s.String() + "\n"
	}
	return out
}

// ToProgram wraps a Block in a Program (used by type checker).
func (b *Block) ToProgram() *Program {
	return &Program{NodePos: b.NodePos, Stmts: b.Statements}
}

type LetStmt struct {
	NodePos Pos
	Name    string
	TypeAnn string
	Value   Expression
}

func (s *LetStmt) Pos() Pos    { return s.NodePos }
func (s *LetStmt) stmtMarker() {}
func (s *LetStmt) nodeMarker() {}
func (s *LetStmt) String() string {
	if s.TypeAnn != "" {
		return fmt.Sprintf("let %s: %s = %s", s.Name, s.TypeAnn, s.Value.String())
	}
	return fmt.Sprintf("let %s = %s", s.Name, s.Value.String())
}

type AssignStmt struct {
	NodePos Pos
	Target  Expression
	Value   Expression
}

func (s *AssignStmt) Pos() Pos    { return s.NodePos }
func (s *AssignStmt) stmtMarker() {}
func (s *AssignStmt) nodeMarker() {}
func (s *AssignStmt) String() string {
	return fmt.Sprintf("%s = %s", s.Target.String(), s.Value.String())
}

type IfStmt struct {
	NodePos   Pos
	Cond      Expression
	Then      *Block
	ElseIf    *IfStmt
	ElseBlock *Block
}

func (s *IfStmt) Pos() Pos    { return s.NodePos }
func (s *IfStmt) stmtMarker() {}
func (s *IfStmt) nodeMarker() {}
func (s *IfStmt) String() string {
	out := fmt.Sprintf("if %s:\n%s", s.Cond.String(), s.Then.String())
	if s.ElseIf != nil {
		out += "elif " + s.ElseIf.String()
	}
	if s.ElseBlock != nil {
		out += "else:\n" + s.ElseBlock.String()
	}
	return out
}

type ForStmt struct {
	NodePos  Pos
	Name     string
	Iterable Expression
	Body     *Block
}

func (s *ForStmt) Pos() Pos    { return s.NodePos }
func (s *ForStmt) stmtMarker() {}
func (s *ForStmt) nodeMarker() {}
func (s *ForStmt) String() string {
	return fmt.Sprintf("for %s in %s:\n%s", s.Name, s.Iterable.String(), s.Body.String())
}

type WhileStmt struct {
	NodePos Pos
	Cond    Expression
	Body    *Block
}

func (s *WhileStmt) Pos() Pos    { return s.NodePos }
func (s *WhileStmt) stmtMarker() {}
func (s *WhileStmt) nodeMarker() {}
func (s *WhileStmt) String() string {
	return fmt.Sprintf("while %s:\n%s", s.Cond.String(), s.Body.String())
}

type MatchArm struct {
	Pattern Expression
	Body    *Block
}

type MatchStmt struct {
	NodePos Pos
	Expr    Expression
	Arms    []MatchArm
}

func (s *MatchStmt) Pos() Pos    { return s.NodePos }
func (s *MatchStmt) stmtMarker() {}
func (s *MatchStmt) nodeMarker() {}
func (s *MatchStmt) String() string {
	out := fmt.Sprintf("match %s:\n", s.Expr.String())
	for _, a := range s.Arms {
		out += fmt.Sprintf("    %s =>\n%s", a.Pattern.String(), a.Body.String())
	}
	return out
}

type ReturnStmt struct {
	NodePos Pos
	Value   Expression
}

func (s *ReturnStmt) Pos() Pos    { return s.NodePos }
func (s *ReturnStmt) stmtMarker() {}
func (s *ReturnStmt) nodeMarker() {}
func (s *ReturnStmt) String() string {
	if s.Value == nil {
		return "return"
	}
	return fmt.Sprintf("return %s", s.Value.String())
}

type BreakStmt struct{ NodePos Pos }

func (s *BreakStmt) Pos() Pos       { return s.NodePos }
func (s *BreakStmt) stmtMarker()    {}
func (s *BreakStmt) nodeMarker()    {}
func (s *BreakStmt) String() string { return "break" }

type ContinueStmt struct{ NodePos Pos }

func (s *ContinueStmt) Pos() Pos       { return s.NodePos }
func (s *ContinueStmt) stmtMarker()    {}
func (s *ContinueStmt) nodeMarker()    {}
func (s *ContinueStmt) String() string { return "continue" }

// CommentStmt is a standalone or trailing `#`/`##` comment, kept as a
// statement so the formatter can reproduce it. Text excludes the leading
// `#`/`##` marker itself (i.e. for `# hello` Text is " hello").
type CommentStmt struct {
	NodePos Pos
	Text    string
	Doc     bool // true for `##` doc comments
}

func (s *CommentStmt) Pos() Pos    { return s.NodePos }
func (s *CommentStmt) stmtMarker() {}
func (s *CommentStmt) nodeMarker() {}
func (s *CommentStmt) String() string {
	if s.Doc {
		return "##" + s.Text
	}
	return "#" + s.Text
}

type ExprStmt struct {
	NodePos Pos
	X       Expression
}

func (s *ExprStmt) Pos() Pos       { return s.NodePos }
func (s *ExprStmt) stmtMarker()    {}
func (s *ExprStmt) nodeMarker()    {}
func (s *ExprStmt) String() string { return s.X.String() }

// ----- Declarations -----

type Param struct {
	Name    string
	TypeAnn string
	Mut     bool // struct fields only: `mut count: int`
}

func (p Param) String() string {
	prefix := ""
	if p.Mut {
		prefix = "mut "
	}
	if p.TypeAnn != "" {
		return fmt.Sprintf("%s%s: %s", prefix, p.Name, p.TypeAnn)
	}
	return prefix + p.Name
}

type FnDecl struct {
	NodePos Pos
	Pub     bool
	Name    string
	Params  []Param
	RetType string
	Body    *Block
}

func (s *FnDecl) Pos() Pos    { return s.NodePos }
func (s *FnDecl) stmtMarker() {}
func (s *FnDecl) nodeMarker() {}
func (s *FnDecl) String() string {
	parts := make([]string, len(s.Params))
	for i, p := range s.Params {
		parts[i] = p.String()
	}
	prefix := ""
	if s.Pub {
		prefix = "pub "
	}
	out := fmt.Sprintf("%sfn %s(%s)", prefix, s.Name, joinComma(parts))
	if s.RetType != "" {
		out += " -> " + s.RetType
	}
	out += ":\n" + s.Body.String()
	return out
}

type StructDecl struct {
	NodePos Pos
	Pub     bool
	Name    string
	Fields  []Param
}

func (s *StructDecl) Pos() Pos    { return s.NodePos }
func (s *StructDecl) stmtMarker() {}
func (s *StructDecl) nodeMarker() {}
func (s *StructDecl) String() string {
	prefix := ""
	if s.Pub {
		prefix = "pub "
	}
	out := fmt.Sprintf("%sstruct %s:\n", prefix, s.Name)
	for _, f := range s.Fields {
		out += fmt.Sprintf("    %s\n", f.String())
	}
	return out
}

// ----- Top-Level -----

type ImportDecl struct {
	NodePos Pos
	Path    string
	Alias   string
}

func (s *ImportDecl) Pos() Pos    { return s.NodePos }
func (s *ImportDecl) stmtMarker() {}
func (s *ImportDecl) nodeMarker() {}
func (s *ImportDecl) String() string {
	out := fmt.Sprintf("import %q", s.Path)
	if s.Alias != "" {
		out += " as " + s.Alias
	}
	return out
}

type MetaBlock struct {
	NodePos Pos
	Name    string
	Fields  map[string]string
}

func (s *MetaBlock) Pos() Pos    { return s.NodePos }
func (s *MetaBlock) stmtMarker() {}
func (s *MetaBlock) nodeMarker() {}
func (s *MetaBlock) String() string {
	out := "meta:\n"
	for k, v := range s.Fields {
		out += fmt.Sprintf("    %s: %s\n", k, v)
	}
	return out
}

type PlanBlock struct {
	NodePos Pos
	Name    string
	Body    *Block
}

func (s *PlanBlock) Pos() Pos    { return s.NodePos }
func (s *PlanBlock) stmtMarker() {}
func (s *PlanBlock) nodeMarker() {}
func (s *PlanBlock) String() string {
	return fmt.Sprintf("plan %q:\n%s", s.Name, s.Body.String())
}

type Program struct {
	NodePos Pos
	Stmts   []Statement
}

func (p *Program) Pos() Pos    { return p.NodePos }
func (p *Program) nodeMarker() {}
func (p *Program) String() string {
	out := ""
	for _, s := range p.Stmts {
		out += s.String() + "\n"
	}
	return out
}
