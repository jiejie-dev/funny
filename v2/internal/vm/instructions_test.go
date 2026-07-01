// v2/internal/vm/instructions_test.go
package vm

import (
	"testing"

	"github.com/jerloo/funny/v2/internal/bytecode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestVM_JumpIfFalse_Taken(t *testing.T) {
	fn := &bytecode.Function{Name: "main", Arity: 0}
	fn.Emit(bytecode.PUSH_BOOL, 0)     // push false
	fn.Emit(bytecode.JUMP_IF_FALSE, 3) // ip=1, jump to ip=3 (HALT) if false
	fn.Emit(bytecode.PUSH_INT, 1)      // ip=2, skipped
	fn.Emit(bytecode.HALT, 0)          // ip=3, returns false
	v := runVM(t, fn, false)
	assert.Equal(t, false, v) // bool still on stack from PUSH_BOOL
}

func TestVM_JumpIfFalse_FallThrough(t *testing.T) {
	fn := &bytecode.Function{Name: "main", Arity: 0}
	fn.Emit(bytecode.PUSH_BOOL, 0)       // push true
	fn.Emit(bytecode.JUMP_IF_FALSE, 100) // won't jump (condition is true)
	fn.Emit(bytecode.PUSH_INT, 1)        // ip=2, push 42
	fn.Emit(bytecode.HALT, 0)
	v := runVM(t, fn, true, 42)
	assert.Equal(t, 42, v)
}

func TestVM_JumpIfTrue_Taken(t *testing.T) {
	fn := &bytecode.Function{Name: "main", Arity: 0}
	fn.Emit(bytecode.PUSH_BOOL, 0)    // push true
	fn.Emit(bytecode.JUMP_IF_TRUE, 3) // ip=1, jump to ip=3 (HALT)
	fn.Emit(bytecode.PUSH_INT, 1)     // ip=2, skipped
	fn.Emit(bytecode.HALT, 0)         // ip=3
	v := runVM(t, fn, true)
	assert.Equal(t, true, v)
}

func TestVM_JumpIfTrue_FallThrough(t *testing.T) {
	fn := &bytecode.Function{Name: "main", Arity: 0}
	fn.Emit(bytecode.PUSH_BOOL, 0)      // push false
	fn.Emit(bytecode.JUMP_IF_TRUE, 100) // won't jump
	fn.Emit(bytecode.PUSH_INT, 1)       // ip=2
	fn.Emit(bytecode.HALT, 0)
	v := runVM(t, fn, false, 99)
	assert.Equal(t, 99, v)
}

func TestVM_JumpUnconditional(t *testing.T) {
	fn := &bytecode.Function{Name: "main", Arity: 0}
	fn.Emit(bytecode.PUSH_INT, 0) // ip=0, push 1
	fn.Emit(bytecode.JUMP, 3)     // ip=1, jump to ip=3
	fn.Emit(bytecode.PUSH_INT, 1) // ip=2, skipped
	fn.Emit(bytecode.HALT, 0)     // ip=3, returns 1 (from PUSH_INT 0)
	v := runVM(t, fn, 1, 2)
	assert.Equal(t, 1, v)
}

// runModule constructs a module from an entry function and helpers, plus constants.
func runModule(t *testing.T, entry *bytecode.Function, helpers []*bytecode.Function, constants ...bytecode.Value) bytecode.Value {
	t.Helper()
	mod := bytecode.NewModule("test")
	mod.AddFunction(entry)
	for _, h := range helpers {
		mod.AddFunction(h)
	}
	for _, c := range constants {
		mod.AddConstant(c)
	}
	m := New(mod)
	v, err := m.Run()
	require.NoError(t, err)
	return v
}

func TestVM_CallReturn(t *testing.T) {
	main := &bytecode.Function{Name: "main", Arity: 0}
	fn1 := &bytecode.Function{Name: "id", Arity: 1, NumLocals: 1}
	fn1.Emit(bytecode.LOAD_LOCAL, 0)
	fn1.Emit(bytecode.RETURN, 0)
	main.Emit(bytecode.PUSH_INT, 0) // push 5
	main.Emit(bytecode.CALL, 1)     // call fn1
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, []*bytecode.Function{fn1}, 5)
	assert.Equal(t, 5, v)
}

func TestVM_NestedCall(t *testing.T) {
	main := &bytecode.Function{Name: "main", Arity: 0}
	add := &bytecode.Function{Name: "add", Arity: 2, NumLocals: 2}
	add.Emit(bytecode.LOAD_LOCAL, 0)
	add.Emit(bytecode.LOAD_LOCAL, 1)
	add.Emit(bytecode.ADD_INT, 0)
	add.Emit(bytecode.RETURN, 0)
	main.Emit(bytecode.PUSH_INT, 0)
	main.Emit(bytecode.PUSH_INT, 1)
	main.Emit(bytecode.CALL, 1)
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, []*bytecode.Function{add}, 2, 3)
	assert.Equal(t, 5, v)
}

func TestVM_CallBuiltin_Println(t *testing.T) {
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_INT, 0)
	main.Emit(bytecode.CALL_BUILTIN, 0) // constant[0] = "println"
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, "println")
	assert.Nil(t, v) // println returns nil
}

func TestVM_CallBuiltin_Len(t *testing.T) {
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_STR, 0)     // "hello" (5 chars)
	main.Emit(bytecode.CALL_BUILTIN, 1) // constant[1] = "len"
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, "hello", "len")
	assert.Equal(t, 5, v)
}

func TestVM_CallBuiltin_TypeOf(t *testing.T) {
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_INT, 0)
	main.Emit(bytecode.CALL_BUILTIN, 1) // constant[1] = "type_of"
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, 42, "type_of")
	assert.Equal(t, "int", v)
}
