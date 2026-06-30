// v2/internal/vm/instructions_test.go
package vm

import (
	"testing"

	"github.com/jerloo/funny/v2/internal/bytecode"
	"github.com/stretchr/testify/assert"
)

func TestVM_SubInt(t *testing.T) {
	fn := &bytecode.Function{Name: "main", Arity: 0}
	fn.Emit(bytecode.PUSH_INT, 0)
	fn.Emit(bytecode.PUSH_INT, 1)
	fn.Emit(bytecode.SUB_INT, 0)
	fn.Emit(bytecode.HALT, 0)
	assert.Equal(t, 7, runVM(t, fn, 10, 3))
}

func TestVM_MulInt(t *testing.T) {
	fn := &bytecode.Function{Name: "main", Arity: 0}
	fn.Emit(bytecode.PUSH_INT, 0)
	fn.Emit(bytecode.PUSH_INT, 1)
	fn.Emit(bytecode.MUL_INT, 0)
	fn.Emit(bytecode.HALT, 0)
	assert.Equal(t, 42, runVM(t, fn, 6, 7))
}

func TestVM_DivInt(t *testing.T) {
	fn := &bytecode.Function{Name: "main", Arity: 0}
	fn.Emit(bytecode.PUSH_INT, 0)
	fn.Emit(bytecode.PUSH_INT, 1)
	fn.Emit(bytecode.DIV_INT, 0)
	fn.Emit(bytecode.HALT, 0)
	assert.Equal(t, 3, runVM(t, fn, 20, 6))
}

func TestVM_ModInt(t *testing.T) {
	fn := &bytecode.Function{Name: "main", Arity: 0}
	fn.Emit(bytecode.PUSH_INT, 0)
	fn.Emit(bytecode.PUSH_INT, 1)
	fn.Emit(bytecode.MOD_INT, 0)
	fn.Emit(bytecode.HALT, 0)
	assert.Equal(t, 1, runVM(t, fn, 10, 3))
}

func TestVM_NegInt(t *testing.T) {
	fn := &bytecode.Function{Name: "main", Arity: 0}
	fn.Emit(bytecode.PUSH_INT, 0)
	fn.Emit(bytecode.NEG_INT, 0)
	fn.Emit(bytecode.HALT, 0)
	assert.Equal(t, -5, runVM(t, fn, 5))
}

func TestVM_NotBool(t *testing.T) {
	fn := &bytecode.Function{Name: "main", Arity: 0}
	fn.Emit(bytecode.PUSH_BOOL, 0)
	fn.Emit(bytecode.NOT_BOOL, 0)
	fn.Emit(bytecode.HALT, 0)
	assert.Equal(t, false, runVM(t, fn, true))
}

func TestVM_AddFloat(t *testing.T) {
	fn := &bytecode.Function{Name: "main", Arity: 0}
	fn.Emit(bytecode.PUSH_FLOAT, 0)
	fn.Emit(bytecode.PUSH_FLOAT, 1)
	fn.Emit(bytecode.ADD_FLOAT, 0)
	fn.Emit(bytecode.HALT, 0)
	assert.InDelta(t, 4.0, runVM(t, fn, 1.5, 2.5), 0.0001)
}

func TestVM_AddStr(t *testing.T) {
	fn := &bytecode.Function{Name: "main", Arity: 0}
	fn.Emit(bytecode.PUSH_STR, 0)
	fn.Emit(bytecode.PUSH_STR, 1)
	fn.Emit(bytecode.ADD_STR, 0)
	fn.Emit(bytecode.HALT, 0)
	assert.Equal(t, "hello world", runVM(t, fn, "hello ", "world"))
}

func TestVM_LTInt(t *testing.T) {
	fn := &bytecode.Function{Name: "main", Arity: 0}
	fn.Emit(bytecode.PUSH_INT, 0)
	fn.Emit(bytecode.PUSH_INT, 1)
	fn.Emit(bytecode.LT_INT, 0)
	fn.Emit(bytecode.HALT, 0)
	assert.Equal(t, true, runVM(t, fn, 1, 2))
}

func TestVM_GTInt(t *testing.T) {
	fn := &bytecode.Function{Name: "main", Arity: 0}
	fn.Emit(bytecode.PUSH_INT, 0)
	fn.Emit(bytecode.PUSH_INT, 1)
	fn.Emit(bytecode.GT_INT, 0)
	fn.Emit(bytecode.HALT, 0)
	assert.Equal(t, false, runVM(t, fn, 1, 2))
}

func TestVM_EQInt(t *testing.T) {
	fn := &bytecode.Function{Name: "main", Arity: 0}
	fn.Emit(bytecode.PUSH_INT, 0)
	fn.Emit(bytecode.PUSH_INT, 1)
	fn.Emit(bytecode.EQ_INT, 0)
	fn.Emit(bytecode.HALT, 0)
	assert.Equal(t, false, runVM(t, fn, 1, 2))
}

func TestVM_EQStr(t *testing.T) {
	fn := &bytecode.Function{Name: "main", Arity: 0}
	fn.Emit(bytecode.PUSH_STR, 0)
	fn.Emit(bytecode.DUP, 0)
	fn.Emit(bytecode.EQ_STR, 0)
	fn.Emit(bytecode.HALT, 0)
	assert.Equal(t, true, runVM(t, fn, "hello"))
}