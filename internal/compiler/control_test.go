// v2/internal/compiler/control_test.go
package compiler

import (
	"testing"

	"github.com/jerloo/funny/internal/bytecode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompile_If_ThenTaken(t *testing.T) {
	mod := compileExpr(t, `if true:
    let x = 1
`)
	fn := mod.Functions[0]
	require.GreaterOrEqual(t, len(fn.Code), 4)
	assert.Equal(t, bytecode.PUSH_BOOL, fn.Code[0].Op)
	assert.Equal(t, bytecode.JUMP_IF_FALSE, fn.Code[1].Op)
	skipTarget := fn.Code[1].Arg
	assert.Greater(t, skipTarget, 1)
}

func TestCompile_IfThenElse(t *testing.T) {
	mod := compileExpr(t, `if false:
    let x = 1
else:
    let y = 2
`)
	fn := mod.Functions[0]
	var jumps []int
	for _, instr := range fn.Code {
		if instr.Op == bytecode.JUMP || instr.Op == bytecode.JUMP_IF_FALSE {
			jumps = append(jumps, instr.Arg)
		}
	}
	assert.GreaterOrEqual(t, len(jumps), 2)
}

func TestCompile_While(t *testing.T) {
	mod := compileExpr(t, `let x = 0
while x < 3:
    x = x + 1
`)
	fn := mod.Functions[0]
	var hasJumpIfFalse, hasJump bool
	for _, instr := range fn.Code {
		if instr.Op == bytecode.JUMP_IF_FALSE {
			hasJumpIfFalse = true
		}
		if instr.Op == bytecode.JUMP {
			hasJump = true
		}
	}
	assert.True(t, hasJumpIfFalse)
	assert.True(t, hasJump)
}

func TestCompile_AssignAfterIf(t *testing.T) {
	mod := compileExpr(t, `let x = 0
if true:
    x = 1
x
`)
	fn := mod.Functions[0]
	var storeLocalCount int
	for _, instr := range fn.Code {
		if instr.Op == bytecode.STORE_LOCAL {
			storeLocalCount++
		}
	}
	assert.GreaterOrEqual(t, storeLocalCount, 2) // let x = 0 + x = 1
}

func TestCompile_For(t *testing.T) {
	mod := compileExpr(t, `for i in [1, 2, 3]:
    let x = i
`)
	fn := mod.Functions[0]
	var hasBuildList, hasIndex, hasJump bool
	for _, instr := range fn.Code {
		switch instr.Op {
		case bytecode.BUILD_LIST:
			hasBuildList = true
		case bytecode.INDEX:
			hasIndex = true
		case bytecode.JUMP:
			hasJump = true
		}
	}
	assert.True(t, hasBuildList, "BUILD_LIST for iterable")
	assert.True(t, hasIndex, "INDEX for iteration")
	assert.True(t, hasJump, "JUMP for loop back")
}