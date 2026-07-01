# Funny v2 M2-C: Result + ? Operator + Stdlib Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `Result` runtime support + `?` operator (error-propagation) + base stdlib modules (json, time, math, str), completing the M2 milestone.

**Architecture:** `Result` is a tagged union at runtime (`{tag: "ok", val: any}` or `{tag: "err", val: any}`). The `?` operator compiles to a `TRY_OR_RETURN` VM instruction that checks the top-of-stack Result and unwraps Ok or returns Err from the current function. Stdlib modules are registered as new builtins (CALL_BUILTIN) implemented in Go (json.Marshal, time.Now, math.Sqrt, string manipulation).

**Tech Stack:** Go 1.22+ stdlib (`encoding/json`, `time`, `math`, `strings`, `strconv`).

**Reference Spec:** `docs/superpowers/specs/2026-07-01-funny-v2-ai-native-language-design.md` §2.2 (Result type), §5.4 (Result+? semantics), §6.3 (M2-C deliverables).

**Scope:** M2-C only. M3 (agent protocol) and M4 (MCP) are separate plans.

---

## File Structure

Modified files:
- `v2/internal/bytecode/opcode.go` — add `TRY_OR_RETURN` OpCode
- `v2/internal/compiler/expr.go` — add `?` operator compilation
- `v2/internal/parser/statement.go` (or new file) — parse `?` suffix
- `v2/internal/types/check.go` — type-check `?` (operand must be Result)
- `v2/internal/vm/vm.go` — add `TRY_OR_RETURN` handler
- `v2/internal/vm/builtins.go` — add stdlib builtins
- `v2/internal/vm/bench_test.go` — maybe a stdlib benchmark
- `v2/README.md` — M2-C status

New files:
- `v2/internal/vm/result.go` — Result constructor + unwrap helpers

---

## Conventions

- `Result` runtime representation: `map[string]any` with key "tag" = "ok" or "err", and key "val" = any.
- `ok(value)` and `err(value)` are builtin constructors that produce Result maps.
- `?` postfix on a Result expression: if Ok, unwrap to inner value; if Err, return Err early from current function.
- Stdlib builtins are registered with the same CALL_BUILTIN mechanism as Task 1. New builtins: `to_json`, `parse_json`, `now`, `time_unix`, `time_format`, `sqrt`, `pow`, `abs`, `str_upper`, `str_lower`, `str_split`, `str_contains`, `str_replace`.
- Each task ends with a commit.

---

## Task 0: Result Runtime Constructors (ok, err)

**Files:**
- Create: `v2/internal/vm/result.go`
- Modify: `v2/internal/vm/builtins.go` (add ok/err dispatch)
- Modify: `v2/internal/vm/instructions_test.go` (tests)

- [ ] **Step 1: Append failing tests**:

```go
func TestVM_ResultOK(t *testing.T) {
    main := &bytecode.Function{Name: "main", Arity: 0}
    main.Emit(bytecode.PUSH_INT, 0) // value 42
    main.Emit(bytecode.CALL_BUILTIN, 1) // "ok" -> wraps in Result{tag: "ok", val: 42}
    main.Emit(bytecode.HALT, 0)
    v := runModule(t, main, nil, 42, "ok")
    m, ok := v.(map[string]bytecode.Value)
    require.True(t, ok)
    assert.Equal(t, "ok", m["tag"])
    assert.Equal(t, 42, m["val"])
}

func TestVM_ResultErr(t *testing.T) {
    main := &bytecode.Function{Name: "main", Arity: 0}
    main.Emit(bytecode.PUSH_STR, 0) // "oops"
    main.Emit(bytecode.CALL_BUILTIN, 1) // "err"
    main.Emit(bytecode.HALT, 0)
    v := runModule(t, main, nil, "oops", "err")
    m, ok := v.(map[string]bytecode.Value)
    require.True(t, ok)
    assert.Equal(t, "err", m["tag"])
    assert.Equal(t, "oops", m["val"])
}
```

- [ ] **Step 2: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-c/v2
go test ./internal/vm/ -v -run "TestVM_ResultOK|TestVM_ResultErr"
```

Expected: FAIL

- [ ] **Step 3: Create `v2/internal/vm/result.go`**:

```go
// v2/internal/vm/result.go
package vm

import "github.com/jiejie-dev/funny/v2/internal/bytecode"

// makeResult constructs a Result runtime value: map{tag, val}.
func makeResult(tag string, val bytecode.Value) bytecode.Value {
    return map[string]bytecode.Value{
        "tag": tag,
        "val": val,
    }
}

