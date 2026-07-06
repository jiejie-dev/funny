package types

import (
	"testing"

	"github.com/jiejie-dev/funny/v2/internal/ast"
	"github.com/jiejie-dev/funny/v2/internal/parser"
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

func TestCheck_MapLiteral_InfersKeyValueTypes(t *testing.T) {
	env := NewEnv(nil)
	got, err := CheckExpr(parseExpr(t, `{"a": 1, "b": 2}`), env)
	require.NoError(t, err)
	assert.True(t, got.Equal(Map{Key: Primitive("str"), Value: Primitive("int")}), "got %s", got)
}

func TestCheck_MapLiteral_EmptyRequiresAnnotation(t *testing.T) {
	env := NewEnv(nil)
	_, err := CheckExpr(parseExpr(t, `{}`), env)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "E2011")
}

func TestCheck_MapLiteral_MismatchedValueTypeErrors(t *testing.T) {
	env := NewEnv(nil)
	_, err := CheckExpr(parseExpr(t, `{"a": 1, "b": "two"}`), env)
	require.Error(t, err)
}

func TestCheck_MapLiteral_MismatchedKeyTypeErrors(t *testing.T) {
	env := NewEnv(nil)
	_, err := CheckExpr(parseExpr(t, `{"a": 1, 2: 3}`), env)
	require.Error(t, err)
}

func TestCheck_MapLiteral_LetWithAnnotation(t *testing.T) {
	src := "let m: map[str, int] = {\n    \"a\": 1,\n    \"b\": 2,\n}\n"
	p := parser.New(src, "t")
	prog, err := p.Parse()
	require.NoError(t, err)
	env := NewEnv(nil)
	require.NoError(t, Check(prog, env))
	vt, ok := env.LookupVar("m")
	require.True(t, ok)
	assert.True(t, vt.Equal(Map{Key: Primitive("str"), Value: Primitive("int")}))
}

func TestCheck_IndexExpr_MapReadUsesKeyType(t *testing.T) {
	env := NewEnv(nil)
	env.DeclareVar("m", Map{Key: Primitive("str"), Value: Primitive("int")})
	got, err := CheckExpr(parseExpr(t, `m["a"]`), env)
	require.NoError(t, err)
	assert.True(t, got.Equal(Primitive("int")))
}

func TestCheck_IndexExpr_MapWrongKeyTypeErrors(t *testing.T) {
	env := NewEnv(nil)
	env.DeclareVar("m", Map{Key: Primitive("str"), Value: Primitive("int")})
	_, err := CheckExpr(parseExpr(t, `m[1]`), env)
	require.Error(t, err)
}

func TestCheck_Assign_IndexIntoMap(t *testing.T) {
	src := "let m: map[str, int] = {\"a\": 1}\nm[\"a\"] = 2\nm[\"b\"] = 3\n"
	p := parser.New(src, "t")
	prog, err := p.Parse()
	require.NoError(t, err)
	env := NewEnv(nil)
	require.NoError(t, Check(prog, env))
}

func TestCheck_Assign_IndexIntoMap_WrongValueTypeErrors(t *testing.T) {
	src := "let m: map[str, int] = {\"a\": 1}\nm[\"a\"] = \"oops\"\n"
	p := parser.New(src, "t")
	prog, err := p.Parse()
	require.NoError(t, err)
	env := NewEnv(nil)
	require.Error(t, Check(prog, env))
}

func TestCheck_Assign_IndexIntoList(t *testing.T) {
	src := "let xs: list[int] = [1, 2, 3]\nxs[0] = 99\n"
	p := parser.New(src, "t")
	prog, err := p.Parse()
	require.NoError(t, err)
	env := NewEnv(nil)
	require.NoError(t, Check(prog, env))
}

