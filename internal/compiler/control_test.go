// v2/internal/compiler/control_test.go
package compiler

import (
	"testing"

	"github.com/jiejie-dev/funny/v2/internal/bytecode"
	"github.com/jiejie-dev/funny/v2/internal/vm"
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

// TestCompile_For_RunsOnVM_VisitsEveryElement is a regression test for a
// severe bug: compileFor initialized the loop index with a hardcoded
// `Emit(bytecode.PUSH_INT, 0)`, i.e. Arg=0, which the VM interprets as
// "push Constants[0]" - not literal 0. Since compiling the loop's iterable
// (e.g. a list literal) registers its own constants *first*,
// Constants[0] was whatever value happened to end up there (e.g. the
// list's first element), not the integer 0. In practice this meant the
// loop index silently started at 1 instead of 0 for almost any real
// iterable, permanently skipping the first element on every `for`
// loop - and no existing test caught it because the prior test only
// asserted on which opcodes were emitted, never actually ran the bytecode.
func TestCompile_For_RunsOnVM_VisitsEveryElement(t *testing.T) {
	mod := compileExpr(t, `let seen = ""
for i in ["a", "b", "c"]:
    seen = seen + i
seen
`)
	got, err := vm.New(mod).Run()
	require.NoError(t, err)
	assert.Equal(t, "abc", got)
}

// TestCompile_For_RunsOnVM_SumsAllElements is a second, numeric variant of
// the same regression: with the bug, `for i in [1, 2, 3]: sum = sum + i`
// produced 5 (2+3) instead of 6 (1+2+3).
func TestCompile_For_RunsOnVM_SumsAllElements(t *testing.T) {
	mod := compileExpr(t, `let sum = 0
for i in [1, 2, 3]:
    sum = sum + i
sum
`)
	got, err := vm.New(mod).Run()
	require.NoError(t, err)
	assert.Equal(t, 6, got)
}