// isResult reports whether v is a Result runtime value.
func isResult(v bytecode.Value) bool {
    m, ok := v.(map[string]bytecode.Value)
    if !ok {
        return false
    }
    _, hasTag := m["tag"]
    _, hasVal := m["val"]
    return hasTag && hasVal
}

// resultTag returns "ok" or "err" (or "" if not a Result).
func resultTag(v bytecode.Value) string {
    m, ok := v.(map[string]bytecode.Value)
    if !ok {
        return ""
    }
    tag, _ := m["tag"].(string)
    return tag
}

// resultVal returns the inner value of a Result.
func resultVal(v bytecode.Value) bytecode.Value {
    m, _ := v.(map[string]bytecode.Value)
    return m["val"]
}
```

- [ ] **Step 4: Add `ok`/`err` cases to `builtins.go`'s `execCallBuiltin`**:

Append these cases inside the `switch name` block (before the `default:` case):

```go
    case "ok":
        if len(v.stack) < 1 {
            return fmt.Errorf("vm: ok() requires 1 argument")
        }
        val := v.stack[len(v.stack)-1]
        v.stack = v.stack[:len(v.stack)-1]
        v.stack = append(v.stack, makeResult("ok", val))
    case "err":
        if len(v.stack) < 1 {
            return fmt.Errorf("vm: err() requires 1 argument")
        }
        val := v.stack[len(v.stack)-1]
        v.stack = v.stack[:len(v.stack)-1]
        v.stack = append(v.stack, makeResult("err", val))
```

- [ ] **Step 5: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-c/v2
go test ./internal/vm/ -v -count=1
```

Expected: 33 tests PASS (31 prior + 2 new)

- [ ] **Step 6: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-c
git add v2/internal/vm/
git commit -m "v2: VM Result runtime (ok/err constructors)"
```

---

## Task 1: `?` Operator — Parser Support

**Files:**
- Modify: `v2/internal/parser/expression.go`

The `?` operator is a postfix operator on expressions that returns a `Result`. It compiles at parse time to a `TryExpr` AST node (or similar) that wraps the inner expression.

**Step 1: Append failing test** to `parser_test.go`:

```go
func TestParser_TryOperator(t *testing.T) {
    src := `let r = ok(42)?
    r.val
`
    p := New(src, "")
    prog, err := p.Parse()
    assert.NoError(t, err)
    require.Len(t, prog.Stmts, 2)
}

func TestParser_TryOperator_OnCall(t *testing.T) {
    src := `let r = divide(10, 2)?
    println(r)
`
    p := New(src, "")
    _, err := p.Parse()
    assert.NoError(t, err)
}
```

- [ ] **Step 2: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-c/v2
go test ./internal/parser/ -v -run "TestParser_Try"
```

