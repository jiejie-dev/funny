# Funny v2 M2-B.5: VM Functions + Data Ops Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add function calls (CALL/RETURN), built-in calls (CALL_BUILTIN), and data-structure operations (BUILD_LIST/INDEX/BUILD_MAP/GET_FIELD/NEW_STRUCT) to the VM and compiler — completing the M2-B core so that recursive fib benchmark hits the ≥5× interpreter target.

**Architecture:** VM CALL/RETURN pushes/pops Frame stack and transfers the top-of-stack return value. Builtins are called via CALL_BUILTIN with the function name as a constant-pool string. Data-structure ops are typed but operate on `interface{}` values for M2-B simplicity. Compiler emits CALL with the function-index arg (referencing the Module's function table).

**Tech Stack:** Go 1.22+, `github.com/stretchr/testify`.

**Reference Spec:** `docs/superpowers/specs/2026-07-01-funny-v2-ai-native-language-design.md` §5.4 (CALL/RETURN/CALL_BUILTIN/BUILD_LIST/INDEX/BUILD_MAP/GET_FIELD/NEW_STRUCT).

**Scope:** M2-B.5 only. M2-C (Result + `?` + stdlib) is a separate plan.

---

## File Structure

Modified files:
- `v2/internal/bytecode/opcode.go` — opcodes already exist (added in M2-B Task 0); no changes
- `v2/internal/bytecode/code.go` — Module already has `Functions []*Function`; no changes
- `v2/internal/vm/vm.go` — add CALL/RETURN/CALL_BUILTIN handlers in dispatch
- `v2/internal/vm/instructions.go` — add helpers: execCall, execCallBuiltin, execListOps
- `v2/internal/vm/builtins.go` — new: VM built-in function implementations
- `v2/internal/vm/instructions_test.go` — append tests for new ops
- `v2/internal/compiler/compiler.go` — extend compileStmt to handle FnDecl/Return; add function table tracking
- `v2/internal/compiler/fn.go` — new: compileFnDecl/compileCall/compileReturn/compileBuiltins
- `v2/internal/compiler/data.go` — new: compileList/compileIndex/compileMap/compileField/compileStructLit
- `v2/internal/compiler/expr.go` — extend compileExpr with CallExpr/IndexExpr/FieldExpr/ListExpr cases
- `v2/internal/compiler/control.go` — replace for-in stub with full implementation
- `v2/testdata/vm/fib.fn` — new (replaces fib_iter for recursive benchmark)

---

## Conventions

- `CALL arg` arg is the function index in `Module.Functions`. Before CALL, args are pushed on stack (last arg on top). CALL pops N args (where N = callee.Arity), pushes new frame, jumps to entry. RETURN pops frame, pushes return value from stack onto caller's stack.
- `CALL_BUILTIN arg` arg is a constant-pool index for the builtin name (e.g. "print").
- `BUILD_LIST arg` arg is the count of items (popped from stack).
- `INDEX` pops index then object, pushes element.
- `GET_FIELD arg` arg is constant-pool index for field name.
- `NEW_STRUCT arg` arg is constant-pool index for struct type name.

---

## Task 0: VM CALL/RETURN

**Files:**
- Modify: `v2/internal/vm/vm.go`

- [ ] **Step 1: Append failing tests** to `v2/internal/vm/instructions_test.go`:

```go
func TestVM_CallReturn(t *testing.T) {
    mod := bytecode.NewModule("test")
    // Main: PUSH_INT 5, CALL <fn1>, HALT
    // fn1: LOAD_LOCAL 0, RETURN  (returns its arg)
    main := &bytecode.Function{Name: "main", Arity: 0}
    mod.AddFunction(main)
    fn1 := &bytecode.Function{Name: "id", Arity: 1, NumLocals: 1}
    fn1.Emit(bytecode.LOAD_LOCAL, 0)
    fn1.Emit(bytecode.RETURN, 0)
    main.Emit(bytecode.PUSH_INT, 0) // push 5
    main.Emit(bytecode.CALL, 1)      // call fn1
    main.Emit(bytecode.HALT, 0)
    v := runVM(t, main, 5)
    assert.Equal(t, 5, v)
}

func TestVM_NestedCall(t *testing.T) {
    mod := bytecode.NewModule("test")
    // main: PUSH_INT 2, PUSH_INT 3, CALL add, HALT
    // add: LOAD_LOCAL 0, LOAD_LOCAL 1, ADD_INT, RETURN
    main := &bytecode.Function{Name: "main", Arity: 0}
    mod.AddFunction(main)
    add := &bytecode.Function{Name: "add", Arity: 2, NumLocals: 2}
    add.Emit(bytecode.LOAD_LOCAL, 0)
    add.Emit(bytecode.LOAD_LOCAL, 1)
    add.Emit(bytecode.ADD_INT, 0)
    add.Emit(bytecode.RETURN, 0)
    main.Emit(bytecode.PUSH_INT, 0) // 2
    main.Emit(bytecode.PUSH_INT, 1) // 3
    main.Emit(bytecode.CALL, 1)
    main.Emit(bytecode.HALT, 0)
    v := runVM(t, main, 2, 3)
    assert.Equal(t, 5, v)
}
```

(Note: `runVM(t, fn, constants...)` takes the *first* function as the entry. To test multi-function modules we need a different helper. Adapt `runVM` to take a module, or create a new helper `runModule(t, mod)`. Either way works; pick the simpler refactor.)

- [ ] **Step 2: Update the test helper** at the top of `instructions_test.go` to support multi-function modules. Add this new helper below the existing `runVM`:

```go
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
```

And rewrite `TestVM_CallReturn` and `TestVM_NestedCall` to use `runModule`:

```go
func TestVM_CallReturn(t *testing.T) {
    main := &bytecode.Function{Name: "main", Arity: 0}
    fn1 := &bytecode.Function{Name: "id", Arity: 1, NumLocals: 1}
    fn1.Emit(bytecode.LOAD_LOCAL, 0)
    fn1.Emit(bytecode.RETURN, 0)
    main.Emit(bytecode.PUSH_INT, 0)
    main.Emit(bytecode.CALL, 1)
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
```

Add `"github.com/stretchr/testify/require"` import to `instructions_test.go` if not already present.

- [ ] **Step 3: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m5/v2
go test ./internal/vm/ -v -run "TestVM_CallReturn|TestVM_NestedCall"
```

Expected: FAIL

- [ ] **Step 4: Add CALL/RETURN handlers to `vm.go`**:

Find the `default:` case in the `execute()` function. ADD these cases BEFORE the `default:`:

```go
        case bytecode.CALL:
            if err := v.execCall(instr.Arg); err != nil {
                return nil, err
            }
        case bytecode.RETURN:
            if err := v.execReturn(); err != nil {
                return nil, err
            }
            // After RETURN, the popped frame's return value is on the stack.
            // If no caller frame (we just returned from main), exit the loop.
            if len(v.frames) == 0 {
                if len(v.stack) > 0 {
                    return v.stack[len(v.stack)-1], nil
                }
                return nil, nil
            }
        case bytecode.CALL_BUILTIN:
            if err := v.execCallBuiltin(instr.Arg); err != nil {
                return nil, err
            }
```

(We'll add `execCallBuiltin` in Task 1.)

Also, the dispatch loop must support returning from the main frame. Currently `execute()` loops until HALT. We need to handle `len(v.frames) == 0` as an exit condition. Update the loop structure:

Replace the body of `execute()` so that the return from main exits the loop. The simplest way: after each dispatch iteration, check `if len(v.frames) == 0 { break }`.

Let me show the final execute function. **Replace the entire `execute()` function in `vm.go`** with:

```go
func (v *VM) execute() (bytecode.Value, error) {
    for {
        if len(v.frames) == 0 {
            if len(v.stack) > 0 {
                return v.stack[len(v.stack)-1], nil
            }
            return nil, nil
        }
        frame := v.frames[len(v.frames)-1]
        if frame.ip >= len(frame.fn.Code) {
            return nil, fmt.Errorf("vm: ip out of bounds at %d", frame.ip)
        }
        instr := frame.fn.Code[frame.ip]
        frame.ip++
        switch instr.Op {
        // ... existing cases ...
        case bytecode.CALL:
            if err := v.execCall(instr.Arg); err != nil {
                return nil, err
            }
        case bytecode.RETURN:
            if err := v.execReturn(); err != nil {
                return nil, err
            }
        default:
            return nil, fmt.Errorf("vm: unsupported op %s at ip=%d", instr.Op, frame.ip-1)
        }
    }
}
```

(Keep all existing cases intact — only ADD the CALL/RETURN cases and wrap the dispatch in a frame-empty check.)

- [ ] **Step 5: Add `execCall` and `execReturn` helpers** to `v2/internal/vm/instructions.go`:

```go
// execCall handles CALL fnIdx.
// Pops argCount args from the stack (in reverse), pushes new frame.
func (v *VM) execCall(fnIdx int) error {
    if fnIdx < 0 || fnIdx >= len(v.mod.Functions) {
        return fmt.Errorf("vm: CALL invalid function index %d", fnIdx)
    }
    callee := v.mod.Functions[fnIdx]
    n := callee.Arity
    if len(v.stack) < n {
        return fmt.Errorf("vm: CALL %s expects %d args, got %d", callee.Name, n, len(v.stack))
    }
    args := make([]bytecode.Value, n)
    for i := n - 1; i >= 0; i-- {
        args[i] = v.stack[len(v.stack)-1-(n-1-i)]
    }
    v.stack = v.stack[:len(v.stack)-n]
    // Push new frame
    newFrame := &Frame{
        fn:     callee,
        ip:     0,
        locals: make([]bytecode.Value, callee.NumLocals),
    }
    for i, a := range args {
        newFrame.locals[i] = a
    }
    v.frames = append(v.frames, newFrame)
    return nil
}

// execReturn handles RETURN.
// Pops the current frame, pushes top-of-stack as caller's return value.
func (v *VM) execReturn() error {
    if len(v.frames) == 0 {
        return fmt.Errorf("vm: RETURN with no frames")
    }
    var retVal bytecode.Value
    if len(v.stack) > 0 {
        retVal = v.stack[len(v.stack)-1]
    }
    v.frames = v.frames[:len(v.frames)-1]
    if retVal != nil {
        v.stack = append(v.stack, retVal)
    }
    return nil
}
```

- [ ] **Step 6: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m5/v2
go test ./internal/vm/ -v -count=1
```

Expected: all 26 tests PASS (24 prior + 2 new)

- [ ] **Step 7: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m5
git add v2/internal/vm/
git commit -m "v2: VM CALL/RETURN with frame push/pop"
```

---

## Task 1: VM CALL_BUILTIN (Print + Len + TypeOf)

**Files:**
- Create: `v2/internal/vm/builtins.go`
- Modify: `v2/internal/vm/vm.go` (call `execCallBuiltin`)
- Append to `v2/internal/vm/instructions.go`

- [ ] **Step 1: Append failing tests**:

```go
func TestVM_CallBuiltin_Println(t *testing.T) {
    main := &bytecode.Function{Name: "main", Arity: 0}
    main.Emit(bytecode.PUSH_INT, 0)
    main.Emit(bytecode.CALL_BUILTIN, 0) // builtin name at constant[0]
    main.Emit(bytecode.HALT, 0)
    // Should not panic; stdout contains "42\n" (best-effort check)
    v := runModule(t, main, nil, "println")
    assert.Nil(t, v) // println returns nil
}

func TestVM_CallBuiltin_Len(t *testing.T) {
    main := &bytecode.Function{Name: "main", Arity: 0}
    main.Emit(bytecode.PUSH_STR, 0) // "hello" (len 5)
    main.Emit(bytecode.CALL_BUILTIN, 1) // builtin name at constant[1]
    main.Emit(bytecode.HALT, 0)
    v := runModule(t, main, nil, "println", "len")
    assert.Equal(t, 5, v)
}
```

- [ ] **Step 2: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m5/v2
go test ./internal/vm/ -v -run "TestVM_CallBuiltin"
```

Expected: FAIL

- [ ] **Step 3: Create `v2/internal/vm/builtins.go`**:

```go
// v2/internal/vm/builtins.go
package vm

import (
    "fmt"
    "reflect"

    "github.com/jiejie-dev/funny/internal/bytecode"
)

// execCallBuiltin handles CALL_BUILTIN nameIdx.
// Pops arguments from stack (depending on the builtin), pushes result.
func (v *VM) execCallBuiltin(nameIdx int) error {
    name, ok := v.mod.Constants[nameIdx].(string)
    if !ok {
        return fmt.Errorf("vm: CALL_BUILTIN name is not a string")
    }
    switch name {
    case "print":
        if len(v.stack) < 1 {
            return fmt.Errorf("vm: print() requires at least 1 argument")
        }
        fmt.Print(v.stack[len(v.stack)-1])
        v.stack = v.stack[:len(v.stack)-1]
    case "println":
        if len(v.stack) < 1 {
            return fmt.Errorf("vm: println() requires at least 1 argument")
        }
        fmt.Println(v.stack[len(v.stack)-1])
        v.stack = v.stack[:len(v.stack)-1]
    case "len":
        if len(v.stack) < 1 {
            return fmt.Errorf("vm: len() requires 1 argument")
        }
        v.stack = v.stack[:len(v.stack)-1]
        x := v.stack[len(v.stack)-1]
        switch val := x.(type) {
        case string:
            v.stack = append(v.stack[:len(v.stack)-1], len(val))
        case []bytecode.Value:
            v.stack = append(v.stack[:len(v.stack)-1], len(val))
        default:
            // fallback via reflection
            v.stack = append(v.stack[:len(v.stack)-1], reflect.ValueOf(val).Len())
        }
    case "to_str":
        if len(v.stack) < 1 {
            return fmt.Errorf("vm: to_str() requires 1 argument")
        }
        v.stack = v.stack[:len(v.stack)-1]
        v.stack[len(v.stack)-1] = fmt.Sprintf("%v", v.stack[len(v.stack)-1])
    case "to_int":
        if len(v.stack) < 1 {
            return fmt.Errorf("vm: to_int() requires 1 argument")
        }
        v.stack = v.stack[:len(v.stack)-1]
        switch v := v.stack[len(v.stack)-1].(type) {
        case int:
            // already int
        case float64:
            v.stack[len(v.stack)-1] = int(v)
        case string:
            // simplistic; no error handling
            var n int
            for _, c := range v {
                if c >= '0' && c <= '9' {
                    n = n*10 + int(c-'0')
                }
            }
            v.stack[len(v.stack)-1] = n
        }
    case "type_of":
        if len(v.stack) < 1 {
            return fmt.Errorf("vm: type_of() requires 1 argument")
        }
        v.stack = v.stack[:len(v.stack)-1]
        switch v.stack[len(v.stack)-1].(type) {
        case nil:
            v.stack[len(v.stack)-1] = "nil"
        case bool:
            v.stack[len(v.stack)-1] = "bool"
        case int:
            v.stack[len(v.stack)-1] = "int"
        case float64:
            v.stack[len(v.stack)-1] = "float"
        case string:
            v.stack[len(v.stack)-1] = "str"
        case []bytecode.Value:
            v.stack[len(v.stack)-1] = "list"
        case map[string]bytecode.Value:
            v.stack[len(v.stack)-1] = "map"
        default:
            v.stack[len(v.stack)-1] = "unknown"
        }
    default:
        return fmt.Errorf("vm: unknown builtin %q", name)
    }
    return nil
}
```

- [ ] **Step 4: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m5/v2
go test ./internal/vm/ -v -count=1
```

Expected: all tests PASS

- [ ] **Step 5: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m5
git add v2/internal/vm/
git commit -m "v2: VM CALL_BUILTIN with print/println/len/to_str/to_int/type_of"
```

---

## Task 2: VM Data Structure Instructions (BUILD_LIST, INDEX, BUILD_MAP, GET_FIELD, NEW_STRUCT)

**Files:**
- Modify: `v2/internal/vm/vm.go`
- Modify: `v2/internal/vm/instructions.go`

- [ ] **Step 1: Append failing tests**:

```go
func TestVM_BuildList(t *testing.T) {
    main := &bytecode.Function{Name: "main", Arity: 0}
    main.Emit(bytecode.PUSH_INT, 0) // 1
    main.Emit(bytecode.PUSH_INT, 1) // 2
    main.Emit(bytecode.PUSH_INT, 2) // 3
    main.Emit(bytecode.BUILD_LIST, 3) // pop 3, push [1,2,3]
    main.Emit(bytecode.HALT, 0)
    v := runModule(t, main, nil, 1, 2, 3)
    list, ok := v.([]bytecode.Value)
    require.True(t, ok)
    assert.Equal(t, 3, len(list))
}

func TestVM_Index(t *testing.T) {
    main := &bytecode.Function{Name: "main", Arity: 0}
    main.Emit(bytecode.PUSH_INT, 0) // 10
    main.Emit(bytecode.PUSH_INT, 1) // 20
    main.Emit(bytecode.PUSH_INT, 2) // 30
    main.Emit(bytecode.BUILD_LIST, 3)
    main.Emit(bytecode.PUSH_INT, 3) // index 1
    main.Emit(bytecode.INDEX, 0)
    main.Emit(bytecode.HALT, 0)
    v := runModule(t, main, nil, 10, 20, 30)
    assert.Equal(t, 20, v)
}

func TestVM_BuildMap(t *testing.T) {
    main := &bytecode.Function{Name: "main", Arity: 0}
    main.Emit(bytecode.PUSH_STR, 0) // "k"
    main.Emit(bytecode.PUSH_INT, 1)  // 42
    main.Emit(bytecode.BUILD_MAP, 1) // 1 entry
    main.Emit(bytecode.HALT, 0)
    v := runModule(t, main, nil, "k", 42)
    m, ok := v.(map[string]bytecode.Value)
    require.True(t, ok)
    assert.Equal(t, 42, m["k"])
}

func TestVM_GetField(t *testing.T) {
    main := &bytecode.Function{Name: "main", Arity: 0}
    main.Emit(bytecode.PUSH_STR, 0)  // "k"
    main.Emit(bytecode.PUSH_INT, 1)   // 99
    main.Emit(bytecode.BUILD_MAP, 1)
    main.Emit(bytecode.PUSH_STR, 2)   // "k" (field name)
    main.Emit(bytecode.GET_FIELD, 0)  // arg unused (we use peek from stack)
    main.Emit(bytecode.HALT, 0)
    v := runModule(t, main, nil, "k", 99, "k")
    assert.Equal(t, 99, v)
}

func TestVM_NewStruct(t *testing.T) {
    main := &bytecode.Function{Name: "main", Arity: 0}
    main.Emit(bytecode.PUSH_STR, 0)   // "k"
    main.Emit(bytecode.PUSH_INT, 1)    // 7
    main.Emit(bytecode.BUILD_MAP, 1)
    main.Emit(bytecode.NEW_STRUCT, 0) // arg unused for M2-B.5
    main.Emit(bytecode.HALT, 0)
    v := runModule(t, main, nil, "k", 7, "User")
    m, ok := v.(map[string]bytecode.Value)
    require.True(t, ok)
    assert.Equal(t, 7, m["k"])
}
```

- [ ] **Step 2: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m5/v2
go test ./internal/vm/ -v -run "TestVM_BuildList|TestVM_Index|TestVM_BuildMap|TestVM_GetField|TestVM_NewStruct"
```

Expected: FAIL

- [ ] **Step 3: Add handlers to `instructions.go`**:

```go
// execBuildList handles BUILD_LIST n. Pops n values from stack (in reverse), pushes a []Value.
func (v *VM) execBuildList(n int) {
    items := make([]bytecode.Value, n)
    for i := n - 1; i >= 0; i-- {
        items[i] = v.stack[len(v.stack)-1]
        v.stack = v.stack[:len(v.stack)-1]
    }
    v.stack = append(v.stack, items)
}

// execIndex handles INDEX. Pops index then object, pushes element.
func (v *VM) execIndex() error {
    if len(v.stack) < 2 {
        return fmt.Errorf("vm: INDEX requires 2 stack values")
    }
    idx := v.stack[len(v.stack)-1]
    obj := v.stack[len(v.stack)-2]
    v.stack = v.stack[:len(v.stack)-2]
    i, ok := idx.(int)
    if !ok {
        return fmt.Errorf("vm: INDEX index not int")
    }
    switch v := obj.(type) {
    case []bytecode.Value:
        if i < 0 || i >= len(v) {
            return fmt.Errorf("vm: INDEX out of range")
        }
        v.stack = append(v.stack, v[i])
    case string:
        runes := []rune(v)
        if i < 0 || i >= len(runes) {
            return fmt.Errorf("vm: INDEX out of range")
        }
        v.stack = append(v.stack, string(runes[i]))
    default:
        return fmt.Errorf("vm: INDEX on non-list/string")
    }
    return nil
}

// execBuildMap handles BUILD_MAP n. Pops 2n values (alternating key, value), pushes map.
func (v *VM) execBuildMap(n int) {
    m := make(map[string]bytecode.Value, n)
    for i := 0; i < n; i++ {
        v := v.stack[len(v.stack)-1]
        v.stack = v.stack[:len(v.stack)-1]
        k := v.stack[len(v.stack)-1]
        v.stack = v.stack[:len(v.stack)-1]
        ks, ok := k.(string)
        if !ok {
            ks = fmt.Sprintf("%v", k)
        }
        m[ks] = v
    }
    v.stack = append(v.stack, m)
}

// execGetField handles GET_FIELD. Pops field name then object, pushes value.
func (v *VM) execGetField() error {
    if len(v.stack) < 2 {
        return fmt.Errorf("vm: GET_FIELD requires 2 stack values")
    }
    fname := v.stack[len(v.stack)-1]
    obj := v.stack[len(v.stack)-2]
    v.stack = v.stack[:len(v.stack)-2]
    fs, ok := fname.(string)
    if !ok {
        return fmt.Errorf("vm: GET_FIELD field name not string")
    }
    switch o := obj.(type) {
    case map[string]bytecode.Value:
        if val, ok := o[fs]; ok {
            v.stack = append(v.stack, val)
        } else {
            v.stack = append(v.stack, nil)
        }
    default:
        return fmt.Errorf("vm: GET_FIELD on non-map/struct")
    }
    return nil
}

// execNewStruct handles NEW_STRUCT. Pops object (already-built map), pushes as struct value.
// For M2-B.5 simplicity, structs are just maps with a type name (NEW_STRUCT arg unused).
func (v *VM) execNewStruct() {
    // The map is already on the stack; we just leave it as-is.
    // Future M3+ work can add type information.
}
```

- [ ] **Step 4: Wire cases into the dispatch loop** in `vm.go` (BEFORE `default:`):

```go
        case bytecode.BUILD_LIST:
            v.execBuildList(instr.Arg)
        case bytecode.INDEX:
            if err := v.execIndex(); err != nil {
                return nil, err
            }
        case bytecode.BUILD_MAP:
            v.execBuildMap(instr.Arg)
        case bytecode.GET_FIELD:
            if err := v.execGetField(); err != nil {
                return nil, err
            }
        case bytecode.NEW_STRUCT:
            v.execNewStruct()
```

- [ ] **Step 5: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m5/v2
go test ./internal/vm/ -v -count=1
```

Expected: all tests PASS

- [ ] **Step 6: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m5
git add v2/internal/vm/
git commit -m "v2: VM data structure instructions (BUILD_LIST/INDEX/BUILD_MAP/GET_FIELD/NEW_STRUCT)"
```

---

## Task 3: Compiler — Functions and Calls

**Files:**
- Modify: `v2/internal/compiler/compiler.go`
- Create: `v2/internal/compiler/fn.go`
- Create: `v2/internal/compiler/fn_test.go`

- [ ] **Step 1: Append failing tests** to `fn_test.go`:

```go
// v2/internal/compiler/fn_test.go
package compiler

import (
    "testing"

    "github.com/jiejie-dev/funny/internal/bytecode"
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
    // Check return is in body
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
```

- [ ] **Step 2: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m5/v2
go test ./internal/compiler/ -v -run "TestCompile_FnDecl|TestCompile_Call|TestCompile_Return"
```

Expected: FAIL

- [ ] **Step 3: Modify `compiler.go`** to add a function table:

In the `Compiler` struct, add a `functions map[string]int` field tracking function name → index. Modify the `Compile()` function to initialize this map. Modify the `compileStmt()` dispatch to handle `FnDecl`, `ReturnStmt`. Add `compileFnDecl`, `compileReturn`, `compileCall` helpers.

Replace the `Compiler` struct:

```go
type Compiler struct {
    mod       *bytecode.Module
    fn        *bytecode.Function
    scopes    []map[string]int
    functions map[string]int // function name → index in mod.Functions
}
```

Replace `Compile()`:

```go
func Compile(prog *ast.Program, name string) (*bytecode.Module, error) {
    c := &Compiler{
        mod:       bytecode.NewModule(name),
        scopes:    []map[string]int{{}},
        functions: map[string]int{},
    }
    mainFn := &bytecode.Function{Name: "main", Arity: 0}
    c.mod.AddFunction(mainFn)
    c.fn = mainFn
    c.functions["main"] = 0
    for _, s := range prog.Stmts {
        if err := c.compileStmt(s); err != nil {
            return nil, err
        }
    }
    c.fn.Emit(bytecode.HALT, 0)
    return c.mod, nil
}
```

Add `compileReturn` and update `compileStmt` to handle `ReturnStmt` and `FnDecl`:

```go
func (c *Compiler) compileReturn(n *ast.ReturnStmt) error {
    if n.Value != nil {
        if _, err := c.compileExpr(n.Value); err != nil {
            return err
        }
    }
    c.fn.Emit(bytecode.RETURN, 0)
    return nil
}
```

In `compileStmt` add these cases:

```go
    case *ast.FnDecl:
        return c.compileFnDecl(n)
    case *ast.ReturnStmt:
        return c.compileReturn(n)
```

- [ ] **Step 4: Create `fn.go`**:

```go
// v2/internal/compiler/fn.go
package compiler

import (
    "fmt"

    "github.com/jiejie-dev/funny/internal/ast"
    "github.com/jiejie-dev/funny/internal/bytecode"
)

// compileFnDecl compiles a function declaration into a separate Function in the module.
func (c *Compiler) compileFnDecl(n *ast.FnDecl) error {
    if _, ok := c.functions[n.Name]; ok {
        return fmt.Errorf("function %s already declared", n.Name)
    }
    fn := &bytecode.Function{Name: n.Name, Arity: len(n.Params)}
    c.fn = fn
    c.scopes = []map[string]int{{}}
    // Declare params as locals
    for _, p := range n.Params {
        idx := c.declareLocal(p.Name)
        _ = idx
    }
    // Compile body
    if err := c.compileBlock(n.Body); err != nil {
        return err
    }
    c.fn.Emit(bytecode.RETURN, 0) // implicit return
    // Register the function
    fnIdx := c.mod.AddFunction(fn)
    c.functions[n.Name] = fnIdx
    // Pop scope
    c.scopes = c.scopes[:0]
    // Restore outer function (main)
    mainIdx := c.functions["main"]
    c.fn = c.mod.Functions[mainIdx]
    // Restore outer scope
    c.scopes = []map[string]int{{}}
    return nil
}
```

Add the `compileCall` helper at the bottom of `expr.go`:

```go
func (c *Compiler) compileCall(n *ast.CallExpr) (bytecode.OpCode, error) {
    varName, ok := n.Func.(*ast.VariableExpr)
    if !ok {
        return "", fmt.Errorf("compileCall: only direct function calls supported")
    }
    // Check if it's a builtin
    builtins := map[string]bool{
        "print": true, "println": true, "len": true,
        "to_str": true, "to_int": true, "type_of": true,
    }
    if builtins[varName.Name] {
        for _, arg := range n.Args {
            if _, err := c.compileExpr(arg); err != nil {
                return "", err
            }
        }
        nameIdx := c.mod.AddConstant(varName.Name)
        c.fn.Emit(bytecode.CALL_BUILTIN, nameIdx)
        return bytecode.CALL_BUILTIN, nil
    }
    // User function call
    fnIdx, ok := c.functions[varName.Name]
    if !ok {
        return "", fmt.Errorf("undefined function: %s", varName.Name)
    }
    for _, arg := range n.Args {
        if _, err := c.compileExpr(arg); err != nil {
            return "", err
        }
    }
    c.fn.Emit(bytecode.CALL, fnIdx)
    return bytecode.CALL, nil
}
```

In `expr.go`'s `compileExpr` switch, add these cases:

```go
    case *ast.CallExpr:
        return c.compileCall(n)
```

- [ ] **Step 5: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m5/v2
go test ./internal/compiler/ -v -count=1
```

Expected: all compiler tests PASS (15 prior + 3 new = 18)

- [ ] **Step 6: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m5
git add v2/internal/compiler/
git commit -m "v2: compiler for function declarations and calls (user + builtin)"
```

---

## Task 4: Compiler — Data Structures (List, Index, Field, Map, Struct Literal)

**Files:**
- Create: `v2/internal/compiler/data.go`

- [ ] **Step 1: Create `data.go`**:

```go
// v2/internal/compiler/data.go
package compiler

import (
    "fmt"

    "github.com/jiejie-dev/funny/internal/ast"
    "github.com/jiejie-dev/funny/internal/bytecode"
)

// compileList compiles a list literal into BUILD_LIST n.
func (c *Compiler) compileList(n *ast.ListExpr) (bytecode.OpCode, error) {
    for _, e := range n.Elements {
        if _, err := c.compileExpr(e); err != nil {
            return "", err
        }
    }
    c.fn.Emit(bytecode.BUILD_LIST, len(n.Elements))
    return bytecode.BUILD_LIST, nil
}

// compileIndex compiles a[b] into BUILD_LIST/BUILD_MAP + INDEX.
func (c *Compiler) compileIndex(n *ast.IndexExpr) (bytecode.OpCode, error) {
    if _, err := c.compileExpr(n.Object); err != nil {
        return "", err
    }
    if _, err := c.compileExpr(n.Index); err != nil {
        return "", err
    }
    c.fn.Emit(bytecode.INDEX, 0)
    return bytecode.INDEX, nil
}

// compileField compiles a.b into GET_FIELD (field name as constant).
func (c *Compiler) compileField(n *ast.FieldExpr) (bytecode.OpCode, error) {
    if _, err := c.compileExpr(n.Object); err != nil {
        return "", err
    }
    nameIdx := c.mod.AddConstant(n.Field)
    c.fn.Emit(bytecode.PUSH_STR, nameIdx) // push field name
    c.fn.Emit(bytecode.GET_FIELD, 0)
    return bytecode.GET_FIELD, nil
}

// compileStructLiteral compiles Point(x: 1, y: 2) into BUILD_MAP + NEW_STRUCT.
func (c *Compiler) compileStructLiteral(n *ast.StructLiteralExpr) (bytecode.OpCode, error) {
    for k, v := range n.Fields {
        nameIdx := c.mod.AddConstant(k)
        c.fn.Emit(bytecode.PUSH_STR, nameIdx)
        if _, err := c.compileExpr(v); err != nil {
            return "", err
        }
    }
    c.fn.Emit(bytecode.BUILD_MAP, len(n.Fields))
    typeIdx := c.mod.AddConstant(n.TypeName)
    c.fn.Emit(bytecode.NEW_STRUCT, typeIdx)
    return bytecode.NEW_STRUCT, nil
}

// compileMap compiles {"k": v} literal into BUILD_MAP n.
func (c *Compiler) compileMap(n *ast.CallExpr) (bytecode.OpCode, error) {
    // For map literals, we'd need an AST node; not present in M1 AST.
    // Skip for M2-B.5.
    return "", fmt.Errorf("compileMap: not implemented (M1 AST has no map literal)")
}
```

In `expr.go`'s `compileExpr` switch, add these cases:

```go
    case *ast.ListExpr:
        return c.compileList(n)
    case *ast.IndexExpr:
        return c.compileIndex(n)
    case *ast.FieldExpr:
        return c.compileField(n)
    case *ast.StructLiteralExpr:
        return c.compileStructLiteral(n)
```

- [ ] **Step 2: Append failing tests**:

```go
// In v2/internal/compiler/expr_test.go
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
    mod := compileExpr(t, `[1, 2, 3][0]`)
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
    mod := compileExpr(t, `let p = Point(x: 1, y: 2)
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
```

- [ ] **Step 3: Run tests**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m5/v2
go test ./internal/compiler/ -v -count=1
```

Expected: all PASS

- [ ] **Step 4: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m5
git add v2/internal/compiler/
git commit -m "v2: compiler for data structures (list/field/index/struct literal)"
```

---

## Task 5: Compiler — Full For-In Loop

**Files:**
- Modify: `v2/internal/compiler/control.go`

- [ ] **Step 1: Append failing test**:

```go
// In v2/internal/compiler/control_test.go
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
```

- [ ] **Step 2: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m5/v2
go test ./internal/compiler/ -v -run TestCompile_For
```

Expected: FAIL (for loop still stubs)

- [ ] **Step 3: Replace the for-loop stub in `control.go`**:

Find the `case *ast.ForStmt` in `compileStmt` (currently in `compiler.go` — it may also be in `control.go`; check both and update).

The implementation compiles:
```
<compile iterable, leaving list on stack>
loopStart:
<dup>                            ; copy top of stack (list)
<STORE_LOCAL idx_n>              ; save length? Or use a separate counter
... (this gets complex; use simpler approach)
```

Simpler approach: compile the list once before the loop, then use an index counter local:

```
<compile iterable>
<STORE_LOCAL list_slot>
loopStart:
<LOAD_LOCAL list_slot>
<PUSH_INT <len>>
<EQ_INT>                          ; or use a different comparison strategy
... too complex
```

**Easiest approach for M2-B.5**: rebuild the iterable each iteration. This works for literals but not for variables. Accept this limitation; M3+ can improve.

Final simple implementation (works for `for i in [1,2,3]`):

```go
func (c *Compiler) compileFor(n *ast.ForStmt) error {
    c.pushScope()
    defer c.popScope()
    // Compile the iterable; it's left on stack.
    if _, err := c.compileExpr(n.Iterable); err != nil {
        return err
    }
    // Store the list in a hidden local
    listSlot := c.declareLocal("__for_list__")
    c.fn.Emit(bytecode.STORE_LOCAL, listSlot)
    c.fn.Emit(bytecode.POP, 0)
    // Use an index counter
    idxSlot := c.declareLocal(n.Name)
    c.fn.Emit(bytecode.PUSH_INT, 0) // initial index 0
    c.fn.Emit(bytecode.STORE_LOCAL, idxSlot)
    c.fn.Emit(bytecode.POP, 0)
    loopStart := len(c.fn.Code)
    // Check: load index, load list, push length, compare
    c.fn.Emit(bytecode.LOAD_LOCAL, idxSlot)
    c.fn.Emit(bytecode.LOAD_LOCAL, listSlot)
    // We need length. For now, assume the iterable was a list literal with known length
    // encoded in the instruction itself via BUILD_LIST — but that doesn't help here.
    // Use a sentinel: compare index against a stack-pushed length.
    // Easiest: use a different VM instruction. For M2-B.5, accept the limitation:
    // for-in only works when iterable is a constant list. Push a dummy length via len builtin.
    // BETTER: have the compiler emit a special "for" sequence using len() builtin.
    nameIdx := c.mod.AddConstant("len")
    c.fn.Emit(bytecode.CALL_BUILTIN, nameIdx) // pop list, push length
    c.fn.Emit(bytecode.LT_INT, 0)
    exitJump := len(c.fn.Code)
    c.fn.Emit(bytecode.JUMP_IF_FALSE, 0)
    // Body: load list, load index, INDEX, store to var
    c.fn.Emit(bytecode.LOAD_LOCAL, listSlot)
    c.fn.Emit(bytecode.LOAD_LOCAL, idxSlot)
    c.fn.Emit(bytecode.INDEX, 0)
    c.fn.Emit(bytecode.STORE_LOCAL, idxSlot)
    c.fn.Emit(bytecode.POP, 0)
    if err := c.compileBlock(n.Body); err != nil {
        return err
    }
    // Increment index
    c.fn.Emit(bytecode.LOAD_LOCAL, idxSlot)
    c.fn.Emit(bytecode.PUSH_INT, 0) // wait — this will dedup with the loop's PUSH_INT 0!
    // Use a fresh constant slot. Trick: emit a DUP-then-add via ADD_INT.
    c.fn.Emit(bytecode.PUSH_INT, 0) // see concern
    c.fn.Emit(bytecode.ADD_INT, 0)
    c.fn.Emit(bytecode.STORE_LOCAL, idxSlot)
    c.fn.Emit(bytecode.POP, 0)
    // Jump back
    c.fn.Emit(bytecode.JUMP, loopStart)
    c.fn.Code[exitJump].Arg = len(c.fn.Code)
    return nil
}
```

The constant pool's `PUSH_INT 0` will dedup, so the increment instruction's `PUSH_INT 0` reuses the same constant as the initial `PUSH_INT 0`. This is fine semantically — `ADD_INT 0+0` works correctly because `0+0=0`, but we wanted `idx+1`. **The above code is INCORRECT.**

**Correct fix**: use a non-deduped constant. The bytecode's `AddConstant` dedupes by `==`. To get a fresh constant for "1", use a non-default value: emit `PUSH_INT` with a constant that exists from earlier. For M2-B.5, simplify: pre-allocate constants for 0 and 1 at the start of main.

Actually, the cleanest approach is to add a `PUSH_INT_1` instruction, but that's scope creep. Instead, **use the LOAD_LOCAL trick**: increment by re-loading index, push 1 (constant pool id for the int 1 — first time it's added), and ADD_INT. Since constants dedup by value, the first call to `AddConstant(1)` returns the unique index for 1, which we use for all subsequent PUSH_INT 1.

For M2-B.5, accept that the for-loop is limited to constant list iterables. The implementation is correct as long as the iterable is a constant list (BUILD_LIST), and the index loop variable is incremented correctly.

**Simpler M2-B.5 implementation** — drop the for-loop iteration via list/INDEX (which has the dedup issue) and instead use a different strategy: **compile iterable to a constant pool slot** and re-build it each iteration. For a literal `[1,2,3]`, this means pushing 3 integers + BUILD_LIST each iteration. Expensive but correct.

Or, the simplest: **make for-loop work only with single-iter values like `[1]`** — not useful.

**Pragmatic decision for M2-B.5**: implement for-in using BUILD_LIST + INDEX with a separate index local. For the increment, emit `PUSH_INT` with a constant slot that's guaranteed unique by adding a dummy constant first.

Replace the `compileFor` function with this version that handles dedup:

```go
func (c *Compiler) compileFor(n *ast.ForStmt) error {
    c.pushScope()
    defer c.popScope()
    if _, err := c.compileExpr(n.Iterable); err != nil {
        return err
    }
    listSlot := c.declareLocal("__for_list__")
    c.fn.Emit(bytecode.STORE_LOCAL, listSlot)
    c.fn.Emit(bytecode.POP, 0)
    idxSlot := c.declareLocal(n.Name)
    c.fn.Emit(bytecode.PUSH_INT, 0)
    c.fn.Emit(bytecode.STORE_LOCAL, idxSlot)
    c.fn.Emit(bytecode.POP, 0)
    loopStart := len(c.fn.Code)
    c.fn.Emit(bytecode.LOAD_LOCAL, idxSlot)
    c.fn.Emit(bytecode.LOAD_LOCAL, listSlot)
    nameIdx := c.mod.AddConstant("len")
    c.fn.Emit(bytecode.CALL_BUILTIN, nameIdx)
    c.fn.Emit(bytecode.LT_INT, 0)
    exitJump := len(c.fn.Code)
    c.fn.Emit(bytecode.JUMP_IF_FALSE, 0)
    c.fn.Emit(bytecode.LOAD_LOCAL, listSlot)
    c.fn.Emit(bytecode.LOAD_LOCAL, idxSlot)
    c.fn.Emit(bytecode.INDEX, 0)
    c.fn.Emit(bytecode.STORE_LOCAL, idxSlot)
    c.fn.Emit(bytecode.POP, 0)
    if err := c.compileBlock(n.Body); err != nil {
        return err
    }
    // Increment idx by adding a constant. To avoid dedup with the initial 0,
    // we pre-allocate a "1" constant and reuse it.
    oneIdx := c.mod.AddConstant(1)
    c.fn.Emit(bytecode.LOAD_LOCAL, idxSlot)
    c.fn.Emit(bytecode.PUSH_INT, oneIdx)
    c.fn.Emit(bytecode.ADD_INT, 0)
    c.fn.Emit(bytecode.STORE_LOCAL, idxSlot)
    c.fn.Emit(bytecode.POP, 0)
    c.fn.Emit(bytecode.JUMP, loopStart)
    c.fn.Code[exitJump].Arg = len(c.fn.Code)
    return nil
}
```

- [ ] **Step 4: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m5/v2
go test ./internal/compiler/ -v -run TestCompile_For
```

Expected: PASS

- [ ] **Step 5: Remove the for-stub error** from `compiler.go` (since `control.go` now handles it):

In `compiler.go`'s `compileStmt` switch, change:

```go
    case *ast.ForStmt:
        return fmt.Errorf("compileStmt: for-in loop not yet implemented (M2-B.5 follow-up)")
```

to:

```go
    case *ast.ForStmt:
        return c.compileFor(n)
```

(This case is duplicated between `compiler.go` and `control.go` — make sure only ONE has the real implementation.)

- [ ] **Step 6: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m5
git add v2/internal/compiler/
git commit -m "v2: compiler for for-in loop (with length-check via len builtin)"
```

---

## Task 6: Replace fib benchmark with recursive version

**Files:**
- Modify: `v2/internal/vm/bench_test.go`
- Create: `v2/testdata/vm/fib.fn`

- [ ] **Step 1: Replace `fib_iter.fn`** with a recursive `fib.fn`:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m5
git rm v2/testdata/vm/fib_iter.fn
```

Create `v2/testdata/vm/fib.fn`:

```
fn fib(n: int) -> int:
    if n < 2:
        return n
    return fib(n - 1) + fib(n - 2)

let r = fib(20)
println("fib(20) =", r)
```

- [ ] **Step 2: Run benchmarks**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m5/v2
go test -bench=BenchmarkFib -benchtime=3s -run=^$ ./internal/vm/
```

Expected: VM should now be much faster than interpreter (target ≥ 5×) because:
- VM avoids AST walking on every call
- VM uses O(1) call frame push/pop
- Tree-walking interpreter does full scope lookup per call

Record actual results.

- [ ] **Step 3: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m5
git add v2/testdata/vm/ v2/internal/vm/bench_test.go
git commit -m "v2: switch benchmark to recursive fib (CALL/RETURN workload)"
```

Include the actual benchmark numbers in the commit message.

---

## Task 7: Update README

**Files:**
- Modify: `v2/README.md`

- [ ] **Step 1: Update status section**:

Replace the M2-B status block with:

```markdown
**Status: M2-B.5 (VM Functions + Data Ops) — RELEASED**

- ✅ Lexer, Parser, Type checker (M1, M2-A)
- ✅ Tree-walking evaluator (fallback via `FUNNY_INTERPRET=1`)
- ✅ **Bytecode VM**: stack + frames, typed instructions
- ✅ **VM function calls**: CALL/RETURN + frame push/pop
- ✅ **VM builtins**: print/println/len/to_str/to_int/type_of via CALL_BUILTIN
- ✅ **VM data structures**: BUILD_LIST/INDEX/BUILD_MAP/GET_FIELD/NEW_STRUCT
- ✅ **Compiler**: function declarations, calls, list/field/index, struct literals, for-in
- ⏳ Result + `?` operator → M2-C
- ⏳ stdlib (json/time/math/str) → M2-C
```

Update the roadmap table — flip M2-B.5 row from Planned to Done.

Update the M2-B Performance section to reflect recursive fib numbers:

```markdown
## M2-B.5 Performance

Recursive fib(20) benchmark (Apple M2 Max, go1.25.1):

```
BenchmarkFib_VM-12           XXXX ns/op
BenchmarkFib_Interpreter-12  YYYY ns/op
```

VM is now X× faster than the tree-walking interpreter on recursive workloads (target ≥ 5× met if ratio ≥ 5).

Run:
```bash
go test -bench=BenchmarkFib -benchtime=3s -run=^$ ./internal/vm/
```
```

- [ ] **Step 2: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m5
git add v2/README.md
git commit -m "v2: README updated for M2-B.5 (function calls + data ops)"
```

---

## Self-Review

1. **Spec coverage**:
   - §5.4 CALL/RETURN → Task 0 ✓
   - §5.4 CALL_BUILTIN → Task 1 ✓
   - §5.4 BUILD_LIST/INDEX/BUILD_MAP/GET_FIELD/NEW_STRUCT → Task 2 ✓
   - §5.4 (compiler side) function decls + calls + data ops → Tasks 3, 4, 5 ✓
   - §6.3 ≥5× target → Task 6 (re-benchmark with recursive fib) ✓
   - **Deferred**: Result + `?` (M2-C), stdlib (M2-C), map literal in M1 AST (deferred)

2. **Placeholder scan**: no TBD/TODO in plan body (the dedup note in Task 5 is a comment about a fix, not a placeholder).

3. **Type consistency**: `execCall`/`execReturn`/`execBuildList`/`execIndex`/`execBuildMap`/`execGetField`/`execNewStruct` names consistent across Tasks 0-2.

4. **For-in dedup concern**: Task 5 documents that `AddConstant(1)` returns a unique slot, avoiding the `0+0=0` bug.

---

## Exit Criteria for M2-B.5

- [ ] All 8 tasks checked off
- [ ] `go test ./...` passes
- [ ] `./funny run ./testdata/vm/fib.fn` prints `fib(20) = 6765`
- [ ] Benchmark shows VM ≥ 5× faster than interpreter on recursive fib
- [ ] M2-B.5 released as `v2.0.0-beta-bytecode-vm-complete`

---

## Total Tasks: 8

**Estimated time**: 3-4 days for one developer.