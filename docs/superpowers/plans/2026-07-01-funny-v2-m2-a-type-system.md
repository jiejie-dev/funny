# Funny v2 M2-A: Strong Type System Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the type system foundation for Funny v2 — type representations, type environment, type parser, type checker, error codes E2xxx, and integration with the parse pipeline.

**Architecture:** Standalone `internal/types` package using the sealed-interface pattern (private marker methods) consistent with `ast` and `lexer`. Types are immutable values compared via `Equal()`. Type checker operates on existing AST (with optional `ResolvedType` field added). Type parsing is a small recursive-descent parser that converts `TypeAnn string` (already produced by lexer/parser) into `Type` values.

**Tech Stack:** Go 1.22+, `github.com/stretchr/testify`.

**Reference Spec:** `docs/superpowers/specs/2026-07-01-funny-v2-ai-native-language-design.md` §2.2 (Base Types), §2.3 (Strong Typing), §5.3 (Module Structure for types), §6.3 (M2 exit criteria).

**Scope:** This plan covers ONLY M2-A (type system). Bytecode VM (M2-B) and stdlib (M2-C) are separate plans.

**Reference:** M1 plan is at `docs/superpowers/plans/2026-07-01-funny-v2-m1-syntax-skeleton.md` — the codebase already has lexer, parser, AST, and tree-walking evaluator; this plan extends it.

---

## File Structure

New files (under `v2/internal/types/`):

```
v2/internal/types/
├── types.go          # Type interface + all concrete types + helpers
├── types_test.go     # Tests for type construction and equality
├── env.go            # Type environment (analogous to evaluator/scope)
├── env_test.go
├── parse.go          # Recursive-descent parser for TypeAnn string → Type
├── parse_test.go
├── check.go          # Type checker for expressions and statements
├── check_test.go
└── errors.go         # Type error codes (E2xxx range)
```

Modified files:
- `v2/internal/ast/ast.go` — add optional `ResolvedType` field to key nodes (LetStmt, Param, FnDecl, StructDecl fields, CallExpr, BinaryExpr, etc.) so the type checker can record resolved types and the evaluator can read them
- `v2/internal/ast/ast_test.go` — update tests if needed
- `v2/internal/cli/run.go` — add `TypeCheck()` step between `Parse()` and `Exec()`
- `v2/internal/cli/run_test.go` — verify error reporting

New testdata:
- `v2/testdata/types/basic.fn` — primitive types
- `v2/testdata/types/lists.fn` — list type annotations
- `v2/testdata/types/structs.fn` — struct declarations and usage
- `v2/testdata/types/functions.fn` — function signatures
- `v2/testdata/types/errors.fn` — sample that should produce type errors

---

## Conventions

- All Type types are immutable values; mutating returns new instances.
- All Type types implement `String() string` returning the type annotation string.
- All Type types implement `Equal(other Type) bool` for structural comparison.
- Tests use `testify/assert` and `testify/require`.
- Each task ends with a commit.
- Error codes: E2xxx for type errors (per spec §5.9).
- When the type checker fails, it returns an error wrapping `*errs.Error` (NOT a panic).

---

## Task 0: types Package Skeleton

**Files:**
- Create: `v2/internal/types/types.go`
- Create: `v2/internal/types/types_test.go`

- [ ] **Step 1: Write failing test** `types_test.go`:

```go
package types

import (
    "testing"

    "github.com/stretchr/testify/assert"
)

func TestType_Primitive_String(t *testing.T) {
    p := Primitive("int")
    assert.Equal(t, "int", p.String())
}

func TestType_Primitive_Equal(t *testing.T) {
    a := Primitive("int")
    b := Primitive("int")
    c := Primitive("str")
    assert.True(t, a.Equal(b))
    assert.False(t, a.Equal(c))
}

func TestType_Equal_Nil(t *testing.T) {
    var nilType Type
    p := Primitive("int")
    assert.False(t, p.Equal(nilType))
    assert.False(t, nilType.Equal(p))
}
```

- [ ] **Step 2: Run test to verify it fails**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2/v2
go test ./internal/types/
```

Expected: FAIL (package doesn't exist)

- [ ] **Step 3: Write implementation** `types.go`:

```go
package types

// Type is the sealed interface for all type system types.
// Only types in this package can implement it (private marker).
type Type interface {
    String() string
    Equal(other Type) bool
    typeMarker()
}

// Primitive is a built-in type like "int", "str", "bool", "float", "nil".
type Primitive string

func (p Primitive) String() string { return string(p) }
func (p Primitive) Equal(other Type) bool {
    o, ok := other.(Primitive)
    return ok && p == o
}
func (p Primitive) typeMarker() {}

// Equal is a convenience for comparing two Types.
// Returns false if either is nil.
func Equal(a, b Type) bool {
    if a == nil || b == nil {
        return false
    }
    return a.Equal(b)
}
```

- [ ] **Step 4: Run test to verify it passes**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2/v2
go test ./internal/types/ -v
```

Expected: 3 tests PASS

- [ ] **Step 5: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2
git add v2/internal/types/
git commit -m "v2: types package skeleton with Primitive"
```

---

## Task 1: List and Map Types

**Files:**
- Modify: `v2/internal/types/types.go`
- Modify: `v2/internal/types/types_test.go`

- [ ] **Step 1: Append failing tests**:

```go
func TestType_List_String(t *testing.T) {
    lt := List{Primitive("int")}
    assert.Equal(t, "list[int]", lt.String())
}

func TestType_List_Equal(t *testing.T) {
    a := List{Primitive("int")}
    b := List{Primitive("int")}
    c := List{Primitive("str")}
    assert.True(t, a.Equal(b))
    assert.False(t, a.Equal(c))
}

func TestType_Map_String(t *testing.T) {
    m := Map{Primitive("str"), Primitive("int")}
    assert.Equal(t, "map[str, int]", m.String())
}

func TestType_Map_Equal(t *testing.T) {
    a := Map{Primitive("str"), Primitive("int")}
    b := Map{Primitive("str"), Primitive("int")}
    c := Map{Primitive("str"), Primitive("str")}
    assert.True(t, a.Equal(b))
    assert.False(t, a.Equal(c))
}
```

- [ ] **Step 2: Run test to verify it fails**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2/v2
go test ./internal/types/ -run "TestType_List|TestType_Map"
```

Expected: FAIL

- [ ] **Step 3: Add List and Map to `types.go`** (append at end):

```go
// List is a homogeneous list type: list[T].
type List struct {
    Elem Type
}

func (l List) String() string {
    return "list[" + l.Elem.String() + "]"
}
func (l List) Equal(other Type) bool {
    o, ok := other.(List)
    return ok && Equal(l.Elem, o.Elem)
}
func (l List) typeMarker() {}

// Map is a key-value map type: map[K, V].
type Map struct {
    Key   Type
    Value Type
}

func (m Map) String() string {
    return "map[" + m.Key.String() + ", " + m.Value.String() + "]"
}
func (m Map) Equal(other Type) bool {
    o, ok := other.(Map)
    return ok && Equal(m.Key, o.Key) && Equal(m.Value, o.Value)
}
func (m Map) typeMarker() {}
```

- [ ] **Step 4: Run test to verify it passes**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2/v2
go test ./internal/types/ -v
```

Expected: 7 tests PASS

- [ ] **Step 5: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2
git add v2/internal/types/
git commit -m "v2: add List and Map types"
```

---

## Task 2: Struct Type

**Files:**
- Modify: `v2/internal/types/types.go`
- Modify: `v2/internal/types/types_test.go`

