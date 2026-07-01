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
	fn.Emit(bytecode.JUMP_IF_FALSE, 3) // ip=1, jump to ip=3 (HALT) if false; pops the bool
	fn.Emit(bytecode.PUSH_INT, 1)      // ip=2, skipped
	fn.Emit(bytecode.HALT, 0)          // ip=3, stack empty -> returns nil
	v := runVM(t, fn, false, 99)
	assert.Nil(t, v)
}

func TestVM_JumpIfFalse_FallThrough(t *testing.T) {
	fn := &bytecode.Function{Name: "main", Arity: 0}
	fn.Emit(bytecode.PUSH_BOOL, 0)       // push true
	fn.Emit(bytecode.JUMP_IF_FALSE, 100) // won't jump; pops the bool
	fn.Emit(bytecode.PUSH_INT, 1)        // ip=2, push 42
	fn.Emit(bytecode.HALT, 0)
	v := runVM(t, fn, true, 42)
	assert.Equal(t, 42, v)
}

func TestVM_JumpIfTrue_Taken(t *testing.T) {
	fn := &bytecode.Function{Name: "main", Arity: 0}
	fn.Emit(bytecode.PUSH_BOOL, 0)    // push true
	fn.Emit(bytecode.JUMP_IF_TRUE, 3) // ip=1, jump to ip=3 (HALT); pops the bool
	fn.Emit(bytecode.PUSH_INT, 1)     // ip=2, skipped
	fn.Emit(bytecode.HALT, 0)         // ip=3, stack empty -> returns nil
	v := runVM(t, fn, true, 99)
	assert.Nil(t, v)
}

func TestVM_JumpIfTrue_FallThrough(t *testing.T) {
	fn := &bytecode.Function{Name: "main", Arity: 0}
	fn.Emit(bytecode.PUSH_BOOL, 0)      // push false
	fn.Emit(bytecode.JUMP_IF_TRUE, 100) // won't jump; pops the bool
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
	main.Emit(bytecode.CALL_BUILTIN, 0) // constant[0] = BuiltinInfo{"println",1}
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, bytecode.BuiltinInfo{Name: "println", Arity: 1}, 42)
	assert.Nil(t, v) // println returns nil
}

func TestVM_CallBuiltin_Len(t *testing.T) {
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_STR, 0)     // "hello" (5 chars)
	main.Emit(bytecode.CALL_BUILTIN, 1) // constant[1] = BuiltinInfo{"len",1}
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, "hello", bytecode.BuiltinInfo{Name: "len", Arity: 1})
	assert.Equal(t, 5, v)
}

func TestVM_CallBuiltin_TypeOf(t *testing.T) {
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_INT, 0)
	main.Emit(bytecode.CALL_BUILTIN, 1) // constant[1] = BuiltinInfo{"type_of",1}
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, 42, bytecode.BuiltinInfo{Name: "type_of", Arity: 1})
	assert.Equal(t, "int", v)
}

func TestVM_BuildList(t *testing.T) {
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_INT, 0) // 1
	main.Emit(bytecode.PUSH_INT, 1) // 2
	main.Emit(bytecode.PUSH_INT, 2) // 3
	main.Emit(bytecode.BUILD_LIST, 3)
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, 1, 2, 3)
	list, ok := v.([]bytecode.Value)
	require.True(t, ok)
	assert.Equal(t, 3, len(list))
	assert.Equal(t, 1, list[0])
	assert.Equal(t, 2, list[1])
	assert.Equal(t, 3, list[2])
}

func TestVM_IndexList(t *testing.T) {
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_INT, 0)
	main.Emit(bytecode.PUSH_INT, 1)
	main.Emit(bytecode.PUSH_INT, 2)
	main.Emit(bytecode.BUILD_LIST, 3)
	main.Emit(bytecode.PUSH_INT, 3) // index 1
	main.Emit(bytecode.INDEX, 0)
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, 10, 20, 30, 1)
	assert.Equal(t, 20, v)
}

func TestVM_IndexString(t *testing.T) {
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_STR, 0) // "abc"
	main.Emit(bytecode.PUSH_INT, 1) // index 1
	main.Emit(bytecode.INDEX, 0)
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, "abc", 1)
	assert.Equal(t, "b", v)
}