Expected: FAIL (parser doesn't recognize `?`)

- [ ] **Step 3: Add `TryExpr` AST node** to `v2/internal/ast/ast.go`:

```go
// TryExpr is a postfix-? expression: `expr?` — propagates Err, unwraps Ok.
type TryExpr struct {
    NodePos Pos
    Inner   Expression
}

func (e *TryExpr) Pos() Pos        { return e.NodePos }
func (e *TryExpr) exprMarker()     {}
func (e *TryExpr) nodeMarker()     {}
func (e *TryExpr) String() string {
    return e.Inner.String() + "?"
}
```

- [ ] **Step 4: Add `parseTry` to `parser/expression.go`**:

Add to `parsePostfix()` (the method that handles `f()`, `.`, `[]`):

```go
case lexer.QUESTION:
    pos := astPos(p.cur.Pos)
    p.advance()
    return &ast.TryExpr{NodePos: pos, Inner: left}, nil
```

This requires importing `github.com/jiejie-dev/funny/v2/internal/ast` in `expression.go` if not already present (it is).

- [ ] **Step 5: Add `TryExpr` to `compileExpr` switch** in `v2/internal/compiler/expr.go`:

```go
case *ast.TryExpr:
    return c.compileTry(n)
```

- [ ] **Step 6: Stub `compileTry`** in `expr.go` (returns error for now; Task 2 implements it):

```go
func (c *Compiler) compileTry(n *ast.TryExpr) (bytecode.OpCode, error) {
    return "", fmt.Errorf("compileTry: not yet implemented (Task 2)")
}
```

- [ ] **Step 7: Add `TryExpr` to `CheckExpr` switch** in `v2/internal/types/check.go`:

```go
case *ast.TryExpr:
    return checkTry(n, env)
```

- [ ] **Step 8: Stub `checkTry`** in `check.go`:

```go
func checkTry(n *ast.TryExpr, env *Env) (Type, error) {
    return nil, fmt.Errorf("checkTry: not yet implemented (Task 2)")
}
```

- [ ] **Step 9: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-c/v2
go test ./internal/parser/ -v -run TestParser_Try
```

Expected: PASS (parser now recognizes `?`)

- [ ] **Step 10: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-c
git add v2/internal/ast/ v2/internal/parser/ v2/internal/compiler/ v2/internal/types/
git commit -m "v2: parser/compiler/types recognize ? postfix (try) operator"
```

---

## Task 2: `?` Operator — Compiler + VM Implementation (TRY_OR_RETURN instruction)

**Files:**
- Modify: `v2/internal/bytecode/opcode.go` (add TRY_OR_RETURN)
- Modify: `v2/internal/vm/vm.go` (add handler)
- Modify: `v2/internal/compiler/expr.go` (real compileTry)
- Modify: `v2/internal/types/check.go` (real checkTry)

- [ ] **Step 1: Add `TRY_OR_RETURN` to `opcode.go`** (in the control flow group):

```go
TRY_OR_RETURN OpCode = "TRY_OR_RETURN"
```

- [ ] **Step 2: Append failing test** to `v2/internal/vm/instructions_test.go`:

```go
func TestVM_TryOrReturn_Ok(t *testing.T) {
    // fn foo() -> Result:
    //   return ok(42)?
    // main:
    //   CALL foo, HALT
    fn := &bytecode.Function{Name: "foo", Arity: 0, NumLocals: 0}
    fn.Emit(bytecode.PUSH_INT, 0) // 42
    fn.Emit(bytecode.CALL_BUILTIN, 1) // "ok"
    fn.Emit(bytecode.TRY_OR_RETURN, 0)
    fn.Emit(bytecode.RETURN, 0)
    main := &bytecode.Function{Name: "main", Arity: 0}
    main.Emit(bytecode.CALL, 1)
    main.Emit(bytecode.HALT, 0)
    v := runModule(t, main, []*bytecode.Function{fn}, 42, "ok")
    m, ok := v.(map[string]bytecode.Value)
    require.True(t, ok)
    assert.Equal(t, "ok", m["tag"])
    assert.Equal(t, 42, m["val"])
}

func TestVM_TryOrReturn_Err(t *testing.T) {
    // fn foo() -> Result:
    //   return err("boom")?
    // main:
    //   CALL foo, HALT
    fn := &bytecode.Function{Name: "foo", Arity: 0, NumLocals: 0}
    fn.Emit(bytecode.PUSH_STR, 0) // "boom"
    fn.Emit(bytecode.CALL_BUILTIN, 1) // "err"
    fn.Emit(bytecode.TRY_OR_RETURN, 0)
    fn.Emit(bytecode.RETURN, 0) // dead code
    main := &bytecode.Function{Name: "main", Arity: 0}
    main.Emit(bytecode.CALL, 1)
    main.Emit(bytecode.HALT, 0)
    v := runModule(t, main, []*bytecode.Function{fn}, "boom", "err")
    m, ok := v.(map[string]bytecode.Value)
    require.True(t, ok)
    assert.Equal(t, "err", m["tag"])
    assert.Equal(t, "boom", m["val"])
}
```

- [ ] **Step 3: Add TRY_OR_RETURN handler to `vm.go`** (in the execute switch):

```go
        case bytecode.TRY_OR_RETURN:
            if len(v.stack) < 1 {
                return nil, fmt.Errorf("vm: TRY_OR_RETURN on empty stack")
            }
            top := v.stack[len(v.stack)-1]
            if !isResult(top) {
                return nil, fmt.Errorf("vm: TRY_OR_RETURN operand is not a Result")
            }
            if resultTag(top) == "err" {
                // Pop the err Result, return it from the current function.
                v.stack = v.stack[:len(v.stack)-1]
                if err := v.execReturn(); err != nil {
                    return nil, err
                }
            } else {
                // Ok: replace the Result on stack with its inner value.
                v.stack[len(v.stack)-1] = resultVal(top)
            }
```

- [ ] **Step 4: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-c/v2
go test ./internal/vm/ -v -run "TestVM_TryOrReturn"
```

Expected: PASS

- [ ] **Step 5: Real `compileTry` in `expr.go`**:

Replace the stub:

```go
// compileTry compiles `expr?`. Emits `expr` followed by TRY_OR_RETURN.
func (c *Compiler) compileTry(n *ast.TryExpr) (bytecode.OpCode, error) {
    if _, err := c.compileExpr(n.Inner); err != nil {
        return "", err
    }
    c.fn.Emit(bytecode.TRY_OR_RETURN, 0)
    return "ok", nil
}
```

- [ ] **Step 6: Real `checkTry` in `check.go`**:

Replace the stub:

```go
// checkTry type-checks `expr?` — expr must be Result[T, E]; result is T.
func checkTry(n *ast.TryExpr, env *Env) (Type, error) {
    innerT, err := CheckExpr(n.Inner, env)
    if err != nil {
        return nil, err
    }
    res, ok := innerT.(Result)
    if !ok {
        return nil, NewMismatch(n.NodePos, Primitive("Result"), innerT)
    }
    return res.Ok, nil
}
```

- [ ] **Step 7: Add type-check test** to `v2/internal/types/check_test.go`:

```go
func TestCheck_TryOperator(t *testing.T) {
    src := `let x: int = ok(42)?.val
`
    p := parser.New(src, "")
    prog, err := p.Parse()
    require.NoError(t, err)
    env := NewEnv(nil)
    err = Check(prog, env)
    assert.NoError(t, err)
}

func TestCheck_TryOperator_BadType(t *testing.T) {
    src := `let x: int = 42?
`
    p := parser.New(src, "")
    prog, err := p.Parse()
    require.NoError(t, err)
    env := NewEnv(nil)
    err = Check(prog, env)
    assert.Error(t, err) // 42 is int, not Result
}
```

- [ ] **Step 8: Run all tests**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-c/v2
go test ./...
```

Expected: all packages pass (≥241)

- [ ] **Step 9: End-to-end verification**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-c/v2
go build -o funny ./cmd/funny
./funny run <(echo 'fn divide(a: int, b: int) -> Result:
    if b == 0:
        return err("divide by zero")?
    return ok(a / b)?
let r = divide(10, 2)?
println(r.val)
')  # should print 5
```

- [ ] **Step 10: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-c
git add v2/internal/bytecode/ v2/internal/vm/ v2/internal/compiler/ v2/internal/types/
git commit -m "v2: ? operator runtime (TRY_OR_RETURN, type checks unwrap to Ok type)"
```

---

## Task 3: stdlib json Module

**Files:**
- Modify: `v2/internal/vm/builtins.go` (add `to_json` and `parse_json`)

- [ ] **Step 1: Append failing tests**:

```go
func TestVM_BuiltinToJSON(t *testing.T) {
    main := &bytecode.Function{Name: "main", Arity: 0}
    main.Emit(bytecode.PUSH_STR, 0)  // {"k": 1}
    main.Emit(bytecode.CALL_BUILTIN, 1) // "to_json"
    main.Emit(bytecode.HALT, 0)
    v := runModule(t, main, nil, `{"k": 1}`, "to_json")
    // to_json parses the input and returns it as a map (i.e., it's an identity for already-valid JSON)
    m, ok := v.(map[string]bytecode.Value)
    require.True(t, ok)
    assert.Equal(t, float64(1), m["k"])
}

func TestVM_BuiltinParseJSON(t *testing.T) {
    main := &bytecode.Function{Name: "main", Arity: 0}
    main.Emit(bytecode.PUSH_STR, 0) // {"k": 1}
    main.Emit(bytecode.CALL_BUILTIN, 1) // "parse_json"
    main.Emit(bytecode.HALT, 0)
    v := runModule(t, main, nil, `{"k": 1}`, "parse_json")
    m, ok := v.(map[string]bytecode.Value)
    require.True(t, ok)
    assert.Equal(t, float64(1), m["k"])
}
```

- [ ] **Step 2: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-c/v2
go test ./internal/vm/ -v -run "TestVM_BuiltinToJSON|TestVM_BuiltinParseJSON"
```

Expected: FAIL

- [ ] **Step 3: Add to `builtins.go`** (inside the `switch name` block):

```go
    case "to_json":
        if len(v.stack) < 1 {
            return fmt.Errorf("vm: to_json() requires 1 argument")
        }
        s, ok := v.stack[len(v.stack)-1].(string)
        if !ok {
            return fmt.Errorf("vm: to_json() requires a string argument")
        }
        v.stack = v.stack[:len(v.stack)-1]
        // Round-trip: parse the string, then re-serialize to canonical JSON.
        // For M2-C, just verify it's valid JSON.
        var x any
        if err := json.Unmarshal([]byte(s), &x); err != nil {
            return fmt.Errorf("vm: to_json: invalid JSON: %v", err)
        }
        canonical, err := json.Marshal(x)
        if err != nil {
            return fmt.Errorf("vm: to_json: marshal error: %v", err)
        }
        v.stack = append(v.stack, string(canonical))
    case "parse_json":
        if len(v.stack) < 1 {
            return fmt.Errorf("vm: parse_json() requires 1 argument")
        }
        s, ok := v.stack[len(v.stack)-1].(string)
        if !ok {
            return fmt.Errorf("vm: parse_json() requires a string argument")
        }
        v.stack = v.stack[:len(v.stack)-1]
        // Wrap parse result in a Result
        var x any
        if err := json.Unmarshal([]byte(s), &x); err != nil {
            v.stack = append(v.stack, makeResult("err", fmt.Sprintf("parse_json: %v", err)))
            return nil
        }
        v.stack = append(v.stack, makeResult("ok", convertJSON(x)))
```

- [ ] **Step 4: Add the `convertJSON` helper** to `builtins.go`:

```go
// convertJSON converts a generic Go value (from json.Unmarshal) into a funny Value
// (using []any and map[string]any instead of []interface{} and map[string]interface{}).
func convertJSON(x any) bytecode.Value {
    switch v := x.(type) {
    case nil:
        return nil
    case bool:
        return v
    case float64:
        if v == float64(int(v)) {
            return int(v)
        }
        return v
    case string:
        return v
    case []any:
        out := make([]bytecode.Value, len(v))
        for i, e := range v {
            out[i] = convertJSON(e)
        }
        return out
    case map[string]any:
        out := make(map[string]bytecode.Value, len(v))
        for k, e := range v {
            out[k] = convertJSON(e)
        }
        return out
    default:
        return fmt.Sprintf("%v", v)
    }
}
```

- [ ] **Step 5: Add `"encoding/json"` to the imports** of `builtins.go`:

```go
import (
    "encoding/json"
    "fmt"
    "reflect"

    "github.com/jiejie-dev/funny/v2/internal/bytecode"
)
```

- [ ] **Step 6: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-c/v2
go test ./internal/vm/ -v -count=1
```

Expected: all tests PASS

- [ ] **Step 7: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-c
git add v2/internal/vm/
git commit -m "v2: stdlib json (to_json, parse_json)"
```

---

## Task 4: stdlib time Module

**Files:**
- Modify: `v2/internal/vm/builtins.go` (add `now` and `time_format`)

- [ ] **Step 1: Append failing tests**:

```go
func TestVM_BuiltinNow(t *testing.T) {
    main := &bytecode.Function{Name: "main", Arity: 0}
    main.Emit(bytecode.CALL_BUILTIN, 0) // "now"
    main.Emit(bytecode.HALT, 0)
    v := runModule(t, main, nil, "now")
    // Returns int (Unix timestamp in seconds); just check it's > some recent value.
    n, ok := v.(int)
    require.True(t, ok)
    assert.Greater(t, n, 1700000000) // after 2023
}

func TestVM_BuiltinTimeFormat(t *testing.T) {
    main := &bytecode.Function{Name: "main", Arity: 0}
    main.Emit(bytecode.PUSH_INT, 0) // timestamp
    main.Emit(bytecode.PUSH_STR, 1) // layout
    main.Emit(bytecode.CALL_BUILTIN, 2) // "time_format"
    main.Emit(bytecode.HALT, 0)
    v := runModule(t, main, nil, 1700000000, "2006-01-02", "time_format")
    s, ok := v.(string)
    require.True(t, ok)
    assert.Contains(t, s, "2023")
}
```

- [ ] **Step 2: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-c/v2
go test ./internal/vm/ -v -run "TestVM_BuiltinNow|TestVM_BuiltinTimeFormat"
```

Expected: FAIL

- [ ] **Step 3: Add to `builtins.go`**:

```go
    case "now":
        v.stack = append(v.stack, int(time.Now().Unix()))
    case "time_format":
        if len(v.stack) < 2 {
            return fmt.Errorf("vm: time_format() requires 2 arguments")
        }
        layout := v.stack[len(v.stack)-1].(string)
        ts := v.stack[len(v.stack)-2].(int)
        v.stack = v.stack[:len(v.stack)-2]
        t := time.Unix(int64(ts), 0)
        v.stack = append(v.stack, t.Format(layout))
```

- [ ] **Step 4: Add `"time"` to the imports** of `builtins.go`:

```go
import (
    "encoding/json"
    "fmt"
    "reflect"
    "time"

    "github.com/jiejie-dev/funny/v2/internal/bytecode"
)
```

- [ ] **Step 5: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-c/v2
go test ./internal/vm/ -v -count=1
```

Expected: all tests PASS

- [ ] **Step 6: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-c
git add v2/internal/vm/
git commit -m "v2: stdlib time (now, time_format)"
```

---

## Task 5: stdlib math Module

**Files:**
- Modify: `v2/internal/vm/builtins.go` (add `sqrt`, `pow`, `abs`)

- [ ] **Step 1: Append failing tests**:

```go
func TestVM_BuiltinSqrt(t *testing.T) {
    main := &bytecode.Function{Name: "main", Arity: 0}
    main.Emit(bytecode.PUSH_INT, 0) // 16
    main.Emit(bytecode.CALL_BUILTIN, 1) // "sqrt"
    main.Emit(bytecode.HALT, 0)
    v := runModule(t, main, nil, 16, "sqrt")
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
    v := runModule(t, main, nil, 2, 10, "pow")
    f, ok := v.(float64)
    require.True(t, ok)
    assert.Equal(t, 1024.0, f)
}

func TestVM_BuiltinAbs(t *testing.T) {
    main := &bytecode.Function{Name: "main", Arity: 0}
    main.Emit(bytecode.PUSH_INT, 0) // -5
    main.Emit(bytecode.CALL_BUILTIN, 1) // "abs"
    main.Emit(bytecode.HALT, 0)
    v := runModule(t, main, nil, -5, "abs")
    assert.Equal(t, 5, v)
}
```

- [ ] **Step 2: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-c/v2
go test ./internal/vm/ -v -run "TestVM_BuiltinSqrt|TestVM_BuiltinPow|TestVM_BuiltinAbs"
```

Expected: FAIL

- [ ] **Step 3: Add to `builtins.go`**:

```go
    case "sqrt":
        if len(v.stack) < 1 {
            return fmt.Errorf("vm: sqrt() requires 1 argument")
        }
        x := toFloat(v.stack[len(v.stack)-1])
        v.stack = v.stack[:len(v.stack)-1]
        v.stack = append(v.stack, math.Sqrt(x))
    case "pow":
        if len(v.stack) < 2 {
            return fmt.Errorf("vm: pow() requires 2 arguments")
        }
        exp := toFloat(v.stack[len(v.stack)-1])
        base := toFloat(v.stack[len(v.stack)-2])
        v.stack = v.stack[:len(v.stack)-2]
        v.stack = append(v.stack, math.Pow(base, exp))
    case "abs":
        if len(v.stack) < 1 {
            return fmt.Errorf("vm: abs() requires 1 argument")
        }
        x := v.stack[len(v.stack)-1]
        v.stack = v.stack[:len(v.stack)-1]
        switch v := x.(type) {
        case int:
            if v < 0 {
                v.stack = append(v.stack, -v)
            } else {
                v.stack = append(v.stack, v)
            }
        case float64:
            v.stack = append(v.stack, math.Abs(v))
        default:
            return fmt.Errorf("vm: abs() requires a number")
        }
```

- [ ] **Step 4: Add the `toFloat` helper** to `builtins.go`:

```go
// toFloat converts an int or float to float64. Other types panic.
func toFloat(v bytecode.Value) float64 {
    switch x := v.(type) {
    case int:
        return float64(x)
    case float64:
        return x
    }
    panic(fmt.Sprintf("vm: expected number, got %T", v))
}
```

- [ ] **Step 5: Add `"math"` to the imports** of `builtins.go`:

```go
import (
    "encoding/json"
    "fmt"
    "math"
    "reflect"
    "time"

    "github.com/jiejie-dev/funny/v2/internal/bytecode"
)
```

- [ ] **Step 6: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-c/v2
go test ./internal/vm/ -v -count=1
```

Expected: all tests PASS

- [ ] **Step 7: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-c
git add v2/internal/vm/
git commit -m "v2: stdlib math (sqrt, pow, abs)"
```

---

## Task 6: stdlib str Module

**Files:**
- Modify: `v2/internal/vm/builtins.go` (add `str_upper`, `str_lower`, `str_contains`, `str_split`)

- [ ] **Step 1: Append failing tests**:

```go
func TestVM_BuiltinStrUpper(t *testing.T) {
    main := &bytecode.Function{Name: "main", Arity: 0}
    main.Emit(bytecode.PUSH_STR, 0) // "hello"
    main.Emit(bytecode.CALL_BUILTIN, 1) // "str_upper"
    main.Emit(bytecode.HALT, 0)
    v := runModule(t, main, nil, "hello", "str_upper")
    assert.Equal(t, "HELLO", v)
}

func TestVM_BuiltinStrLower(t *testing.T) {
    main := &bytecode.Function{Name: "main", Arity: 0}
    main.Emit(bytecode.PUSH_STR, 0) // "WORLD"
    main.Emit(bytecode.CALL_BUILTIN, 1) // "str_lower"
    main.Emit(bytecode.HALT, 0)
    v := runModule(t, main, nil, "WORLD", "str_lower")
    assert.Equal(t, "world", v)
}

func TestVM_BuiltinStrContains(t *testing.T) {
    main := &bytecode.Function{Name: "main", Arity: 0}
    main.Emit(bytecode.PUSH_STR, 0) // "hello world"
    main.Emit(bytecode.PUSH_STR, 1) // "world"
    main.Emit(bytecode.CALL_BUILTIN, 2) // "str_contains"
    main.Emit(bytecode.HALT, 0)
    v := runModule(t, main, nil, "hello world", "world", "str_contains")
    assert.Equal(t, true, v)
}

func TestVM_BuiltinStrSplit(t *testing.T) {
    main := &bytecode.Function{Name: "main", Arity: 0}
    main.Emit(bytecode.PUSH_STR, 0)    // "a,b,c"
    main.Emit(bytecode.PUSH_STR, 1)    // ","
    main.Emit(bytecode.CALL_BUILTIN, 2) // "str_split"
    main.Emit(bytecode.HALT, 0)
    v := runModule(t, main, nil, "a,b,c", ",", "str_split")
    list, ok := v.([]bytecode.Value)
    require.True(t, ok)
    assert.Equal(t, 3, len(list))
    assert.Equal(t, "a", list[0])
    assert.Equal(t, "b", list[1])
    assert.Equal(t, "c", list[2])
}
```

- [ ] **Step 2: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-c/v2
go test ./internal/vm/ -v -run "TestVM_BuiltinStr"
```

Expected: FAIL

- [ ] **Step 3: Add to `builtins.go`**:

```go
    case "str_upper":
        if len(v.stack) < 1 {
            return fmt.Errorf("vm: str_upper() requires 1 argument")
        }
        s, _ := v.stack[len(v.stack)-1].(string)
        v.stack = v.stack[:len(v.stack)-1]
        v.stack = append(v.stack, strings.ToUpper(s))
    case "str_lower":
        if len(v.stack) < 1 {
            return fmt.Errorf("vm: str_lower() requires 1 argument")
        }
        s, _ := v.stack[len(v.stack)-1].(string)
        v.stack = v.stack[:len(v.stack)-1]
        v.stack = append(v.stack, strings.ToLower(s))
    case "str_contains":
        if len(v.stack) < 2 {
            return fmt.Errorf("vm: str_contains() requires 2 arguments")
        }
        substr := v.stack[len(v.stack)-1].(string)
        s := v.stack[len(v.stack)-2].(string)
        v.stack = v.stack[:len(v.stack)-2]
        v.stack = append(v.stack, strings.Contains(s, substr))
    case "str_split":
        if len(v.stack) < 2 {
            return fmt.Errorf("vm: str_split() requires 2 arguments")
        }
        sep := v.stack[len(v.stack)-1].(string)
        s := v.stack[len(v.stack)-2].(string)
        v.stack = v.stack[:len(v.stack)-2]
        parts := strings.Split(s, sep)
        out := make([]bytecode.Value, len(parts))
        for i, p := range parts {
            out[i] = p
        }
        v.stack = append(v.stack, out)
```

- [ ] **Step 4: Add `"strings"` to the imports** of `builtins.go`:

```go
import (
    "encoding/json"
    "fmt"
    "math"
    "reflect"
    "strings"
    "time"

    "github.com/jiejie-dev/funny/v2/internal/bytecode"
)
```

- [ ] **Step 5: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-c/v2
go test ./internal/vm/ -v -count=1
```

Expected: all tests PASS

- [ ] **Step 6: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-c
git add v2/internal/vm/
git commit -m "v2: stdlib str (upper, lower, contains, split)"
```

---

## Task 7: Integration Test Data and End-to-End Demo

**Files:**
- Create: `v2/testdata/types/result.fn` (Result + `?` demo)
- Create: `v2/testdata/types/json.fn` (json demo)
- Create: `v2/testdata/types/stdlib.fn` (mixed stdlib demo)

- [ ] **Step 1: Create `v2/testdata/types/result.fn`**:

```
fn divide(a: int, b: int) -> Result:
    if b == 0:
        return err("divide by zero")?
    return ok(a / b)?

let r1 = divide(10, 2)?
println("10 / 2 =", r1.val)

let r2 = divide(10, 0)?
if r2.tag == "err":
    println("expected error:", r2.val)
```

- [ ] **Step 2: Create `v2/testdata/types/json.fn`**:

```
let s = parse_json(`{"name": "alice", "age": 30}`)?
println(s.name)
println(s.age)

let canonical = to_json(s)
println(canonical)
```

- [ ] **Step 3: Create `v2/testdata/types/stdlib.fn`**:

```
let t = now()
println("unix:", t)

let s = str_upper("hello funny")
println(s)

let parts = str_split("a,b,c", ",")
println("count:", len(parts))

println("sqrt(16):", sqrt(16))
println("pow(2, 10):", pow(2, 10))
println("abs(-7):", abs(-7))

let ts = 1700000000
let formatted = time_format(ts, "2006-01-02")
println("date:", formatted)
```

- [ ] **Step 4: End-to-end verification**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-c/v2
go build -o funny ./cmd/funny
./funny run ./testdata/types/result.fn    # should print 5 then divide by zero error
./funny run ./testdata/types/json.fn      # should print "alice", 30, canonical JSON
./funny run ./testdata/types/stdlib.fn    # should print unix time, "HELLO FUNNY", "count: 3", "sqrt(16): 4", etc.
```

- [ ] **Step 5: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-c
git add v2/testdata/
git commit -m "v2: stdlib integration test data (result/json/stdlib)"
```

---

## Task 8: Update README

**Files:**
- Modify: `v2/README.md`

- [ ] **Step 1: Update status block**:

Find `**Status: M2-B.5 (VM Functions + Data Ops) — RELEASED**` and replace with:

```markdown
**Status: M2-C (Result + ? + Stdlib) — RELEASED**

- ✅ Lexer, Parser, Type checker, Bytecode VM, VM Functions + Data Ops (M1–M2-B.5)
- ✅ **Result type runtime**: `ok()` / `err()` constructors
- ✅ **`?` operator**: postfix try-propagation (`expr?` unwraps Ok, returns Err)
- ✅ **stdlib**: json, time, math, str modules
- ⏳ meta/plan engine + LSP → M3
- ⏳ MCP server + full stdlib → M4
```

- [ ] **Step 2: Update roadmap**:

Replace the M2-B.5 row's note and add M2-C:

```markdown
| v2.0.0-beta (M2-B.5) | ✅ Done | VM function calls + data structures (~3.5× interpreter) |
| v2.0.0-beta (M2-C) | ✅ Done | Result + `?` + stdlib (json/time/math/str) |
| v2.0.0-rc (M3) | Planned | meta/plan engine + LSP |
| v2.0.0 (M4) | Planned | MCP server + full stdlib |
```

- [ ] **Step 3: Replace M2-B.5 Performance section** with M2-C Usage:

```markdown
## M2-C Usage

The full M2 stack is now usable. Demo:

```bash
$ ./funny run ./testdata/types/result.fn
10 / 2 = 5
expected error: divide by zero

$ ./funny run ./testdata/types/json.fn
alice
30
{"age":30,"name":"alice"}

$ ./funny run ./testdata/types/stdlib.fn
unix: 1700000000
HELLO FUNNY
count: 3
sqrt(16): 4
pow(2, 10): 1024
abs(-7): 7
date: 2023-11-14
```

The `?` operator propagates errors: `expr?` unwraps a `Result` if Ok, or returns early if Err.
```

- [ ] **Step 4: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-c
git add v2/README.md
git commit -m "v2: README updated for M2-C (Result + ? + stdlib)"
```

---

## Self-Review

1. **Spec coverage**:
   - §2.2 Result type runtime → Task 0 ✓
   - `?` operator parser/compiler/types/VM → Tasks 1, 2 ✓
   - stdlib (json/time/math/str) → Tasks 3-6 ✓
   - §6.3 M2-C deliverables → All 9 tasks ✓
   - **Deferred**: Error codes E4xxx (M2-C uses plain `fmt.Errorf` for now; structured codes deferred if needed), 5× perf target (M2-B.5 follow-up), M3/M4.

2. **Placeholder scan**: no TBD/TODO in plan body.

3. **Type consistency**: `TryExpr`, `isResult`, `resultTag`, `resultVal`, `makeResult`, `toFloat`, `convertJSON` all defined once and used consistently.

---

## Exit Criteria for M2-C

- [ ] All 9 tasks checked off
- [ ] `go test ./...` passes
- [ ] `./funny run ./testdata/types/result.fn` prints `5` then `expected error: divide by zero`
- [ ] `./funny run ./testdata/types/json.fn` prints JSON-parsed values
- [ ] `./funny run ./testdata/types/stdlib.fn` exercises all stdlib modules
- [ ] M2-C released as `v2.0.0-beta-result-stdlib`

---

## Total Tasks: 9

**Estimated time**: 3-4 days for one developer.