- [ ] **Step 1: Append failing tests**:

```go
func TestType_Struct_String(t *testing.T) {
    s := Struct{
        Name: "User",
        Fields: map[string]Type{
            "name": Primitive("str"),
            "age":  Primitive("int"),
        },
    }
    out := s.String()
    assert.Contains(t, out, "User")
    assert.Contains(t, out, "name: str")
    assert.Contains(t, out, "age: int")
}

func TestType_Struct_Equal(t *testing.T) {
    a := Struct{
        Name: "User",
        Fields: map[string]Type{
            "name": Primitive("str"),
            "age":  Primitive("int"),
        },
    }
    b := Struct{
        Name: "User",
        Fields: map[string]Type{
            "name": Primitive("str"),
            "age":  Primitive("int"),
        },
    }
    c := Struct{
        Name: "User",
        Fields: map[string]Type{"name": Primitive("str")},
    }
    assert.True(t, a.Equal(b))
    assert.False(t, a.Equal(c))
}

func TestType_Struct_Field(t *testing.T) {
    s := Struct{
        Name: "User",
        Fields: map[string]Type{"name": Primitive("str")},
    }
    f, ok := s.Field("name")
    assert.True(t, ok)
    assert.Equal(t, Primitive("str"), f)
    _, ok = s.Field("missing")
    assert.False(t, ok)
}
```

- [ ] **Step 2: Run test to verify it fails**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2/v2
go test ./internal/types/ -run TestType_Struct
```

Expected: FAIL

- [ ] **Step 3: Add Struct to `types.go`**:

```go
// Struct is a user-defined struct type with named fields.
type Struct struct {
    Name   string
    Fields map[string]Type
}

func (s Struct) String() string {
    out := s.Name + ":\n"
    for k, v := range s.Fields {
        out += "    " + k + ": " + v.String() + "\n"
    }
    return out
}

func (s Struct) Equal(other Type) bool {
    o, ok := other.(Struct)
    if !ok || s.Name != o.Name || len(s.Fields) != len(o.Fields) {
        return false
    }
    for k, v := range s.Fields {
        ov, ok := o.Fields[k]
        if !ok || !Equal(v, ov) {
            return false
        }
    }
    return true
}

func (s Struct) typeMarker() {}

// Field looks up a field by name. Returns (nil, false) if not found.
func (s Struct) Field(name string) (Type, bool) {
    t, ok := s.Fields[name]
    return t, ok
}

func (s Struct) FieldNames() []string {
    out := make([]string, 0, len(s.Fields))
    for k := range s.Fields {
        out = append(out, k)
    }
    return out
}
```

- [ ] **Step 4: Run test to verify it passes**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2/v2
go test ./internal/types/ -v
```

Expected: 10 tests PASS

- [ ] **Step 5: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2
git add v2/internal/types/
git commit -m "v2: add Struct type"
```

---

## Task 3: Function Type

**Files:**
- Modify: `v2/internal/types/types.go`
- Modify: `v2/internal/types/types_test.go`

- [ ] **Step 1: Append failing tests**:

```go
func TestType_Func_String(t *testing.T) {
    f := Func{
        Params: []Type{Primitive("int"), Primitive("int")},
        Return: Primitive("int"),
    }
    assert.Equal(t, "(int, int) -> int", f.String())
}

func TestType_Func_StringNoParams(t *testing.T) {
    f := Func{Return: Primitive("str")}
    assert.Equal(t, "() -> str", f.String())
}

func TestType_Func_Equal(t *testing.T) {
    a := Func{Params: []Type{Primitive("int")}, Return: Primitive("str")}
    b := Func{Params: []Type{Primitive("int")}, Return: Primitive("str")}
    c := Func{Params: []Type{Primitive("str")}, Return: Primitive("str")}
    d := Func{Params: []Type{Primitive("int")}, Return: Primitive("int")}
    assert.True(t, a.Equal(b))
    assert.False(t, a.Equal(c))
    assert.False(t, a.Equal(d))
}

func TestType_Func_Arity(t *testing.T) {
    f := Func{Params: []Type{Primitive("int"), Primitive("str")}, Return: Primitive("bool")}
    assert.Equal(t, 2, f.Arity())
}
```

- [ ] **Step 2: Run test to verify it fails**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2/v2
go test ./internal/types/ -run TestType_Func
```

Expected: FAIL

- [ ] **Step 3: Add Func to `types.go`**:

```go
// Func is a function type: (params) -> return.
type Func struct {
    Params []Type
    Return Type
}

func (f Func) String() string {
    out := "("
    for i, p := range f.Params {
        if i > 0 {
            out += ", "
        }
        out += p.String()
    }
    out += ") -> " + f.Return.String()
    return out
}

func (f Func) Equal(other Type) bool {
    o, ok := other.(Func)
    if !ok || len(f.Params) != len(o.Params) {
        return false
    }
    for i := range f.Params {
        if !Equal(f.Params[i], o.Params[i]) {
            return false
        }
    }
    return Equal(f.Return, o.Return)
}

func (f Func) typeMarker() {}

// Arity returns the number of parameters.
func (f Func) Arity() int { return len(f.Params) }
```

- [ ] **Step 4: Run test to verify it passes**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2/v2
go test ./internal/types/ -v
```

Expected: 14 tests PASS

- [ ] **Step 5: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2
git add v2/internal/types/
git commit -m "v2: add Func type"
```

---

## Task 4: Result and Optional Types

**Files:**
- Modify: `v2/internal/types/types.go`
- Modify: `v2/internal/types/types_test.go`

- [ ] **Step 1: Append failing tests**:

```go
func TestType_Result_String(t *testing.T) {
    r := Result{Ok: Primitive("int"), Err: Primitive("str")}
    assert.Equal(t, "Result[int, str]", r.String())
}

func TestType_Result_Equal(t *testing.T) {
    a := Result{Ok: Primitive("int"), Err: Primitive("str")}
    b := Result{Ok: Primitive("int"), Err: Primitive("str")}
    c := Result{Ok: Primitive("str"), Err: Primitive("str")}
    assert.True(t, a.Equal(b))
    assert.False(t, a.Equal(c))
}

func TestType_Optional_String(t *testing.T) {
    o := Optional{Primitive("int")}
    assert.Equal(t, "int?", o.String())
}

func TestType_Optional_Equal(t *testing.T) {
    a := Optional{Primitive("int")}
    b := Optional{Primitive("int")}
    c := Optional{Primitive("str")}
    assert.True(t, a.Equal(b))
    assert.False(t, a.Equal(c))
}
```

- [ ] **Step 2: Run test to verify it fails**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2/v2
go test ./internal/types/ -run "TestType_Result|TestType_Optional"
```

Expected: FAIL

- [ ] **Step 3: Add Result and Optional to `types.go`**:

```go
// Result is a fallible operation result: Result[T, E].
type Result struct {
    Ok  Type
    Err Type
}

func (r Result) String() string {
    return "Result[" + r.Ok.String() + ", " + r.Err.String() + "]"
}

func (r Result) Equal(other Type) bool {
    o, ok := other.(Result)
    return ok && Equal(r.Ok, o.Ok) && Equal(r.Err, o.Err)
}

func (r Result) typeMarker() {}

// Optional is a nullable type: T?.
type Optional struct {
    Inner Type
}

func (o Optional) String() string {
    return o.Inner.String() + "?"
}

func (o Optional) Equal(other Type) bool {
    inner, ok := other.(Optional)
    return ok && Equal(o.Inner, inner.Inner)
}

