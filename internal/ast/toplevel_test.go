package ast

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestImportDecl_String(t *testing.T) {
	i := &ImportDecl{Path: "std/http.fn"}
	assert.Equal(t, `import "std/http.fn"`, i.String())
}

func TestImportDecl_StringWithAlias(t *testing.T) {
	i := &ImportDecl{Path: "std/http.fn", Alias: "h"}
	assert.Equal(t, `import "std/http.fn" as h`, i.String())
}

func TestMetaBlock_String(t *testing.T) {
	m := &MetaBlock{Fields: map[string]string{"name": "demo", "version": "1.0"}}
	out := m.String()
	assert.Contains(t, out, "meta:")
	assert.Contains(t, out, "name: demo")
}

func TestPlanBlock_String(t *testing.T) {
	p := &PlanBlock{
		Name: "my_plan",
		Body: &Block{Statements: []Statement{&ExprStmt{X: &VariableExpr{Name: "x"}}}},
	}
	out := p.String()
	assert.Contains(t, out, `plan "my_plan":`)
}

func TestProgram_String(t *testing.T) {
	p := &Program{
		Stmts: []Statement{
			&LetStmt{Name: "x", Value: &LiteralExpr{Value: 1}},
			&ExprStmt{X: &VariableExpr{Name: "x"}},
		},
	}
	out := p.String()
	assert.Contains(t, out, "let x = 1")
}
