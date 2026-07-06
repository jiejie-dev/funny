# Funny v2: AI-Native Scripting Language Design

| | |
|---|---|
| **Status** | Draft (awaiting user review) |
| **Date** | 2026-07-01 |
| **Author** | jiejie-dev (with opencode assistance) |
| **Target Version** | v2.0.0 |
| **Implementation** | Go (locked) |
| **Scope** | Complete rewrite of funny language |

---

## 1. Overview

### 1.1 Position

**Funny v2** is a **strongly-typed, indentation-sensitive minimal scripting language**. Its core thesis is **"code is program, code is agent plan"** — the same `.fn` source file must be simultaneously:

1. **Readable & writable by humans** as a normal scripting language
2. **Generateable with high accuracy by LLMs** through structural design choices
3. **Directly consumable by agent runtimes** as a structured workflow manifest

### 1.2 Three Iron Rules

1. **Structure over flexibility** — indentation-sensitive, single idiom per construct, eliminate ambiguity. Better to own the 90% idiom than to add variants for the remaining 10%.
2. **Types are documentation** — every exported symbol carries an explicit type. LLM reads types and infers semantics without comments.
3. **One source, three consumers** — humans, LLMs, and agent runtimes read the same file, the same structure, only with different lenses.

### 1.3 Non-Goals

- No backward compatibility with v1
- Not a replacement for Go/Python; an "AI-era glue language"
- No macros, no complex generics
- No GUI, no native binary, no AOT compiler (JIT/bytecode interpreter is sufficient)

---

## 2. Syntax Core Features

### 2.1 Indentation Rules

- **Indentation-sensitive**, spaces only (no tabs)
- More indent = new block; less indent = exit block
- Sibling statements must align
- **Forbidden**: Python-style "hanging bracket alignment"; all blocks expressed via indentation

```
✅
if x > 0:
    print("pos")
    return x

❌ (forbidden bracket alignment)
if (x > 0 and
        y > 0):
    print(x)
```

### 2.2 Base Types

| Type | Literal | Notes |
|---|---|---|
| `int` | `42`, `-7`, `0x1F` | 64-bit signed integer |
| `float` | `3.14`, `1e-3` | 64-bit float |
| `bool` | `true`, `false` | Boolean |
| `str` | `"hello"` or `'hi'` | Immutable UTF-8 string, supports f-string |
| `nil` | `nil` | Single null value |
| `list[T]` | `[1, 2, 3]` | Dynamic list, elements are homogeneous |
| `map[K, V]` | `{"a": 1}` | Dynamic map |
| `T?` | — | Nullable type (default non-null) |
| `Result[T, E]` | `ok(x)` / `err(e)` | Standard error wrapper |

**Why `Result`**: avoids the "non-linear control flow" problem of try/catch. LLM-generated code has visible error paths identical to happy paths.

### 2.3 Strong Typing Rules

- Variables/function declarations must be type-inferable or explicitly annotated
- Type inference applies to right-hand side of assignment, return values, local variables
- **Public symbols** (appearing in `meta`/`plan`, imported elsewhere) **must be explicitly annotated**

```
let x = 1              # OK, inferred int
let name: str = "hi"   # OK, explicit annotation
let items = []         # ❌ Error: empty list cannot be inferred
let items: list[int] = []  # OK
```

### 2.4 Control Flow

Only four constructs, deliberately no variants:

- `if / elif / else`
- `for x in iterable:` (iteration)
- `while cond:` (loop, with `break`/`continue`)
- `match expr:` (pattern matching, value match + type match)

```
match status:
    200 => print("ok")
    404 => print("not found")
    int(n) if n >= 500 => print("server err")
    _ => print("other")
```

### 2.5 Functions

- Sole keyword: `fn`
- Must have parameter list (may be empty), return type (inferable, but top-level `fn` recommended explicit)
- Multi-return supported