func (o Optional) typeMarker() {}
```

- [ ] **Step 4: Run test to verify it passes**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2/v2
go test ./internal/types/ -v
```

Expected: 18 tests PASS

- [ ] **Step 5: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2
git add v2/internal/types/
git commit -m "v2: add Result and Optional types"
```

---

## Task 5: Type Errors (E2xxx)

**Files:**
- Create: `v2/internal/types/errors.go`
- Create: `v2/internal/types/errors_test.go`

- [ ] **Step 1: Write failing test** `errors_test.go`:

```go
package types

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/jiejie-dev/funny/v2/internal/ast"
)

func TestError_Format(t *testing.T) {
    e := &Error{
        Code:    "E2010",
        Message: "type mismatch",
        Pos:     ast.Pos{Line: 3, Col: 5},
        Expected: Primitive("int"),
        Actual:   Primitive("str"),
    }
    s := e.Format()
    assert.Contains(t, s, "E2010")
    assert.Contains(t, s, "type mismatch")
    assert.Contains(t, s, "int")
    assert.Contains(t, s, "str")
}

func TestError_Error(t *testing.T) {
    e := &Error{Code: "E2001", Message: "undefined variable: x", Pos: ast.Pos{}}
    assert.Contains(t, e.Error(), "undefined variable")
}
```

- [ ] **Step 2: Run test to verify it fails**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2/v2
go test ./internal/types/ -run TestError
```

Expected: FAIL

- [ ] **Step 3: Write `errors.go`**:

```go
package types

import "fmt"
import "github.com/jiejie-dev/funny/v2/internal/ast"

// Error is a type-checking error.
type Error struct {
    Code     string
    Message  string
    Pos      ast.Pos
    Expected Type
    Actual   Type
    Hint     string
}

// New creates a new Error with the given code and message.
func New(code, msg string, pos ast.Pos) *Error {
    return &Error{Code: code, Message: msg, Pos: pos}
}

// NewMismatch creates a type-mismatch error with both types annotated.
func NewMismatch(pos ast.Pos, expected, actual Type) *Error {
    return &Error{
        Code:     "E2010",
        Message:  fmt.Sprintf("type mismatch: expected %s, got %s", expected, actual),
        Pos:      pos,
        Expected: expected,
        Actual:   actual,
        Hint:     fmt.Sprintf("expected %s here", expected),
    }
}

// Error implements the error interface.
func (e *Error) Error() string { return e.Format() }

// Format produces the unified error format:
//   error[E2010]: type mismatch: expected int, got str
//    --> <file>:<line>:<col>
//   help: expected int here
func (e *Error) Format() string {
    s := fmt.Sprintf("error[%s]: %s\n --> %s:%d:%d\n",
        e.Code, e.Message, e.Pos.File, e.Pos.Line+1, e.Pos.Col+1)
    if e.Hint != "" {
        s += fmt.Sprintf("\nhelp: %s", e.Hint)
    }
    return s
}
```

- [ ] **Step 4: Run test to verify it passes**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2/v2
go test ./internal/types/ -v
```

Expected: 20 tests PASS

- [ ] **Step 5: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2
git add v2/internal/types/errors.go v2/internal/types/errors_test.go
git commit -m "v2: type error type with E2010 mismatch support"
```

---

## Task 6: Type Environment

**Files:**
- Create: `v2/internal/types/env.go`
- Create: `v2/internal/types/env_test.go`

- [ ] **Step 1: Write failing tests** `env_test.go`:

```go
package types

import (
    "testing"

    "github.com/stretchr/testify/assert"
)

func TestEnv_DeclareLookup(t *testing.T) {
    e := NewEnv(nil)
    e.DeclareVar("x", Primitive("int"))
    t, ok := e.LookupVar("x")
    assert.True(t, ok)
    assert.Equal(t, Primitive("int"), t)
}

func TestEnv_NestedLookup(t *testing.T) {
    outer := NewEnv(nil)
    outer.DeclareVar("a", Primitive("int"))
    inner := NewEnv(outer)
    inner.DeclareVar("b", Primitive("str"))
    t, _ := inner.LookupVar("a")
    assert.Equal(t, Primitive("int"), t)
    t, _ = inner.LookupVar("b")
    assert.Equal(t, Primitive("str"), t)
}

func TestEnv_FuncLookup(t *testing.T) {
    e := NewEnv(nil)
    f := Func{Params: []Type{Primitive("int")}, Return: Primitive("int")}
    e.DeclareFunc("add", f)
    got, ok := e.LookupFunc("add")
    assert.True(t, ok)
    assert.True(t, got.Equal(f))
}

func TestEnv_StructLookup(t *testing.T) {
    e := NewEnv(nil)
    s := Struct{Name: "User", Fields: map[string]Type{"name": Primitive("str")}}
    e.DeclareStruct("User", s)
    got, ok := e.LookupStruct("User")
    assert.True(t, ok)
    assert.True(t, got.Equal(s))
}

func TestEnv_Shadowing(t *testing.T) {
    outer := NewEnv(nil)
    outer.DeclareVar("x", Primitive("int"))
    inner := NewEnv(outer)
    inner.DeclareVar("x", Primitive("str"))
    t, _ := inner.LookupVar("x")
    assert.Equal(t, Primitive("str"), t)
}
```

- [ ] **Step 2: Run test to verify it fails**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2/v2
go test ./internal/types/ -run TestEnv
```

Expected: FAIL

- [ ] **Step 3: Write `env.go`**:

```go
package types

// Env is a type environment that tracks variables, functions, and structs.
type Env struct {
    parent  *Env
    vars    map[string]Type
    funcs   map[string]Func
    structs map[string]Struct
}

// NewEnv creates a new Env, optionally nested inside parent.
func NewEnv(parent *Env) *Env {
    return &Env{
        parent:  parent,
        vars:    map[string]Type{},
        funcs:   map[string]Func{},
        structs: map[string]Struct{},
    }
}

// DeclareVar defines a variable in this scope (no parent traversal).
func (e *Env) DeclareVar(name string, t Type) {
    e.vars[name] = t
}

// LookupVar finds a variable, walking up parent scopes.
func (e *Env) LookupVar(name string) (Type, bool) {
    if t, ok := e.vars[name]; ok {
        return t, true
    }
    if e.parent != nil {
        return e.parent.LookupVar(name)
    }
    return nil, false
}

// DeclareFunc registers a function in this scope.
func (e *Env) DeclareFunc(name string, f Func) {
    e.funcs[name] = f
}

// LookupFunc finds a function by name.
func (e *Env) LookupFunc(name string) (Func, bool) {
    if f, ok := e.funcs[name]; ok {
        return f, true
    }
    if e.parent != nil {
        return e.parent.LookupFunc(name)
    }
    return Func{}, false
}

// DeclareStruct registers a struct type in this scope.
func (e *Env) DeclareStruct(name string, s Struct) {
    e.structs[name] = s
}

// LookupStruct finds a struct type by name.
func (e *Env) LookupStruct(name string) (Struct, bool) {
    if s, ok := e.structs[name]; ok {
        return s, true
    }
    if e.parent != nil {
        return e.parent.LookupStruct(name)
    }
    return Struct{}, false
}

// Has reports whether any binding with this name exists in this scope chain.
func (e *Env) Has(name string) bool {
    if _, ok := e.vars[name]; ok {
        return true
    }
    if _, ok := e.funcs[name]; ok {
        return true
    }
    if _, ok := e.structs[name]; ok {
        return true
    }
    if e.parent != nil {
        return e.parent.Has(name)
    }
    return false
}
```

- [ ] **Step 4: Run test to verify it passes**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2/v2
go test ./internal/types/ -v
```