func TestCheck_FString_ValidInterpolation(t *testing.T) {
	env := NewEnv(nil)
	env.DeclareVar("name", Primitive("str"))
	got, err := CheckExpr(parseExpr(t, `f"hello {name}"`), env)
	require.NoError(t, err)
	assert.True(t, got.Equal(Primitive("str")))
}

func TestCheck_FString_UndefinedVariableErrors(t *testing.T) {
	env := NewEnv(nil)
	_, err := CheckExpr(parseExpr(t, `f"hello {missing}"`), env)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "E2001")
}

func TestCheck_FString_InvalidSpecErrors(t *testing.T) {
	env := NewEnv(nil)
	env.DeclareVar("x", Primitive("int"))
	_, err := CheckExpr(parseExpr(t, `f"{x:zz}"`), env)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "E2090")
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

// TestCheck_ExtendedStdlibBuiltins guards against the regex/env/file/http/
// crypto/jwt/sql builtins silently falling out of builtinTypeNames again -
// they were implemented in internal/vm/builtins.go and documented in
// docs/language-manual.md, but were never registered here, so every call
// to them failed with E2002 "undefined function" before the compiler (and
// therefore the VM) ever saw them.
func TestCheck_ExtendedStdlibBuiltins(t *testing.T) {
	srcs := []string{
		`regex_match("a.c", "abc")`,
		`regex_replace("a", "b", "c")`,
		`env_get("HOME")`,
		`file_read("x")`,
		`file_exists("x")`,
		`http_get("http://x")`,
		`md5("x")`,
		`sha256("x")`,
		`b64_encode("x")`,
		`b64_decode("x")`,
		`jwt_encode("h", "c", "s")`,
		`jwt_decode("t", "s")`,
		`sql_open(":memory:")`,
	}
	for _, src := range srcs {
		env := NewEnv(nil)
		_, err := CheckExpr(parseExpr(t, src), env)
		assert.NoError(t, err, src)
	}
}

// TestCheck_ExtendedStdlibBuiltins_ResultReturnsSupportTry confirms the `?`
// operator type-checks against builtins whose VM implementation
// consistently returns a Result on both the success and failure path.
func TestCheck_ExtendedStdlibBuiltins_ResultReturnsSupportTry(t *testing.T) {
	srcs := []string{
		`file_read("x")?`,
		`http_get("http://x")?`,
		`b64_decode("x")?`,
		`jwt_decode("t", "s")?`,
	}
	for _, src := range srcs {
		env := NewEnv(nil)
		_, err := CheckExpr(parseExpr(t, src), env)
		assert.NoError(t, err, src)
	}
}

// TestCheck_BuiltinReturnType_ConcreteTypes is a regression test:
// builtinReturnType used to report every non-Result builtin as
// Primitive("any"), which has no special-cased compatibility with
// concrete types in Equal - so e.g. `fn f() -> float: return sqrt(x)`
// failed with a spurious "expected float, got any" mismatch even though
// sqrt always returns a float.
func TestCheck_BuiltinReturnType_ConcreteTypes(t *testing.T) {
	cases := []struct {
		src  string
		want Type
	}{
		{`len("x")`, Primitive("int")},
		{`to_int("1")`, Primitive("int")},
		{`now()`, Primitive("int")},
		{`sqrt(4.0)`, Primitive("float")},
		{`pow(2.0, 3.0)`, Primitive("float")},
		{`abs(-1)`, Primitive("int")},
		{`abs(-1.5)`, Primitive("float")},
		{`to_str(1)`, Primitive("str")},
		{`type_of(1)`, Primitive("str")},
		{`str_contains("a", "b")`, Primitive("bool")},
		{`regex_match("a", "b")`, Primitive("bool")},
		{`file_exists("x")`, Primitive("bool")},
		{`str_split("a,b", ",")`, List{Elem: Primitive("str")}},
		{`to_float("3.14")`, Primitive("float")},
		{`jwt_encode("h", "c", "s")`, Primitive("str")},
	}
	for _, c := range cases {
		env := NewEnv(nil)
		got, err := CheckExpr(parseExpr(t, c.src), env)
		require.NoError(t, err, c.src)
		assert.True(t, got.Equal(c.want), "%s: got %s want %s", c.src, got, c.want)
	}
}

