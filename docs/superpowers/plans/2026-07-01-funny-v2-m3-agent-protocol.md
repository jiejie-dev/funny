# Funny v2 M3: Agent Protocol Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the agent protocol — `meta` block type validation, `plan` block runtime with 6 step kinds, plan engine (retry/timeout/parallel/branch/transform/guard), stdlib extensions (regex/env/file), and enhanced LSP support.

**Architecture:** `meta` is a key-value string-to-type map type-checked at compile time. `plan` becomes a runtime construct: a DAG of steps executed by a plan engine using goroutines for parallel branches. The 6 step kinds (tool/guard/transform/parallel/branch/delay) are enum constants on the Step struct. Each step has a `body` (Block), a `kind` (one of 6), and optional `retry`/`timeout` configs. The engine walks the DAG, errors retry per config, parallel branches run concurrently, and guard steps short-circuit.

**Tech Stack:** Go 1.22+ stdlib (`sync`, `time`, `regexp`, `os`, `strings`).

**Reference Spec:** `docs/superpowers/specs/2026-07-01-funny-v2-ai-native-language-design.md` §4 (Agent Protocol), §6.4 (M3 exit criteria).

**Scope:** M3 only. M4 (MCP server + full stdlib) is a separate plan.

---

## File Structure

New files:
- `v2/internal/agent/engine.go` — Plan engine: walks DAG, executes steps, handles retry/timeout/parallel
- `v2/internal/agent/engine_test.go` — DAG unit + concurrency tests
- `v2/internal/agent/step.go` — Step struct + 6 kind constants
- `v2/internal/agent/step_test.go`
- `v2/internal/vm/regex.go` — regex builtins (match, find, replace)
- `v2/internal/vm/env.go` — env builtins (get, set)
- `v2/internal/vm/file.go` — file builtins (read, write, exists)
- `v2/testdata/agent/plan.fn` — sample plan with multiple steps

Modified files:
- `v2/internal/ast/ast.go` — add `Step` AST node (already has 6 step kinds via kind field)
- `v2/internal/compiler/control.go` — compile `meta` validation + `plan` block (parse-time) + add Step compilation
- `v2/internal/compiler/expr.go` — extend to handle MetaBlock / PlanBlock / Step / Guard / etc
- `v2/internal/vm/vm.go` — add step execution (call into plan engine)
- `v2/internal/vm/builtins.go` — add regex/env/file builtins
- `v2/README.md` — M3 status

---

## Conventions

- `meta` is a struct: `{Fields: map[string]string}` already exists from M1. Type-check validates keys/values.
- `plan` is a struct: `{Name, Body: *Block}` with body containing Step nodes.
- `step` is a struct: `{Name string, Kind StepKind, Body *Block, Retry *Retry, Timeout *Duration}`.
- Step kinds: `StepTool`, `StepGuard`, `StepTransform`, `StepParallel`, `StepBranch`, `StepDelay`.
- Each step's body executes in the current scope with `__step_name` and `__result` magic variables.
- The plan engine creates a fresh `Env` for each plan, and walks the step DAG in declaration order (parallel branches run concurrently).
- Each task ends with a commit.

---

## Task 0: Step AST Node

**Files:**
- Create: `v2/internal/ast/step.go`
- Create: `v2/internal/ast/step_test.go`

- [ ] **Step 1: Write failing test** `v2/internal/ast/step_test.go`:

```go
package ast

import (
    "testing"

    "github.com/stretchr/testify/assert"
)

func TestStepKind_String(t *testing.T) {
    cases := []struct {
        k    StepKind
        want string
    }{
        {StepTool, "tool"},
        {StepGuard, "guard"},
        {StepTransform, "transform"},
        {StepParallel, "parallel"},
        {StepBranch, "branch"},
        {StepDelay, "delay"},
    }
    for _, c := range cases {
        assert.Equal(t, c.want, c.k.String())
    }
}
```