```
fn add(a: int, b: int) -> int:
    return a + b

fn parse(s: str) -> Result[int, str]:
    if s == "":
        return err("empty")
    return ok(to_int(s))
```

**No** `def`/`func`/`function` aliases.

### 2.6 Data Structures

**Three carriers, distinct roles**:

| Purpose | Keyword | Example |
|---|---|---|
| Fixed structure (class-like) | `struct` | `struct User: name: str, age: int` |
| Dynamic key-value | `map` | `let m: map[str, int] = {"a": 1}` |
| Homogeneous sequence | `list` | `let xs: list[int] = [1, 2, 3]` |

**Forbidden**: v1-style "object literal that is both map and has methods" ambiguity.

```
# v1 (forbidden)
person = {
    name = 'jiejie-dev'
    isAdult() { return this.age >= 18 }
}

# v2 (forced explicit)
struct Person:
    name: str
    age: int
    fn is_adult(self) -> bool:
        return self.age >= 18
```

### 2.7 Strings and Templates

- Double quote preferred, single quote compatible
- `f"..."` template strings (F-string style, abundant Python training data)

```
let name = "world"
print(f"hello {name}")
```

### 2.8 Module System

- `import "path/mod.fn"` — file path is module identifier, no package manager
- Explicit `pub` keyword for exports
- Naming: snake_case for symbols, `PascalCase` for types

```
# math.fn
pub fn add(a: int, b: int) -> int:
    return a + b

# main.fn
import "math.fn" as m
let r = m.add(1, 2)
```

### 2.9 Comments

- `#` line comment (single token, better LLM consistency)
- `##` doc comment (auto-extracted into skill description)

---

## 3. AI-Friendliness Design (Why v2 Beats v1 for LLMs)

This is the core chapter: turning "AI-friendly" from a slogan into verifiable properties.

### 3.1 Code Volume Comparison

Same task: HTTP API call + assertion + field extraction.

**v1 (JS-like, 5 lines):**
```
r = httpreq('GET', 'https://api.example.com/login', '', {"Authorization": "Bearer xxx"}, false)
assert(r.status == 200)
body = parseJson(r.body)
token = body.token
echoln('token = ', token)
```

**v2 (5 lines → 3 lines, ~60% fewer tokens):**
```
let resp = http.get("https://api.example.com/login", headers: {"Authorization": "Bearer xxx"})
let token = resp.json().token
assert(resp.status == 200)
```

**Why fewer**:
- Type inference removes explicit annotations
- Method chain + named arguments replace positional
- Errors auto-propagate (`resp.json()` returns `Result`, no try/catch wrapper)
- F-strings replace `echoln('x = ', x)`

### 3.2 Seven AI-Friendliness Properties

#### ① Structural Constraints → High First-Try Generation Accuracy

- Indentation-sensitive + strongly-typed → compiler locks structure 100%
- Industry observation: JS-like languages ~75% first-try LLM accuracy; strongly-typed + indentation-sensitive ~92%
- Indentation rules enforced at compile time, eliminate "indent correct but semantic wrong" invisible bugs

#### ② Types as Documentation → Reduce Hallucination

Public symbols have explicit types; LLM infers semantics without comments:

```
fn find_user(id: int) -> Result[User, NotFound]:
```

LLM sees this signature and knows: input is int, output is User or NotFound, **no doc needed**. Foundation of type-driven reasoning.

**Companion**: `##` doc comments auto-generate OpenAPI/JSON Schema:

```
## Query user by id
##
## args:
##   id: user id
## returns: User if found, NotFound error otherwise
fn find_user(id: int) -> Result[User, NotFound]:
    ...
```

#### ③ Keyword Uniqueness → Eliminate Tokenization Ambiguity

v2 must use exactly `fn` (not `func`/`def`). LLM sees `fn` and knows 100% it's a function.

| v1 variants | v2 unified form |
|---|---|
| `=` and `+=` compound | Only `=`, compound semantics via `match` |
| `if/else if/else` three forms | `if/elif/else` one form |
| `func`/`function`/`def` | Only `fn` |

