package parser

import (
	"os"
	"testing"

	"github.com/jiejie-dev/funny/v2/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParser_Empty(t *testing.T) {
	p := New("", "")
	prog, err := p.Parse()
	assert.NoError(t, err)
	assert.Empty(t, prog.Stmts)
}

func TestParse_StandaloneComment_DoesNotError(t *testing.T) {
	p := New("let x = 1\n# a comment\nlet y = 2\n", "t")
	prog, err := p.Parse()
	require.NoError(t, err)
	require.Len(t, prog.Stmts, 3)
	_, ok := prog.Stmts[1].(*ast.CommentStmt)
	assert.True(t, ok)
}

func TestParse_TrailingComment_DoesNotError(t *testing.T) {
	p := New("let x = 1  # trailing\n", "t")
	prog, err := p.Parse()
	require.NoError(t, err)
	require.Len(t, prog.Stmts, 2)
	_, ok := prog.Stmts[0].(*ast.LetStmt)
	assert.True(t, ok)
	c, ok := prog.Stmts[1].(*ast.CommentStmt)
	require.True(t, ok)
	assert.Equal(t, c.Pos().Line, prog.Stmts[0].Pos().Line)
}

func TestParse_CommentInsideBlock(t *testing.T) {
	p := New("if true:\n    # note\n    let x = 1\n", "t")
	_, err := p.Parse()
	require.NoError(t, err)
}

func TestParse_DocCommentStandalone(t *testing.T) {
	p := New("## doc note\nlet x = 1\n", "t")
	prog, err := p.Parse()
	require.NoError(t, err)
	c, ok := prog.Stmts[0].(*ast.CommentStmt)
	require.True(t, ok)
	assert.True(t, c.Doc)
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

// TestParse_CommentAfterMultiLevelDedent_AttachesToOuterBlock guards against
// a lexer bug where a comment line dedenting across more than one nesting
// level (e.g. from inside a nested if/else back out to a plan's top level)
// was incorrectly attached to the inner block instead of the outer one,
// because only one DEDENT was emitted instead of the two required.
func TestParse_CommentAfterMultiLevelDedent_AttachesToOuterBlock(t *testing.T) {
	src := "plan \"p\":\n    step \"a\":\n        if true:\n            let x = 1\n        else:\n            let y = 2\n\n    # outer comment\n    step \"b\":\n        let z = 3\n"
	p := New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	plan := prog.Stmts[0].(*ast.PlanBlock)
	require.Len(t, plan.Body.Statements, 3)
	_, ok := plan.Body.Statements[1].(*ast.CommentStmt)
	assert.True(t, ok, "expected comment as a direct sibling of the plan's steps, got %T", plan.Body.Statements[1])
}

func TestParser_Match(t *testing.T) {
	src := "match x:\n    1 =>\n        a\n    2 =>\n        b\n"
	p := New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	ms := prog.Stmts[0].(*ast.MatchStmt)
	assert.Len(t, ms.Arms, 2)
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

func TestParser_StructDecl_MutFields(t *testing.T) {
	src := "struct Counter:\n    mut count: int\n    label: str\n"
	p := New(src, "")
	prog, err := p.Parse()
	assert.NoError(t, err)
	s := prog.Stmts[0].(*ast.StructDecl)
	assert.Equal(t, "Counter", s.Name)
	require.Len(t, s.Fields, 2)
	assert.True(t, s.Fields[0].Mut)
	assert.Equal(t, "count", s.Fields[0].Name)
	assert.False(t, s.Fields[1].Mut)
	assert.Equal(t, "label", s.Fields[1].Name)
}

func TestParser_PubFn(t *testing.T) {
	src := "pub fn hello() -> int:\n    return 1\n"
	p := New(src, "")
	prog, err := p.Parse()
	assert.NoError(t, err)
	fn := prog.Stmts[0].(*ast.FnDecl)
	assert.True(t, fn.Pub)
}

func TestParser_Meta(t *testing.T) {
	src := "meta:\n    name: \"demo\"\n    version: \"1.0\"\n"
	p := New(src, "")
	prog, err := p.Parse()
	assert.NoError(t, err)
	m := prog.Stmts[0].(*ast.MetaBlock)
	assert.Equal(t, "demo", m.Fields["name"])
}

func TestParser_Plan(t *testing.T) {
	src := "plan \"my_plan\":\n    pass\n"
	p := New(src, "")
	prog, err := p.Parse()
	assert.NoError(t, err)
	pl := prog.Stmts[0].(*ast.PlanBlock)
	assert.Equal(t, "my_plan", pl.Name)
}

func TestParser_Import(t *testing.T) {
	p := New("import \"std/http.fn\"", "")
	prog, err := p.Parse()
	assert.NoError(t, err)
	imp := prog.Stmts[0].(*ast.ImportDecl)
	assert.Equal(t, "std/http.fn", imp.Path)
}

func TestParser_FromFile(t *testing.T) {
	cases := []string{
		"../../testdata/parser/control_flow.fn",
		"../../testdata/parser/function.fn",
	}
	for _, path := range cases {
		data, err := os.ReadFile(path)
		assert.NoError(t, err)
		p := New(string(data), path)
		_, err = p.Parse()
		assert.NoError(t, err, "file=%s", path)
	}
}

func TestParser_TryOperator(t *testing.T) {
	src := `let r = ok(42)?
r.val
`
	p := New(src, "")
	prog, err := p.Parse()
	assert.NoError(t, err)
	require.Len(t, prog.Stmts, 2)
}

func TestParser_TryOperator_OnCall(t *testing.T) {
	src := `let r = divide(10, 2)?
println(r)
`
	p := New(src, "")
	_, err := p.Parse()
	assert.NoError(t, err)
}