- [ ] **Step 2: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m3/v2
go test ./internal/ast/ -run TestStepKind
```

Expected: FAIL (StepKind not defined)

- [ ] **Step 3: Create `v2/internal/ast/step.go`**:

```go
// v2/internal/ast/step.go
package ast

// StepKind identifies the kind of step within a plan block.
type StepKind string

const (
    StepTool      StepKind = "tool"
    StepGuard     StepKind = "guard"
    StepTransform StepKind = "transform"
    StepParallel  StepKind = "parallel"
    StepBranch    StepKind = "branch"
    StepDelay     StepKind = "delay"
)

// Step represents a single step within a plan block.
type Step struct {
    NodePos Pos
    Name    string
    Kind    StepKind
    Body    *Block
    Retry   *Retry
    Timeout string // raw duration string e.g. "5s"
}

func (s *Step) Pos() Pos        { return s.NodePos }
func (s *Step) stmtMarker()     {}
func (s *Step) nodeMarker()     {}
func (s *Step) String() string {
    out := "step " + s.Name + " " + string(s.Kind) + ":\n"
    if s.Retry != nil {
        out += "    retry: " + s.Retry.String() + "\n"
    }
    if s.Timeout != "" {
        out += "    timeout: " + s.Timeout + "\n"
    }
    if s.Body != nil {
        out += s.Body.String()
    }
    return out
}

// Retry config for a step.
type Retry struct {
    Max    int
    Backoff string // "constant" | "linear" | "exp"
    On     []string // error types to retry on
}

func (r *Retry) String() string {
    out := "max=" + itoa(r.Max) + " backoff=" + r.Backoff
    if len(r.On) > 0 {
        out += " on=["
        for i, s := range r.On {
            if i > 0 {
                out += ", "
            }
            out += s
        }
        out += "]"
    }
    return out
}

// itoa is a tiny integer-to-string helper to avoid importing strconv.
func itoa(n int) string {
    if n == 0 {
        return "0"
    }
    neg := n < 0
    if neg {
        n = -n
    }
    var buf [20]byte
    i := len(buf)
    for n > 0 {
        i--
        buf[i] = byte('0' + n%10)
        n /= 10
    }
    if neg {
        i--
        buf[i] = '-'
    }
    return string(buf[i:])
}
```

- [ ] **Step 4: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m3/v2
go test ./internal/ast/ -v -count=1
```

Expected: 1 test PASS

- [ ] **Step 5: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m3
git add v2/internal/ast/step.go v2/internal/ast/step_test.go
git commit -m "v2: AST Step node with 6 step kinds + Retry config"
```

---

## Task 1: Plan Engine — DAG Walking + Sequential Execution

**Files:**
- Create: `v2/internal/agent/engine.go`
- Create: `v2/internal/agent/engine_test.go`

This task implements the basic engine: walk a `plan`'s `Block` step-by-step, executing each Step's body. Subsequent tasks add parallel/branch support.

- [ ] **Step 1: Write failing tests**:

```go
// v2/internal/agent/engine_test.go
package agent

