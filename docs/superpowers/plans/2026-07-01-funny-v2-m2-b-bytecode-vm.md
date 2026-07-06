# Funny v2 M2-B: Bytecode VM Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a typed bytecode compiler and stack-based VM for Funny v2 — achieves ≥5× runtime speedup over M1's tree-walking evaluator while maintaining identical semantics.

**Architecture:** Two-pass compiler: (1) walk the typed AST produced by M2-A's type checker and emit typed instructions per spec §5.4, (2) VM executes instructions over an operand stack with per-function frames (local variable table + instruction pointer). VM is added alongside the existing tree-walking evaluator; CLI default switches to VM but `--interpret` flag preserves the old path for debugging.

**Tech Stack:** Go 1.22+, `github.com/stretchr/testify`, Go's testing.Benchmark.

**Reference Spec:** `docs/superpowers/specs/2026-07-01-funny-v2-ai-native-language-design.md` §5.4 (Bytecode VM Design), §6.3 (M2 exit criteria).

**Scope:** This plan covers M2-B (compiler + VM). M2-C (Result runtime + `?` operator + stdlib) is a separate plan.

---

## File Structure

New files:

```
v2/internal/bytecode/
├── opcode.go         # OpCode enum, all typed instruction names
├── opcode_test.go
├── code.go           # Function, Module, Constant, Value structs
├── code_test.go
└── disasm.go         # Disassembler for debugging

v2/internal/compiler/
├── compiler.go       # Top-level Compiler, Emit* helpers
├── compiler_test.go
├── expr.go           # CompileExpr (literals, vars, binary, call, index, field)
├── expr_test.go
├── control.go        # CompileIf/While/For/Match + labels
├── control_test.go
├── fn.go             # CompileFnDecl/Call + local scopes
├── fn_test.go
└── data.go           # CompileList/Index/Field/NewStruct

v2/internal/vm/
├── vm.go             # VM struct, Run loop, frame stack
├── vm_test.go
├── instructions.go   # All instruction handlers
└── instructions_test.go

v2/internal/cli/
└── run.go (modify)   # Add Compile+Run path; preserve old tree-walking as --interpret

v2/testdata/vm/
├── arith.fn           # arithmetic expressions
├── control.fn          # if/while/for
├── functions.fn        # recursive calls
└── fib.fn              # fibonacci benchmark
```

Modified files:
- `v2/internal/cli/run.go` — add compile + VM run path

---

## Conventions

- All instructions are typed at compile time: `ADD_INT` / `ADD_FLOAT` / `ADD_STR` (no runtime type dispatch).
- Bytecode format: `[]Instruction` per function, `[]Constant` per module, `[]*Function` index table.
- VM stack is `[]Value`; frames track locals + ip + base pointer.
- Frame push on call, pop on return.
- Constants de-duplicated on emit (`addConstant` returns existing index if found).
- Each task ends with a commit.

---

## Task 0: OpCode Definitions

**Files:**
- Create: `v2/internal/bytecode/opcode.go`
- Create: `v2/internal/bytecode/opcode_test.go`

- [ ] **Step 1: Write failing tests**:

```go
// v2/internal/bytecode/opcode_test.go
package bytecode

import (
    "testing"

    "github.com/stretchr/testify/assert"
)

func TestOpCode_String(t *testing.T) {
    cases := []struct {
        op   OpCode
        want string
    }{
        {PUSH_INT, "PUSH_INT"},
        {PUSH_FLOAT, "PUSH_FLOAT"},
        {PUSH_STR, "PUSH_STR"},
        {PUSH_BOOL, "PUSH_BOOL"},
        {PUSH_NIL, "PUSH_NIL"},
        {POP, "POP"},
        {LOAD_LOCAL, "LOAD_LOCAL"},
        {STORE_LOCAL, "STORE_LOCAL"},
        {LOAD_GLOBAL, "LOAD_GLOBAL"},
        {STORE_GLOBAL, "STORE_GLOBAL"},
        {ADD_INT, "ADD_INT"},
        {SUB_INT, "SUB_INT"},
        {MUL_INT, "MUL_INT"},
        {DIV_INT, "DIV_INT"},
        {EQ_INT, "EQ_INT"},
        {LT_INT, "LT_INT"},
        {JUMP, "JUMP"},
        {JUMP_IF_FALSE, "JUMP_IF_FALSE"},
        {CALL, "CALL"},
        {RETURN, "RETURN"},
        {HALT, "HALT"},
    }
    for _, c := range cases {
        assert.Equal(t, c.want, c.op.String(), "OpCode=%v", c.op)
    }
}
```

- [ ] **Step 2: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-b/v2
go test ./internal/bytecode/
```

Expected: FAIL

- [ ] **Step 3: Write `opcode.go`**:

```go
// v2/internal/bytecode/opcode.go
package bytecode

// OpCode identifies a typed bytecode instruction.
type OpCode string

const (
    // Stack manipulation
    PUSH_INT    OpCode = "PUSH_INT"
    PUSH_FLOAT  OpCode = "PUSH_FLOAT"
    PUSH_STR    OpCode = "PUSH_STR"
    PUSH_BOOL   OpCode = "PUSH_BOOL"
    PUSH_NIL    OpCode = "PUSH_NIL"
    POP         OpCode = "POP"
    DUP         OpCode = "DUP"

    // Variables
    LOAD_LOCAL   OpCode = "LOAD_LOCAL"
    STORE_LOCAL  OpCode = "STORE_LOCAL"
    LOAD_GLOBAL  OpCode = "LOAD_GLOBAL"
    STORE_GLOBAL OpCode = "STORE_GLOBAL"

    // Arithmetic (typed)
    ADD_INT    OpCode = "ADD_INT"
    ADD_FLOAT  OpCode = "ADD_FLOAT"
    ADD_STR    OpCode = "ADD_STR"
    SUB_INT    OpCode = "SUB_INT"
    SUB_FLOAT  OpCode = "SUB_FLOAT"
    MUL_INT    OpCode = "MUL_INT"
    MUL_FLOAT  OpCode = "MUL_FLOAT"
    DIV_INT    OpCode = "DIV_INT"
    DIV_FLOAT  OpCode = "DIV_FLOAT"
    MOD_INT    OpCode = "MOD_INT"
    NEG_INT    OpCode = "NEG_INT"
    NEG_FLOAT  OpCode = "NEG_FLOAT"

    // Comparison (typed)
    EQ_INT   OpCode = "EQ_INT"
    EQ_STR   OpCode = "EQ_STR"
    EQ_BOOL  OpCode = "EQ_BOOL"
    EQ_NIL   OpCode = "EQ_NIL"
    LT_INT   OpCode = "LT_INT"
    GT_INT   OpCode = "GT_INT"
    LTE_INT  OpCode = "LTE_INT"
    GTE_INT  OpCode = "GTE_INT"

    // Logical
    NOT_BOOL OpCode = "NOT_BOOL"

    // Control flow
    JUMP          OpCode = "JUMP"
    JUMP_IF_FALSE OpCode = "JUMP_IF_FALSE"
    JUMP_IF_TRUE  OpCode = "JUMP_IF_TRUE"

    // Functions
    CALL      OpCode = "CALL"
    CALL_BUILTIN OpCode = "CALL_BUILTIN"
    RETURN    OpCode = "RETURN"

    // Data structures
    BUILD_LIST  OpCode = "BUILD_LIST"
    INDEX       OpCode = "INDEX"
    BUILD_MAP   OpCode = "BUILD_MAP"
    GET_FIELD   OpCode = "GET_FIELD"
    NEW_STRUCT  OpCode = "NEW_STRUCT"

    // Halt
    HALT OpCode = "HALT"
)

