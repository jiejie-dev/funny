// v2/internal/compiler/fn_test.go
package compiler

import (
	"testing"

	"github.com/jerloo/funny/v2/internal/bytecode"
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