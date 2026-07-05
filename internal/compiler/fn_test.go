// v2/internal/compiler/fn_test.go
package compiler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jiejie-dev/funny/internal/bytecode"
	"github.com/jiejie-dev/funny/internal/vm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompile_FnDecl(t *testing.T) {
	src := `fn add(a: int, b: int) -> int:
    return a + b
`
	mod := compileExpr(t, src)
	require.Len(t, mod.Functions, 2) // main + add
	assert.Equal(t, "main", mod.Functions[0].Name)
	assert.Equal(t, "add", mod.Functions[1].Name)
	assert.Equal(t, 2, mod.Functions[1].Arity)
	var hasReturn bool
	for _, instr := range mod.Functions[1].Code {
		if instr.Op == bytecode.RETURN {
			hasReturn = true
			break
		}
	}
	assert.True(t, hasReturn)
}

func TestCompile_Call(t *testing.T) {
	src := `fn add(a: int, b: int) -> int:
    return a + b
let r = add(1, 2)
`
	mod := compileExpr(t, src)
	var hasCall bool
	for _, instr := range mod.Functions[0].Code {
		if instr.Op == bytecode.CALL {
			hasCall = true
			break
		}
	}
	assert.True(t, hasCall)
}

func TestCompile_Return(t *testing.T) {
	src := `fn foo() -> int:
    return 42
`
	mod := compileExpr(t, src)
	var hasReturn bool
	for _, instr := range mod.Functions[1].Code {
		if instr.Op == bytecode.RETURN {
			hasReturn = true
			break
		}
	}
	assert.True(t, hasReturn)
}

// Regression test: compileFnDecl used to reset c.scopes to a brand-new
// empty map (instead of saving/restoring the enclosing scope) after
// compiling a function body, so any top-level local declared *before* the
// `fn` became permanently unreachable by name and fell back to an
// unimplemented LOAD_GLOBAL lookup for every subsequent reference.
func TestCompile_TopLevelVarSurvivesFnDeclInBetween_RunsOnVM(t *testing.T) {
	src := `let a = 10
fn add_one(x: int) -> int:
    return x + 1
a + 5
`
	mod := compileExpr(t, src)
	got, err := vm.New(mod).Run()
	require.NoError(t, err)
	assert.Equal(t, 15, got)
}

func TestCompile_TopLevelVarSurvivesMultipleFnDecls_RunsOnVM(t *testing.T) {
	src := `let a = 10
fn add_one(x: int) -> int:
    return x + 1
fn double(x: int) -> int:
    return x * 2
add_one(double(a))
`
	mod := compileExpr(t, src)
	got, err := vm.New(mod).Run()
	require.NoError(t, err)
	assert.Equal(t, 21, got)
}

// Regression test: c.varTypes is indexed by local slot number, which is
// *not* globally unique across functions (each function's own locals start
// at slot 0). Without saving/restoring c.varTypes around a nested
// function's compilation, a top-level variable and an unrelated function
// parameter that happen to share a slot number could clobber each other's
// recorded value type, corrupting codegen for type-sensitive operators
// like `+` (int add vs. string concat).
func TestCompile_VarTypeSurvivesFnDeclWithConflictingSlot_RunsOnVM(t *testing.T) {
	src := `let name = "alice"
fn greet(n: int) -> int:
    return n + 100
name + " smith"
`
	mod := compileExpr(t, src)
	got, err := vm.New(mod).Run()
	require.NoError(t, err)
	assert.Equal(t, "alice smith", got)
}

func TestCompile_CallBuiltin(t *testing.T) {
	src := `println(42)`
	mod := compileExpr(t, src)
	var hasCallBuiltin bool
	for _, instr := range mod.Functions[0].Code {
		if instr.Op == bytecode.CALL_BUILTIN {
			hasCallBuiltin = true
			break
		}
	}
	assert.True(t, hasCallBuiltin)
}

