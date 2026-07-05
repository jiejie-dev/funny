// v2/internal/vm/instructions_test.go
package vm

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jiejie-dev/funny/internal/bytecode"
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
func TestVM_FormatValue_Default(t *testing.T) {
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_INT, 0)
	main.Emit(bytecode.FORMAT_VALUE, 1)
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, 42, "")
	assert.Equal(t, "42", v)
}

func TestVM_FormatValue_WithSpec(t *testing.T) {
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_FLOAT, 0)
	main.Emit(bytecode.FORMAT_VALUE, 1)
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, 3.14159, ".2f")
	assert.Equal(t, "3.14", v)
}

func TestVM_FormatValue_ThenAddStr(t *testing.T) {
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_STR, 0) // "hi "
	main.Emit(bytecode.PUSH_INT, 1) // 42
	main.Emit(bytecode.FORMAT_VALUE, 2)
	main.Emit(bytecode.ADD_STR, 0)
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, "hi ", 42, "")
	assert.Equal(t, "hi 42", v)
}

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

// TestVM_CallBuiltin_Append is a regression test: funny had no way to grow
// a list at all (no `lst[i] = x` past the end, no `+` on lists), so a
// loop could never collect its per-iteration results into a list. append
// must also leave the original list untouched (it returns a new list).
func TestVM_CallBuiltin_Append(t *testing.T) {
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_INT, 0) // 1
	main.Emit(bytecode.PUSH_INT, 1) // 2
	main.Emit(bytecode.BUILD_LIST, 2)
	main.Emit(bytecode.PUSH_INT, 2)    // 3
	main.Emit(bytecode.CALL_BUILTIN, 3) // constant[3] = BuiltinInfo{"append",2}
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, 1, 2, 3, bytecode.BuiltinInfo{Name: "append", Arity: 2})
	list, ok := v.([]bytecode.Value)
	require.True(t, ok)
	assert.Equal(t, []bytecode.Value{1, 2, 3}, list)
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
	main.Emit(bytecode.PUSH_INT, 1) // 42
	main.Emit(bytecode.BUILD_MAP, 1)
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, "k", 42)
	m, ok := v.(map[string]bytecode.Value)
	require.True(t, ok)
	assert.Equal(t, 42, m["k"])
}

func TestVM_IndexMap(t *testing.T) {
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_STR, 0) // "k"
	main.Emit(bytecode.PUSH_INT, 1) // 42
	main.Emit(bytecode.BUILD_MAP, 1)
	main.Emit(bytecode.PUSH_STR, 0) // "k" (same deduped constant, used as index)
	main.Emit(bytecode.INDEX, 0)
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, "k", 42)
	assert.Equal(t, 42, v)
}

func TestVM_IndexMap_MissingKeyErrors(t *testing.T) {
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_STR, 0) // "k"
	main.Emit(bytecode.PUSH_INT, 1) // 42
	main.Emit(bytecode.BUILD_MAP, 1)
	main.Emit(bytecode.PUSH_STR, 2) // "missing"
	main.Emit(bytecode.INDEX, 0)
	main.Emit(bytecode.HALT, 0)
	mod := bytecode.NewModule("test")
	mod.AddFunction(main)
	mod.AddConstant("k")
	mod.AddConstant(42)
	mod.AddConstant("missing")
	_, err := New(mod).Run()
	require.Error(t, err)
}