// TestCheck_StrSplit_IndexableAndIterable is a regression test: before
// str_split had a concrete List(str) return type, both indexing into its
// result (`str_split(...)[0]`) and iterating over it (`for p in
// str_split(...)`) failed type-checking with E2050 "cannot index into
// any" / "for-in requires list, got any", even though the VM always
// returns an actual list of strings at runtime.
func TestCheck_StrSplit_IndexableAndIterable(t *testing.T) {
	src := `
let parts = str_split("a,b,c", ",")
let first = parts[0]
for p in parts:
    println(p)
`
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	env := NewEnv(nil)
	require.NoError(t, Check(prog, env))
}

// TestCheck_LetEmptyListWithAnnotation_Allowed is a regression test:
// checkLet used to type-check the RHS *before* ever consulting the type
// annotation, so `let xs: list[int] = []` failed with E2011 "cannot infer
// type of empty list" even though the annotation fully determines the
// type - the only way to seed an accumulator that starts empty (e.g. for
// use with append() in a loop).
func TestCheck_LetEmptyListWithAnnotation_Allowed(t *testing.T) {
	p := parser.New(`let xs: list[int] = []`, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	env := NewEnv(nil)
	require.NoError(t, Check(prog, env))
	got, ok := env.LookupVar("xs")
	require.True(t, ok)
	assert.True(t, got.Equal(List{Elem: Primitive("int")}))
}

// TestCheck_LetEmptyListWithoutAnnotation_StillErrors ensures the E2011
// diagnostic is preserved when there's genuinely no way to infer the
// element type.
func TestCheck_LetEmptyListWithoutAnnotation_StillErrors(t *testing.T) {
	p := parser.New(`let xs = []`, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	env := NewEnv(nil)
	err = Check(prog, env)
	require.Error(t, err)
}

// TestCheck_ResultReturningBuiltins_ConcreteOkType is a regression test:
// file_read/http_get/b64_decode's Result Ok payload is always a string
// in practice (file contents / response body / decoded bytes as text),
// but was reported as Primitive("any") - so the extremely common
// "unwrap and use the body as a string" pattern (`result.val + "..."`,
// or returning it from a `-> str` function) failed to type-check with
// "expected str, got any". jwt_decode's Ok payload is a genuine claims
// map, so it's deliberately left at "any".
func TestCheck_ResultReturningBuiltins_ConcreteOkType(t *testing.T) {
	cases := []struct {
		src     string
		wantOk  Type
		wantErr Type
	}{
		{`file_read("x")`, Primitive("str"), Primitive("str")},
		{`http_get("x")`, Primitive("str"), Primitive("str")},
		{`b64_decode("x")`, Primitive("str"), Primitive("str")},
		{`jwt_decode("x", "y")`, Primitive("any"), Primitive("str")},
	}
	for _, c := range cases {
		env := NewEnv(nil)
		got, err := CheckExpr(parseExpr(t, c.src), env)
		require.NoError(t, err, c.src)
		r, ok := got.(Result)
		require.True(t, ok, "%s: expected Result, got %T", c.src, got)
		assert.True(t, r.Ok.Equal(c.wantOk), "%s: Ok = %s want %s", c.src, r.Ok, c.wantOk)
		assert.True(t, r.Err.Equal(c.wantErr), "%s: Err = %s want %s", c.src, r.Err, c.wantErr)
	}
}

// TestCheck_BuiltinCallArgs_AreTypeChecked is a regression test: builtin
// call arguments used to never be visited by CheckExpr at all (the
// builtinTypeNames branch returned before looking at n.Args), so
// something like `sqrt(undefined_var)` type-checked fine and only
// surfaced as a confusing "vm: unsupported op LOAD_GLOBAL" at runtime
// instead of a proper E2001 here.
func TestCheck_BuiltinCallArgs_AreTypeChecked(t *testing.T) {
	env := NewEnv(nil)
	_, err := CheckExpr(parseExpr(t, "sqrt(undefined_var_xyz)"), env)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "E2001")
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

func TestIntegration_TypeCheck_StructTypeAnnotation(t *testing.T) {
	// Regression test: `let p: Point = ...` used to fail with a spurious
	// type mismatch because ParseType has no environment access and could
	// only produce an opaque Primitive("Point") for a bare struct name,
	// which never compared equal to the real Struct type.
	src := `struct Point:
    x: int
    y: int

let p: Point = Point(x: 1, y: 2)
let px: int = p.x
`
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	env := NewEnv(nil)
	err = Check(prog, env)
	assert.NoError(t, err)
}

func TestIntegration_TypeCheck_StructTypeAnnotation_FunctionParamAndReturn(t *testing.T) {
	src := `struct Point:
    x: int
    y: int

fn make_point(x: int, y: int) -> Point:
    return Point(x: x, y: y)

fn sum_x(p: Point) -> int:
    return p.x

let p: Point = make_point(1, 2)
let total: int = sum_x(p)
`
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	env := NewEnv(nil)
	err = Check(prog, env)
	assert.NoError(t, err)
}

func TestIntegration_TypeCheck_StructTypeAnnotation_NestedInStructFieldAndList(t *testing.T) {
	src := `struct Point:
    x: int
    y: int

struct Wrapper:
    inner: Point

let p: Point = Point(x: 1, y: 2)
let w: Wrapper = Wrapper(inner: p)
let pts: list[Point] = [p, w.inner]
`
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	env := NewEnv(nil)
	err = Check(prog, env)
	assert.NoError(t, err)
}

func TestIntegration_TypeCheck_StructTypeAnnotation_MismatchStillErrors(t *testing.T) {
	// Make sure the fix doesn't accidentally make struct-typed annotations
	// vacuously true: a genuine mismatch must still be caught.
	src := `struct Point:
    x: int
    y: int

struct Other:
    z: int

let o: Other = Other(z: 1)
let p: Point = o
`
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	env := NewEnv(nil)
	err = Check(prog, env)
	assert.Error(t, err)
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

func TestCheck_Match(t *testing.T) {
	src := `let code: int = 200
match code:
    200 =>
        let ok = true
    404 =>
        let missing = true
    _ =>
        let other = true
`
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	env := NewEnv(nil)
	require.NoError(t, Check(prog, env))
}

func TestCheck_MatchPatternMismatch(t *testing.T) {
	src := `let code: int = 200
match code:
    "bad" =>
        let x = 1
`
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	env := NewEnv(nil)
	err = Check(prog, env)
	require.Error(t, err)
}

func TestCheck_BreakOutsideLoop(t *testing.T) {
	src := `break
`
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	env := NewEnv(nil)
	err = Check(prog, env)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "E2012")
}

func TestCheck_ContinueInLoop(t *testing.T) {
	src := `for i in [1, 2, 3]:
    continue
`
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	env := NewEnv(nil)
	require.NoError(t, Check(prog, env))
}

func TestCheck_ExprStmt_UndefinedCallErrors(t *testing.T) {
	src := `println(undefined_fn())
`
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	env := NewEnv(nil)
	err = Check(prog, env)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "E2002")
}

func TestCheck_ExprStmt_ValidCallOk(t *testing.T) {
	src := `println("hello")
`
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	env := NewEnv(nil)
	require.NoError(t, Check(prog, env))
}

func TestCheck_ExprStmt_TypeMismatchInCallArgs(t *testing.T) {
	src := `println(1 + "two")
`
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	env := NewEnv(nil)
	err = Check(prog, env)
	require.Error(t, err)
}
