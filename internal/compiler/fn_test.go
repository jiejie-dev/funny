// v2/internal/compiler/fn_test.go
package compiler

import (
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