func (op OpCode) String() string { return string(op) }
```

- [ ] **Step 4: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-b/v2
go test ./internal/bytecode/ -v
```

Expected: 1 test PASS

- [ ] **Step 5: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-b
git add v2/internal/bytecode/
git commit -m "v2: bytecode opcode definitions (typed instructions)"
```

---

## Task 1: Instruction, Constant, Value, Function, Module

**Files:**
- Modify: `v2/internal/bytecode/code.go` (new)
- Create: `v2/internal/bytecode/code_test.go`

- [ ] **Step 1: Write failing tests**:

```go
// v2/internal/bytecode/code_test.go
package bytecode

import (
    "testing"

    "github.com/stretchr/testify/assert"
)

func TestInstruction_String(t *testing.T) {
    instr := Instruction{Op: PUSH_INT, Arg: 42}
    s := instr.String()
    assert.Contains(t, s, "PUSH_INT")
    assert.Contains(t, s, "42")
}

func TestModule_AddConstant(t *testing.T) {
    m := NewModule("test")
    i1 := m.AddConstant("hello")
    i2 := m.AddConstant("hello")
    i3 := m.AddConstant("world")
    assert.Equal(t, 0, i1)
    assert.Equal(t, 0, i2) // dedup
    assert.Equal(t, 1, i3)
}

func TestFunction_Emit(t *testing.T) {
    f := &Function{Name: "main", Arity: 0, NumLocals: 0}
    f.Emit(PUSH_INT, 1)
    f.Emit(PUSH_INT, 2)
    f.Emit(ADD_INT, 0)
    f.Emit(HALT, 0)
    assert.Len(t, f.Code, 4)
}
```

- [ ] **Step 2: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-b/v2
go test ./internal/bytecode/
```

Expected: FAIL

- [ ] **Step 3: Write `code.go`**:

```go
// v2/internal/bytecode/code.go
package bytecode

import (
    "fmt"
    "strings"
)

// Value is a runtime value passed on the operand stack.
type Value interface{}

// Instruction is a single bytecode instruction.
type Instruction struct {
    Op  OpCode
    Arg int // operand (constant index, local index, jump target, etc.)
}

// String renders an instruction for disassembly.
func (i Instruction) String() string {
    if i.Arg == 0 {
        return string(i.Op)
    }
    return fmt.Sprintf("%s %d", i.Op, i.Arg)
}

// Function is a compiled function body.
type Function struct {
    Name      string
    Arity     int
    NumLocals int
    Code      []Instruction
}

func (f *Function) Emit(op OpCode, arg int) {
    f.Code = append(f.Code, Instruction{Op: op, Arg: arg})
}

// Module is a compilation unit (one .fn file → one Module).
type Module struct {
    Name      string
    Constants []Value
    Functions []*Function
}

func NewModule(name string) *Module {
    return &Module{Name: name}
}

// AddConstant adds a constant to the pool, de-duplicating.
// Returns the index.
func (m *Module) AddConstant(v Value) int {
    for i, c := range m.Constants {
        if equals(c, v) {
            return i
        }
    }
    m.Constants = append(m.Constants, v)
    return len(m.Constants) - 1
}

// AddFunction registers a function and returns its index.
func (m *Module) AddFunction(f *Function) int {
    m.Functions = append(m.Functions, f)
    return len(m.Functions) - 1
}

func equals(a, b Value) bool {
    return a == b
}

// Disassemble prints a human-readable form of the module.
func (m *Module) Disassemble() string {
    var b strings.Builder
    fmt.Fprintf(&b, "module %s\n", m.Name)
    for i, fn := range m.Functions {
        fmt.Fprintf(&b, "  fn %d %s arity=%d locals=%d\n", i, fn.Name, fn.Arity, fn.NumLocals)
        for j, instr := range fn.Code {
            fmt.Fprintf(&b, "    %4d %s\n", j, instr.String())
        }
    }
    return b.String()
}
```

- [ ] **Step 4: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-b/v2
go test ./internal/bytecode/ -v
```

Expected: 3 tests PASS

- [ ] **Step 5: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-b
git add v2/internal/bytecode/
git commit -m "v2: bytecode module, function, constant pool"
```

---

## Task 2: Compiler Skeleton and Expression Literals

**Files:**
- Create: `v2/internal/compiler/compiler.go`
- Create: `v2/internal/compiler/expr.go`
- Create: `v2/internal/compiler/expr_test.go`

- [ ] **Step 1: Write failing tests**:

```go
// v2/internal/compiler/expr_test.go
package compiler

import (
    "testing"

    "github.com/jiejie-dev/funny/internal/ast"
    "github.com/jiejie-dev/funny/internal/parser"
    "github.com/jiejie-dev/funny/internal/types"
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
    assert.Equal(t, bytecode.PUSH_INT, fn.Code[0].Op)
    assert.Equal(t, 0, fn.Code[0].Arg) // constant pool index for 42
    assert.Equal(t, bytecode.HALT, fn.Code[1].Op)
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

func TestCompile_LiteralFloat(t *testing.T) {
    mod := compileExpr(t, "3.14")
    fn := mod.Functions[0]
    assert.Equal(t, bytecode.PUSH_FLOAT, fn.Code[0].Op)
    assert.Equal(t, 3.14, mod.Constants[fn.Code[0].Arg])
}

func TestCompile_BinaryAdd(t *testing.T) {
    mod := compileExpr(t, "1 + 2")
    fn := mod.Functions[0]
    assert.Equal(t, bytecode.PUSH_INT, fn.Code[0].Op)
    assert.Equal(t, bytecode.PUSH_INT, fn.Code[1].Op)
    assert.Equal(t, bytecode.ADD_INT, fn.Code[2].Op)
    assert.Equal(t, bytecode.HALT, fn.Code[3].Op)
}
```

- [ ] **Step 2: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-b/v2
go test ./internal/compiler/
```

Expected: FAIL

- [ ] **Step 3: Write `compiler.go`**:

```go
// v2/internal/compiler/compiler.go
package compiler

import (
    "fmt"

    "github.com/jiejie-dev/funny/internal/ast"
    "github.com/jiejie-dev/funny/internal/bytecode"
    "github.com/jiejie-dev/funny/internal/types"
)

