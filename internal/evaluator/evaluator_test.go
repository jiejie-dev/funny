package evaluator

import (
	"testing"

	"github.com/jiejie-dev/funny/v2/internal/ast"
	"github.com/jiejie-dev/funny/v2/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func evalExpr(t *testing.T, src string) any {
	t.Helper()
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	exprStmt := prog.Stmts[0].(*ast.ExprStmt)
	e := New(nil)
	v, err := e.Eval(exprStmt.X)
	require.NoError(t, err)
	return v
}

func execProgram(t *testing.T, src string) *Evaluator {
	t.Helper()
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	e := New(nil)
	require.NoError(t, e.Exec(prog))
	return e
}

func TestEval_IndexExpr_MapRead(t *testing.T) {
	v := evalExpr(t, `{"a": 1, "b": 2}["a"]`)
	assert.Equal(t, 1, v)
}

func TestEval_IndexExpr_MapRead_MissingKeyErrors(t *testing.T) {
	p := parser.New(`{"a": 1}["missing"]`, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	e := New(nil)
	_, err = e.Eval(prog.Stmts[0].(*ast.ExprStmt).X)
	require.Error(t, err)
}

func TestEval_Assign_IndexIntoMap(t *testing.T) {
	e := execProgram(t, "let m = {\"a\": 1}\nm[\"a\"] = 100\nm[\"b\"] = 2\n")
	v, ok := e.Scope().Get("m")
	require.True(t, ok)
	m := v.(map[string]any)
	assert.Equal(t, 100, m["a"])
	assert.Equal(t, 2, m["b"])
}

func TestEval_Assign_IndexIntoList(t *testing.T) {
	e := execProgram(t, "let xs = [10, 20, 30]\nxs[1] = 99\n")
	v, ok := e.Scope().Get("xs")
	require.True(t, ok)
	xs := v.([]any)
	assert.Equal(t, []any{10, 99, 30}, xs)
}

func TestEval_MapLiteral(t *testing.T) {
	v := evalExpr(t, `{"a": 1, "b": 2}`)
	m, ok := v.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, 1, m["a"])
	assert.Equal(t, 2, m["b"])
}

func TestEval_MapLiteral_MultiLine(t *testing.T) {
	src := "{\n    \"a\": 1,\n    \"b\": 2,\n}"
	v := evalExpr(t, src)
	m, ok := v.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, 1, m["a"])
	assert.Equal(t, 2, m["b"])
}

func TestEval_MapLiteral_Empty(t *testing.T) {
	v := evalExpr(t, `{}`)
	m, ok := v.(map[string]any)
	require.True(t, ok)
	assert.Empty(t, m)
}

func TestEval_FString_Interpolation(t *testing.T) {
	e := New(nil)
	e.Scope().Set("name", "world")
	v, err := e.Eval(&ast.FStringExpr{Parts: []ast.FStringPart{
		{Text: "hello "},
		{Expr: &ast.VariableExpr{Name: "name"}},
		{Text: "!"},
	}})
	require.NoError(t, err)
	assert.Equal(t, "hello world!", v)
}

func TestEval_FString_WithSpec(t *testing.T) {
	e := New(nil)
	e.Scope().Set("price", 3.14159)
	v, err := e.Eval(&ast.FStringExpr{Parts: []ast.FStringPart{
		{Expr: &ast.VariableExpr{Name: "price"}, Spec: ".2f"},
	}})
	require.NoError(t, err)
	assert.Equal(t, "3.14", v)
}

func TestEval_FString_ViaParser(t *testing.T) {
	p := parser.New(`f"hi {name}, total {price:.2f}"`, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	e := New(nil)
	e.Scope().Set("name", "alice")
	e.Scope().Set("price", 9.5)
	v, err := e.Eval(prog.Stmts[0].(*ast.ExprStmt).X)
	require.NoError(t, err)
	assert.Equal(t, "hi alice, total 9.50", v)
}

func TestEval_Literal(t *testing.T) {
	assert.Equal(t, 42, evalExpr(t, "42"))
	assert.Equal(t, 3.14, evalExpr(t, "3.14"))
	assert.Equal(t, "hi", evalExpr(t, `"hi"`))
	assert.Equal(t, true, evalExpr(t, "true"))
}

func TestEval_Binary(t *testing.T) {
	assert.Equal(t, 5, evalExpr(t, "2 + 3"))
	assert.Equal(t, true, evalExpr(t, "5 > 3"))
	assert.Equal(t, true, evalExpr(t, "5 == 5"))
	assert.Equal(t, "ab", evalExpr(t, `"a" + "b"`))
}

func TestEval_Variable(t *testing.T) {
	e := New(nil)
	e.scope.Set("x", 10)
	p := parser.New("x + 5", "")
	prog, _ := p.Parse()
	v, _ := e.Eval(prog.Stmts[0].(*ast.ExprStmt).X)
	assert.Equal(t, 15, v)
}

func TestEval_LetAndAssign(t *testing.T) {
	src := `let x = 1
x = 2
`
	p := parser.New(src, "")
	prog, _ := p.Parse()
	e := New(nil)
	require.NoError(t, e.Exec(prog))
	v, _ := e.scope.Get("x")
	assert.Equal(t, 2, v)
}

func TestEval_If(t *testing.T) {
	src := `let x = 10
if x > 5:
    x = 1
else:
    x = 2
`
	p := parser.New(src, "")
	prog, _ := p.Parse()
	e := New(nil)
	require.NoError(t, e.Exec(prog))
	v, _ := e.scope.Get("x")
	assert.Equal(t, 1, v)
}

func TestEval_For(t *testing.T) {
	src := `let sum = 0
for i in [1, 2, 3, 4, 5]:
    sum = sum + i
`
	p := parser.New(src, "")
	prog, _ := p.Parse()
	e := New(nil)
	require.NoError(t, e.Exec(prog))
	v, _ := e.scope.Get("sum")
	assert.Equal(t, 15, v)
}

func TestEval_While(t *testing.T) {
	src := `let x = 0
while x < 5:
    x = x + 1
`
	p := parser.New(src, "")
	prog, _ := p.Parse()
	e := New(nil)
	require.NoError(t, e.Exec(prog))
	v, _ := e.scope.Get("x")
	assert.Equal(t, 5, v)
}

func TestIntegration_Fib(t *testing.T) {
	src := `fn fib(n: int) -> int:
    if n < 2:
        return n
    return fib(n - 1) + fib(n - 2)
let r = fib(10)
`
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	e := New(nil)
	require.NoError(t, e.Exec(prog))
	v, _ := e.scope.Get("r")
	assert.Equal(t, 55, v)
}

func TestIntegration_Sum(t *testing.T) {
	src := `let sum = 0
for i in [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]:
    sum = sum + i
`
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	e := New(nil)
	require.NoError(t, e.Exec(prog))
	v, _ := e.scope.Get("sum")
	assert.Equal(t, 55, v)
}