Expected: 25 tests PASS

- [ ] **Step 5: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2
git add v2/internal/types/env.go v2/internal/types/env_test.go
git commit -m "v2: type environment with var/func/struct scopes"
```

---

## Task 7: Type Annotation Parser

**Files:**
- Create: `v2/internal/types/parse.go`
- Create: `v2/internal/types/parse_test.go`

- [ ] **Step 1: Write failing tests** `parse_test.go`:

```go
package types

import (
    "testing"

    "github.com/stretchr/testify/assert"
)

func TestParseType_Primitive(t *testing.T) {
    tests := []struct {
        in   string
        want Type
    }{
        {"int", Primitive("int")},
        {"str", Primitive("str")},
        {"bool", Primitive("bool")},
        {"float", Primitive("float")},
    }
    for _, c := range tests {
        got, err := ParseType(c.in)
        assert.NoError(t, err, c.in)
        assert.True(t, got.Equal(c.want), c.in)
    }
}

func TestParseType_List(t *testing.T) {
    got, err := ParseType("list[int]")
    assert.NoError(t, err)
    want := List{Primitive("int")}
    assert.True(t, got.Equal(want))
}

func TestParseType_Map(t *testing.T) {
    got, err := ParseType("map[str, int]")
    assert.NoError(t, err)
    want := Map{Primitive("str"), Primitive("int")}
    assert.True(t, got.Equal(want))
}

func TestParseType_Optional(t *testing.T) {
    got, err := ParseType("int?")
    assert.NoError(t, err)
    want := Optional{Primitive("int")}
    assert.True(t, got.Equal(want))
}

func TestParseType_Result(t *testing.T) {
    got, err := ParseType("Result[int, str]")
    assert.NoError(t, err)
    want := Result{Ok: Primitive("int"), Err: Primitive("str")}
    assert.True(t, got.Equal(want))
}

func TestParseType_Nested(t *testing.T) {
    got, err := ParseType("list[map[str, int?]]")
    assert.NoError(t, err)
    want := List{Map{Primitive("str"), Optional{Primitive("int")}}}
    assert.True(t, got.Equal(want))
}

func TestParseType_Func(t *testing.T) {
    got, err := ParseType("(int, str) -> bool")
    assert.NoError(t, err)
    want := Func{Params: []Type{Primitive("int"), Primitive("str")}, Return: Primitive("bool")}
    assert.True(t, got.Equal(want))
}

func TestParseType_Invalid(t *testing.T) {
    _, err := ParseType("list[")
    assert.Error(t, err)
}
```

- [ ] **Step 2: Run test to verify it fails**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2/v2
go test ./internal/types/ -run TestParseType
```

Expected: FAIL

- [ ] **Step 3: Write `parse.go`** (recursive-descent parser for type annotation strings):

```go
package types

import (
    "fmt"
    "strings"
    "unicode"
)

// ParseType parses a type annotation string into a Type.
// Grammar:
//   type      := primary
//   primary   := 'list' '[' type ']'
//              | 'map' '[' type ',' type ']'
//              | 'Result' '[' type ',' type ']'
//              | func-param-list '->' type
//              | IDENT ('.' IDENT)* '?'
//   func-param-list := '(' (type (',' type)*)? ')'
func ParseType(src string) (Type, error) {
    p := &typeParser{src: src}
    p.next()
    t, err := p.parseType()
    if err != nil {
        return nil, err
    }
    p.skipSpace()
    if p.pos < len(p.src) {
        return nil, fmt.Errorf("unexpected trailing characters at position %d in %q", p.pos, p.src)
    }
    return t, nil
}

type typeParser struct {
    src string
    pos int
}

func (p *typeParser) peek() byte {
    if p.pos >= len(p.src) {
        return 0
    }
    return p.src[p.pos]
}

func (p *typeParser) next() {
    if p.pos < len(p.src) {
        p.pos++
    }
}

func (p *typeParser) skipSpace() {
    for p.pos < len(p.src) && unicode.IsSpace(rune(p.src[p.pos])) {
        p.pos++
    }
}

func (p *typeParser) readIdent() string {
    start := p.pos
    for p.pos < len(p.src) && (isIdentByte(p.src[p.pos]) || p.src[p.pos] == '.') {
        p.pos++
    }
    return p.src[start:p.pos]
}

func isIdentByte(b byte) bool {
    return b == '_' || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

func (p *typeParser) expect(ch byte) error {
    p.skipSpace()
    if p.pos >= len(p.src) || p.src[p.pos] != ch {
        return fmt.Errorf("expected %q at position %d in %q, got %q", ch, p.pos, p.src, string(p.peek()))
    }
    p.pos++
    return nil
}

func (p *typeParser) parseType() (Type, error) {
    p.skipSpace()
    if p.pos >= len(p.src) {
        return nil, fmt.Errorf("unexpected end of type annotation")
    }
    ch := p.src[p.pos]

    switch {
    case ch == '(':
        return p.parseFuncType()
    case strings.HasPrefix(p.src[p.pos:], "list["):
        return p.parseListType()
    case strings.HasPrefix(p.src[p.pos:], "map["):
        return p.parseMapType()
    case strings.HasPrefix(p.src[p.pos:], "Result["):
        return p.parseResultType()
    }

    return p.parseNamedType()
}

func (p *typeParser) parseNamedType() (Type, error) {
    p.skipSpace()
    ident := p.readIdent()
    if ident == "" {
        return nil, fmt.Errorf("expected type name at position %d", p.pos)
    }
    if strings.Contains(ident, ".") {
        return nil, fmt.Errorf("qualified type names not supported: %s", ident)
    }
    base := Primitive(ident)
    // Optional suffix
    p.skipSpace()
    if p.pos < len(p.src) && p.src[p.pos] == '?' {
        p.pos++
        return Optional{Inner: base}, nil
    }
    return base, nil
}

func (p *typeParser) parseListType() (Type, error) {
    p.pos += len("list[")
    elem, err := p.parseType()
    if err != nil {
        return nil, err
    }
    if err := p.expect(']'); err != nil {
        return nil, fmt.Errorf("malformed list type: %w", err)
    }
    return List{Elem: elem}, nil
}

func (p *typeParser) parseMapType() (Type, error) {
    p.pos += len("map[")
    key, err := p.parseType()
    if err != nil {
        return nil, err
    }
    if err := p.expect(','); err != nil {
        return nil, fmt.Errorf("malformed map type (expected ','): %w", err)
    }
    val, err := p.parseType()
    if err != nil {
        return nil, err
    }
    if err := p.expect(']'); err != nil {
        return nil, fmt.Errorf("malformed map type: %w", err)
    }
    return Map{Key: key, Value: val}, nil
}

func (p *typeParser) parseResultType() (Type, error) {
    p.pos += len("Result[")
    ok, err := p.parseType()
    if err != nil {
        return nil, err
    }
    if err := p.expect(','); err != nil {
        return nil, fmt.Errorf("malformed Result type (expected ','): %w", err)
    }
    errT, err := p.parseType()
    if err != nil {
        return nil, err
    }
    if err := p.expect(']'); err != nil {
        return nil, fmt.Errorf("malformed Result type: %w", err)
    }
    return Result{Ok: ok, Err: errT}, nil
}

func (p *typeParser) parseFuncType() (Type, error) {
    if err := p.expect('('); err != nil {
        return nil, err
    }
    p.skipSpace()
    var params []Type
    if p.pos < len(p.src) && p.src[p.pos] != ')' {
        for {
            t, err := p.parseType()
            if err != nil {
                return nil, err
            }
            params = append(params, t)
            p.skipSpace()
            if p.pos < len(p.src) && p.src[p.pos] == ',' {
                p.pos++
                continue
            }
            break
        }
    }
    if err := p.expect(')'); err != nil {
        return nil, fmt.Errorf("malformed func type: %w", err)
    }
    if err := p.expect('-'); err != nil {
        return nil, err
    }
    if p.pos >= len(p.src) || p.src[p.pos] != '>' {
        return nil, fmt.Errorf("expected '->' in func type")
    }
    p.pos++
    ret, err := p.parseType()
    if err != nil {
        return nil, err
    }
    return Func{Params: params, Return: ret}, nil
}
```

