package parser

import (
	"testing"

	"github.com/jerloo/funny/v2/internal/ast"
	"github.com/stretchr/testify/assert"
)

func TestParser_Empty(t *testing.T) {
	p := New("", "")
	prog, err := p.Parse()
	assert.NoError(t, err)
	assert.Empty(t, prog.Stmts)
}

func TestParser_ExprLiteral(t *testing.T) {
	p := New("42", "")
	prog, err := p.Parse()
	assert.NoError(t, err)
	exprStmt := prog.Stmts[0].(*ast.ExprStmt)
	lit := exprStmt.X.(*ast.LiteralExpr)
	assert.Equal(t, 42, lit.Value)
}

func TestParser_ExprBinary(t *testing.T) {
	p := New("1 + 2 * 3", "")
	prog, err := p.Parse()
	assert.NoError(t, err)
	expr := prog.Stmts[0].(*ast.ExprStmt).X.(*ast.BinaryExpr)
	assert.Equal(t, "+", expr.Op)
	rhs := expr.Right.(*ast.BinaryExpr)
	assert.Equal(t, "*", rhs.Op)
}

func TestParser_ExprSub(t *testing.T) {
	p := New("(1 + 2) * 3", "")
	prog, err := p.Parse()
	assert.NoError(t, err)
	expr := prog.Stmts[0].(*ast.ExprStmt).X.(*ast.BinaryExpr)
	assert.Equal(t, "*", expr.Op)
}

func TestParser_Let(t *testing.T) {
	p := New("let x = 1", "")
	prog, err := p.Parse()
	assert.NoError(t, err)
	let := prog.Stmts[0].(*ast.LetStmt)
	assert.Equal(t, "x", let.Name)
}

func TestParser_LetWithType(t *testing.T) {
	p := New("let x: int = 1", "")
	prog, err := p.Parse()
	assert.NoError(t, err)
	let := prog.Stmts[0].(*ast.LetStmt)
	assert.Equal(t, "int", let.TypeAnn)
}

func TestParser_Assign(t *testing.T) {
	p := New("x = 2", "")
	prog, err := p.Parse()
	assert.NoError(t, err)
	assign := prog.Stmts[0].(*ast.AssignStmt)
	assert.Equal(t, "2", assign.Value.String())
}

func TestParser_ExprStatement(t *testing.T) {
	p := New("foo()", "")
	prog, err := p.Parse()
	assert.NoError(t, err)
	exprStmt := prog.Stmts[0].(*ast.ExprStmt)
	call := exprStmt.X.(*ast.CallExpr)
	assert.Equal(t, "foo", call.Func.String())
}

func TestParser_If(t *testing.T) {
	src := "if x > 0:\n    print(\"pos\")\n"
	p := New(src, "")
	prog, err := p.Parse()
	assert.NoError(t, err)
	ifs := prog.Stmts[0].(*ast.IfStmt)
	assert.NotNil(t, ifs.Then)
}

func TestParser_IfElseIf(t *testing.T) {
	src := "if x:\n    a\nelif y:\n    b\nelse:\n    c\n"
	p := New(src, "")
	prog, err := p.Parse()
	assert.NoError(t, err)
	ifs := prog.Stmts[0].(*ast.IfStmt)
	assert.NotNil(t, ifs.ElseIf)
	assert.NotNil(t, ifs.ElseBlock)
}

func TestParser_For(t *testing.T) {
	src := "for i in items:\n    print(i)\n"
	p := New(src, "")
	prog, err := p.Parse()
	assert.NoError(t, err)
	fs := prog.Stmts[0].(*ast.ForStmt)
	assert.Equal(t, "i", fs.Name)
}

func TestParser_While(t *testing.T) {
	src := "while x > 0:\n    x = x - 1\n"
	p := New(src, "")
	prog, err := p.Parse()
	assert.NoError(t, err)
	ws := prog.Stmts[0].(*ast.WhileStmt)
	assert.NotNil(t, ws.Body)
}

func TestParser_Return(t *testing.T) {
	p := New("return 42", "")
	prog, err := p.Parse()
	assert.NoError(t, err)
	ret := prog.Stmts[0].(*ast.ReturnStmt)
	assert.Equal(t, "42", ret.Value.String())
}

func TestParser_FnDecl(t *testing.T) {
	src := "fn add(a: int, b: int) -> int:\n    return a + b\n"
	p := New(src, "")
	prog, err := p.Parse()
	assert.NoError(t, err)
	fn := prog.Stmts[0].(*ast.FnDecl)
	assert.Equal(t, "add", fn.Name)
	assert.Equal(t, "int", fn.RetType)
	assert.Len(t, fn.Params, 2)
}

func TestParser_StructDecl(t *testing.T) {
	src := "struct User:\n    name: str\n    age: int\n"
	p := New(src, "")
	prog, err := p.Parse()
	assert.NoError(t, err)
	s := prog.Stmts[0].(*ast.StructDecl)
	assert.Equal(t, "User", s.Name)
	assert.Len(t, s.Fields, 2)
}

func TestParser_PubFn(t *testing.T) {
	src := "pub fn hello() -> int:\n    return 1\n"
	p := New(src, "")
	prog, err := p.Parse()
	assert.NoError(t, err)
	fn := prog.Stmts[0].(*ast.FnDecl)
	assert.True(t, fn.Pub)
}
