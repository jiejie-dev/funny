package types

import (
	"testing"

	"github.com/jerloo/funny/v2/internal/ast"
	"github.com/jerloo/funny/v2/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parseExpr(t *testing.T, src string) ast.Expression {
	t.Helper()
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	return prog.Stmts[0].(*ast.ExprStmt).X
}

func TestCheck_Literal(t *testing.T) {
	cases := []struct {
		src  string
		want Type
	}{
		{"42", Primitive("int")},
		{"3.14", Primitive("float")},
		{`"hi"`, Primitive("str")},
		{"true", Primitive("bool")},
		{"nil", Primitive("nil")},
	}
	for _, c := range cases {
		env := NewEnv(nil)
		got, err := CheckExpr(parseExpr(t, c.src), env)
		assert.NoError(t, err, c.src)
		assert.True(t, got.Equal(c.want), "%s: got %s want %s", c.src, got, c.want)
	}
}

func TestCheck_Variable(t *testing.T) {
	env := NewEnv(nil)
	env.DeclareVar("x", Primitive("int"))
	got, err := CheckExpr(parseExpr(t, "x"), env)
	assert.NoError(t, err)
	assert.Equal(t, Primitive("int"), got)
}

func TestCheck_UndefinedVariable(t *testing.T) {
	env := NewEnv(nil)
	_, err := CheckExpr(parseExpr(t, "undefined_xyz"), env)
	assert.Error(t, err)
}

func TestCheck_Binary_Int(t *testing.T) {
	env := NewEnv(nil)
	got, err := CheckExpr(parseExpr(t, "1 + 2"), env)
	assert.NoError(t, err)
	assert.Equal(t, Primitive("int"), got)
}

func TestCheck_Binary_Mismatch(t *testing.T) {
	env := NewEnv(nil)
	_, err := CheckExpr(parseExpr(t, `1 + "x"`), env)
	assert.Error(t, err)
}

func TestCheck_Binary_Bool(t *testing.T) {
	env := NewEnv(nil)
	got, err := CheckExpr(parseExpr(t, "1 < 2"), env)
	assert.NoError(t, err)
	assert.Equal(t, Primitive("bool"), got)
}

func TestCheck_And_NonBool(t *testing.T) {
	env := NewEnv(nil)
	_, err := CheckExpr(parseExpr(t, "1 and 2"), env)
	assert.Error(t, err)
}

func TestCheck_Unary(t *testing.T) {
	env := NewEnv(nil)
	got, err := CheckExpr(parseExpr(t, "-42"), env)
	assert.NoError(t, err)
	assert.Equal(t, Primitive("int"), got)
	got, err = CheckExpr(parseExpr(t, "not true"), env)
	assert.NoError(t, err)
	assert.Equal(t, Primitive("bool"), got)
}

func TestCheck_UnaryBad(t *testing.T) {
	env := NewEnv(nil)
	_, err := CheckExpr(parseExpr(t, `-"x"`), env)
	assert.Error(t, err)
	_, err = CheckExpr(parseExpr(t, "not 42"), env)
	assert.Error(t, err)
}

func TestCheck_Call(t *testing.T) {
	env := NewEnv(nil)
	env.DeclareFunc("id", Func{Params: []Type{Primitive("int")}, Return: Primitive("int")})
	got, err := CheckExpr(parseExpr(t, "id(42)"), env)
	assert.NoError(t, err)
	assert.Equal(t, Primitive("int"), got)
}

func TestCheck_Call_Undefined(t *testing.T) {
	env := NewEnv(nil)
	_, err := CheckExpr(parseExpr(t, "unknown(42)"), env)
	assert.Error(t, err)
}

func TestCheck_Call_WrongArity(t *testing.T) {
	env := NewEnv(nil)
	env.DeclareFunc("id", Func{Params: []Type{Primitive("int")}, Return: Primitive("int")})
	_, err := CheckExpr(parseExpr(t, "id(1, 2)"), env)
	assert.Error(t, err)
}

func TestCheck_Call_WrongType(t *testing.T) {
	env := NewEnv(nil)
	env.DeclareFunc("id", Func{Params: []Type{Primitive("int")}, Return: Primitive("int")})
	_, err := CheckExpr(parseExpr(t, `id("x")`), env)
	assert.Error(t, err)
}

func TestCheck_Index_List(t *testing.T) {
	env := NewEnv(nil)
	got, err := CheckExpr(parseExpr(t, "[1, 2, 3][0]"), env)
	assert.NoError(t, err)
	assert.Equal(t, Primitive("int"), got)
}

func TestCheck_Index_NonInt(t *testing.T) {
	env := NewEnv(nil)
	_, err := CheckExpr(parseExpr(t, `[1, 2][true]`), env)
	assert.Error(t, err)
}

func TestCheck_Field(t *testing.T) {
	env := NewEnv(nil)
	env.DeclareVar("p", Struct{
		Name:   "Point",
		Fields: map[string]Type{"x": Primitive("int")},
	})
	got, err := CheckExpr(parseExpr(t, "p.x"), env)
	assert.NoError(t, err)
	assert.Equal(t, Primitive("int"), got)
}

func TestCheck_Field_Missing(t *testing.T) {
	env := NewEnv(nil)
	env.DeclareVar("p", Struct{
		Name:   "Point",
		Fields: map[string]Type{"x": Primitive("int")},
	})
	_, err := CheckExpr(parseExpr(t, "p.y"), env)
	assert.Error(t, err)
}

func TestCheck_ListLiteral(t *testing.T) {
	env := NewEnv(nil)
	got, err := CheckExpr(parseExpr(t, "[1, 2, 3]"), env)
	assert.NoError(t, err)
	assert.True(t, got.Equal(List{Primitive("int")}))
}

func TestCheck_ListLiteral_Mismatch(t *testing.T) {
	env := NewEnv(nil)
	_, err := CheckExpr(parseExpr(t, `[1, "x", 3]`), env)
	assert.Error(t, err)
}

func TestCheck_FString(t *testing.T) {
	env := NewEnv(nil)
	got, err := CheckExpr(parseExpr(t, `f"hello"`), env)
	assert.NoError(t, err)
	assert.Equal(t, Primitive("str"), got)
}
