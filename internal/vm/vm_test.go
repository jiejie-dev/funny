// v2/internal/vm/vm_test.go
package vm

import (
	"testing"

	"github.com/jiejie-dev/funny/internal/bytecode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runVM creates a module from bytecode instructions, runs the VM, returns the top of stack (or nil).
func runVM(t *testing.T, fn *bytecode.Function, constants ...bytecode.Value) bytecode.Value {
	t.Helper()
	mod := bytecode.NewModule("test")
	mod.AddFunction(fn)
	for _, c := range constants {
		mod.AddConstant(c)
	}
	m := New(mod)
	v, err := m.Run()
	require.NoError(t, err)
	return v
}

func TestVM_PushInt(t *testing.T) {
	fn := &bytecode.Function{Name: "main", Arity: 0}
	fn.Emit(bytecode.PUSH_INT, 0) // constant[0] = 42
	fn.Emit(bytecode.HALT, 0)
	v := runVM(t, fn, 42)
	assert.Equal(t, 42, v)
}

func TestVM_Pop(t *testing.T) {
	fn := &bytecode.Function{Name: "main", Arity: 0}
	fn.Emit(bytecode.PUSH_INT, 0) // push 1
	fn.Emit(bytecode.PUSH_INT, 1) // push 2
	fn.Emit(bytecode.POP, 0)      // discard 2
	fn.Emit(bytecode.HALT, 0)     // result: top of stack = 1
	v := runVM(t, fn, 1, 2)
	assert.Equal(t, 1, v)
}

func TestVM_LoadStoreLocal(t *testing.T) {
	fn := &bytecode.Function{Name: "main", Arity: 0, NumLocals: 2}
	fn.Emit(bytecode.PUSH_INT, 0)    // push 10
	fn.Emit(bytecode.STORE_LOCAL, 0) // x = 10
	fn.Emit(bytecode.PUSH_INT, 1)    // push 20
	fn.Emit(bytecode.STORE_LOCAL, 1) // y = 20
	fn.Emit(bytecode.LOAD_LOCAL, 0)  // push x
	fn.Emit(bytecode.LOAD_LOCAL, 1)  // push y (top of stack)
	fn.Emit(bytecode.HALT, 0)
	v := runVM(t, fn, 10, 20)
	assert.Equal(t, 20, v) // top of stack is y
}

func TestVM_String(t *testing.T) {
	fn := &bytecode.Function{Name: "main", Arity: 0}
	fn.Emit(bytecode.PUSH_STR, 0)
	fn.Emit(bytecode.HALT, 0)
	v := runVM(t, fn, "hello")
	assert.Equal(t, "hello", v)
}

func TestVM_Bool(t *testing.T) {
	fn := &bytecode.Function{Name: "main", Arity: 0}
	fn.Emit(bytecode.PUSH_BOOL, 0)
	fn.Emit(bytecode.HALT, 0)
	v := runVM(t, fn, true)
	assert.Equal(t, true, v)
}

func TestVM_Nil(t *testing.T) {
	fn := &bytecode.Function{Name: "main", Arity: 0}
	fn.Emit(bytecode.PUSH_NIL, 0)
	fn.Emit(bytecode.HALT, 0)
	v := runVM(t, fn)
	assert.Nil(t, v)
}

func TestVM_HaltWithEmptyStack(t *testing.T) {
	fn := &bytecode.Function{Name: "main", Arity: 0}
	fn.Emit(bytecode.HALT, 0)
	v := runVM(t, fn)
	assert.Nil(t, v)
}