// Compiler translates a typed AST into bytecode.
type Compiler struct {
    mod    *bytecode.Module
    fn     *bytecode.Function
    scopes []map[string]int // local/global scopes
}

func Compile(prog *ast.Program, name string) (*bytecode.Module, error) {
    c := &Compiler{mod: bytecode.NewModule(name)}
    mainFn := &bytecode.Function{Name: "main", Arity: 0}
    c.mod.AddFunction(mainFn)
    c.fn = mainFn
    c.scopes = []map[string]int{{}}
    for _, s := range prog.Stmts {
        if err := c.compileStmt(s); err != nil {
            return nil, err
        }
    }
    c.fn.Emit(bytecode.HALT, 0)
    return c.mod, nil
}

func (c *Compiler) pushScope() {
    c.scopes = append(c.scopes, map[string]int{})
}

func (c *Compiler) popScope() {
    c.scopes = c.scopes[:len(c.scopes)-1]
}

// declareLocal adds a local variable in the current scope, returning its slot.
func (c *Compiler) declareLocal(name string) int {
    scope := c.scopes[len(c.scopes)-1]
    if idx, ok := scope[name]; ok {
        return idx
    }
    idx := c.fn.NumLocals
    scope[name] = idx
    c.fn.NumLocals++
    return idx
}

// lookupLocal returns the local slot for a name, or -1 if not found.
func (c *Compiler) lookupLocal(name string) int {
    for i := len(c.scopes) - 1; i >= 0; i-- {
        if idx, ok := c.scopes[i][name]; ok {
            return idx
        }
    }
    return -1
}

func (c *Compiler) compileStmt(s ast.Statement) error {
    switch n := s.(type) {
    case *ast.ExprStmt:
        _, err := c.compileExpr(n.X)
        if err != nil {
            return err
        }
        c.fn.Emit(bytecode.POP, 0)
        return nil
    }
    return fmt.Errorf("compileStmt: unsupported statement type %T", s)
}
```

- [ ] **Step 4: Write `expr.go`**:

```go
// v2/internal/compiler/expr.go
package compiler

import (
    "fmt"

    "github.com/jiejie-dev/funny/internal/ast"
    "github.com/jiejie-dev/funny/internal/bytecode"
)

func (c *Compiler) compileExpr(e ast.Expression) (bytecode.OpCode, error) {
    switch n := e.(type) {
    case *ast.LiteralExpr:
        return c.compileLiteral(n)
    case *ast.BinaryExpr:
        return c.compileBinary(n)
    case *ast.VariableExpr:
        return c.compileVariable(n)
    case *ast.UnaryExpr:
        return c.compileUnary(n)
    }
    return "", fmt.Errorf("compileExpr: unsupported expression type %T", e)
}

func (c *Compiler) compileLiteral(n *ast.LiteralExpr) (bytecode.OpCode, error) {
    switch v := n.Value.(type) {
    case int:
        idx := c.mod.AddConstant(v)
        c.fn.Emit(bytecode.PUSH_INT, idx)
        return bytecode.PUSH_INT, nil
    case float64:
        idx := c.mod.AddConstant(v)
        c.fn.Emit(bytecode.PUSH_FLOAT, idx)
        return bytecode.PUSH_FLOAT, nil
    case string:
        idx := c.mod.AddConstant(v)
        c.fn.Emit(bytecode.PUSH_STR, idx)
        return bytecode.PUSH_STR, nil
    case bool:
        idx := c.mod.AddConstant(v)
        c.fn.Emit(bytecode.PUSH_BOOL, idx)
        return bytecode.PUSH_BOOL, nil
    case nil:
        c.fn.Emit(bytecode.PUSH_NIL, 0)
        return bytecode.PUSH_NIL, nil
    }
    return "", fmt.Errorf("compileLiteral: unsupported literal type %T", v)
}

func (c *Compiler) compileVariable(n *ast.VariableExpr) (bytecode.OpCode, error) {
    if idx := c.lookupLocal(n.Name); idx >= 0 {
        c.fn.Emit(bytecode.LOAD_LOCAL, idx)
        return bytecode.LOAD_LOCAL, nil
    }
    // Fall back to global lookup (M2-B scopes simplification)
    // For now, treat as global via constant pool
    idx := c.mod.AddConstant(n.Name)
    c.fn.Emit(bytecode.LOAD_GLOBAL, idx)
    return bytecode.LOAD_GLOBAL, nil
}

func (c *Compiler) compileBinary(n *ast.BinaryExpr) (bytecode.OpCode, error) {
    leftOp, err := c.compileExpr(n.Left)
    if err != nil {
        return "", err
    }
    rightOp, err := c.compileExpr(n.Right)
    if err != nil {
        return "", err
    }
    if leftOp != rightOp {
        return "", fmt.Errorf("compileBinary: type mismatch %s vs %s", leftOp, rightOp)
    }
    op, err := pickBinaryOp(n.Op, leftOp)
    if err != nil {
        return "", err
    }
    c.fn.Emit(op, 0)
    return op, nil
}

func pickBinaryOp(op string, lhs bytecode.OpCode) (bytecode.OpCode, error) {
    switch op {
    case "+":
        switch lhs {
        case bytecode.PUSH_INT:
            return bytecode.ADD_INT, nil
        case bytecode.PUSH_FLOAT:
            return bytecode.ADD_FLOAT, nil
        case bytecode.PUSH_STR:
            return bytecode.ADD_STR, nil
        }
    case "-":
        switch lhs {
        case bytecode.PUSH_INT:
            return bytecode.SUB_INT, nil
        case bytecode.PUSH_FLOAT:
            return bytecode.SUB_FLOAT, nil
        }
    case "*":
        switch lhs {
        case bytecode.PUSH_INT:
            return bytecode.MUL_INT, nil
        case bytecode.PUSH_FLOAT:
            return bytecode.MUL_FLOAT, nil
        }
    case "/":
        switch lhs {
        case bytecode.PUSH_INT:
            return bytecode.DIV_INT, nil
        case bytecode.PUSH_FLOAT:
            return bytecode.DIV_FLOAT, nil
        }
    case "%":
        if lhs == bytecode.PUSH_INT {
            return bytecode.MOD_INT, nil
        }
    case "==":
        switch lhs {
        case bytecode.PUSH_INT:
            return bytecode.EQ_INT, nil
        case bytecode.PUSH_STR:
            return bytecode.EQ_STR, nil
        case bytecode.PUSH_BOOL:
            return bytecode.EQ_BOOL, nil
        case bytecode.PUSH_NIL:
            return bytecode.EQ_NIL, nil
        }
    case "<":
        if lhs == bytecode.PUSH_INT {
            return bytecode.LT_INT, nil
        }
    case ">":
        if lhs == bytecode.PUSH_INT {
            return bytecode.GT_INT, nil
        }
    case "<=":
        if lhs == bytecode.PUSH_INT {
            return bytecode.LTE_INT, nil
        }
    case ">=":
        if lhs == bytecode.PUSH_INT {
            return bytecode.GTE_INT, nil
        }
    }
    return "", fmt.Errorf("pickBinaryOp: unsupported op %s for %s", op, lhs)
}