- [ ] **Step 4: Run test to verify it passes**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2/v2
go test ./internal/types/ -v
```

Expected: 32 tests PASS

- [ ] **Step 5: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2
git add v2/internal/types/parse.go v2/internal/types/parse_test.go
git commit -m "v2: recursive-descent parser for type annotations"
```

---

## Task 8: Type Checker - Expression Primitives

**Files:**
- Create: `v2/internal/types/check.go`
- Create: `v2/internal/types/check_test.go`

- [ ] **Step 1: Write failing tests** `check_test.go` (start with expression primitives only):

```go
package types

import (
    "testing"

    "github.com/jiejie-dev/funny/v2/internal/ast"
    "github.com/jiejie-dev/funny/v2/internal/parser"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func parseExpr(t *testing.T, src string) ast.Expression {
    t.Helper()
    p := parser.New(src, "")
    prog, err := p.Parse()
    require.NoError(t, err)
    return prog.Stmts[0].(*ast.ExprStmt).X
}

func TestCheck_Literal(t *testing.T) {
    cases := []struct {
        src  string
        want Type
    }{
        {"42", Primitive("int")},
        {"3.14", Primitive("float")},
        {`"hi"`, Primitive("str")},
        {"true", Primitive("bool")},
        {"nil", Primitive("nil")},
    }
    for _, c := range cases {
        env := NewEnv(nil)
        got, err := CheckExpr(parseExpr(t, c.src), env)
        assert.NoError(t, err, c.src)
        assert.True(t, got.Equal(c.want), "%s: got %s want %s", c.src, got, c.want)
    }
}

func TestCheck_Variable(t *testing.T) {
    env := NewEnv(nil)
    env.DeclareVar("x", Primitive("int"))
    got, err := CheckExpr(parseExpr(t, "x"), env)
    assert.NoError(t, err)
    assert.Equal(t, Primitive("int"), got)
}

func TestCheck_UndefinedVariable(t *testing.T) {
    env := NewEnv(nil)
    _, err := CheckExpr(parseExpr(t, "undefined_xyz"), env)
    assert.Error(t, err)
}
```

- [ ] **Step 2: Run test to verify it fails**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2/v2
go test ./internal/types/ -run "TestCheck_Literal|TestCheck_Variable|TestCheck_UndefinedVariable"
```

Expected: FAIL

- [ ] **Step 3: Write initial `check.go`** (expression primitives only):

```go
package types

import (
    "fmt"

    "github.com/jiejie-dev/funny/v2/internal/ast"
)

// CheckExpr type-checks an expression and returns its type.
func CheckExpr(expr ast.Expression, env *Env) (Type, error) {
    switch n := expr.(type) {
    case *ast.LiteralExpr:
        return literalType(n.Value), nil
    case *ast.VariableExpr:
        t, ok := env.LookupVar(n.Name)
        if !ok {
            return nil, New("E2001", fmt.Sprintf("undefined variable: %s", n.Name), n.NodePos)
        }
        return t, nil
    case *ast.BinaryExpr:
        return checkBinaryExpr(n, env)
    case *ast.UnaryExpr:
        return checkUnaryExpr(n, env)
    case *ast.CallExpr:
        return checkCallExpr(n, env)
    case *ast.IndexExpr:
        return checkIndexExpr(n, env)
    case *ast.FieldExpr:
        return checkFieldExpr(n, env)
    case *ast.ListExpr:
        return checkListLiteral(n, env)
    case *ast.SubExpr:
        return CheckExpr(n.Inner, env)
    case *ast.FStringExpr:
        return Primitive("str"), nil
    }
    return nil, New("E2099", fmt.Sprintf("type checker: unsupported expression %T", expr), expr.Pos())
}

// literalType infers a Type from a Go value (after evaluation semantics).
func literalType(v any) Type {
    switch v.(type) {
    case int:
        return Primitive("int")
    case float64:
        return Primitive("float")
    case string:
        return Primitive("str")
    case bool:
        return Primitive("bool")
    case nil:
        return Primitive("nil")
    }
    return Primitive("unknown")
}

func checkBinaryExpr(n *ast.BinaryExpr, env *Env) (Type, error) {
    leftT, err := CheckExpr(n.Left, env)
    if err != nil {
        return nil, err
    }
    rightT, err := CheckExpr(n.Right, env)
    if err != nil {
        return nil, err
    }
    switch n.Op {
    case "+", "-", "*", "/", "%":
        if !Equal(leftT, rightT) {
            return nil, NewMismatch(n.NodePos, leftT, rightT)
        }
        return leftT, nil
    case "==", "!=", "<", ">", "<=", ">=":
        if !Equal(leftT, rightT) {
            return nil, NewMismatch(n.NodePos, leftT, rightT)
        }
        return Primitive("bool"), nil
    case "and", "or":
        if !Equal(leftT, Primitive("bool")) || !Equal(rightT, Primitive("bool")) {
            return nil, NewMismatch(n.NodePos, Primitive("bool"), leftT)
        }
        return Primitive("bool"), nil
    case "in":
        // Element on left, list/map on right
        rightList, ok := rightT.(List)
        if !ok {
            return nil, New("E2050", fmt.Sprintf("'in' requires list on right side, got %s", rightT), n.NodePos)
        }
        if !Equal(leftT, rightList.Elem) {
            return nil, NewMismatch(n.NodePos, rightList.Elem, leftT)
        }
        return Primitive("bool"), nil
    }
    return nil, New("E2098", fmt.Sprintf("unsupported binary operator: %s", n.Op), n.NodePos)
}

func checkUnaryExpr(n *ast.UnaryExpr, env *Env) (Type, error) {
    inner, err := CheckExpr(n.Expr, env)
    if err != nil {
        return nil, err
    }
    switch n.Op {
    case "-":
        if !Equal(inner, Primitive("int")) && !Equal(inner, Primitive("float")) {
            return nil, NewMismatch(n.NodePos, Primitive("int"), inner)
        }
        return inner, nil
    case "not":
        if !Equal(inner, Primitive("bool")) {
            return nil, NewMismatch(n.NodePos, Primitive("bool"), inner)
        }
        return Primitive("bool"), nil
    }
    return nil, New("E2098", fmt.Sprintf("unsupported unary operator: %s", n.Op), n.NodePos)
}