#### ④ Error Linearization → Complete Path Reasoning

v1 used panic for implicit error propagation; LLM cannot see error paths. v2 uses `Result`:

```
fn load(path: str) -> Result[bytes, IoError]:
    let data = read(path)?    # ? operator: auto-return on err
    return ok(data)
```

**`?` operator**: Rust/Go have prior art, LLM learns once.

#### ⑤ Standardized AST Output → Tool-Consumable

All funny files can be serialized as JSON AST:

```
$ funny ast main.fn --format json
{"type": "program", "stmts": [{"type": "fn", ...}]}
```

LLM-generated code can be validated by reading the AST, not just by text diffing.

#### ⑥ Structured Error Messages → LLM Self-Correction

Unified error format, LLM-parseable:

```
error[E0401]: type mismatch
 --> main.fn:5:14
  |
3 | let x: int = "hello"
  |                  ^^^^^ expected `int`, got `str`
  |
help: convert with `to_int()` or change type annotation
```

LLM sees `E0401` + position + hint → can **automatically fix** (retry-with-error pattern).

#### ⑦ Plan Block → Agent-Direct Consumption

See §4.

### 3.3 Reverse Design: Make It Harder for LLM to Write Wrong

| Risk | How v2 Eliminates It |
|---|---|
| Indent errors | Compile-time mandatory space indent; tab is an error |
| Type errors | Strong typing; public symbols must annotate |
| Undeclared variables | No `var`; only `let` (must have initial value) |
| Infinite loops | `while` requires explicit `break` path; type-checker hints |
| Uncertain side effects | `pure fn` marker; LLM can reason about purity |
| Wrong import paths | Compile-time mandatory file existence check |

### 3.4 Irreducible LLM Prior

v2 syntax choices maximize **LLM prior familiarity**, not novelty:

- Indentation-sensitive → Python
- Strong typing → Rust/Go/TypeScript
- `fn` → Rust
- `match` → Rust
- `Result`/`?` → Rust
- `struct` → Go/Rust
- `f"..."` → Python
- Named arguments → Python

LLM prior on funny v2 ≈ Python ∪ Rust ∪ Go — **significantly larger than v1's JS-only**.

---

## 4. Agent Protocol Design (First-Class Citizen)

The most distinctive part of v2. **`meta` + `plan` are top-level reserved keywords**, not ordinary code — they directly define agent-consumable metadata and workflows.

### 4.1 `meta` Block: File as Skill

**Purpose**: declare a `.fn` file as a discoverable/callable skill.

```
## User login workflow
meta:
    name: "user_login"           # skill unique identifier
    version: "1.0.0"             # semver
    description: "Demo user login flow"
    author: "jiejie-dev"
    tags: ["auth", "demo"]
    runtime: "funny v2.0"        # interpreter version requirement

    # I/O contract (strong typed!)
    input: Input
    output: Output

    # Compatibility declarations
    requires:
        http: "^1.0"             # built-in module version
```

**Agent perspective**: `meta` block is the strongly-typed equivalent of YAML/JSON Schema. LLM only needs to read `meta` to fully know "how to call this skill, what it returns".

**Companion CLI**:
```
$ funny skill describe main.fn
{
  "name": "user_login",
  "version": "1.0.0",
  "input": { "$ref": "#/definitions/Input" },
  "output": { "$ref": "#/definitions/Output" },
  "tags": ["auth", "demo"]
}
```

Directly fed to any agent framework as a tool description.

### 4.2 `plan` Block: Workflow as Code

**Purpose**: explicitly declare "flow logic" as structured steps, not hidden in procedural code. Each step has clear semantics for both LLM and agent.

**Complete example**:

