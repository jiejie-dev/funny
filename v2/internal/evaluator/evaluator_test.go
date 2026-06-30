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