func checkCallExpr(n *ast.CallExpr, env *Env) (Type, error) {
    varName, ok := n.Func.(*ast.VariableExpr)
    if !ok {
        return nil, New("E2070", "only direct function calls supported in M2-A", n.NodePos)
    }
    fn, ok := env.LookupFunc(varName.Name)
    if !ok {
        return nil, New("E2002", fmt.Sprintf("undefined function: %s", varName.Name), n.NodePos)
    }
    if len(n.Args) != fn.Arity() {
        return nil, New("E2020",
            fmt.Sprintf("%s expects %d args, got %d", varName.Name, fn.Arity(), len(n.Args)),
            n.NodePos)
    }
    for i, arg := range n.Args {
        argT, err := CheckExpr(arg, env)
        if err != nil {
            return nil, err
        }
        if !Equal(argT, fn.Params[i]) {
            return nil, NewMismatch(n.NodePos, fn.Params[i], argT)
        }
    }
    return fn.Return, nil
}

func checkIndexExpr(n *ast.IndexExpr, env *Env) (Type, error) {
    objT, err := CheckExpr(n.Object, env)
    if err != nil {
        return nil, err
    }
    idxT, err := CheckExpr(n.Index, env)
    if err != nil {
        return nil, err
    }
    if !Equal(idxT, Primitive("int")) {
        return nil, NewMismatch(n.NodePos, Primitive("int"), idxT)
    }
    switch t := objT.(type) {
    case List:
        return t.Elem, nil
    case Map:
        return t.Value, nil
    }
    return nil, New("E2050", fmt.Sprintf("cannot index into %s", objT), n.NodePos)
}

func checkFieldExpr(n *ast.FieldExpr, env *Env) (Type, error) {
    objT, err := CheckExpr(n.Object, env)
    if err != nil {
        return nil, err
    }
    s, ok := objT.(Struct)
    if !ok {
        return nil, New("E2051", fmt.Sprintf("field access requires struct, got %s", objT), n.NodePos)
    }
    f, ok := s.Field(n.Field)
    if !ok {
        return nil, New("E2052", fmt.Sprintf("struct %s has no field %q", s.Name, n.Field), n.NodePos)
    }
    return f, nil
}

func checkListLiteral(n *ast.ListExpr, env *Env) (Type, error) {
    if len(n.Elements) == 0 {
        return nil, New("E2011", "cannot infer type of empty list; add type annotation", n.NodePos)
    }
    first, err := CheckExpr(n.Elements[0], env)
    if err != nil {
        return nil, err
    }
    for i := 1; i < len(n.Elements); i++ {
        t, err := CheckExpr(n.Elements[i], env)
        if err != nil {
            return nil, err
        }
        if !Equal(t, first) {
            return nil, NewMismatch(n.Elements[i].Pos(), first, t)
        }
    }
    return List{Elem: first}, nil
}
```

- [ ] **Step 4: Run test to verify it passes**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2/v2
go test ./internal/types/ -v
```

Expected: 35 tests PASS

- [ ] **Step 5: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2
git add v2/internal/types/check.go v2/internal/types/check_test.go
git commit -m "v2: type checker for expressions (literals, vars, binary, call, index, field)"
```

---

## Task 9: Type Checker - Statements

**Files:**
- Modify: `v2/internal/types/check.go`
- Modify: `v2/internal/types/check_test.go`

- [ ] **Step 1: Append failing tests**:

```go
func TestCheck_Let(t *testing.T) {
    src := `let x: int = 42`
    p := parser.New(src, "")
    prog, err := p.Parse()
    require.NoError(t, err)
    env := NewEnv(nil)
    err = Check(prog, env)
    assert.NoError(t, err)
    t2, _ := env.LookupVar("x")
    assert.Equal(t, Primitive("int"), t2)
}

func TestCheck_LetInfer(t *testing.T) {
    src := `let x = 42`
    p := parser.New(src, "")
    prog, err := p.Parse()
    require.NoError(t, err)
    env := NewEnv(nil)
    err = Check(prog, env)
    assert.NoError(t, err)
    t2, _ := env.LookupVar("x")
    assert.Equal(t, Primitive("int"), t2)
}

func TestCheck_LetTypeMismatch(t *testing.T) {
    src := `let x: int = "hello"`
    p := parser.New(src, "")
    prog, err := p.Parse()
    require.NoError(t, err)
    env := NewEnv(nil)
    err = Check(prog, env)
    assert.Error(t, err)
}

func TestCheck_Assign(t *testing.T) {
    src := `let x: int = 1
x = 2`
    p := parser.New(src, "")
    prog, err := p.Parse()
    require.NoError(t, err)
    env := NewEnv(nil)
    err = Check(prog, env)
    assert.NoError(t, err)
}

func TestCheck_AssignMismatch(t *testing.T) {
    src := `let x: int = 1
x = "hello"`
    p := parser.New(src, "")
    prog, err := p.Parse()
    require.NoError(t, err)
    env := NewEnv(nil)
    err = Check(prog, env)
    assert.Error(t, err)
}

func TestCheck_If_NonBool(t *testing.T) {
    src := `if 42:
    pass`
    p := parser.New(src, "")
    prog, err := p.Parse()
    require.NoError(t, err)
    env := NewEnv(nil)
    err = Check(prog, env)
    assert.Error(t, err)
}

func TestCheck_For_NonList(t *testing.T) {
    src := `for i in 42:
    pass`
    p := parser.New(src, "")
    prog, err := p.Parse()
    require.NoError(t, err)
    env := NewEnv(nil)
    err = Check(prog, env)
    assert.Error(t, err)
}

func TestCheck_ReturnType(t *testing.T) {
    src := `fn foo() -> int:
    return "hello"
`
    p := parser.New(src, "")
    prog, err := p.Parse()
    require.NoError(t, err)
    env := NewEnv(nil)
    err = Check(prog, env)
    assert.Error(t, err)
}
```

- [ ] **Step 2: Run test to verify it fails**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2/v2
go test ./internal/types/ -run "TestCheck_Let|TestCheck_Assign|TestCheck_If|TestCheck_For|TestCheck_Return"
```

Expected: FAIL (Check function not defined)

- [ ] **Step 3: Add Check function and statement-level checkers** to `check.go`:

```go
// Check type-checks a full program.
func Check(prog *ast.Program, env *Env) error {
    for _, s := range prog.Stmts {
        if err := checkStmt(s, env); err != nil {
            return err
        }
    }
    return nil
}

func checkStmt(s ast.Statement, env *Env) error {
    switch n := s.(type) {
    case *ast.LetStmt:
        return checkLet(n, env)
    case *ast.AssignStmt:
        return checkAssign(n, env)
    case *ast.IfStmt:
        return checkIf(n, env)
    case *ast.ForStmt:
        return checkFor(n, env)
    case *ast.WhileStmt:
        return checkWhile(n, env)
    case *ast.ReturnStmt:
        return checkReturn(n, env)
    case *ast.FnDecl:
        return checkFnDecl(n, env)
    case *ast.StructDecl:
        return checkStructDecl(n, env)
    case *ast.ExprStmt, *ast.BreakStmt, *ast.ContinueStmt, *ast.MetaBlock, *ast.PlanBlock, *ast.ImportDecl:
        return nil // M2-A doesn't type-check these
    }
    return New("E2099", fmt.Sprintf("unsupported statement %T", s), s.Pos())
}

func checkLet(n *ast.LetStmt, env *Env) error {
    valT, err := CheckExpr(n.Value, env)
    if err != nil {
        return err
    }
    var declared Type
    if n.TypeAnn != "" {
        declared, err = ParseType(n.TypeAnn)
        if err != nil {
            return New("E2012", fmt.Sprintf("invalid type annotation %q: %v", n.TypeAnn, err), n.NodePos)
        }
        if !Equal(valT, declared) {
            return NewMismatch(n.NodePos, declared, valT)
        }
    } else {
        declared = valT
    }
    env.DeclareVar(n.Name, declared)
    return nil
}

func checkAssign(n *ast.AssignStmt, env *Env) error {
    valT, err := CheckExpr(n.Value, env)
    if err != nil {
        return err
    }
    targetT, err := CheckExpr(n.Target, env)
    if err != nil {
        return err
    }
    if !Equal(valT, targetT) {
        return NewMismatch(n.NodePos, targetT, valT)
    }
    return nil
}

func checkIf(n *ast.IfStmt, env *Env) error {
    condT, err := CheckExpr(n.Cond, env)
    if err != nil {
        return err
    }
    if !Equal(condT, Primitive("bool")) {
        return NewMismatch(n.NodePos, Primitive("bool"), condT)
    }
    if err := Check(n.Then.ToProgram(), env); err != nil {
        return err
    }
    if n.ElseIf != nil {
        return checkIf(n.ElseIf, env)
    }
    if n.ElseBlock != nil {
        return Check(n.ElseBlock.ToProgram(), env)
    }
    return nil
}

func checkFor(n *ast.ForStmt, env *Env) error {
    iterT, err := CheckExpr(n.Iterable, env)
    if err != nil {
        return err
    }
    listT, ok := iterT.(List)
    if !ok {
        return New("E2050", fmt.Sprintf("for-in requires list, got %s", iterT), n.NodePos)
    }
    bodyEnv := NewEnv(env)
    bodyEnv.DeclareVar(n.Name, listT.Elem)
    return Check(n.Body.ToProgram(), bodyEnv)
}

func checkWhile(n *ast.WhileStmt, env *Env) error {
    condT, err := CheckExpr(n.Cond, env)
    if err != nil {
        return err
    }
    if !Equal(condT, Primitive("bool")) {
        return NewMismatch(n.NodePos, Primitive("bool"), condT)
    }
    return Check(n.Body.ToProgram(), env)
}

func checkReturn(n *ast.ReturnStmt, env *Env) error {
    if n.Value == nil {
        return nil
    }
    valT, err := CheckExpr(n.Value, env)
    if err != nil {
        return err
    }
    retT, ok := env.LookupVar("__return_type__")
    if !ok {
        return nil
    }
    if !Equal(valT, retT.(Type)) {
        return NewMismatch(n.NodePos, retT.(Type), valT)
    }
    return nil
}

func checkFnDecl(n *ast.FnDecl, env *Env) error {
    var retType Type = Primitive("nil")
    if n.RetType != "" {
        var err error
        retType, err = ParseType(n.RetType)
        if err != nil {
            return New("E2012", fmt.Sprintf("invalid return type %q: %v", n.RetType, err), n.NodePos)
        }
    }
    var paramTypes []Type
    for _, p := range n.Params {
        if p.TypeAnn == "" {
            return New("E2013", fmt.Sprintf("parameter %q missing type annotation", p.Name), n.NodePos)
        }
        pt, err := ParseType(p.TypeAnn)
        if err != nil {
            return New("E2012", fmt.Sprintf("invalid type for parameter %q: %v", p.Name, err), n.NodePos)
        }
        paramTypes = append(paramTypes, pt)
    }
    env.DeclareFunc(n.Name, Func{Params: paramTypes, Return: retType})
    bodyEnv := NewEnv(env)
    bodyEnv.DeclareVar("__return_type__", retType)
    for i, p := range n.Params {
        bodyEnv.DeclareVar(p.Name, paramTypes[i])
    }
    return Check(n.Body.ToProgram(), bodyEnv)
}

func checkStructDecl(n *ast.StructDecl, env *Env) error {
    fields := map[string]Type{}
    for _, f := range n.Fields {
        if f.TypeAnn == "" {
            return New("E2013", fmt.Sprintf("struct field %q missing type annotation", f.Name), n.NodePos)
        }
        ft, err := ParseType(f.TypeAnn)
        if err != nil {
            return New("E2012", fmt.Sprintf("invalid type for field %q: %v", f.Name, err), n.NodePos)
        }
        fields[f.Name] = ft
    }
    env.DeclareStruct(n.Name, Struct{Name: n.Name, Fields: fields})
    return nil
}
```

- [ ] **Step 4: Add helper method** to `v2/internal/ast/ast.go` (modify the file):

Find the Block type definition and add a `ToProgram()` method after it:

```go
// ToProgram wraps a Block in a Program (used by type checker).
func (b *Block) ToProgram() *Program {
    return &Program{NodePos: b.NodePos, Stmts: b.Statements}
}
```

(Add this right after the existing `Block` String method.)

- [ ] **Step 5: Run test to verify it passes**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2/v2
go test ./internal/types/ -v
```

Expected: 43 tests PASS

- [ ] **Step 6: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2
git add v2/internal/types/check.go v2/internal/types/check_test.go v2/internal/ast/ast.go
git commit -m "v2: type checker for statements (let/assign/if/for/while/return/fn/struct)"
```

---

## Task 10: Integration Test Data Files

**Files:**
- Create: `v2/testdata/types/basic.fn`
- Create: `v2/testdata/types/lists.fn`
- Create: `v2/testdata/types/structs.fn`
- Create: `v2/testdata/types/functions.fn`

- [ ] **Step 1: Create `v2/testdata/types/basic.fn`**:

```
let x: int = 42
let y: float = 3.14
let name: str = "hello"
let flag: bool = true

let sum = x + 0
let neg = -x
```

- [ ] **Step 2: Create `v2/testdata/types/lists.fn`**:

```
let xs: list[int] = [1, 2, 3]
let first = xs[0]
let ys = [1.0, 2.0, 3.0]
```

- [ ] **Step 3: Create `v2/testdata/types/structs.fn`**:

```
struct Point:
    x: int
    y: int

struct User:
    name: str
    age: int

let p = Point(x: 1, y: 2)
let px = p.x
let u = User(name: "alice", age: 30)
```

- [ ] **Step 4: Create `v2/testdata/types/functions.fn`**:

```
fn add(a: int, b: int) -> int:
    return a + b

fn greet(name: str) -> str:
    return "hello " + name

let r = add(1, 2)
let s = greet("world")
```

- [ ] **Step 5: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2
git add v2/testdata/types/
git commit -m "v2: type system integration test data"
```

---

## Task 11: Integration Tests for Type Checker

**Files:**
- Create: `v2/testdata/types/errors_test.go`
- Modify: `v2/testdata/types/integration_test.go`

Actually create tests at `v2/internal/types/integration_test.go` for type checking against the .fn files.

- [ ] **Step 1: Append integration tests** to `v2/internal/types/check_test.go`:

```go
func TestIntegration_TypeCheck_Basic(t *testing.T) {
    src := `let x: int = 42
let y: float = 3.14
let name: str = "hello"
let sum = x + 0
`
    p := parser.New(src, "")
    prog, err := p.Parse()
    require.NoError(t, err)
    env := NewEnv(nil)
    err = Check(prog, env)
    assert.NoError(t, err)
}

func TestIntegration_TypeCheck_Lists(t *testing.T) {
    src := `let xs: list[int] = [1, 2, 3]
let first = xs[0]
`
    p := parser.New(src, "")
    prog, err := p.Parse()
    require.NoError(t, err)
    env := NewEnv(nil)
    err = Check(prog, env)
    assert.NoError(t, err)
}