func TestVM_BuildMap(t *testing.T) {
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_STR, 0) // "k"
	main.Emit(bytecode.PUSH_INT, 1)  // 42
	main.Emit(bytecode.BUILD_MAP, 1)
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, "k", 42)
	m, ok := v.(map[string]bytecode.Value)
	require.True(t, ok)
	assert.Equal(t, 42, m["k"])
}

func TestVM_GetField(t *testing.T) {
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_STR, 0) // "k"
	main.Emit(bytecode.PUSH_INT, 1)  // 99
	main.Emit(bytecode.BUILD_MAP, 1)
	main.Emit(bytecode.PUSH_STR, 0) // "k" (field name, same deduped constant)
	main.Emit(bytecode.GET_FIELD, 0)
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, "k", 99)
	assert.Equal(t, 99, v)
}

func TestVM_NewStruct(t *testing.T) {
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_STR, 0) // "k"
	main.Emit(bytecode.PUSH_INT, 1)  // 7
	main.Emit(bytecode.BUILD_MAP, 1)
	main.Emit(bytecode.NEW_STRUCT, 0) // type name at constant[0] = "User"
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, "k", 7) // note: constant[0] is "k" but NEW_STRUCT arg unused
	m, ok := v.(map[string]bytecode.Value)
	require.True(t, ok)
	assert.Equal(t, 7, m["k"])
}

func TestVM_BuiltinToJSON(t *testing.T) {
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_STR, 0)
	main.Emit(bytecode.CALL_BUILTIN, 1) // "to_json"
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, `{"k": 1}`, bytecode.BuiltinInfo{Name: "to_json", Arity: 1})
	m, ok := v.(map[string]bytecode.Value)
	require.True(t, ok)
	assert.Equal(t, float64(1), m["k"])
}

func TestVM_BuiltinParseJSON(t *testing.T) {
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_STR, 0)
	main.Emit(bytecode.CALL_BUILTIN, 1) // "parse_json"
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, `{"k": 1}`, bytecode.BuiltinInfo{Name: "parse_json", Arity: 1})
	m, ok := v.(map[string]bytecode.Value)
	require.True(t, ok)
	assert.Equal(t, float64(1), m["k"])
}

func TestVM_BuiltinNow(t *testing.T) {
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.CALL_BUILTIN, 0) // "now"
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, bytecode.BuiltinInfo{Name: "now", Arity: 0})
	n, ok := v.(int)
	require.True(t, ok)
	assert.Greater(t, n, 1700000000) // after 2023
}

func TestVM_BuiltinTimeFormat(t *testing.T) {
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_INT, 0)  // timestamp
	main.Emit(bytecode.PUSH_STR, 1)  // layout
	main.Emit(bytecode.CALL_BUILTIN, 2) // "time_format"
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, 1700000000, "2006-01-02", bytecode.BuiltinInfo{Name: "time_format", Arity: 2})
	s, ok := v.(string)
	require.True(t, ok)
	assert.Contains(t, s, "2023")
}

func TestVM_BuiltinSqrt(t *testing.T) {
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_INT, 0) // 16
	main.Emit(bytecode.CALL_BUILTIN, 1) // "sqrt"
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, 16, bytecode.BuiltinInfo{Name: "sqrt", Arity: 1})
	f, ok := v.(float64)
	require.True(t, ok)
	assert.Equal(t, 4.0, f)
}

func TestVM_BuiltinPow(t *testing.T) {
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_INT, 0) // 2
	main.Emit(bytecode.PUSH_INT, 1) // 10
	main.Emit(bytecode.CALL_BUILTIN, 2) // "pow"
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, 2, 10, bytecode.BuiltinInfo{Name: "pow", Arity: 2})
	f, ok := v.(float64)
	require.True(t, ok)
	assert.Equal(t, 1024.0, f)
}

func TestVM_BuiltinAbs(t *testing.T) {
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_INT, 0) // -5
	main.Emit(bytecode.CALL_BUILTIN, 1) // "abs"
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, -5, bytecode.BuiltinInfo{Name: "abs", Arity: 1})
	assert.Equal(t, 5, v)
}