// TestVM_SetIndex_List builds [10, 20, 30], stores it in a local, sets
// index 1 to 99 via SET_INDEX, then reads it back through the same local
// to confirm the mutation is visible (lists are reference types).
func TestVM_SetIndex_List(t *testing.T) {
	fn := &bytecode.Function{Name: "main", Arity: 0, NumLocals: 1}
	fn.Emit(bytecode.PUSH_INT, 0) // 10
	fn.Emit(bytecode.PUSH_INT, 1) // 20
	fn.Emit(bytecode.PUSH_INT, 2) // 30
	fn.Emit(bytecode.BUILD_LIST, 3)
	fn.Emit(bytecode.STORE_LOCAL, 0)
	fn.Emit(bytecode.POP, 0)
	fn.Emit(bytecode.PUSH_INT, 3) // value 99
	fn.Emit(bytecode.LOAD_LOCAL, 0)
	fn.Emit(bytecode.PUSH_INT, 4) // index 1
	fn.Emit(bytecode.SET_INDEX, 0)
	fn.Emit(bytecode.POP, 0)
	fn.Emit(bytecode.LOAD_LOCAL, 0)
	fn.Emit(bytecode.PUSH_INT, 4) // index 1 again
	fn.Emit(bytecode.INDEX, 0)
	fn.Emit(bytecode.HALT, 0)
	v := runModule(t, fn, nil, 10, 20, 30, 99, 1)
	assert.Equal(t, 99, v)
}

// TestVM_SetIndex_Map builds {"a": 1}, overwrites "a" via SET_INDEX, then
// reads it back to confirm the mutation is visible (maps are reference
// types).
func TestVM_SetIndex_Map(t *testing.T) {
	fn := &bytecode.Function{Name: "main", Arity: 0, NumLocals: 1}
	fn.Emit(bytecode.PUSH_STR, 0) // "a"
	fn.Emit(bytecode.PUSH_INT, 1) // 1
	fn.Emit(bytecode.BUILD_MAP, 1)
	fn.Emit(bytecode.STORE_LOCAL, 0)
	fn.Emit(bytecode.POP, 0)
	fn.Emit(bytecode.PUSH_INT, 2) // value 100
	fn.Emit(bytecode.LOAD_LOCAL, 0)
	fn.Emit(bytecode.PUSH_STR, 0) // "a"
	fn.Emit(bytecode.SET_INDEX, 0)
	fn.Emit(bytecode.POP, 0)
	fn.Emit(bytecode.LOAD_LOCAL, 0)
	fn.Emit(bytecode.PUSH_STR, 0) // "a"
	fn.Emit(bytecode.INDEX, 0)
	fn.Emit(bytecode.HALT, 0)
	v := runModule(t, fn, nil, "a", 1, 100)
	assert.Equal(t, 100, v)
}

func TestVM_GetField(t *testing.T) {
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_STR, 0) // "k"
	main.Emit(bytecode.PUSH_INT, 1) // 99
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
	main.Emit(bytecode.PUSH_INT, 1) // 7
	main.Emit(bytecode.BUILD_MAP, 1)
	main.Emit(bytecode.NEW_STRUCT, 0) // type name at constant[0] = "User"
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, "k", 7) // note: constant[0] is "k" but NEW_STRUCT arg unused
	m, ok := v.(map[string]bytecode.Value)
	require.True(t, ok)
	assert.Equal(t, 7, m["k"])
}

func TestVM_ResultOK(t *testing.T) {
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_INT, 0)     // value 42
	main.Emit(bytecode.CALL_BUILTIN, 1) // BuiltinInfo{"ok",1} -> wraps in Result{tag: "ok", val: 42}
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, 42, bytecode.BuiltinInfo{Name: "ok", Arity: 1})
	m, ok := v.(map[string]bytecode.Value)
	require.True(t, ok)
	assert.Equal(t, "ok", m["tag"])
	assert.Equal(t, 42, m["val"])
}

func TestVM_ResultErr(t *testing.T) {
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_STR, 0)     // "oops"
	main.Emit(bytecode.CALL_BUILTIN, 1) // BuiltinInfo{"err",1}
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, "oops", bytecode.BuiltinInfo{Name: "err", Arity: 1})
	m, ok := v.(map[string]bytecode.Value)
	require.True(t, ok)
	assert.Equal(t, "err", m["tag"])
	assert.Equal(t, "oops", m["val"])
}