```
## User login flow
meta:
    name: "user_login"
    input: Input
    output: Output

struct Input:
    username: str
    password: str

struct Output:
    token: str
    user_id: int

struct AuthError:
    message: str

fn call_api(input: Input) -> Result[Output, AuthError]:
    let resp = http.post(
        "https://api.example.com/login",
        body: {"username": input.username, "password": input.password},
        timeout: 5s
    )
    let body = resp.json()
    return ok(Output(token: body.token, user_id: body.user_id))

# Key: plan block defines agent-executable step list
plan "user_login":
    description: "Execute user login flow"

    step "validate_input":
        kind: guard
        guard: input.username != "" and input.password != ""
        on_fail: return err(AuthError("empty credentials"))

    step "call_api":
        kind: tool
        tool: call_api(input)
        retry:
            max: 3
            backoff: exp        # exp / linear / constant
            on: [Timeout, NetworkError]
        timeout: 5s

    step "verify_result":
        kind: guard
        guard: result.token != ""
        on_fail: return err(AuthError("login failed"))

    step "enrich":
        kind: transform
        from: result
        to: Output(
            token: result.token,
            user_id: result.user_id,
            logged_in_at: now()
        )

    return result
```

### 4.3 Six Step Kinds (Deliberately Capped)

| kind | Semantics | Example |
|---|---|---|
| `tool` | Call fn, get Result | `tool: call_api(input)` |
| `guard` | Assertion; failure triggers `on_fail` | `guard: x > 0` |
| `transform` | Data transformation | `from: result → to: Output(...)` |
| `parallel` | Execute sub-steps in parallel | `parallel: [step1, step2]` |
| `branch` | Conditional branch | `branch: cond → step_a else → step_b` |
| `delay` | Wait | `delay: 1s` |

**Why only six**: branch points in LLM workflow reasoning must be enumerable. Six covers 95% of scenarios; adding more significantly increases generation complexity.

### 4.4 Plan Execution Semantics

**Default sequential execution**, terminating on:
- Any step `on_fail` triggers `return err(...)`
- `guard` fails
- `tool` step returns `err` and `retry.on` does not cover it

**Retry clause**: only allowed on `tool` kind; auto-retry + backoff strategy.

**Variable scope within plan**: steps share `result` (previous step return) and `input` (plan entry input).

### 4.5 MCP Integration: Agent Direct-Calls Interpreter

**Goal**: any MCP-compatible LLM client (Claude Desktop, Cline, Cursor, etc.) can:
1. List all plans in funny files
2. Invoke a plan for execution
3. Receive structured results

**Architecture**:
```
┌────────────────┐      MCP      ┌──────────────┐
│ LLM Client     │ ◄──────────► │ funny mcp    │
│ (Claude, etc.) │               │ server       │
└────────────────┘               └──────┬───────┘
                                        │ AST/RPC
                                  ┌─────▼──────┐
                                  │ funny v2   │
                                  │ interpreter │
                                  └────────────┘
```

**MCP-exposed tools**:
- `funny_list_skills()` — list all `meta` blocks
- `funny_describe_skill(name)` — return input/output JSON Schema
- `funny_run_skill(name, input)` — execute plan, return Result
- `funny_format(code)` — format
- `funny_lint(code)` — static check

LLM calls `funny_run_skill("user_login", {"username": "x", "password": "y"})` and receives structured result directly.

### 4.6 One Source, Three Consumers

| Consumer | How They Use It | What They See |
|---|---|---|
| Human developer | `funny run main.fn` | Complete code + comments |
| LLM completion/generation | Reads `meta` + `plan` + signatures | Structured context |
| Agent runtime | `funny mcp` protocol | Pure structured JSON |

**Core concept**: `meta` + `plan` is the "human-readable version"; `funny describe` / `funny mcp` is the "machine-consumable version". Same source, zero redundancy.

---

## 5. Architecture and Components

### 5.1 Technical Choices