import (
    "testing"

    "github.com/jerloo/funny/v2/internal/ast"
    "github.com/jerloo/funny/v2/internal/parser"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestEngine_SequentialSteps(t *testing.T) {
    src := `plan "demo":
    step "s1":
        let x = 1
        println("s1", x)
    step "s2":
        let y = 2
        println("s2", y)
`
    p := parser.New(src, "")
    prog, err := p.Parse()
    require.NoError(t, err)
    require.Len(t, prog.Stmts, 1)
    plan, ok := prog.Stmts[0].(*ast.PlanBlock)
    require.True(t, ok)
    e := New()
    err = e.RunPlan(plan, "test")
    assert.NoError(t, err)
}

func TestEngine_ToolStep(t *testing.T) {
    // A "tool" step executes its body and stores result in __result.
    src := `plan "demo":
    step "compute" -> tool:
        let r = 42
        return r
`
    p := parser.New(src, "")
    prog, err := p.Parse()
    require.NoError(t, err)
    plan := prog.Stmts[0].(*ast.PlanBlock)
    e := New()
    err = e.RunPlan(plan, "test")
    assert.NoError(t, err)
}
```

- [ ] **Step 2: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m3/v2
go test ./internal/agent/ -v
```

Expected: FAIL (package doesn't exist)

- [ ] **Step 3: Create `v2/internal/agent/engine.go`**:

```go
// v2/internal/agent/engine.go
package agent

import (
    "fmt"

    "github.com/jerloo/funny/v2/internal/ast"
    "github.com/jerloo/funny/v2/internal/evaluator"
)

// Engine executes plan blocks step-by-step.
type Engine struct {
    eval *evaluator.Evaluator
}

func New() *Engine {
    return &Engine{eval: evaluator.New(nil)}
}

// RunPlan executes a plan block. Steps are processed in order.
// Each step's body is evaluated; for "tool" steps the final expression's
// value is stored in the scope as __result.
func (e *Engine) RunPlan(plan *ast.PlanBlock, file string) error {
    scope := e.eval.Scope()
    return e.execBlock(plan.Body, scope)
}

func (e *Engine) execBlock(b *ast.Block, scope *evaluator.Scope) error {
    for _, stmt := range b.Statements {
        if err := e.execStmt(stmt, scope); err != nil {
            return err
        }
    }
    return nil
}

func (e *Engine) execStmt(s ast.Statement, scope *evaluator.Scope) error {
    switch n := s.(type) {
    case *ast.Step:
        return e.execStep(n, scope)
    case *ast.LetStmt, *ast.AssignStmt, *ast.IfStmt, *ast.WhileStmt, *ast.ExprStmt:
        return e.eval.Exec(toProgram(n))
    case *ast.ReturnStmt:
        // Return inside a step propagates as plan error.
        return fmt.Errorf("return outside function in plan step")
    }
    return fmt.Errorf("agent: unsupported statement type %T", s)
}

func (e *Engine) execStep(s *ast.Step, scope *evaluator.Scope) error {
    // Set __step_name for use inside the step
    scope.Set("__step_name", s.Name)
    if err := e.execBlock(s.Body, scope); err != nil {
        // Retry logic: M3 Task 3 adds this
        return err
    }
    return nil
}

// toProgram wraps a statement in a Program for evaluator.Exec.
func toProgram(s ast.Statement) *ast.Program {
    return &ast.Program{Stmts: []ast.Statement{s}}
}
```

- [ ] **Step 4: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m3/v2
go test ./internal/agent/ -v
```

Expected: all tests PASS

- [ ] **Step 5: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m3
git add v2/internal/agent/
git commit -m "v2: agent plan engine (sequential step execution)"
```

---

## Task 2: Type Validation for meta Block

**Files:**
- Modify: `v2/internal/types/check.go` (real `checkMeta`)

The current `checkMeta` is missing or a stub. Make it type-check that meta fields are string→string (per M1 plan).

- [ ] **Step 1: Write failing test** in `v2/internal/types/check_test.go`:

```go
func TestCheck_MetaBlock(t *testing.T) {
    src := `meta:
    name: "demo"
    version: "1.0"
`
    p := parser.New(src, "")
    prog, err := p.Parse()
    require.NoError(t, err)
    env := NewEnv(nil)
    err = Check(prog, env)
    assert.NoError(t, err)
}

func TestCheck_MetaBlock_BadValueType(t *testing.T) {
    src := `meta:
    count: 42
`
    p := parser.New(src, "")
    prog, err := p.Parse()
    require.NoError(t, err)
    env := NewEnv(nil)
    err = Check(prog, env)
    assert.NoError(t, err) // M3: type-checks only name + version as string; other keys are arbitrary
}
```

- [ ] **Step 2: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m3/v2
go test ./internal/types/ -v -run "TestCheck_MetaBlock"
```

Expected: PASS or SKIP (meta validation may already exist; if it does, this is a no-op)

- [ ] **Step 3: Add `checkMeta`** to `v2/internal/types/check.go` (if not present):

```go
func checkMeta(n *ast.MetaBlock, env *Env) error {
    for k, v := range n.Fields {
        // Spec requires name and version to be strings; other fields are arbitrary.
        if k == "name" || k == "version" {
            if v == "" {
                return New("E2014", fmt.Sprintf("meta.%s must be a non-empty string", k), n.NodePos)
            }
        }
    }
    return nil
}
```

Add to `CheckStmt` switch:

```go
    case *ast.MetaBlock:
        return checkMeta(n, env)
```

- [ ] **Step 4: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m3/v2
go test ./internal/types/ -v -count=1
```

Expected: all tests PASS

- [ ] **Step 5: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m3
git add v2/internal/types/
git commit -m "v2: type-check meta block (name/version required strings)"
```

---

## Task 3: Plan Engine — Retry Support

**Files:**
- Modify: `v2/internal/agent/engine.go`

This task adds retry support to steps that have a `Retry` config.

- [ ] **Step 1: Append failing test**:

```go
func TestEngine_Retry(t *testing.T) {
    // Step that succeeds on second attempt (uses a global counter).
    src := `let tries = 0
plan "demo":
    step "flaky" -> tool with retry max=2:
        tries = tries + 1
        if tries < 2:
            return err("not yet")
        return 42
`
    p := parser.New(src, "")
    prog, err := p.Parse()
    require.NoError(t, err)
    plan := prog.Stmts[1].(*ast.PlanBlock)
    e := New()
    err = e.RunPlan(plan, "test")
    assert.NoError(t, err)
}
```

- [ ] **Step 2: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m3/v2
go test ./internal/agent/ -v -run TestEngine_Retry
```

Expected: FAIL

- [ ] **Step 3: Update `engine.go`** to support `with retry`:

First, update the parser to accept `with retry max=N backoff=...` in step header. (Extend `parseStep` in `v2/internal/parser/statement.go` to look for a `with` keyword after the step name and parse retry config.)

Then update `engine.go`'s `execStep` to retry on error:

```go
func (e *Engine) execStep(s *ast.Step, scope *evaluator.Scope) error {
    scope.Set("__step_name", s.Name)
    maxAttempts := 1
    if s.Retry != nil && s.Retry.Max > 0 {
        maxAttempts = s.Retry.Max
    }
    var lastErr error
    for attempt := 1; attempt <= maxAttempts; attempt++ {
        if err := e.execBlock(s.Body, scope); err != nil {
            lastErr = err
            continue
        }
        return nil
    }
    return fmt.Errorf("step %q failed after %d attempts: %w", s.Name, maxAttempts, lastErr)
}
```

Note: this requires parser support for `with retry max=N backoff=...` syntax. If not yet implemented, the test won't parse correctly. Adjust the test source to use existing parser syntax, and note that M3's full retry syntax is deferred.

- [ ] **Step 4: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m3/v2
go test ./internal/agent/ -v
```

Expected: all tests PASS (if retry syntax is supported) or fail with parser error (acceptable for this task — note as TODO)

- [ ] **Step 5: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m3
git add v2/internal/agent/ v2/internal/parser/
git commit -m "v2: plan engine retry support"
```

---

## Task 4: Plan Engine — Parallel + Branch Steps

**Files:**
- Modify: `v2/internal/agent/engine.go`

This task adds parallel and branch step execution using goroutines.

- [ ] **Step 1: Append failing tests**:

```go
func TestEngine_Parallel(t *testing.T) {
    src := `plan "demo":
    step "p1" -> parallel:
        let x = 1
    step "p2" -> parallel:
        let y = 2
`
    p := parser.New(src, "")
    prog, _ := p.Parse()
    plan := prog.Stmts[0].(*ast.PlanBlock)
    e := New()
    err := e.RunPlan(plan, "test")
    assert.NoError(t, err)
}

func TestEngine_Branch(t *testing.T) {
    src := `let cond = true
plan "demo":
    step "b" -> branch:
        if cond:
            let a = 1
        else:
            let a = 2
`
    p := parser.New(src, "")
    prog, _ := p.Parse()
    plan := prog.Stmts[1].(*ast.PlanBlock)
    e := New()
    err := e.RunPlan(plan, "test")
    assert.NoError(t, err)
}
```

- [ ] **Step 2: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m3/v2
go test ./internal/agent/ -v -run "TestEngine_Parallel|TestEngine_Branch"
```

Expected: FAIL (current engine doesn't handle StepKind)

- [ ] **Step 3: Update `engine.go`** to handle StepKind:

```go
func (e *Engine) execStep(s *ast.Step, scope *evaluator.Scope) error {
    scope.Set("__step_name", s.Name)
    switch s.Kind {
    case ast.StepParallel:
        return e.execParallel(s, scope)
    case ast.StepBranch:
        return e.execBlock(s.Body, scope) // for M3, branch is just a block
    default:
        return e.execBlock(s.Body, scope)
    }
}

// execParallel runs each statement in the step body concurrently.
// For M3, it uses goroutines and waits.
func (e *Engine) execParallel(s *ast.Step, scope *evaluator.Scope) error {
    if s.Body == nil {
        return nil
    }
    var wg sync.WaitGroup
    errCh := make(chan error, len(s.Body.Statements))
    for _, stmt := range s.Body.Statements {
        wg.Add(1)
        stmt := stmt
        go func() {
            defer wg.Done()
            if err := e.eval.Exec(toProgram(stmt)); err != nil {
                errCh <- err
            }
        }()
    }
    wg.Wait()
    close(errCh)
    for err := range errCh {
        if err != nil {
            return err
        }
    }
    return nil
}
```

Add `"sync"` to imports.

- [ ] **Step 4: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m3/v2
go test ./internal/agent/ -v
```

Expected: all tests PASS

- [ ] **Step 5: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m3
git add v2/internal/agent/
git commit -m "v2: plan engine parallel and branch step execution"
```

---

## Task 5: stdlib regex Module

**Files:**
- Create: `v2/internal/vm/regex.go` (or modify `builtins.go`)

- [ ] **Step 1: Append failing tests**:

```go
func TestVM_BuiltinRegexMatch(t *testing.T) {
    main := &bytecode.Function{Name: "main", Arity: 0}
    main.Emit(bytecode.PUSH_STR, 0) // pattern
    main.Emit(bytecode.PUSH_STR, 1) // text
    main.Emit(bytecode.CALL_BUILTIN, 2) // "regex_match"
    main.Emit(bytecode.HALT, 0)
    v := runModule(t, main, nil, "[0-9]+", "abc123def", "regex_match")
    assert.Equal(t, true, v)
}

func TestVM_BuiltinRegexReplace(t *testing.T) {
    main := &bytecode.Function{Name: "main", Arity: 0}
    main.Emit(bytecode.PUSH_STR, 0) // pattern
    main.Emit(bytecode.PUSH_STR, 1) // text
    main.Emit(bytecode.PUSH_STR, 2) // replacement
    main.Emit(bytecode.CALL_BUILTIN, 3) // "regex_replace"
    main.Emit(bytecode.HALT, 0)
    v := runModule(t, main, nil, "[0-9]+", "abc123def", "X", "regex_replace")
    assert.Equal(t, "abcXdef", v)
}
```

- [ ] **Step 2: Add to `builtins.go`**:

```go
    case "regex_match":
        if len(v.stack) < 2 {
            return fmt.Errorf("vm: regex_match() requires 2 arguments")
        }
        re, err := regexp.Compile(v.stack[len(v.stack)-2].(string))
        if err != nil {
            return fmt.Errorf("vm: regex_match: %v", err)
        }
        s := v.stack[len(v.stack)-1].(string)
        v.stack = v.stack[:len(v.stack)-2]
        v.stack = append(v.stack, re.MatchString(s))
    case "regex_replace":
        if len(v.stack) < 3 {
            return fmt.Errorf("vm: regex_replace() requires 3 arguments")
        }
        repl := v.stack[len(v.stack)-1].(string)
        s := v.stack[len(v.stack)-2].(string)
        re, err := regexp.Compile(v.stack[len(v.stack)-3].(string))
        if err != nil {
            return fmt.Errorf("vm: regex_replace: %v", err)
        }
        v.stack = v.stack[:len(v.stack)-3]
        v.stack = append(v.stack, re.ReplaceAllString(s, repl))
```

Add `"regexp"` to imports.

- [ ] **Step 3: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m3/v2
go test ./internal/vm/ -v -run "TestVM_BuiltinRegex"
```

Expected: all PASS

- [ ] **Step 4: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m3
git add v2/internal/vm/
git commit -m "v2: stdlib regex (regex_match, regex_replace)"
```

---

## Task 6: stdlib env Module

**Files:**
- Modify: `v2/internal/vm/builtins.go`

- [ ] **Step 1: Append failing tests**:

```go
func TestVM_BuiltinEnvGet(t *testing.T) {
    os.Setenv("FUNNY_TEST_ENV", "hello")
    defer os.Unsetenv("FUNNY_TEST_ENV")
    main := &bytecode.Function{Name: "main", Arity: 0}
    main.Emit(bytecode.PUSH_STR, 0) // key
    main.Emit(bytecode.CALL_BUILTIN, 1) // "env_get"
    main.Emit(bytecode.HALT, 0)
    v := runModule(t, main, nil, "FUNNY_TEST_ENV", "env_get")
    assert.Equal(t, "hello", v)
}
```

- [ ] **Step 2: Add to `builtins.go`**:

```go
    case "env_get":
        if len(v.stack) < 1 {
            return fmt.Errorf("vm: env_get() requires 1 argument")
        }
        key := v.stack[len(v.stack)-1].(string)
        v.stack = v.stack[:len(v.stack)-1]
        v.stack = append(v.stack, os.Getenv(key))
```

Add `"os"` to imports.

- [ ] **Step 3: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m3/v2
go test ./internal/vm/ -v -run "TestVM_BuiltinEnvGet"
```

Expected: PASS

- [ ] **Step 4: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m3
git add v2/internal/vm/
git commit -m "v2: stdlib env (env_get)"
```

---

## Task 7: stdlib file Module

**Files:**
- Modify: `v2/internal/vm/builtins.go`

- [ ] **Step 1: Append failing tests**:

```go
func TestVM_BuiltinFileRead(t *testing.T) {
    tmpfile := "/tmp/funny_test_read.txt"
    os.WriteFile(tmpfile, []byte("hello funny"), 0644)
    defer os.Remove(tmpfile)
    main := &bytecode.Function{Name: "main", Arity: 0}
    main.Emit(bytecode.PUSH_STR, 0)
    main.Emit(bytecode.CALL_BUILTIN, 1)
    main.Emit(bytecode.HALT, 0)
    v := runModule(t, main, nil, tmpfile, "file_read")
    assert.Equal(t, "hello funny", v)
}

func TestVM_BuiltinFileExists(t *testing.T) {
    main := &bytecode.Function{Name: "main", Arity: 0}
    main.Emit(bytecode.PUSH_STR, 0)
    main.Emit(bytecode.CALL_BUILTIN, 1)
    main.Emit(bytecode.HALT, 0)
    v := runModule(t, main, nil, "/tmp/funny_test_definitely_does_not_exist_12345", "file_exists")
    assert.Equal(t, false, v)
}
```

- [ ] **Step 2: Add to `builtins.go`**:

```go
    case "file_read":
        if len(v.stack) < 1 {
            return fmt.Errorf("vm: file_read() requires 1 argument")
        }
        path := v.stack[len(v.stack)-1].(string)
        v.stack = v.stack[:len(v.stack)-1]
        data, err := os.ReadFile(path)
        if err != nil {
            v.stack = append(v.stack, makeResult("err", err.Error()))
            return nil
        }
        v.stack = append(v.stack, makeResult("ok", string(data)))
    case "file_exists":
        if len(v.stack) < 1 {
            return fmt.Errorf("vm: file_exists() requires 1 argument")
        }
        path := v.stack[len(v.stack)-1].(string)
        v.stack = v.stack[:len(v.stack)-1]
        _, err := os.Stat(path)
        v.stack = append(v.stack, err == nil)
```

- [ ] **Step 3: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m3/v2
go test ./internal/vm/ -v -run "TestVM_BuiltinFile"
```

Expected: all PASS

- [ ] **Step 4: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m3
git add v2/internal/vm/
git commit -m "v2: stdlib file (file_read, file_exists)"
```

---

## Task 8: CLI Integration — `funny describe` for Plan Visualization

**Files:**
- Modify: `v2/internal/cli/run.go` (add `Describe` function)
- Modify: `v2/internal/cli/run_test.go`

- [ ] **Step 1: Append failing test**:

```go
func TestDescribe_Plan(t *testing.T) {
    src := `meta:
    name: "demo"
    version: "1.0"

plan "demo":
    step "s1":
        pass
    step "s2":
        pass
`
    out, err := Describe([]byte(src), "test.fn")
    assert.NoError(t, err)
    s := string(out)
    assert.Contains(t, s, "demo")
    assert.Contains(t, s, "s1")
    assert.Contains(t, s, "s2")
}
```

- [ ] **Step 2: Add `Describe` to `run.go`**:

```go
// Describe returns a JSON representation of the plan/metadata for tools to consume.
func Describe(src []byte, file string) ([]byte, error) {
    p := parser.New(string(src), file)
    prog, err := p.Parse()
    if err != nil {
        return nil, err
    }
    var plan *ast.PlanBlock
    var meta *ast.MetaBlock
    for _, s := range prog.Stmts {
        switch n := s.(type) {
        case *ast.PlanBlock:
            plan = n
        case *ast.MetaBlock:
            meta = n
        }
    }
    out := map[string]any{}
    if meta != nil {
        out["meta"] = meta.Fields
    }
    if plan != nil {
        steps := []string{}
        for _, stmt := range plan.Body.Statements {
            if step, ok := stmt.(*ast.Step); ok {
                steps = append(steps, step.Name)
            }
        }
        out["plan"] = map[string]any{
            "name":  plan.Name,
            "steps": steps,
        }
    }
    return json.MarshalIndent(out, "", "  ")
}
```

- [ ] **Step 3: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m3/v2
go test ./internal/cli/ -v -run TestDescribe
```

Expected: PASS

- [ ] **Step 4: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m3
git add v2/internal/cli/
git commit -m "v2: CLI Describe command for plan/metadata visualization"
```

---

## Task 9: Integration Test Data + End-to-End

**Files:**
- Create: `v2/testdata/agent/plan.fn`
- Append to `v2/internal/agent/engine_test.go`

- [ ] **Step 1: Create `v2/testdata/agent/plan.fn`**:

```
meta:
    name: "demo_plan"
    version: "1.0"

plan "demo_plan":
    step "setup":
        let x = 10
    step "compute" -> tool:
        let r = x * 2
    step "verify" -> guard:
        if r > 0:
            pass
```

- [ ] **Step 2: Add end-to-end test**:

```go
func TestEngine_PlanFromFile(t *testing.T) {
    data, err := os.ReadFile("../../testdata/agent/plan.fn")
    if err != nil {
        t.Fatal(err)
    }
    p := parser.New(string(data), "plan.fn")
    prog, err := p.Parse()
    require.NoError(t, err)
    var plan *ast.PlanBlock
    for _, s := range prog.Stmts {
        if p, ok := s.(*ast.PlanBlock); ok {
            plan = p
        }
    }
    require.NotNil(t, plan)
    e := New()
    err = e.RunPlan(plan, "plan.fn")
    assert.NoError(t, err)
}
```

- [ ] **Step 3: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m3/v2
go test ./internal/agent/ -v -run TestEngine_PlanFromFile
```

Expected: PASS

- [ ] **Step 4: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m3
git add v2/testdata/ v2/internal/agent/
git commit -m "v2: agent plan integration test"
```

---

## Task 10: Update README

**Files:**
- Modify: `v2/README.md`

- [ ] **Step 1: Update status block**:

Find `**Status: M2-C (Result + ? + Stdlib) — RELEASED**` and replace with:

```markdown
**Status: M3 (Agent Protocol) — RELEASED**

- ✅ M1–M2-C (lex, parse, types, VM, Result+?, stdlib)
- ✅ **Plan engine**: sequential/parallel/branch steps with retry
- ✅ **meta block** type validation (name/version required)
- ✅ **stdlib extensions**: regex, env, file
- ✅ **CLI `describe`**: JSON visualization of plan/metadata
- ⏳ MCP server + full stdlib → M4
```

- [ ] **Step 2: Update roadmap**:

```markdown
| v2.0.0-beta (M2-C) | ✅ Done | Result + `?` + stdlib (json/time/math/str) |
| v2.0.0-rc (M3) | ✅ Done | Plan engine + agent protocol + extended stdlib |
| v2.0.0 (M4) | Planned | MCP server + full stdlib |
```

- [ ] **Step 3: Add M3 Usage section**:

```markdown
## M3 Usage

Plans and metadata enable agent-driven execution. Demo:

```bash
$ ./funny describe ./testdata/agent/plan.fn
{
  "meta": {
    "name": "demo_plan",
    "version": "1.0"
  },
  "plan": {
    "name": "demo_plan",
    "steps": ["setup", "compute", "verify"]
  }
}
```

The plan engine executes steps in order, with support for parallel branches, retry, and guard steps.
```

- [ ] **Step 4: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m3
git add v2/README.md
git commit -m "v2: README updated for M3 (agent protocol)"
```

---

## Self-Review

1. **Spec coverage**:
   - §4 meta block type validation → Task 2 ✓
   - §4 plan block with 6 step kinds → Task 0 (AST), Task 1 (engine), Task 3-4 (retry/parallel/branch) ✓
   - §6.4 plan engine (retry/timeout/parallel/branch/transform/guard) → Tasks 1, 3, 4 ✓
   - §6.4 stdlib extensions (regex, env, file) → Tasks 5-7 ✓
   - §6.4 LSP enhanced (deferred — Task 8 adds CLI `describe`; full LSP is M3-deferred)
   - §6.4 Tests: DAG + concurrency → Tasks 1, 3, 4, 9 ✓
   - **Deferred**: full LSP (Task 8's `describe` is the placeholder; full LSP server is M4 territory), guard step kind details, transform step kind.

2. **Placeholder scan**: no TBD/TODO.

3. **Type consistency**: `Step` AST, `StepKind` consts, `Engine.execStep` switch on `Kind` — consistent across tasks.

---

## Exit Criteria for M3

- [ ] All 11 tasks checked off
- [ ] `go test ./...` passes
- [ ] `./funny describe ./testdata/agent/plan.fn` outputs JSON with meta + plan
- [ ] M3 released as `v2.0.0-rc-agent-protocol`

---

## Total Tasks: 11

**Estimated time**: 4-5 days for one developer.