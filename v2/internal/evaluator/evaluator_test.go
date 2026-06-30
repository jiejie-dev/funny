package evaluator

import (
	"testing"

	"github.com/jerloo/funny/v2/internal/ast"
	"github.com/jerloo/funny/v2/internal/parser"
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