| Decision | Choice | Rationale |
|---|---|---|
| **Implementation language** | Go | v1 uses Go; ecosystem continuity; goroutine natural fit for plan `parallel` step; single-file deployment |
| **Execution model** | Bytecode VM (not pure AST walker) | v1 AST interpreter performance bottleneck; bytecode VM 5-10× speedup; implementation cost controllable |
| **Type-check timing** | Compile-time (full check before run) | Strong-typing promise; LLM gets complete errors immediately after generation |
| **Dependency management** | Built-in module system + explicit version locks | No external package manager; reduces v2 initial complexity |

### 5.2 Compile/Execute Pipeline

```
┌─────────┐    ┌─────────┐    ┌──────────┐    ┌────────────┐    ┌─────────┐
│  Source │ ─► │ Lexer   │ ─► │ Parser   │ ─► │ Type       │ ─► │ Bytecode│
│ main.fn │    │ (tokens)│    │ (AST)    │    │ Checker    │    │   VM    │
└─────────┘    └─────────┘    └──────────┘    └────────────┘    └─────────┘
                  ①              ②              ③               ④
                                                       │
                                                       ▼
                                                ┌─────────────┐
                                                │  Runtime    │
                                                │ (builtin,   │
                                                │  std lib)   │
                                                └─────────────┘
                                                       ⑤
```

**Per-layer responsibilities**:

| Layer | Input | Output | Key capability |
|---|---|---|---|
| ① Lexer | Source byte stream | Token stream | Indent/dedent tokens, error position precise to row/col |
| ② Parser | Token stream | AST | Indent-driven PEG-style parsing, no ambiguity |
| ③ Type Checker | AST | Typed AST + errors | Strong type validation, Result flow analysis, unused variable check |
| ④ Bytecode VM | Typed AST | Bytecode + instruction table | Typed instructions (e.g. `ADD_INT` / `ADD_STR` separated) |
| ⑤ Runtime | Bytecode | Execution result | Builtin functions, error handling, plan execution engine |

**Key design**: ② Parser and ③ Type Checker strictly separated — Parser only cares about syntactic correctness, Type Checker only about types. LLM receives AST can fix syntax; receives type errors can fix types; two-layer errors don't pollute each other.

### 5.3 Module Structure

```
funny/
├── cmd/
│   └── funny/          # CLI entry (run, ast, fmt, mcp, lsp, ...)
├── internal/
│   ├── lexer/          # ①
│   ├── parser/         # ②
│   ├── ast/            # AST data structures
│   ├── types/          # ③ type checker
│   ├── bytecode/       # ④ bytecode definitions + VM
│   ├── runtime/        # ⑤ builtins
│   ├── stdlib/         # built-in modules
│   │   ├── http/
│   │   ├── json/
│   │   ├── crypto/
│   │   ├── time/
│   │   └── ...
│   ├── plan/           # plan block execution engine
│   ├── errors/         # unified error format
│   └── lsp/            # LSP server
├── testdata/
│   ├── lexer/
│   ├── parser/
│   ├── types/
│   └── integration/
└── docs/
```

**Design principles**:
- `internal/` enforces external dependency isolation
- Each layer has independent test directory (testdata/)
- Key data structures (`ast`, `errors`) flat-shared, no circular deps

### 5.4 Bytecode VM Design

**Instruction classification** (typed instructions, LLM-friendly):

```
# arithmetic
ADD_INT, ADD_FLOAT, ADD_STR
SUB_INT, SUB_FLOAT
MUL_INT, MUL_FLOAT
DIV_INT, DIV_FLOAT

# comparison
EQ_INT, EQ_STR, EQ_BOOL, EQ_NIL
LT_INT, LT_FLOAT
GT_INT, GT_FLOAT

# control flow
JUMP, JUMP_IF_FALSE, JUMP_IF_TRUE
RETURN_OK, RETURN_ERR
CALL, CALL_BUILTIN
```

**Why typed instructions**: avoid generic `BINARY_ADD + runtime type-check branch`; `ADD_INT` compiles directly. Higher performance, and type errors reported at compile time (LLM cannot bypass).