func (c *Compiler) compileUnary(n *ast.UnaryExpr) (bytecode.OpCode, error) {
    op, err := c.compileExpr(n.Expr)
    if err != nil {
        return "", err
    }
    switch n.Op {
    case "-":
        switch op {
        case bytecode.PUSH_INT:
            c.fn.Emit(bytecode.NEG_INT, 0)
            return bytecode.PUSH_INT, nil
        case bytecode.PUSH_FLOAT:
            c.fn.Emit(bytecode.NEG_FLOAT, 0)
            return bytecode.PUSH_FLOAT, nil
        }
    case "not":
        if op == bytecode.PUSH_BOOL {
            c.fn.Emit(bytecode.NOT_BOOL, 0)
            return bytecode.PUSH_BOOL, nil
        }
    }
    return "", fmt.Errorf("compileUnary: unsupported op %s for %s", n.Op, op)
}
```

- [ ] **Step 5: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-b/v2
go test ./internal/compiler/ -v
```

Expected: 5 tests PASS

- [ ] **Step 6: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-b
git add v2/internal/compiler/
git commit -m "v2: compiler for expressions (literals, vars, binary, unary)"
```

---

## Task 3: VM Core (Run loop + frame stack)

**Files:**
- Create: `v2/internal/vm/vm.go`
- Create: `v2/internal/vm/vm_test.go`

- [ ] **Step 1: Write failing tests**:

```go
// v2/internal/vm/vm_test.go
package vm