func TestVM_TryOrReturn_Ok(t *testing.T) {
	// fn foo():
	//   return ok(42)?
	// main: CALL foo, HALT
	// ? leaves Result on the stack on Ok; the Result is then returned.
	fn := &bytecode.Function{Name: "foo", Arity: 0, NumLocals: 0}
	fn.Emit(bytecode.PUSH_INT, 0)      // constant[0] = 42
	fn.Emit(bytecode.CALL_BUILTIN, 1)  // constant[1] = BuiltinInfo{"ok",1} -> Result{tag: "ok", val: 42}
	fn.Emit(bytecode.TRY_OR_RETURN, 0) // Ok: leave Result on stack
	fn.Emit(bytecode.RETURN, 0)
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.CALL, 1)
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, []*bytecode.Function{fn}, 42, bytecode.BuiltinInfo{Name: "ok", Arity: 1})
	m, ok := v.(map[string]bytecode.Value)
	require.True(t, ok)
	assert.Equal(t, "ok", m["tag"])
	assert.Equal(t, 42, m["val"])
}

func TestVM_TryOrReturn_Err(t *testing.T) {
	// fn foo():
	//   return err("boom")?
	// main: CALL foo, HALT
	fn := &bytecode.Function{Name: "foo", Arity: 0, NumLocals: 0}
	fn.Emit(bytecode.PUSH_STR, 0)      // constant[0] = "boom"
	fn.Emit(bytecode.CALL_BUILTIN, 1)  // constant[1] = BuiltinInfo{"err",1} -> Result{tag: "err", val: "boom"}
	fn.Emit(bytecode.TRY_OR_RETURN, 0) // Err: return from current function with Result
	fn.Emit(bytecode.RETURN, 0)        // dead code (unreachable after TRY_OR_RETURN on err)
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.CALL, 1)
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, []*bytecode.Function{fn}, "boom", bytecode.BuiltinInfo{Name: "err", Arity: 1})
	m, ok := v.(map[string]bytecode.Value)
	require.True(t, ok)
	assert.Equal(t, "err", m["tag"])
	assert.Equal(t, "boom", m["val"])
}

func TestVM_BuiltinToJSON(t *testing.T) {
	// to_json maps a funny value (map[string]bytecode.Value) to its canonical
	// JSON string representation.
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_STR, 0)
	main.Emit(bytecode.CALL_BUILTIN, 1) // "to_json"
	main.Emit(bytecode.HALT, 0)
	m := map[string]bytecode.Value{"k": float64(1)}
	v := runModule(t, main, nil, m, bytecode.BuiltinInfo{Name: "to_json", Arity: 1})
	s, ok := v.(string)
	require.True(t, ok)
	assert.Equal(t, `{"k":1}`, s)
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
	main.Emit(bytecode.PUSH_INT, 0)     // timestamp
	main.Emit(bytecode.PUSH_STR, 1)     // layout
	main.Emit(bytecode.CALL_BUILTIN, 2) // "time_format"
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, 1700000000, "2006-01-02", bytecode.BuiltinInfo{Name: "time_format", Arity: 2})
	s, ok := v.(string)
	require.True(t, ok)
	assert.Contains(t, s, "2023")
}

func TestVM_BuiltinSqrt(t *testing.T) {
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_INT, 0)     // 16
	main.Emit(bytecode.CALL_BUILTIN, 1) // "sqrt"
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, 16, bytecode.BuiltinInfo{Name: "sqrt", Arity: 1})
	f, ok := v.(float64)
	require.True(t, ok)
	assert.Equal(t, 4.0, f)
}

func TestVM_BuiltinPow(t *testing.T) {
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_INT, 0)     // 2
	main.Emit(bytecode.PUSH_INT, 1)     // 10
	main.Emit(bytecode.CALL_BUILTIN, 2) // "pow"
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, 2, 10, bytecode.BuiltinInfo{Name: "pow", Arity: 2})
	f, ok := v.(float64)
	require.True(t, ok)
	assert.Equal(t, 1024.0, f)
}

func TestVM_BuiltinAbs(t *testing.T) {
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_INT, 0)     // -5
	main.Emit(bytecode.CALL_BUILTIN, 1) // "abs"
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, -5, bytecode.BuiltinInfo{Name: "abs", Arity: 1})
	assert.Equal(t, 5, v)
}