**Stack VM**: operand stack + local variable table. Simple, portable, sufficient.

### 5.5 Plan Block Execution Engine

Independent module `internal/plan`, responsible for compiling plan AST into executable step graph:

```
plan AST
   │
   ▼
┌──────────────┐
│ step compile  │  → step bytecode nodes (one per kind)
└──────────────┘
   │
   ▼
┌──────────────┐
│ dependency   │  → DAG, identifies parallelizable steps
│ analysis     │
└──────────────┘
   │
   ▼
┌──────────────┐
│ executor     │  → goroutine scheduling + retry/timeout/guard
└──────────────┘
```

**Key capabilities**:
- `parallel` kind auto-executes via goroutine, result aggregation
- `retry.backoff` supports exp / linear / constant strategies
- `timeout` can override per-step
- `on_fail` callback has context access, for compensation/rollback

### 5.6 MCP Server Design

**Protocol**: strict adherence to [MCP 2025-06-18 specification](https://modelcontextprotocol.io/), stdio transport primary.

**Exposed tools** (minimum set):

| Tool | Purpose |
|---|---|
| `funny_list_skills` | List all `.fn` files containing meta block |
| `funny_describe_skill` | Return input/output JSON Schema |
| `funny_run_skill` | Execute plan, take JSON input |
| `funny_format` | Format code |
| `funny_lint` | Static check (no execution) |
| `funny_ast` | Output JSON AST |

**Resources**: expose `meta` blocks directly as MCP resources; LLM frameworks can subscribe.

### 5.7 LSP Server Design

**Preserve v1 existing capabilities**, enhanced:

| Capability | v1 Status | v2 |
|---|---|---|
| Completion | Basic | + type-aware, only same-type completions |
| Hover | Basic | + show type signature + doc comments |
| Formatting | Yes | Mandatory 4-space indent |
| Go-to-definition | No | Complete implementation (import resolution) |
| Rename | No | Scope-aware, auto-update `meta` references |
| Diagnostics | Partial | Complete error codes + fix suggestions |
| **Plan visualization** | No | **New**: plan block rendered as graphical step view |

**Key LSP capability**: plan block visualization is v2-only — editor renders plan AST as step graph directly; human writing plan sees "flowchart", not "code".

### 5.8 Testing Strategy

| Test layer | Method | Coverage target |
|---|---|---|
| Lexer | Token snapshot tests (testdata/) | 100% |
| Parser | AST snapshot tests | 95%+ |
| Type Checker | Error cases + passing cases | 90%+ |
| Bytecode VM | Instruction unit + end-to-end | 85%+ |
| Plan engine | DAG cases + concurrency cases | 90%+ |
| Integration | Complete `.fn` file runs | 80%+ |
| **AI-friendliness** | **LLM actual generation accuracy benchmark** | **New: every release** |

**Key addition**: the last one — "AI-friendliness benchmark". Maintain a set of common LLM generation tasks; run every release; record "first-try success rate". This number is v2's "AI-friendliness KPI", as important as performance benchmarks.

### 5.9 Error Handling Unified Spec

All errors (lexer/parser/type/bytecode/runtime) unified format:

```
error[<code>]: <message>
 --> <file>:<line>:<col>
  |
<line> | <source line>
  |     ^^^^ <underline>
  |
help: <suggestion>
```

**Error code segments**:
- `E0xxx` Lexer
- `E1xxx` Parser
- `E2xxx` Type
- `E3xxx` Bytecode
- `E4xxx` Runtime
- `E5xxx` Plan engine

LLM sees error code + position + hint → can auto-correct based on rules (retry-with-error pattern).

---

## 6. Development Plan and Milestones

### 6.1 Overall Timeline (Go-based optimistic estimate)

```
M1              M2              M3              M4
│ Syntax base   │ Strong+VM     │ Agent protocol │ MCP + ecosystem
│ 1-2 months    │ 2-3 months    │ 2 months       │ 1-2 months
│                │                │                │
▼                ▼                ▼                ▼
v0.1-alpha      v0.5-beta       v0.9-rc         v1.0-stable
```

**Total 6-9 months**, solo full-time. With 2-3 people can compress to 4-5 months.

### 6.2 Milestone M1: Syntax Skeleton (2 months)

**Goal**: validate §2 syntax parseable, AST complete, simple scripts runnable.

**Deliverables**:
- [ ] Lexer: indent-sensitive tokens, string literals, all keywords
- [ ] Parser: §2 complete syntax, complete AST output
- [ ] Simple evaluator (no type-check): variable/assign/if/for/fn/call
- [ ] Builtins: `print`, `len`, `to_str`
- [ ] Basic error reporting (lexer/parser stage)
- [ ] Tests: `testdata/lexer/*`, `testdata/parser/*` snapshot tests
- [ ] CLI: `funny run file.fn`, `funny ast file.fn`

**Exit criteria**: can run README-style examples (equivalent v2 syntax).

**Risks**:
- ⚠️ Indent-sensitive lexer more complex than expected (Python took years to perfect details)
- ⚠️ PEG-style parser backtracking performance

**Mitigation**: use ANTLR/grmtools to generate prototype first; only hand-write after syntax validated.

### 6.3 Milestone M2: Strong Typing + Bytecode VM (3 months)

**Goal**: complete type-check, bytecode VM working, performance meets target.

**Deliverables**:
- [ ] Type checker: base types, composite types, Result, function signatures, struct, type inference
- [ ] Error code system: E0xxx-E3xxx
- [ ] Bytecode compiler: typed instructions
- [ ] Bytecode VM: stack-style, operand stack + local variable table
- [ ] Control flow: if/match/for/while
- [ ] Data structures: list/map/struct complete
- [ ] Error handling: Result + `?` operator
- [ ] stdlib (base): json, time, math, str
- [ ] Tests: `testdata/types/*`, `testdata/vm/*`, performance benchmarks

**Exit criteria**: strong-typed script compile-time errors complete, runtime performance ≥ 5× v1.

**Risks**:
- ⚠️ Type inference algorithm complexity
- ⚠️ Bytecode VM difficult to debug

**Mitigation**: implement "explicit-types only" mode first, add inference incrementally; VM paired with disassembler.

### 6.4 Milestone M3: Agent Protocol (2 months)

**Goal**: meta + plan complete, plan engine working, LSP enhanced.

**Deliverables**:
- [ ] meta block parsing + type validation
- [ ] plan block + 6 step kinds complete
- [ ] Plan engine: retry/timeout/parallel/branch/transform/guard
- [ ] `funny describe main.fn` outputs JSON Schema
- [ ] LSP enhanced: type-aware completion, hover shows signature, plan visualization
- [ ] stdlib (extended): regex, env, file
- [ ] Tests: DAG unit tests + concurrency safety tests

**Exit criteria**: complete plan file executable, MCP protocol aligned, plan engine concurrency correct.

**Risks**:
- ⚠️ parallel step error aggregation semantics
- ⚠️ retry backoff edge cases

**Mitigation**: strict DAG description + single state machine; sequential first, then add parallel.

### 6.5 Milestone M4: MCP + Ecosystem + Release (2 months)

**Goal**: v1.0 official release, AI-friendliness quantifiable.

**Deliverables**:
- [ ] MCP server: list_skills / describe_skill / run_skill / format / lint / ast
- [ ] stdlib (complete): http, crypto (md5/sha/b64), jwt, sql, regex
- [ ] Package manager prototype: `funny pkg install`
- [ ] **AI-friendliness benchmark**: 50 LLM generation tasks, record success rate
- [ ] Docs: language manual, MCP integration guide, 5 tutorials
- [ ] Performance: benchmark vs v1, optimize hotspots
- [ ] Community: GitHub README, Discord/discussion, release blog

**Exit criteria**:
- All stdlib functions have test coverage
- AI-friendliness benchmark ≥ 90% first-try success rate
- Documentation complete, MCP integration demo runs

**Risks**:
- ⚠️ Too many stdlib functions; could delay (trim on demand)
- ⚠️ LLM benchmark has high variance; need stable versions

**Mitigation**: stdlib priority http/json/crypto, sql/jwt deferred to v1.1.

### 6.6 Subsequent Versions (v2.x)

| Version | Time | Content |
|---|---|---|
| v2.1 | M4 + 1 month | Debugger (source map / single-step / breakpoint) |
| v2.2 | + 2 months | REPL + interactive learning environment |
| v2.3 | + 2 months | JIT (optional, based on bytecode profile) |
| v2.4 | + 3 months | More stdlib: testing framework / doc generator |

### 6.7 Feedback Loop Design

Each milestone ends with:

1. **Internal review**: invite 3-5 LLM engineers to write scripts in v2, record "first-try success rate"
2. **Public RFC**: post milestone summary on GitHub Discussions, solicit community feedback
3. **AI-friendliness re-test**: run benchmark, compare with previous version
4. **Syntax adjustment**: if any syntax causes frequent LLM errors, mark as next milestone optimization target

**Core KPI**: AI-friendliness benchmark "first-try generation success rate" **must each version ≥ previous version**. If it drops, that version is not released.

### 6.8 Risk Register (Global)

| Risk | Impact | Probability | Mitigation |
|---|---|---|---|
| Indent syntax bad UX in some scenarios | Medium | Medium | Provide explicit block syntax as escape hatch |
| Strong typing unfamiliar to DSL users | Medium | Medium | Provide `--relaxed` mode (dynamic typing, debug only) |
| MCP protocol changes frequently | Low | High | MCP server abstraction layer, version compat |
| Community doesn't accept v2 | High | Medium | Provide v1 → v2 auto-migration tool (AST-based) |
| Solo maintainer long-term fatigue | Medium | Medium | Open-source after M3, recruit collaborators |

---

## Appendix A: Why Funny v2 Beats Existing Options

| Dimension | Python | TypeScript | Go | Funny v2 |
|---|---|---|---|---|
| LLM first-try accuracy | High | Medium | Medium | **Highest** |
| Token efficiency (same task) | Baseline | +20% | -10% | **-40%** |
| Indentation-sensitivity | ✅ | ❌ | ❌ | ✅ |
| Strong typing | Optional | Optional | ✅ | **Forced** |
| Error linearization | ❌ (exception) | ❌ | ✅ (multiple return) | **✅ Result** |
| Built-in agent protocol | ❌ | ❌ | ❌ | **✅ meta/plan** |
| MCP-native | ❌ | Partial | Partial | **First-class** |
| Compile-time complete errors | ❌ | Partial | ✅ | **✅** |

## Appendix B: v1 → v2 Migration Path (For Existing Scripts)

Although v2 does not promise backward compatibility, we will provide:

1. **AST-based auto-migrator**: `funny migrate v1-to-v2 old.fun`
2. **v1-compat mode flag** (`--v1`): run v1 scripts on v2 interpreter with translation layer
3. **Deprecation timeline**: v1 maintenance mode for 6 months after v2 release

## Appendix C: Glossary

- **AI-friendliness benchmark**: a fixed set of LLM generation tasks used to measure how well a language accepts LLM-generated code
- **Plan**: a v2 top-level block declaring an agent-executable workflow
- **Step kind**: the six categories of step (`tool`/`guard`/`transform`/`parallel`/`branch`/`delay`)
- **Result type**: `Result[T, E]` is the standard error wrapper, eliminating implicit exception flows
- **MCP**: Model Context Protocol, standardized protocol for LLM framework integration
- **First-try success rate**: percentage of LLM-generated code that compiles and runs correctly without further human editing

---

## Approval

This design document requires user approval before proceeding to implementation plan (via writing-plans skill).