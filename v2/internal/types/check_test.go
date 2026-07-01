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

func TestCheck_Let(t *testing.T) {
	src := `let x: int = 42`
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	env := NewEnv(nil)
	err = Check(prog, env)
	assert.NoError(t, err)
	t2, _ := env.LookupVar("x")
	assert.Equal(t, Primitive("int"), t2)
}

func TestCheck_LetInfer(t *testing.T) {
	src := `let x = 42`
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	env := NewEnv(nil)
	err = Check(prog, env)
	assert.NoError(t, err)
	t2, _ := env.LookupVar("x")
	assert.Equal(t, Primitive("int"), t2)
}

func TestCheck_LetTypeMismatch(t *testing.T) {
	src := `let x: int = "hello"`
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	env := NewEnv(nil)
	err = Check(prog, env)
	assert.Error(t, err)
}

func TestCheck_Assign(t *testing.T) {
	src := `let x: int = 1
x = 2`
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	env := NewEnv(nil)
	err = Check(prog, env)
	assert.NoError(t, err)
}

func TestCheck_AssignMismatch(t *testing.T) {
	src := `let x: int = 1
x = "hello"`
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	env := NewEnv(nil)
	err = Check(prog, env)
	assert.Error(t, err)
}

func TestCheck_If_NonBool(t *testing.T) {
	src := `if 42:
    pass`
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	env := NewEnv(nil)
	err = Check(prog, env)
	assert.Error(t, err)
}

func TestCheck_For_NonList(t *testing.T) {
	src := `for i in 42:
    pass`
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	env := NewEnv(nil)
	err = Check(prog, env)
	assert.Error(t, err)
}

func TestCheck_ReturnType(t *testing.T) {
	src := `fn foo() -> int:
    return "hello"
`
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	env := NewEnv(nil)
	err = Check(prog, env)
	assert.Error(t, err)
}

func TestCheck_FnDecl_ParamMissingType(t *testing.T) {
	src := `fn foo(a) -> int:
    return a
`
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	env := NewEnv(nil)
	err = Check(prog, env)
	assert.Error(t, err)
}

func TestCheck_StructDecl_FieldMissingType(t *testing.T) {
	src := `struct User:
    name: str
    bad
`
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	env := NewEnv(nil)
	err = Check(prog, env)
	assert.Error(t, err)
}

func TestIntegration_TypeCheck_Basic(t *testing.T) {
	src := `let x: int = 42
let y: float = 3.14
let name: str = "hello"
let sum = x + 0
`
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	env := NewEnv(nil)
	err = Check(prog, env)
	assert.NoError(t, err)
}

func TestIntegration_TypeCheck_Lists(t *testing.T) {
	src := `let xs: list[int] = [1, 2, 3]
let first = xs[0]
`
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	env := NewEnv(nil)
	err = Check(prog, env)
	assert.NoError(t, err)
}

func TestIntegration_TypeCheck_Structs(t *testing.T) {
	src := `struct Point:
    x: int
    y: int

let p = Point(x: 1, y: 2)
let px = p.x
`
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	env := NewEnv(nil)
	err = Check(prog, env)
	assert.NoError(t, err)
}

func TestIntegration_TypeCheck_Functions(t *testing.T) {
	src := `fn add(a: int, b: int) -> int:
    return a + b

let r = add(1, 2)
let s = add("x", "y")
`
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	env := NewEnv(nil)
	err = Check(prog, env)
	assert.Error(t, err)
}

func TestIntegration_TypeCheck_VariousErrors(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{"undefined var", "let y = undefined_xyz"},
		{"if non-bool", `if 42:
    pass`},
		{"for non-list", `for i in 42:
    pass`},
		{"return mismatch", `fn foo() -> int:
    return "hello"
`},
		{"assign mismatch", `let x: int = 1
x = "hello"`},
		{"param missing type", `fn foo(a) -> int:
    return a
`},
	}
	for _, c := range cases {
		p := parser.New(c.src, "")
		prog, err := p.Parse()
		require.NoError(t, err, c.name)
		env := NewEnv(nil)
		err = Check(prog, env)
		assert.Error(t, err, c.name+": expected type error")
	}
}

func TestCheck_TryOperator(t *testing.T) {
	// ok(42) is Result[int, str]; ? keeps the Result type (Err propagates, Ok leaves Result on stack).
	// Accessing .val on the Result returns the inner Ok value.
	src := `let x = ok(42)?.val
`
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	env := NewEnv(nil)
	err = Check(prog, env)
	assert.NoError(t, err)
	t2, _ := env.LookupVar("x")
	assert.Equal(t, Primitive("int"), t2)
}

func TestCheck_TryOperator_Annotated(t *testing.T) {
	// Same but with explicit type annotation.
	src := `let x: int = ok(42)?.val
`
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	env := NewEnv(nil)
	err = Check(prog, env)
	assert.NoError(t, err)
	t2, _ := env.LookupVar("x")
	assert.Equal(t, Primitive("int"), t2)
}

func TestCheck_TryOperator_NonResult(t *testing.T) {
	// `?` on a non-Result expression is now a transparent pass-through:
	// the value's type is preserved and no Result unwrap is performed.
	src := `let x: int = 42?
`
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	env := NewEnv(nil)
	err = Check(prog, env)
	assert.NoError(t, err)
	t2, _ := env.LookupVar("x")
	assert.Equal(t, Primitive("int"), t2)
}

func TestCheck_MetaBlock(t *testing.T) {
	src := `meta:
    name: "demo"
    version: "1.0"
`
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	env := NewEnv(nil)
	err = Check(prog, env)
	assert.NoError(t, err)
}

func TestCheck_MetaBlock_MissingName(t *testing.T) {
	src := `meta:
    version: "1.0"
`
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	env := NewEnv(nil)
	err = Check(prog, env)
	assert.Error(t, err)
}