func TestVM_BuiltinStrUpper(t *testing.T) {
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_STR, 0)     // "hello"
	main.Emit(bytecode.CALL_BUILTIN, 1) // "str_upper"
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, "hello", bytecode.BuiltinInfo{Name: "str_upper", Arity: 1})
	assert.Equal(t, "HELLO", v)
}

func TestVM_BuiltinStrLower(t *testing.T) {
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_STR, 0)     // "WORLD"
	main.Emit(bytecode.CALL_BUILTIN, 1) // "str_lower"
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, "WORLD", bytecode.BuiltinInfo{Name: "str_lower", Arity: 1})
	assert.Equal(t, "world", v)
}

func TestVM_BuiltinStrContains(t *testing.T) {
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_STR, 0)     // "hello world"
	main.Emit(bytecode.PUSH_STR, 1)     // "world"
	main.Emit(bytecode.CALL_BUILTIN, 2) // "str_contains"
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, "hello world", "world", bytecode.BuiltinInfo{Name: "str_contains", Arity: 2})
	assert.Equal(t, true, v)
}

func TestVM_BuiltinStrSplit(t *testing.T) {
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_STR, 0)     // "a,b,c"
	main.Emit(bytecode.PUSH_STR, 1)     // ","
	main.Emit(bytecode.CALL_BUILTIN, 2) // "str_split"
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, "a,b,c", ",", bytecode.BuiltinInfo{Name: "str_split", Arity: 2})
	list, ok := v.([]bytecode.Value)
	require.True(t, ok)
	assert.Equal(t, 3, len(list))
	assert.Equal(t, "a", list[0])
	assert.Equal(t, "b", list[1])
	assert.Equal(t, "c", list[2])
}

func TestVM_BuiltinRegexMatch(t *testing.T) {
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_STR, 0)     // pattern
	main.Emit(bytecode.PUSH_STR, 1)     // text
	main.Emit(bytecode.CALL_BUILTIN, 2) // "regex_match"
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, "[0-9]+", "abc123def", bytecode.BuiltinInfo{Name: "regex_match", Arity: 2})
	assert.Equal(t, true, v)
}

func TestVM_BuiltinRegexReplace(t *testing.T) {
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_STR, 0)     // pattern
	main.Emit(bytecode.PUSH_STR, 1)     // text
	main.Emit(bytecode.PUSH_STR, 2)     // replacement
	main.Emit(bytecode.CALL_BUILTIN, 3) // "regex_replace"
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, "[0-9]+", "abc123def", "X", bytecode.BuiltinInfo{Name: "regex_replace", Arity: 3})
	assert.Equal(t, "abcXdef", v)
}

func TestVM_BuiltinEnvGet(t *testing.T) {
	os.Setenv("FUNNY_TEST_ENV", "hello")
	defer os.Unsetenv("FUNNY_TEST_ENV")
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_STR, 0)
	main.Emit(bytecode.CALL_BUILTIN, 1)
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, "FUNNY_TEST_ENV", bytecode.BuiltinInfo{Name: "env_get", Arity: 1})
	assert.Equal(t, "hello", v)
}

func TestVM_BuiltinFileRead(t *testing.T) {
	tmpfile := "/tmp/funny_test_read.txt"
	os.WriteFile(tmpfile, []byte("hello funny"), 0644)
	defer os.Remove(tmpfile)
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_STR, 0)
	main.Emit(bytecode.CALL_BUILTIN, 1)
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, tmpfile, bytecode.BuiltinInfo{Name: "file_read", Arity: 1})
	m, ok := v.(map[string]bytecode.Value)
	require.True(t, ok, "expected Result, got %T", v)
	assert.Equal(t, "ok", m["tag"])
	assert.Equal(t, "hello funny", m["val"])
}

func TestVM_BuiltinFileExists(t *testing.T) {
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_STR, 0)
	main.Emit(bytecode.CALL_BUILTIN, 1)
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, "/tmp/funny_test_definitely_does_not_exist_12345", bytecode.BuiltinInfo{Name: "file_exists", Arity: 1})
	assert.Equal(t, false, v)
}