// Regression test: compileCall used to report valNil as the value type of
// *every* builtin call regardless of what it actually returns, so using a
// builtin's result directly as an operand of a typed arithmetic/comparison
// operator failed with "compileBinary: type mismatch nil vs X" even though
// the exact same source ran fine under the tree-walking evaluator (which
// has no static value-type tracking to trip over). builtinValueType fixes
// this by inferring a concrete type for the builtins that return one.
func TestCompile_BuiltinResultUsableInComparison_RunsOnVM(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want any
	}{
		{"len_int", `println(len("hello") > 0)`, nil},
		{"sqrt_float", `println(sqrt(4.0) < 3.0)`, nil},
		{"abs_preserves_int", `let n = -5
println(abs(n) + 1)`, nil},
		{"abs_preserves_float", `let f = -2.5
println(abs(f) + 1.0)`, nil},
		{"str_upper_concat", `println(str_upper("a") + "b")`, nil},
		{"str_contains_and", `println(str_contains("abc", "b") and true)`, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mod := compileExpr(t, c.src)
			_, err := vm.New(mod).Run()
			require.NoError(t, err)
		})
	}
}

// TestCompile_StructFieldArithmetic_RunsOnVM is a regression test:
// compileField used to unconditionally report every field access as
// valStr regardless of the field's real type, so a non-string struct
// field used in a typed operator (`p.x * p.x`, `item.price + tax`, ...)
// failed to compile with a confusing "unsupported op * for str".
func TestCompile_StructFieldArithmetic_RunsOnVM(t *testing.T) {
	src := `struct Point:
    x: int
    y: int
let p = Point(x: 3, y: 4)
p.x * p.x + p.y * p.y
`
	mod := compileExpr(t, src)
	got, err := vm.New(mod).Run()
	require.NoError(t, err)
	assert.Equal(t, 25, got)
}

// TestCompile_StructFieldStillWorksAsResultTag_RunsOnVM guards the
// deliberate special case in compileField: a `.tag` access on an
// object whose static type isn't a tracked struct (e.g. the Result
// returned by ok()/err(), which the compiler doesn't model as a struct)
// must still be treated as a string, since real-world code compares it
// directly against a string literal (`r.tag == "err"`).
func TestCompile_StructFieldStillWorksAsResultTag_RunsOnVM(t *testing.T) {
	src := `let r = ok(42)
r.tag == "ok"
`
	mod := compileExpr(t, src)
	got, err := vm.New(mod).Run()
	require.NoError(t, err)
	assert.Equal(t, true, got)
}

// TestCompile_ListParamElementType_RunsOnVM is a regression test:
// annotationValueType didn't parse `list[T]` annotations at all, so a
// `list[int]` function parameter fell back to valNil ("unknown"), and a
// `for` loop over it produced an untyped loop variable that failed to
// compile the moment it was used in a typed operator like `x > 0`.
func TestCompile_ListParamElementType_RunsOnVM(t *testing.T) {
	src := `fn count_positive(xs: list[int]) -> int:
    let c = 0
    for x in xs:
        if x > 0:
            c = c + 1
    return c
count_positive([1, -2, 3, -4, 5])
`
	mod := compileExpr(t, src)
	got, err := vm.New(mod).Run()
	require.NoError(t, err)
	assert.Equal(t, 3, got)
}

// TestCompile_ElifChain_AllBranchesReachable_RunsOnVM is a regression
// test: compileIf used to never look at ast.IfStmt.ElseIf at all. The
// parser hoists an elif chain's trailing `else:` block onto the
// *outermost* IfStmt's ElseBlock (see parseIf's comment and
// compileIfChain's doc comment for why), so the old code - which only
// ever compiled n.Then and n.ElseBlock - treated any "if/elif.../else"
// chain as just "if cond: then else: <final else>", silently skipping
// every elif's condition and body entirely.
func TestCompile_ElifChain_AllBranchesReachable_RunsOnVM(t *testing.T) {
	src := `fn classify(status: int) -> str:
    if status < 300:
        return "2xx"
    elif status < 400:
        return "3xx"
    elif status < 500:
        return "4xx"
    elif status < 600:
        return "5xx"
    else:
        return "other"

classify(404)
`
	mod := compileExpr(t, src)
	got, err := vm.New(mod).Run()
	require.NoError(t, err)
	assert.Equal(t, "4xx", got)
}

