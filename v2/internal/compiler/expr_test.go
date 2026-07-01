package compiler

import (
	"testing"

	"github.com/jerloo/funny/v2/internal/bytecode"
	"github.com/jerloo/funny/v2/internal/parser"
	"github.com/jerloo/funny/v2/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func compileExpr(t *testing.T, src string) *bytecode.Module {
	t.Helper()
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	env := types.NewEnv(nil)
	require.NoError(t, types.Check(prog, env))
	mod, err := Compile(prog, "test")
	require.NoError(t, err)
	return mod
}

func TestCompile_LiteralInt(t *testing.T) {
	mod := compileExpr(t, "42")
	fn := mod.Functions[0]
	require.Len(t, fn.Code, 2)
	assert.Equal(t, bytecode.PUSH_INT, fn.Code[0].Op)
	assert.Equal(t, 0, fn.Code[0].Arg)
	assert.Equal(t, bytecode.HALT, fn.Code[1].Op)
	assert.Equal(t, 42, mod.Constants[0])
}

func TestCompile_LiteralFloat(t *testing.T) {
	mod := compileExpr(t, "3.14")
	fn := mod.Functions[0]
	assert.Equal(t, bytecode.PUSH_FLOAT, fn.Code[0].Op)
	assert.Equal(t, 3.14, mod.Constants[fn.Code[0].Arg])
}

func TestCompile_LiteralString(t *testing.T) {
	mod := compileExpr(t, `"hi"`)
	fn := mod.Functions[0]
	assert.Equal(t, bytecode.PUSH_STR, fn.Code[0].Op)
	assert.Equal(t, "hi", mod.Constants[fn.Code[0].Arg])
}

func TestCompile_LiteralBool(t *testing.T) {
	mod := compileExpr(t, "true")
	fn := mod.Functions[0]
	assert.Equal(t, bytecode.PUSH_BOOL, fn.Code[0].Op)
}

func TestCompile_LiteralNil(t *testing.T) {
	mod := compileExpr(t, "nil")
	fn := mod.Functions[0]
	assert.Equal(t, bytecode.PUSH_NIL, fn.Code[0].Op)
}

func TestCompile_LocalVar(t *testing.T) {
	mod := compileExpr(t, `let x = 1
x
`)
	fn := mod.Functions[0]
	var loadLocal *bytecode.Instruction
	for i := range fn.Code {
		if fn.Code[i].Op == bytecode.LOAD_LOCAL {
			loadLocal = &fn.Code[i]
			break
		}
	}
	require.NotNil(t, loadLocal)
	assert.Equal(t, 0, loadLocal.Arg)
	assert.Equal(t, 1, fn.NumLocals)
}

func TestCompile_BinaryAddInt(t *testing.T) {
	mod := compileExpr(t, "1 + 2")
	fn := mod.Functions[0]
	require.GreaterOrEqual(t, len(fn.Code), 4)
	assert.Equal(t, bytecode.PUSH_INT, fn.Code[0].Op)
	assert.Equal(t, bytecode.PUSH_INT, fn.Code[1].Op)
	assert.Equal(t, bytecode.ADD_INT, fn.Code[2].Op)
	assert.Equal(t, bytecode.HALT, fn.Code[3].Op)
}

func TestCompile_BinaryAddFloat(t *testing.T) {
	mod := compileExpr(t, "1.0 + 2.0")
	fn := mod.Functions[0]
	assert.Equal(t, bytecode.ADD_FLOAT, fn.Code[2].Op)
}

func TestCompile_BinaryEqInt(t *testing.T) {
	mod := compileExpr(t, "1 == 2")
	fn := mod.Functions[0]
	assert.Equal(t, bytecode.EQ_INT, fn.Code[2].Op)
}

func TestCompile_UnaryNeg(t *testing.T) {
	mod := compileExpr(t, "-5")
	fn := mod.Functions[0]
	require.GreaterOrEqual(t, len(fn.Code), 3)
	assert.Equal(t, bytecode.PUSH_INT, fn.Code[0].Op)
	assert.Equal(t, bytecode.NEG_INT, fn.Code[1].Op)
	assert.Equal(t, bytecode.HALT, fn.Code[2].Op)
}

func TestCompile_UnaryNot(t *testing.T) {
	mod := compileExpr(t, "not true")
	fn := mod.Functions[0]
	assert.Equal(t, bytecode.NOT_BOOL, fn.Code[1].Op)
}

func TestCompile_ListLiteral(t *testing.T) {
	mod := compileExpr(t, "[1, 2, 3]")
	fn := mod.Functions[0]
	var hasBuildList bool
	for _, instr := range fn.Code {
		if instr.Op == bytecode.BUILD_LIST {
			hasBuildList = true
			break
		}
	}
	assert.True(t, hasBuildList)
}

func TestCompile_Index(t *testing.T) {
	mod := compileExpr(t, "[1, 2, 3][0]")
	fn := mod.Functions[0]
	var hasIndex bool
	for _, instr := range fn.Code {
		if instr.Op == bytecode.INDEX {
			hasIndex = true
			break
		}
	}
	assert.True(t, hasIndex)
}

func TestCompile_Field(t *testing.T) {
	mod := compileExpr(t, `struct Point:
    x: int
    y: int

let p = Point(x: 1, y: 2)
p.x
`)
	fn := mod.Functions[0]
	var hasGetField bool
	for _, instr := range fn.Code {
		if instr.Op == bytecode.GET_FIELD {
			hasGetField = true
			break
		}
	}
	assert.True(t, hasGetField)
}