func TestVM_BuiltinHttpGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	}))
	defer srv.Close()

	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_STR, 0)     // URL
	main.Emit(bytecode.CALL_BUILTIN, 1) // "http_get"
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, srv.URL, bytecode.BuiltinInfo{Name: "http_get", Arity: 1})
	m, ok := v.(map[string]bytecode.Value)
	require.True(t, ok, "expected Result, got %T", v)
	assert.Equal(t, "ok", m["tag"])
	assert.Equal(t, "hello", m["val"])
}

func TestVM_BuiltinCrypto(t *testing.T) {
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_STR, 0)     // "hello"
	main.Emit(bytecode.CALL_BUILTIN, 1) // "md5"
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, "hello", bytecode.BuiltinInfo{Name: "md5", Arity: 1})
	assert.Equal(t, "5d41402abc4b2a76b9719d911017c592", v)
}

func TestVM_BuiltinB64Encode(t *testing.T) {
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_STR, 0)     // "hello"
	main.Emit(bytecode.CALL_BUILTIN, 1) // "b64_encode"
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, "hello", bytecode.BuiltinInfo{Name: "b64_encode", Arity: 1})
	assert.Equal(t, "aGVsbG8=", v)
}

func TestVM_BuiltinB64Decode(t *testing.T) {
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_STR, 0)     // "aGVsbG8="
	main.Emit(bytecode.CALL_BUILTIN, 1) // "b64_decode"
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, "aGVsbG8=", bytecode.BuiltinInfo{Name: "b64_decode", Arity: 1})
	m, ok := v.(map[string]bytecode.Value)
	require.True(t, ok)
	assert.Equal(t, "ok", m["tag"])
	assert.Equal(t, "hello", m["val"])
}

func TestVM_BuiltinJwtEncode(t *testing.T) {
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_STR, 0)     // header json
	main.Emit(bytecode.PUSH_STR, 1)     // claims json
	main.Emit(bytecode.PUSH_STR, 2)     // secret
	main.Emit(bytecode.CALL_BUILTIN, 3) // "jwt_encode"
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil,
		`{"alg":"HS256","typ":"JWT"}`,
		`{"sub":"alice"}`,
		"secret",
		bytecode.BuiltinInfo{Name: "jwt_encode", Arity: 3})
	s, ok := v.(string)
	require.True(t, ok)
	assert.Contains(t, s, ".")
}

func TestVM_BuiltinJwtRoundTrip(t *testing.T) {
	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_STR, 0)     // header json
	main.Emit(bytecode.PUSH_STR, 1)     // claims json
	main.Emit(bytecode.PUSH_STR, 2)     // secret (encode)
	main.Emit(bytecode.CALL_BUILTIN, 3) // jwt_encode (arity 3)
	main.Emit(bytecode.PUSH_STR, 2)     // same secret (decode)
	main.Emit(bytecode.CALL_BUILTIN, 4) // jwt_decode (arity 2)
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil,
		`{"alg":"HS256","typ":"JWT"}`,
		`{"sub":"alice"}`,
		"secret",
		bytecode.BuiltinInfo{Name: "jwt_encode", Arity: 3},
		bytecode.BuiltinInfo{Name: "jwt_decode", Arity: 2})
	m, ok := v.(map[string]bytecode.Value)
	require.True(t, ok)
	assert.Equal(t, "ok", m["tag"])
}

func TestVM_BuiltinSqlOpen(t *testing.T) {
	tmpdb := "/tmp/funny_test_m4.db"
	os.Remove(tmpdb)
	defer os.Remove(tmpdb)

	main := &bytecode.Function{Name: "main", Arity: 0}
	main.Emit(bytecode.PUSH_STR, 0)
	main.Emit(bytecode.CALL_BUILTIN, 1) // "sql_open"
	main.Emit(bytecode.HALT, 0)
	v := runModule(t, main, nil, tmpdb, bytecode.BuiltinInfo{Name: "sql_open", Arity: 1})
	s, ok := v.(string)
	require.True(t, ok)
	assert.NotEmpty(t, s)
}