// TestCompile_ElifChain_NoTrailingElse_RunsOnVM covers an elif chain with
// no final `else:` at all, which must fall through to nothing (not error,
// not the wrong branch) when every condition is false.
func TestCompile_ElifChain_NoTrailingElse_RunsOnVM(t *testing.T) {
	src := `let status = 999
let label = "unset"
if status < 300:
    label = "2xx"
elif status < 400:
    label = "3xx"
elif status < 500:
    label = "4xx"
label
`
	mod := compileExpr(t, src)
	got, err := vm.New(mod).Run()
	require.NoError(t, err)
	assert.Equal(t, "unset", got)
}

// TestCompile_IndexIntoListParam_PreservesElementType_RunsOnVM is a
// regression test: compileIndex used to always report valNil regardless
// of the indexed object's tracked type, even though this compiler
// already tracks a list-valued expression's type as its *element* type
// (see compileList/annotationValueType) - so `xs[0]` losing that type
// broke anything built on top of it, like `xs[0].field` or `xs[i] + 1`.
func TestCompile_IndexIntoListParam_PreservesElementType_RunsOnVM(t *testing.T) {
	src := `struct Point:
    x: int
    y: int

fn first_x(pts: list[Point]) -> int:
    let p = pts[0]
    return p.x + 1

first_x([Point(x: 10, y: 20), Point(x: 99, y: 0)])
`
	mod := compileExpr(t, src)
	got, err := vm.New(mod).Run()
	require.NoError(t, err)
	assert.Equal(t, 11, got)
}

// TestCompile_ConcreteTypePlusUntrackedResultVal_RunsOnVM is a
// regression test: compileBinary used to hard-reject combining a
// concretely-typed operand with a valNil ("statically untracked", not
// necessarily an actual nil) one - e.g. `str + result.val` where
// result.val comes off a Result this compiler doesn't model field types
// for. The type checker (a separate, authoritative pass) already accepts
// this - http_get's Ok payload is a real string at runtime - so the
// compiler's stricter same-valueType-on-both-sides check broke code the
// type checker had already approved.
func TestCompile_ConcreteTypePlusUntrackedResultVal_RunsOnVM(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hi"))
	}))
	defer srv.Close()

	src := `fn describe(url: str) -> str:
    let result = http_get(url)
    if result.tag == "ok":
        return "ok: " + result.val
    return "unreachable: " + result.val

describe("` + srv.URL + `")
`
	mod := compileExpr(t, src)
	got, err := vm.New(mod).Run()
	require.NoError(t, err)
	assert.Equal(t, "ok: hi", got)
}

func TestBuiltinValueType(t *testing.T) {
	cases := []struct {
		name     string
		argTypes []valueType
		want     valueType
	}{
		{"len", nil, valInt},
		{"to_int", nil, valInt},
		{"now", nil, valInt},
		{"sqrt", nil, valFloat},
		{"pow", nil, valFloat},
		{"abs", []valueType{valInt}, valInt},
		{"abs", []valueType{valFloat}, valFloat},
		{"abs", []valueType{valNil}, valNil},
		{"to_str", nil, valStr},
		{"type_of", nil, valStr},
		{"str_upper", nil, valStr},
		{"str_lower", nil, valStr},
		{"regex_replace", nil, valStr},
		{"env_get", nil, valStr},
		{"time_format", nil, valStr},
		{"md5", nil, valStr},
		{"sha256", nil, valStr},
		{"b64_encode", nil, valStr},
		{"str_contains", nil, valBool},
		{"regex_match", nil, valBool},
		{"file_exists", nil, valBool},
		{"file_read", nil, valNil},
		{"println", nil, valNil},
	}
	for _, c := range cases {
		got := builtinValueType(c.name, c.argTypes)
		assert.Equal(t, c.want, got, "%s(%v)", c.name, c.argTypes)
	}
}