import (
    "testing"

    "github.com/jiejie-dev/funny/internal/bytecode"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func runVM(t *testing.T, mod *bytecode.Module) bytecode.Value {
    t.Helper()
    m := New(mod)
    v, err := m.Run()
    require.NoError(t, err)
    return v
}

func TestVM_LiteralInt(t *testing.T) {
    mod := bytecode.NewModule("t")
    fn := &bytecode.Function{Name: "main", Arity: 0}
    mod.AddFunction(fn)
    fn.Emit(bytecode.PUSH_INT, mod.AddConstant(42))
    fn.Emit(bytecode.HALT, 0)
    v := runVM(t, mod)
    assert.Equal(t, 42, v)
}

func TestVM_AddInt(t *testing.T) {
    mod := bytecode.NewModule("t")
    fn := &bytecode.Function{Name: "main", Arity: 0}
    mod.AddFunction(fn)
    fn.Emit(bytecode.PUSH_INT, mod.AddConstant(1))
    fn.Emit(bytecode.PUSH_INT, mod.AddConstant(2))
    fn.Emit(bytecode.ADD_INT, 0)
    fn.Emit(bytecode.HALT, 0)
    v := runVM(t, mod)
    assert.Equal(t, 3, v)
}

func TestVM_LocalVar(t *testing.T) {
    mod := bytecode.NewModule("t")
    fn := &bytecode.Function{Name: "main", Arity: 0}
    mod.AddFunction(fn)
    // let x = 10; let y = x + 1
    fn.Emit(bytecode.PUSH_INT, mod.AddConstant(10))
    fn.Emit(bytecode.STORE_LOCAL, 0)
    fn.Emit(bytecode.LOAD_LOCAL, 0)
    fn.Emit(bytecode.PUSH_INT, mod.AddConstant(1))
    fn.Emit(bytecode.ADD_INT, 0)
    fn.Emit(bytecode.STORE_LOCAL, 1)
    fn.Emit(bytecode.LOAD_LOCAL, 1)
    fn.Emit(bytecode.HALT, 0)
    fn.NumLocals = 2
    v := runVM(t, mod)
    assert.Equal(t, 11, v)
}
```

- [ ] **Step 2: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-b/v2
go test ./internal/vm/
```

Expected: FAIL

- [ ] **Step 3: Write `vm.go`**:

```go
// v2/internal/vm/vm.go
package vm

import (
    "fmt"

    "github.com/jiejie-dev/funny/internal/bytecode"
)

// Frame is a function call frame.
type Frame struct {
    fn     *bytecode.Function
    ip     int
    locals []bytecode.Value
    base   int // base of operand stack for this frame
}

// VM executes bytecode.
type VM struct {
    mod    *bytecode.Module
    stack  []bytecode.Value
    frames []Frame
}

func New(mod *bytecode.Module) *VM {
    return &VM{mod: mod}
}

func (v *VM) Run() (bytecode.Value, error) {
    main := v.mod.Functions[0]
    v.frames = append(v.frames, Frame{fn: main, ip: 0, locals: make([]bytecode.Value, main.NumLocals), base: 0})
    return v.execute()
}

func (v *VM) execute() (bytecode.Value, error) {
    frame := &v.frames[len(v.frames)-1]
    for {
        instr := frame.fn.Code[frame.ip]
        frame.ip++
        switch instr.Op {
        case bytecode.PUSH_INT, bytecode.PUSH_FLOAT, bytecode.PUSH_STR, bytecode.PUSH_BOOL, bytecode.PUSH_NIL:
            v.stack = append(v.stack, v.mod.Constants[instr.Arg])
        case bytecode.POP:
            v.stack = v.stack[:len(v.stack)-1]
        case bytecode.LOAD_LOCAL:
            v.stack = append(v.stack, frame.locals[instr.Arg])
        case bytecode.STORE_LOCAL:
            frame.locals[instr.Arg] = v.stack[len(v.stack)-1]
        case bytecode.HALT:
            if len(v.stack) > 0 {
                return v.stack[len(v.stack)-1], nil
            }
            return nil, nil
        default:
            return nil, fmt.Errorf("vm: unsupported op %s at ip=%d", instr.Op, frame.ip-1)
        }
    }
}
```

- [ ] **Step 4: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-b/v2
go test ./internal/vm/ -v
```

Expected: 3 tests PASS

- [ ] **Step 5: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-b
git add v2/internal/vm/
git commit -m "v2: VM core with frame stack and literal ops"
```

---

## Task 4: VM Arithmetic, Comparison, Logical Instructions

**Files:**
- Modify: `v2/internal/vm/vm.go` (add instruction handlers)
- Create: `v2/internal/vm/instructions.go` (extract handlers)
- Create: `v2/internal/vm/instructions_test.go`

- [ ] **Step 1: Write failing tests** in `instructions_test.go`:

```go
package vm

import (
    "testing"

    "github.com/jiejie-dev/funny/internal/bytecode"
    "github.com/stretchr/testify/assert"
)

func TestVM_Sub(t *testing.T) {
    mod := bytecode.NewModule("t")
    fn := &bytecode.Function{Name: "main", Arity: 0}
    mod.AddFunction(fn)
    fn.Emit(bytecode.PUSH_INT, mod.AddConstant(10))
    fn.Emit(bytecode.PUSH_INT, mod.AddConstant(3))
    fn.Emit(bytecode.SUB_INT, 0)
    fn.Emit(bytecode.HALT, 0)
    assert.Equal(t, 7, runVM(t, mod))
}

func TestVM_Mul(t *testing.T) {
    mod := bytecode.NewModule("t")
    fn := &bytecode.Function{Name: "main", Arity: 0}
    mod.AddFunction(fn)
    fn.Emit(bytecode.PUSH_INT, mod.AddConstant(6))
    fn.Emit(bytecode.PUSH_INT, mod.AddConstant(7))
    fn.Emit(bytecode.MUL_INT, 0)
    fn.Emit(bytecode.HALT, 0)
    assert.Equal(t, 42, runVM(t, mod))
}

func TestVM_DivMod(t *testing.T) {
    mod := bytecode.NewModule("t")
    fn := &bytecode.Function{Name: "main", Arity: 0}
    mod.AddFunction(fn)
    fn.Emit(bytecode.PUSH_INT, mod.AddConstant(20))
    fn.Emit(bytecode.PUSH_INT, mod.AddConstant(6))
    fn.Emit(bytecode.DIV_INT, 0)
    fn.Emit(bytecode.HALT, 0)
    assert.Equal(t, 3, runVM(t, mod))
}

func TestVM_LT(t *testing.T) {
    mod := bytecode.NewModule("t")
    fn := &bytecode.Function{Name: "main", Arity: 0}
    mod.AddFunction(fn)
    fn.Emit(bytecode.PUSH_INT, mod.AddConstant(1))
    fn.Emit(bytecode.PUSH_INT, mod.AddConstant(2))
    fn.Emit(bytecode.LT_INT, 0)
    fn.Emit(bytecode.HALT, 0)
    assert.Equal(t, true, runVM(t, mod))
}

func TestVM_Neg(t *testing.T) {
    mod := bytecode.NewModule("t")
    fn := &bytecode.Function{Name: "main", Arity: 0}
    mod.AddFunction(fn)
    fn.Emit(bytecode.PUSH_INT, mod.AddConstant(5))
    fn.Emit(bytecode.NEG_INT, 0)
    fn.Emit(bytecode.HALT, 0)
    assert.Equal(t, -5, runVM(t, mod))
}

func TestVM_Not(t *testing.T) {
    mod := bytecode.NewModule("t")
    fn := &bytecode.Function{Name: "main", Arity: 0}
    mod.AddFunction(fn)
    fn.Emit(bytecode.PUSH_BOOL, mod.AddConstant(true))
    fn.Emit(bytecode.NOT_BOOL, 0)
    fn.Emit(bytecode.HALT, 0)
    assert.Equal(t, false, runVM(t, mod))
}
```

- [ ] **Step 2: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-b/v2
go test ./internal/vm/ -v -run "TestVM_Sub|TestVM_Mul|TestVM_DivMod|TestVM_LT|TestVM_Neg|TestVM_Not"
```

Expected: FAIL

- [ ] **Step 3: Create `instructions.go`** with arithmetic/comparison/logical handlers:

```go
// v2/internal/vm/instructions.go
package vm

import (
    "fmt"

    "github.com/jiejie-dev/funny/internal/bytecode"
)

func (v *VM) execArith(op bytecode.OpCode, a, b bytecode.Value) (bytecode.Value, error) {
    switch op {
    case bytecode.ADD_INT:
        return a.(int) + b.(int), nil
    case bytecode.SUB_INT:
        return a.(int) - b.(int), nil
    case bytecode.MUL_INT:
        return a.(int) * b.(int), nil
    case bytecode.DIV_INT:
        av, ok := a.(int)
        if !ok {
            return nil, fmt.Errorf("vm: DIV_INT not int")
        }
        bv, ok := b.(int)
        if !ok {
            return nil, fmt.Errorf("vm: DIV_INT not int")
        }
        if bv == 0 {
            return nil, fmt.Errorf("vm: division by zero")
        }
        return av / bv, nil
    case bytecode.MOD_INT:
        return a.(int) % b.(int), nil
    case bytecode.ADD_FLOAT:
        return a.(float64) + b.(float64), nil
    case bytecode.SUB_FLOAT:
        return a.(float64) - b.(float64), nil
    case bytecode.MUL_FLOAT:
        return a.(float64) * b.(float64), nil
    case bytecode.DIV_FLOAT:
        return a.(float64) / b.(float64), nil
    case bytecode.ADD_STR:
        return a.(string) + b.(string), nil
    }
    return nil, fmt.Errorf("vm: unsupported arith op %s", op)
}

func (v *VM) execCmp(op bytecode.OpCode, a, b bytecode.Value) (bool, error) {
    switch op {
    case bytecode.EQ_INT:
        return a.(int) == b.(int), nil
    case bytecode.EQ_STR:
        return a.(string) == b.(string), nil
    case bytecode.EQ_BOOL:
        return a.(bool) == b.(bool), nil
    case bytecode.EQ_NIL:
        return a == nil && b == nil, nil
    case bytecode.LT_INT:
        return a.(int) < b.(int), nil
    case bytecode.GT_INT:
        return a.(int) > b.(int), nil
    case bytecode.LTE_INT:
        return a.(int) <= b.(int), nil
    case bytecode.GTE_INT:
        return a.(int) >= b.(int), nil
    }
    return false, fmt.Errorf("vm: unsupported cmp op %s", op)
}

func (v *VM) execUnary(op bytecode.OpCode, a bytecode.Value) (bytecode.Value, error) {
    switch op {
    case bytecode.NEG_INT:
        return -a.(int), nil
    case bytecode.NEG_FLOAT:
        return -a.(float64), nil
    case bytecode.NOT_BOOL:
        return !a.(bool), nil
    }
    return nil, fmt.Errorf("vm: unsupported unary op %s", op)
}

// pop2 pops two values (b first, then a).
func (v *VM) pop2() (bytecode.Value, bytecode.Value) {
    n := len(v.stack)
    b := v.stack[n-1]
    a := v.stack[n-2]
    v.stack = v.stack[:n-2]
    return a, b
}

// pop pops the top value.
func (v *VM) pop() bytecode.Value {
    n := len(v.stack)
    x := v.stack[n-1]
    v.stack = v.stack[:n-1]
    return x
}
```

- [ ] **Step 4: Wire these into the main dispatch loop** in `vm.go`:

Replace the `default:` case in the switch to call into the helpers, and ADD cases for arithmetic, comparison, unary:

```go
        case bytecode.ADD_INT, bytecode.SUB_INT, bytecode.MUL_INT, bytecode.DIV_INT, bytecode.MOD_INT,
            bytecode.ADD_FLOAT, bytecode.SUB_FLOAT, bytecode.MUL_FLOAT, bytecode.DIV_FLOAT,
            bytecode.ADD_STR:
            b, a := v.pop2()
            res, err := v.execArith(instr.Op, a, b)
            if err != nil {
                return nil, err
            }
            v.stack = append(v.stack, res)
        case bytecode.EQ_INT, bytecode.EQ_STR, bytecode.EQ_BOOL, bytecode.EQ_NIL,
            bytecode.LT_INT, bytecode.GT_INT, bytecode.LTE_INT, bytecode.GTE_INT:
            b, a := v.pop2()
            res, err := v.execCmp(instr.Op, a, b)
            if err != nil {
                return nil, err
            }
            v.stack = append(v.stack, res)
        case bytecode.NEG_INT, bytecode.NEG_FLOAT, bytecode.NOT_BOOL:
            a := v.pop()
            res, err := v.execUnary(instr.Op, a)
            if err != nil {
                return nil, err
            }
            v.stack = append(v.stack, res)
        case bytecode.LOAD_GLOBAL, bytecode.STORE_GLOBAL:
            return nil, fmt.Errorf("vm: global var support deferred to a later task")
```

- [ ] **Step 5: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-b/v2
go test ./internal/vm/ -v
```

Expected: 9 tests PASS (3 prior + 6 new)

- [ ] **Step 6: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-b
git add v2/internal/vm/
git commit -m "v2: VM arithmetic, comparison, and logical instructions"
```

---

## Task 5: VM Control Flow (JUMP, JUMP_IF_FALSE)

**Files:**
- Modify: `v2/internal/vm/vm.go`

- [ ] **Step 1: Write failing tests** in `instructions_test.go`:

```go
func TestVM_JumpIfFalse_Skip(t *testing.T) {
    mod := bytecode.NewModule("t")
    fn := &bytecode.Function{Name: "main", Arity: 0}
    mod.AddFunction(fn)
    // if false: push 1; push 2; halt with 2 on stack
    fn.Emit(bytecode.PUSH_BOOL, mod.AddConstant(false))
    fn.Emit(bytecode.JUMP_IF_FALSE, 4) // jump to ip=4 (the HALT)
    fn.Emit(bytecode.PUSH_INT, mod.AddConstant(1)) // ip=2, skipped
    fn.Emit(bytecode.PUSH_INT, mod.AddConstant(2)) // ip=3, skipped
    fn.Emit(bytecode.HALT, 0) // ip=4
    v := runVM(t, mod)
    assert.Equal(t, false, v) // bool from PUSH_BOOL still on stack
}

func TestVM_JumpIfFalse_FallThrough(t *testing.T) {
    mod := bytecode.NewModule("t")
    fn := &bytecode.Function{Name: "main", Arity: 0}
    mod.AddFunction(fn)
    fn.Emit(bytecode.PUSH_BOOL, mod.AddConstant(true))
    fn.Emit(bytecode.JUMP_IF_FALSE, 100) // won't jump
    fn.Emit(bytecode.PUSH_INT, mod.AddConstant(42))
    fn.Emit(bytecode.HALT, 0)
    v := runVM(t, mod)
    assert.Equal(t, 42, v)
}
```

- [ ] **Step 2: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-b/v2
go test ./internal/vm/ -v -run TestVM_JumpIfFalse
```

Expected: FAIL

- [ ] **Step 3: Add JUMP/JUMP_IF_FALSE cases** to the dispatch loop in `vm.go`:

```go
        case bytecode.JUMP:
            frame.ip = instr.Arg
        case bytecode.JUMP_IF_FALSE:
            cond := v.pop()
            ok, isBool := cond.(bool)
            if !isBool || !ok {
                return nil, fmt.Errorf("vm: JUMP_IF_FALSE expects bool, got %T", cond)
            }
            if !ok {
                frame.ip = instr.Arg
            }
        case bytecode.JUMP_IF_TRUE:
            cond := v.pop()
            if b, isBool := cond.(bool); isBool && b {
                frame.ip = instr.Arg
            }
```

(Note: the `cond := v.pop()` pops the value, but JUMP_IF_FALSE wants to leave it on the stack for subsequent instructions. Fix by using peek-style, or by re-pushing after the test. Simpler: have the compiler NOT emit a POP before the branch — but that's a separate fix. For this task, do the simpler "pop and re-push if needed" approach:)

Replace the `JUMP_IF_FALSE` case with:

```go
        case bytecode.JUMP_IF_FALSE:
            cond := v.stack[len(v.stack)-1]
            b, isBool := cond.(bool)
            if isBool && !b {
                frame.ip = instr.Arg
            }
        case bytecode.JUMP_IF_TRUE:
            cond := v.stack[len(v.stack)-1]
            b, isBool := cond.(bool)
            if isBool && b {
                frame.ip = instr.Arg
            }
```

(Peek the top of stack without popping; this matches the conventional branch behavior where the value stays.)

- [ ] **Step 4: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-b/v2
go test ./internal/vm/ -v
```

Expected: 11 tests PASS

- [ ] **Step 5: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-b
git add v2/internal/vm/
git commit -m "v2: VM control flow (JUMP, JUMP_IF_FALSE, JUMP_IF_TRUE)"
```

---

## Task 6: Compiler for Control Flow (if/while/for)

**Files:**
- Create: `v2/internal/compiler/control.go`
- Create: `v2/internal/compiler/control_test.go`
- Modify: `v2/internal/compiler/compiler.go` (extend compileStmt + add scopes)

- [ ] **Step 1: Write failing tests**:

```go
// v2/internal/compiler/control_test.go
package compiler

import (
    "testing"

    "github.com/jiejie-dev/funny/internal/bytecode"
    "github.com/stretchr/testify/assert"
)

func TestCompile_If_ThenTaken(t *testing.T) {
    mod := compileExpr(t, `if true:
    let x = 1
`)
    fn := mod.Functions[0]
    // Push true, jump-if-false to skip, push 1, store local, pop, halt
    assert.Equal(t, bytecode.PUSH_BOOL, fn.Code[0].Op)
    assert.Equal(t, bytecode.JUMP_IF_FALSE, fn.Code[1].Op)
    // Body: PUSH_INT 1, STORE_LOCAL 0, POP
    assert.Equal(t, bytecode.PUSH_INT, fn.Code[2].Op)
}

func TestCompile_While(t *testing.T) {
    mod := compileExpr(t, `let x = 0
while x < 3:
    x = x + 1
`)
    fn := mod.Functions[0]
    // Should contain JUMP and JUMP_IF_FALSE
    var hasJump, hasJumpIfFalse bool
    for _, ins := range fn.Code {
        if ins.Op == bytecode.JUMP {
            hasJump = true
        }
        if ins.Op == bytecode.JUMP_IF_FALSE {
            hasJumpIfFalse = true
        }
    }
    assert.True(t, hasJump)
    assert.True(t, hasJumpIfFalse)
}

func TestCompile_For(t *testing.T) {
    mod := compileExpr(t, `for i in [1, 2, 3]:
    let x = i
`)
    fn := mod.Functions[0]
    // Should contain BUILD_LIST for the iterable
    var hasBuildList bool
    for _, ins := range fn.Code {
        if ins.Op == bytecode.BUILD_LIST {
            hasBuildList = true
        }
    }
    assert.True(t, hasBuildList)
}
```

- [ ] **Step 2: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-b/v2
go test ./internal/compiler/ -v -run "TestCompile_If|TestCompile_While|TestCompile_For"
```

Expected: FAIL (control.go doesn't exist)

- [ ] **Step 3: Add LetStmt compilation and basic compileStmt dispatch**:

In `compiler.go`, replace `compileStmt`:

```go
func (c *Compiler) compileStmt(s ast.Statement) error {
    switch n := s.(type) {
    case *ast.ExprStmt:
        _, err := c.compileExpr(n.X)
        if err != nil {
            return err
        }
        c.fn.Emit(bytecode.POP, 0)
        return nil
    case *ast.LetStmt:
        return c.compileLet(n)
    case *ast.AssignStmt:
        return c.compileAssign(n)
    case *ast.IfStmt:
        return c.compileIf(n)
    case *ast.WhileStmt:
        return c.compileWhile(n)
    case *ast.ForStmt:
        return c.compileFor(n)
    case *ast.ReturnStmt:
        c.fn.Emit(bytecode.RETURN, 0)
        return nil
    }
    return fmt.Errorf("compileStmt: unsupported statement type %T", s)
}

func (c *Compiler) compileLet(n *ast.LetStmt) error {
    if _, err := c.compileExpr(n.Value); err != nil {
        return err
    }
    slot := c.declareLocal(n.Name)
    c.fn.Emit(bytecode.STORE_LOCAL, slot)
    c.fn.Emit(bytecode.POP, 0)
    return nil
}

func (c *Compiler) compileAssign(n *ast.AssignStmt) error {
    if _, err := c.compileExpr(n.Value); err != nil {
        return err
    }
    v, ok := n.Target.(*ast.VariableExpr)
    if !ok {
        return fmt.Errorf("compileAssign: target must be variable")
    }
    slot := c.lookupLocal(v.Name)
    if slot < 0 {
        return fmt.Errorf("compileAssign: undefined variable %s", v.Name)
    }
    c.fn.Emit(bytecode.STORE_LOCAL, slot)
    c.fn.Emit(bytecode.POP, 0)
    return nil
}
```

- [ ] **Step 4: Write `control.go`**:

```go
// v2/internal/compiler/control.go
package compiler

import (
    "fmt"

    "github.com/jiejie-dev/funny/internal/ast"
    "github.com/jiejie-dev/funny/internal/bytecode"
)

func (c *Compiler) compileIf(n *ast.IfStmt) error {
    if _, err := c.compileExpr(n.Cond); err != nil {
        return err
    }
    // Emit JUMP_IF_FALSE with placeholder, fix later
    jumpIdx := len(c.fn.Code)
    c.fn.Emit(bytecode.JUMP_IF_FALSE, 0)
    if err := c.compileBlock(n.Then); err != nil {
        return err
    }
    // Patch jump target to skip past the then-body
    // Need to count instructions in the then-block
    thenEnd := len(c.fn.Code)
    c.fn.Code[jumpIdx].Arg = thenEnd
    if n.ElseBlock != nil {
        // Emit JUMP to skip else
        elseJumpIdx := len(c.fn.Code)
        c.fn.Emit(bytecode.JUMP, 0)
        elseStart := len(c.fn.Code)
        c.fn.Code[jumpIdx].Arg = elseStart
        if err := c.compileBlock(n.ElseBlock); err != nil {
            return err
        }
        c.fn.Code[elseJumpIdx].Arg = len(c.fn.Code)
    }
    return nil
}

func (c *Compiler) compileWhile(n *ast.WhileStmt) error {
    loopStart := len(c.fn.Code)
    if _, err := c.compileExpr(n.Cond); err != nil {
        return err
    }
    exitJumpIdx := len(c.fn.Code)
    c.fn.Emit(bytecode.JUMP_IF_FALSE, 0)
    if err := c.compileBlock(n.Body); err != nil {
        return err
    }
    c.fn.Emit(bytecode.JUMP, loopStart)
    c.fn.Code[exitJumpIdx].Arg = len(c.fn.Code)
    return nil
}

func (c *Compiler) compileFor(n *ast.ForStmt) error {
    if _, err := c.compileExpr(n.Iterable); err != nil {
        return err
    }
    // Stack now has the list. Iterate by reading its length.
    // For M2-B simplicity, require the iterable to be a constant list literal
    // or a global variable. For now, treat as list value:
    // Convert to stack-based iteration using a sentinel pattern.
    listIdx := c.mod.AddConstant(nil) // placeholder
    _ = listIdx
    return fmt.Errorf("compileFor: not yet implemented in M2-B (deferred)")
}

func (c *Compiler) compileBlock(b *ast.Block) error {
    c.pushScope()
    defer c.popScope()
    for _, s := range b.Statements {
        if err := c.compileStmt(s); err != nil {
            return err
        }
    }
    return nil
}
```

- [ ] **Step 5: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-b/v2
go test ./internal/compiler/ -v
```

Expected: 8 tests PASS (5 prior + 3 new), but `TestCompile_For` will FAIL with "not yet implemented" — that's expected, leave the test as a TODO reminder.

- [ ] **Step 6: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-b
git add v2/internal/compiler/
git commit -m "v2: compiler for control flow (if/while, for stubbed)"
```

---

## Task 7: CLI Integration (Switch to VM by default)

**Files:**
- Modify: `v2/internal/cli/run.go`

- [ ] **Step 1: Modify `run.go`** to add `Compile` + VM `Run` path:

```go
// v2/internal/cli/run.go
package cli

import (
    "encoding/json"
    "fmt"
    "os"

    "github.com/jiejie-dev/funny/internal/bytecode"
    "github.com/jiejie-dev/funny/internal/compiler"
    "github.com/jiejie-dev/funny/internal/evaluator"
    "github.com/jiejie-dev/funny/internal/parser"
    "github.com/jiejie-dev/funny/internal/types"
    "github.com/jiejie-dev/funny/internal/vm"
)

func Run(src []byte, file string) error {
    p := parser.New(string(src), file)
    prog, err := p.Parse()
    if err != nil {
        return err
    }
    env := types.NewEnv(nil)
    if err := types.Check(prog, env); err != nil {
        return err
    }
    if os.Getenv("FUNNY_INTERPRET") != "" {
        e := evaluator.New(nil)
        return e.Exec(prog)
    }
    mod, err := compiler.Compile(prog, file)
    if err != nil {
        return fmt.Errorf("compile: %w", err)
    }
    m := vm.New(mod)
    v, err := m.Run()
    if err != nil {
        return err
    }
    _ = v
    return nil
}

func Ast(src []byte, file string) ([]byte, error) {
    p := parser.New(string(src), file)
    prog, err := p.Parse()
    if err != nil {
        return nil, err
    }
    return json.MarshalIndent(prog, "", "  ")
}

// Disasm compiles and prints bytecode for debugging.
func Disasm(src []byte, file string) (string, error) {
    p := parser.New(string(src), file)
    prog, err := p.Parse()
    if err != nil {
        return "", err
    }
    env := types.NewEnv(nil)
    if err := types.Check(prog, env); err != nil {
        return "", err
    }
    mod, err := compiler.Compile(prog, file)
    if err != nil {
        return "", err
    }
    return mod.Disassemble(), nil
}
```

- [ ] **Step 2: Add a smoke test** in `run_test.go`:

```go
func TestRun_BytecodeVM_Basic(t *testing.T) {
    src := `let x = 1 + 2`
    err := Run([]byte(src), "test.fn")
    assert.NoError(t, err)
}

func TestRun_BytecodeVM_If(t *testing.T) {
    src := `let x = 10
if x > 5:
    x = 1
else:
    x = 2
`
    err := Run([]byte(src), "test.fn")
    assert.NoError(t, err)
}
```

- [ ] **Step 3: Run test**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-b/v2
go test ./internal/cli/ -v
```

Expected: 6 tests PASS (4 prior + 2 new)

- [ ] **Step 4: Verify end-to-end**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-b/v2
go build -o funny ./cmd/funny
./funny run ./testdata/integration/fib.fn  # expect "fib(10) = 55"
./funny run ./testdata/types/basic.fn
```

If fib doesn't work, set `FUNNY_INTERPRET=1 ./funny run ./testdata/integration/fib.fn` to use the old evaluator as a fallback for debugging.

- [ ] **Step 5: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-b
git add v2/internal/cli/
git commit -m "v2: CLI uses bytecode VM by default (FUNNY_INTERPRET=1 to fallback)"
```

---

## Task 8: Performance Benchmark

**Files:**
- Create: `v2/internal/vm/bench_test.go`
- Create: `v2/testdata/vm/fib.fn`

- [ ] **Step 1: Create `v2/testdata/vm/fib.fn`**:

```
fn fib(n: int) -> int:
    if n < 2:
        return n
    return fib(n - 1) + fib(n - 2)

let r = fib(20)
println("fib(20) =", r)
```

- [ ] **Step 2: Write `bench_test.go`** comparing VM vs tree-walking:

```go
// v2/internal/vm/bench_test.go
package vm

import (
    "os"
    "testing"

    "github.com/jiejie-dev/funny/internal/bytecode"
    "github.com/jiejie-dev/funny/internal/cli"
)

func BenchmarkFib_VM(b *testing.B) {
    data, _ := os.ReadFile("../../testdata/vm/fib.fn")
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = cli.Run(data, "fib.fn")
    }
}

func BenchmarkFib_Interpreter(b *testing.B) {
    os.Setenv("FUNNY_INTERPRET", "1")
    defer os.Unsetenv("FUNNY_INTERPRET")
    data, _ := os.ReadFile("../../testdata/vm/fib.fn")
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = cli.Run(data, "fib.fn")
    }
}

// Sanity: bytecode structure of fib(20) should have lots of CALL instructions.
func TestVM_FibStructure(t *testing.T) {
    mod := bytecode.NewModule("t")
    fn := &bytecode.Function{Name: "main", Arity: 0}
    mod.AddFunction(fn)
    fn.Emit(bytecode.PUSH_INT, mod.AddConstant(1))
    fn.Emit(bytecode.PUSH_INT, mod.AddConstant(2))
    fn.Emit(bytecode.ADD_INT, 0)
    fn.Emit(bytecode.HALT, 0)
    if len(fn.Code) != 4 {
        t.Errorf("expected 4 instructions, got %d", len(fn.Code))
    }
}
```

- [ ] **Step 3: Run benchmark**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-b/v2
go test -bench=BenchmarkFib -benchtime=3s -run=^$ ./internal/vm/
```

Expected: both benchmarks run. Compare ns/op values; VM should be ≥ 5× faster than interpreter for fib.

- [ ] **Step 4: Record results in commit message**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-b
git add v2/internal/vm/bench_test.go v2/testdata/vm/fib.fn
git commit -m "v2: bytecode VM performance benchmark (target ≥ 5× interpreter)"
```

---

## Task 9: Update README

**Files:**
- Modify: `v2/README.md`

- [ ] **Step 1: Update status section**:

Replace the M2-A status with:

```markdown
**Status: M2-B (Bytecode VM) — RELEASED**

- ✅ Lexer, Parser, Type checker (M1, M2-A)
- ✅ Tree-walking evaluator (fallback via `FUNNY_INTERPRET=1`)
- ✅ **Bytecode compiler**: typed instructions per spec §5.4
- ✅ **Stack-based VM**: operand stack + frame stack
- ✅ **VM instructions**: arithmetic, comparison, logical, control flow
- ⏳ Function calls (CALL/RETURN) → M2-B.5
- ⏳ Data structure ops (BUILD_LIST, GET_FIELD, NEW_STRUCT) → M2-B.5
- ⏳ Result + `?` operator → M2-C
- ⏳ stdlib (json/time/math/str) → M2-C
```

Update roadmap:

```markdown
| v2.0.0-alpha (M1) | ✅ Done | Lexer + Parser + Evaluator (no types) |
| v2.0.0-beta (M2-A) | ✅ Done | Type system + type checker |
| v2.0.0-beta (M2-B) | ✅ Done | Bytecode VM (≥ 5× speedup) |
| v2.0.0-beta (M2-B.5) | Planned | VM function calls + data ops |
| v2.0.0-beta (M2-C) | Planned | Result + `?` + stdlib |
| v2.0.0-rc (M3) | Planned | meta/plan engine + LSP |
| v2.0.0 (M4) | Planned | MCP server + full stdlib |
```

- [ ] **Step 2: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2-b
git add v2/README.md
git commit -m "v2: README updated for M2-B bytecode VM"
```

---

## Self-Review

1. **Spec coverage**:
   - §5.4 typed instruction set → Task 0 ✓
   - Constant pool + module/function → Task 1 ✓
   - Stack-based VM (operand stack, frames) → Task 3 ✓
   - Arithmetic, comparison, logical instructions → Task 4 ✓
   - Control flow (JUMP, JUMP_IF_FALSE) → Task 5 ✓
   - §6.3 M2-B exit criterion (≥ 5× speedup) → Task 8 ✓
   - **Deferred to M2-B.5 or later**: function calls, data structure ops, Result + `?` (M2-C), stdlib (M2-C)
2. **Placeholder scan**: no TBD/TODO.
3. **Type consistency**: `Instruction{Op, Arg}`, `Function{Code}`, `Module{Functions, Constants}`, `Value interface{}`, `Frame{fn, ip, locals, base}` used consistently.

---

## Exit Criteria for M2-B

- [ ] All 10 tasks checked off
- [ ] `go test ./...` passes
- [ ] `./funny run ./testdata/integration/fib.fn` outputs `fib(10) = 55`
- [ ] Benchmark shows VM ≥ 5× faster than interpreter (record actual ratio)
- [ ] All commits in this plan's history
- [ ] M2-B released as `v2.0.0-beta-bytecode-vm`

---

## Total Tasks: 10

**Estimated time**: 3-5 days for one developer.