func TestIntegration_TypeCheck_Structs(t *testing.T) {
    src := `struct Point:
    x: int
    y: int

let p = Point(x: 1, y: 2)
let px = p.x
`
    p := parser.New(src, "")
    prog, err := p.Parse()
    require.NoError(t, err)
    env := NewEnv(nil)
    err = Check(prog, env)
    assert.NoError(t, err)
}

func TestIntegration_TypeCheck_Functions(t *testing.T) {
    src := `fn add(a: int, b: int) -> int:
    return a + b

let r = add(1, 2)
let s = add("x", "y")
`
    p := parser.New(src, "")
    prog, err := p.Parse()
    require.NoError(t, err)
    env := NewEnv(nil)
    err = Check(prog, env)
    assert.Error(t, err) // second call passes str to int params
}

func TestIntegration_TypeCheck_VariousErrors(t *testing.T) {
    cases := []struct {
        name string
        src  string
    }{
        {"undefined var", "let y = undefined_xyz"},
        {"if non-bool", `if 42:
    pass`},
        {"for non-list", `for i in 42:
    pass`},
        {"return mismatch", `fn foo() -> int:
    return "hello"
`},
        {"assign mismatch", `let x: int = 1
x = "hello"`},
        {"param missing type", `fn foo(a) -> int:
    return a
`},
    }
    for _, c := range cases {
        p := parser.New(c.src, "")
        prog, err := p.Parse()
        require.NoError(t, err, c.name)
        env := NewEnv(nil)
        err = Check(prog, env)
        assert.Error(t, err, c.name+": expected type error")
    }
}
```

- [ ] **Step 2: Run test to verify all pass**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2/v2
go test ./internal/types/ -v
```

Expected: 49 tests PASS

- [ ] **Step 3: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2
git add v2/internal/types/check_test.go
git commit -m "v2: type checker integration tests"
```

---

## Task 12: CLI Integration (Type Check Step)

**Files:**
- Modify: `v2/internal/cli/run.go`
- Modify: `v2/internal/cli/run_test.go`

- [ ] **Step 1: Append failing test** to `run_test.go`:

```go
func TestRun_TypeCheckPasses(t *testing.T) {
    src := `let x: int = 42
let y: int = x + 1
`
    err := Run([]byte(src), "test.fn")
    assert.NoError(t, err)
}

func TestRun_TypeCheckFails(t *testing.T) {
    src := `let x: int = "hello"`
    err := Run([]byte(src), "test.fn")
    assert.Error(t, err)
}
```

- [ ] **Step 2: Run test to verify second test fails**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2/v2
go test ./internal/cli/ -run TestRun_TypeCheck
```

Expected: FAIL (type check not called)

- [ ] **Step 3: Modify `run.go`** to add type checking step:

```go
package cli

import (
    "encoding/json"

    "github.com/jiejie-dev/funny/v2/internal/evaluator"
    "github.com/jiejie-dev/funny/v2/internal/parser"
    "github.com/jiejie-dev/funny/v2/internal/types"
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
    e := evaluator.New(nil)
    return e.Exec(prog)
}

func Ast(src []byte, file string) ([]byte, error) {
    p := parser.New(string(src), file)
    prog, err := p.Parse()
    if err != nil {
        return nil, err
    }
    return json.MarshalIndent(prog, "", "  ")
}
```

- [ ] **Step 4: Run test to verify both tests pass**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2/v2
go test ./internal/cli/ -v
```

Expected: 4 tests PASS (2 original + 2 new)

- [ ] **Step 5: Verify full test suite still passes**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2/v2
go test ./...
```

Expected: all packages OK

- [ ] **Step 6: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2
git add v2/internal/cli/run.go v2/internal/cli/run_test.go
git commit -m "v2: CLI integrates type checker between parse and exec"
```

---

## Task 13: Update README for M2-A

**Files:**
- Modify: `v2/README.md`

- [ ] **Step 1: Update README status**:

Find the `**Status: M1 (Syntax Skeleton) — RELEASED**` line and update to:

```markdown
**Status: M2-A (Strong Typing Foundation) — RELEASED**

- ✅ Lexer (indent-sensitive, all operators, strings, f-strings)
- ✅ Parser (Pratt expressions, control flow, fn/struct/meta/plan)
- ✅ Tree-walking evaluator (no type checking at runtime — types are compile-time only)
- ✅ **Type system** (M2-A): primitive types, list, map, struct, func, Result, optional
- ✅ **Type checker**: validates expressions, statements, function calls, returns
- ✅ **Compile-time error reporting**: E2001-E2099 with unified format
- ⏳ Bytecode VM → M2-B
- ⏳ Result + `?` operator runtime → M2-C
- ⏳ stdlib (json/time/math/str) → M2-C
```

Update the Roadmap section:

```markdown
| v2.0.0-alpha (M1) | ✅ Done | Lexer + Parser + Evaluator (no types) |
| v2.0.0-beta (M2-A) | ✅ Done | Type system + type checker |
| v2.0.0-beta (M2-B) | Planned | Bytecode VM (5×+ perf) |
| v2.0.0-beta (M2-C) | Planned | Result + `?` + stdlib |
| v2.0.0-rc (M3) | Planned | meta/plan engine + LSP |
| v2.0.0 (M4) | Planned | MCP server + full stdlib |
```

- [ ] **Step 2: Commit**:

```bash
cd /Users/j/repos/funny/.worktrees/funny-v2-m2
git add v2/README.md
git commit -m "v2: README updated for M2-A type system"
```

---

## Self-Review

1. **Spec coverage**:
   - §2.2 Base types (int/float/str/bool/nil/list/map/struct/Result/optional) → Tasks 1, 2, 3, 5 ✓
   - §2.2 Function types (Task 4) ✓
   - §5.9 Error code system E2xxx → Task 5 ✓
   - §2.3 Strong typing: variable type annotations, parameter types, return types, type mismatch detection → Tasks 9, 11, 12 ✓
   - §2.3 Explicit type required for public symbols (function params + struct fields) → Tasks 9, 11 ✓
   - M2 §6.3 type checker base types, composite types, Result (types only — runtime in M2-C), function signatures, struct, type inference (basic — let x = 42 → int) → Tasks 8, 9 ✓
   - **Deferred**: Result + `?` runtime (M2-C), bytecode VM (M2-B), stdlib (M2-C)

2. **Placeholder scan**: No TBD/TODO in this plan.

3. **Type consistency**:
   - `Type` interface defined in Task 0 with `Equal`, `String`, `typeMarker`
   - All concrete types (`Primitive`, `List`, `Map`, `Struct`, `Func`, `Result`, `Optional`) implement all three
   - `Env.LookupVar/LookupFunc/LookupStruct` signatures consistent across tasks
   - `Check(prog, env)` entry point consistent in all task tests
   - `ast.Block.ToProgram()` helper added in Task 9, used in `checkIf`, `checkFor`, `checkWhile`

4. **Spec requirement with no task**: None — every M2-A scope item has a task.

---

## Exit Criteria for M2-A

The M2-A plan is complete when:
- [ ] All 13 tasks checked off
- [ ] `go test ./...` passes (≥ 49 type tests + existing 98 = ~147 total)
- [ ] `go build -o funny ./cmd/funny && ./funny run ./testdata/types/basic.fn` produces `0` (the let sum = x + 0 = 42 + 0 = 42)
- [ ] Type errors are caught at compile time (before evaluation)
- [ ] All commits in this plan's history
- [ ] M2-A released as `v2.0.0-beta-type-system`

---

## Total Tasks: 13

**Estimated time**: 4-6 days for one developer, parallelizable.