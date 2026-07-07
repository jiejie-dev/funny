# Funny v2 Language Manual

Complete reference for Funny v2 (M1–M3).

## Lexical Elements

- **Indentation**: 4 spaces per level. Tabs are forbidden (runtime panic).
- **Identifiers**: `[a-zA-Z_][a-zA-Z0-9_]*`
- **Numbers**: `int` (decimal or `0x` hex), `float64`
- **Strings**: `"..."` or `'...'` with `\n \t \\ \" \'` escapes
- **F-strings**: `f"hello {name}"` — full `{expr}` interpolation with optional Python/Rust-flavored format specs, e.g. `f"{price:.2f}"`, `f"{n:>10}"` (see [Format Strings](#format-strings))
- **Comments**: `#` line comment, `##` doc comment (for agent metadata)
- **Operators**: `+ - * / % == != < > <= >= and or not in`
- **Punctuation**: `( ) [ ] { } , : . -> ?`

## Format Strings

F-strings (`f"..."`) support `{expr}` interpolation: any expression may appear
inside `{}`, and its value is converted to a string and spliced into the
result.

```
let name = "world"
let price = 19.5
println(f"hello {name}! total: {price:.2f}")   # hello world! total: 19.50
```

Use `{{` and `}}` to embed a literal brace:

```
println(f"{{literal braces}}")   # {literal braces}
```

### Format spec

An optional `:spec` after the expression controls how the value is rendered,
following a Python/Rust-flavored mini-grammar:

```
{expr:[[fill]align][sign][0][width][.precision][type]}
```

| Field | Values | Meaning |
|---|---|---|
| `fill` | any single char | padding character (default: space); only valid with an explicit `align` |
| `align` | `<` `>` `^` | left / right / center within `width` (default: `<` for strings, `>` for numbers) |
| `sign` | `+` | force a leading `+` on non-negative numbers |
| `0` | `0` | zero-pad shorthand (equivalent to fill `0`, align `>`) |
| `width` | decimal digits | minimum field width |
| `.precision` | `.` + decimal digits | decimal places for `f`/`%`; max length for `s`/default |
| `type` | `d f x X o b s %` | integer, fixed-point float, hex (lower/upper), octal, binary, string, percent |

Examples:

```
f"{n:5d}"      # right-aligned int in a 5-wide field:  "   42"
f"{n:05d}"     # zero-padded:                          "00042"
f"{pi:.2f}"    # fixed-point, 2 decimals:               "3.14"
f"{x:>10}"     # right-align in a 10-wide field
f"{x:^10}"     # center in a 10-wide field
f"{255:X}"     # uppercase hex:                          "FF"
f"{0.5:%}"     # percent:                          "50.000000%"
```

Omitting the spec (`{expr}`) falls back to the same default stringification
used by `to_str`/`println` (`true`/`false` for bools, `nil` for nil).

## Types

- **Primitives**: `int float bool str nil`
- **Composite**: `list[T]`, `map[K, V]`, `Result[T, E]`
- **Nullable**: `T?`
- **Function**: `(P1, P2) -> R`
- **Struct**: declared via `struct Name: field: T, ...`

## Declarations

### Variables
```
let x = 42                       # type inferred as int
let name: str = "hello"          # explicit type
let items: list[int] = [1, 2, 3] # explicit type
```

### Collections

List literals use `[...]`; map literals use `{key: value, ...}`. Both infer
their element/key/value types from the first entry when there's no explicit
annotation, and require an annotation when empty (`let xs: list[int] = []`,
`let m: map[str, int] = {}`).

```
let xs = [1, 2, 3]
let m: map[str, int] = {"a": 1, "b": 2}
```

Any bracketed literal - `[...]`, `(...)`, and `{...}` - may span multiple
lines; a newline inside an open bracket is insignificant whitespace, so the
usual convention is one entry per line ending with a trailing comma:

```
let m: map[str, int] = {
    "a": 1,
    "b": 2,
    "c": 3,
}
```

Map values can be read and written either with `.field` (like a struct) or
with `[key]` indexing; index assignment adds the key if it's absent:

```
println(m.a)      # 1
println(m["a"])   # 1
m["a"] = 100
m["c"] = 3        # adds a new key
xs[0] = 99        # list index assignment works the same way
```

List indices must be `int`; map indices must match the map's declared key
type (`str` in the examples above).

### Functions
```
fn add(a: int, b: int) -> int:
    return a + b

pub fn greet(name: str) -> str:
    return "hello " + name
```

### Structs
```
struct User:
    name: str
    age: int

let u = User(name: "alice", age: 30)
println(u.name)  # field access
```

Fields are immutable by default. Mark a field with `mut` to allow assignment
after construction:

```
struct Counter:
    mut count: int
    label: str

let c = Counter(count: 0, label: "hits")
c.count = c.count + 1   # OK
c.label = "other"       # compile error: field is not mutable
```

### Modules and Imports

`import "path/to/file.fn"` loads real declarations from another file on
disk - it is not just syntax. The path is resolved relative to the
*importing file's* directory. Only top-level `fn` and `struct` declarations
are extracted from the imported file; other top-level statements (`let`,
bare expressions, `meta`, `plan`, ...) are ignored, since dependency files
are treated as function/struct libraries.

Without an alias, the module's `pub` functions and all of its `struct`
types are merged directly into the importing file's namespace and called
like any local function:

```
# math.fn
pub fn add(a: int, b: int) -> int:
    return a + b

# main.fn
import "math.fn"
let r = add(1, 2)
```

With `as alias`, the module isn't renamed - `alias` is just a local nickname
used at the call site, similar to Python's `import numpy as np`. Only `pub`
functions are reachable this way; calling a non-`pub` function through an
alias (`m.helper()`) is a compile error:

```
import "math.fn" as m
let r = m.add(1, 2)
```

Struct types are always merged under their bare name regardless of alias
(there is no `m.Point(...)` construction syntax); use the struct name
directly after importing it.

#### Package dependencies (`funny pkg`)

Projects declare third-party modules in `funny.pkg` (JSON) at the project root.
`funny pkg install` copies or fetches them into `.funny/packages/<name>/` and
writes `funny.lock` with SHA-256 checksums.

```json
{
  "name": "my-app",
  "dependencies": {
    "math": {
      "source": "path:vendor/math.fn",
      "version": "^1.0.0"
    }
  }
}
```

Each dependency may include an optional `version` constraint:

| Form | Meaning |
|------|---------|
| `1.2.3` | exact version |
| `>=1.0.0` | minimum version |
| `^1.2.0` | compatible within major (≥1.2.0 and &lt;2.0.0) |
| `*` or omitted | any version |

For `git+<url>@<ref>` sources, the `@ref` tag is treated as the resolved
version and checked against the constraint at install time. `path:` and
`https://` sources resolve to `0.0.0` unless you pin with an exact constraint.

Supported `source` forms: `path:<file-or-dir>`, `https://...` (single `.fn`),
`git+<url>@<ref>` (shallow clone).

Import installed packages with the `pkg:` prefix:

```
import "pkg:math"
let r = add(1, 2)
```

```bash
funny pkg add math path:vendor/math.fn          # declare + install
funny pkg add math --source path:vendor/math.fn --version "^1.0.0"
funny pkg install
funny pkg install math
funny pkg update              # refresh all locked packages
funny pkg update math         # refresh one package
funny pkg list
```

Other rules:
- A module's own private (non-`pub`) functions are still usable by that
  module's `pub` functions, but are invisible to (and cannot collide with)
  everything else - they're internally renamed to a hygienic, unwritable
  name.
- A file is only ever read and merged once per run, even if reached through
  multiple import paths (diamond dependencies).
- Circular imports (`a.fn` -> `b.fn` -> `a.fn`) are a compile error.
- Two distinct files declaring a `pub fn`/`struct` with the same name that
  both end up merged into the same program (e.g. two unaliased imports, or
  an import colliding with a name declared in the importing file) is a
  compile error; use `as` to disambiguate, or rename one of them.

## Control Flow

### If
```
if x > 0:
    print("positive")
elif x == 0:
    print("zero")
else:
    print("negative")
```

### Loops
```
for i in [1, 2, 3]:
    print(i)

while x > 0:
    x = x - 1

# break exits the nearest for/while; continue skips to the next iteration.
for i in [1, 2, 3]:
    if i == 2:
        continue
    print(i)
```

### Match

Value matching on an expression. Patterns are literals, variables (compared
by value), or `_` (wildcard). The first matching arm runs; if none match,
execution continues after the `match`.

```
match status:
    200 => print("ok")
    404 => print("not found")
    _   => print("other")
```

## Result + `?` Operator

`Result[T, E]` is a tagged union: Ok(value) or Err(error). The `?` postfix unwraps Ok or returns Err from the enclosing function.

```
fn divide(a: int, b: int) -> Result:
    if b == 0:
        return err("divide by zero")?
    return ok(a / b)?

let r = divide(10, 2)?
if r.tag == "err":
    print("error: " + r.val)
else:
    print("result: " + r.val)
```

## Plans (Agent Protocol)

```
meta:
    name = "my_skill"
    version = "1.0"

plan "my_skill":
    step "setup":
        let x = 1
    step "compute" -> tool with retry max=3 backoff=exp:
        let r = x * 2
    step "verify" -> guard:
        r > 0
    step "pause" -> delay with timeout="200ms":
        pass
```

A step's kind (`tool`/`guard`/`transform`/`parallel`/`branch`/`delay`, after `->`; `tool` if
omitted) and its `with` options are executed by `internal/agent.Engine` as follows:

- **`tool`** / **`transform`**: run the body once (subject to retry below). If the body's
  final statement is a bare expression or `return <value>` (not `let`/`assign`), its value
  is published to the plan's scope as `__result`, so a later step can read what the
  previous one produced.
- **`guard`**: same as `tool`, but if the body's final statement is a bare
  expression/`return <value>`, that value is also treated as an assertion — an `err(...)`
  Result or anything falsy fails the step (triggering retry, if configured); an `ok(...)`
  Result always passes regardless of its payload. A body that ends in `let`/`assign`
  (nothing to assert) always passes.
- **`branch`**: evaluates a case-list and runs exactly one named target step
  (`cond => "step_name"`, with `_ => "fallback"` as the default arm). Target
  steps are skipped during normal sequential plan execution and only run when
  selected. A legacy `if`/`else` body is still accepted for backward
  compatibility.
- **`delay`**: requires `with timeout="<duration>"`; sleeps for that duration before
  running the body (which is typically empty or just `pass`).
- **`parallel`**: every statement directly in the body runs concurrently, one goroutine
  each (not a list of named sub-steps — there's no nested-step syntax); the step
  completes once all of them do, failing with the first error seen if any failed. Retry,
  timeout, and guard assertions do not apply to `parallel` steps.
- **`with retry max=<N>`**: retries the body up to `N` times on failure (an error, a
  timeout, or — for `guard` — a failed assertion).
- **`with ... backoff=<constant|linear|exp>`**: adds a delay between retry attempts
  (constant, `N`× the base unit, or `2^(N-1)`× the base unit); omitting `backoff`
  retries immediately, matching pre-v2.1 behavior.
- **`with ... on=<Type1>,<Type2>`**: only retry when the failure's error type matches
  one of the listed names. Struct-typed errors use the struct name (e.g.
  `return err(NetworkError(message: "timeout"))`); plain string errors use `str`.
  Omitting `on` retries every failure.
- **`with ... timeout="<duration>"`** (e.g. `"500ms"`, `"5s"`): bounds a single attempt's
  wall-clock time. When the deadline passes the plan engine cancels the step's evaluator
  context; the tree-walking interpreter stops at the next preemption point (loop head,
  statement boundary), so infinite loops no longer keep mutating scope in the background.

Struct instances created via struct literals carry a runtime `__type` field with the
struct name so plan `retry.on` can distinguish typed errors from plain strings.

## Builtin Functions

| Function | Description |
|---|---|
| `print(...)` | Print to stdout (no newline) |
| `println(...)` | Print with newline |
| `len(x)` | Length of string or list |
| `to_str(x)` | Convert to string |
| `to_int(x)` | Convert to int |
| `type_of(x)` | Type name as string |
| `ok(x)` / `err(x)` | Construct Result |
| `regex_match(p, t)` | Test regex |
| `regex_replace(p, t, r)` | Replace matches |
| `env_get(name)` | Read environment variable |
| `file_read(path)` | Read file (returns Result) |
| `file_exists(path)` | Test file existence |
| `http_get(url)` | HTTP GET (returns Result) |
| `md5(s)` / `sha256(s)` | Hash functions |
| `b64_encode(s)` / `b64_decode(s)` | Base64 encoding |
| `jwt_encode(h, c, s)` | Sign JWT (HS256) |
| `jwt_decode(t, s)` | Verify and decode JWT |
| `sql_open(path)` | Open SQLite database |
| `assert(cond)` | Fail the current test if `cond` is false |
| `assert_eq(a, b)` | Fail if `a` and `b` are not equal |

## Testing

Write tests in `*_test.fn` files using `test "name":` blocks:

```
fn add(a: int, b: int) -> int:
    return a + b

test "addition":
    assert(add(1, 2) == 3)

test "zero":
    assert_eq(add(0, 0), 0)
```

Run all tests under a directory or a single file:

```bash
funny test                  # discover *_test.fn under .
funny test path/to/pkg      # run tests in a tree
funny test math_test.fn     # one file
funny test -v               # verbose (print each case as it runs)
funny test --json           # machine-readable report
```

Test bodies share the same helpers and imports as the rest of the file; the
runner type-checks the full module, then executes each `test` block in isolation
with supporting declarations available.

## Documentation (`funny doc`)

Place `##` doc comments immediately before `pub fn` / `pub struct` declarations.
The doc generator extracts summaries, argument descriptions, and return notes:

```
## Add two integers
##
## args:
##   a: first summand
##   b: second summand
## returns: sum of a and b
pub fn add(a: int, b: int) -> int:
    return a + b
```

Generate Markdown or JSON API reference:

```bash
funny doc .                         # markdown to stdout
funny doc lib/ --out docs/api       # write one .md per .fn file
funny doc skill.fn --format json    # JSON schema-like output
```

## CLI Usage

```bash
funny run script.fn         # execute
funny ast script.fn         # JSON AST
funny fmt script.fn         # print canonically-formatted source to stdout
funny fmt script.fn -w      # reformat the file in place
funny describe script.fn    # JSON plan/metadata
funny disasm script.fn      # print bytecode disassembly
funny debug script.fn       # interactive bytecode debugger
funny debug script.fn --source-map  # JSON instruction→source map
funny debug script.fn -b 10 # break at line 10, then step/continue
funny dap                   # Debug Adapter Protocol (VS Code Run and Debug)
funny pkg add <name> [source] # add dependency to funny.pkg and install
funny pkg install           # install dependencies from funny.pkg
funny pkg update [name...]  # re-fetch and refresh funny.lock
funny pkg list              # list locked packages
funny repl                  # interactive REPL
funny test [path]           # run *_test.fn test blocks
funny doc [path]            # generate docs from ## comments
funny mcp                   # start MCP server
funny lsp                   # start LSP server
```

Install the single `funny` binary (CLI + MCP + LSP):

```bash
go install github.com/jiejie-dev/funny/cmd/funny@latest
```

## Debugger

The bytecode VM records a **source map** (instruction index → file:line:col) at compile
time. Use `funny debug` for an interactive session, or export the map as JSON.

```bash
# JSON source map (per-function instructions + local names)
funny debug script.fn --source-map

# Interactive debugger (pauses at entry, then on breakpoints / after each step)
funny debug script.fn
funny debug script.fn -b 12 -b other.fn:5
```

`funny disasm` also annotates each instruction with its source location (`; file:line:col`).

### VS Code debugging (DAP)

`funny dap` speaks the [Debug Adapter Protocol](https://microsoft.github.io/debug-adapter-protocol/) over stdio. The Funny VS Code extension (`editors/vscode/`) registers a **Debug Funny File** launch configuration that starts `funny dap`, sets editor breakpoints, and inspects **Locals** and **Stack** scopes while stepping.

Debugger commands at the `(dbg)` prompt:

| Command | Alias | Action |
|---|---|---|
| `step` | `s` | Execute one bytecode instruction |
| `continue` | `c` | Run until the next breakpoint |
| `break N` | `b` | Set breakpoint at line N (or `file:line`) |
| `locals` | `l` | Show local variables |
| `stack` | `p` | Show operand stack |
| `where` | `w` | Show current source location |
| `quit` | `q` | End the session |

Set `FUNNY_INTERPRET=1` to use the tree-walking evaluator instead of the VM; the
debugger applies only to the default bytecode path.

## REPL

`funny repl` starts an interactive read-eval-print loop for learning and
experimentation. State (variables, functions, structs) persists across inputs.
The REPL uses the tree-walking evaluator with the same type checker as `funny run`.

```bash
funny repl
funny repl --project /path/to/project   # for import/pkg: resolution
```

Multi-line cells use indentation (and open brackets); continuation lines show a
`...` prompt. Trailing expressions print their value (including the last
expression in an `if`/`for`/`while` body).

| Command | Alias | Action |
|---|---|---|
| `:help` | `:h` | Show REPL help |
| `:vars` | `:v` | List bindings |
| `:reset` | | Clear session |
| `:quit` | `:q` | Exit |
| `:lessons` | `:ls` | List `tutorial-*.funny` lessons |
| `:lesson N` | | Start guided tutorial *N* |
| `:step` | | Run current tutorial step (demo) |
| `:hint` | | Show current step hint |
| `:show` | | Reveal current step source |
| `:load PATH` | | Load and run a script into the session |
| `:type EXPR` | | Show expression type |
| `:desc NAME` | | Describe a binding |
| `:complete PREFIX` | | Suggest identifier completions |
| `:history` | | Show recent inputs |
| `:install [PKG]` | | Run `funny.pkg` install |

Start a guided tutorial directly:

```bash
funny repl --lesson 1
funny repl --lessons-dir ./docs
```

## MCP Server

The `funny mcp` subcommand exposes 6 tools over stdio:
- `ast`: parse source, return JSON AST
- `format`: format source code (canonical 4-space indentation, preserves comments)
- `list_skills`: list .fn files in a directory
- `describe_skill`: meta + plan info for one file
- `run_skill`: execute a .fn file
- `lint`: type-check only, no execution

## LSP Server

The `funny lsp` subcommand speaks LSP 3.17 (a hand-rolled minimal subset, no third-party
protocol dependency) over stdio, framed as standard `Content-Length`-prefixed JSON-RPC
2.0. Point any LSP-capable editor at it for `.fn` files. Supported capabilities:

- **Diagnostics** (`textDocument/publishDiagnostics`, sent on `didOpen`/`didChange`):
  parser, module-resolution, and type-checker errors, each anchored at its precise
  position and carrying its structured error code (`E1xxx`/`E2xxx`). Since the type
  checker stops at the first error (fail-fast, matching the compiler), only one
  diagnostic is reported per analysis pass. An error inside an *imported* file is
  still surfaced in the importing document (anchored at the top of the file, with the
  imported file's path/line embedded in the message), so it isn't silently invisible.
- **Hover**: shows the type of local variables/parameters, full function signatures,
  struct field layouts, builtin functions, and keyword documentation.
- **Completion**: keywords, builtins, declared functions/structs, and locals in scope
  everywhere; immediately after `<expr>.`, only that expression's fields (struct
  fields, or `tag`/`val` for a `Result`) are offered — type-aware, same-type-only
  completion, not a generic symbol dump.
- **Signature help**: shows the enclosing call's signature and highlights the active
  parameter while typing arguments.
- **Go-to-definition**: jumps to local variable/parameter declarations, and to
  function/struct declarations — including across `import`ed files, reusing the same
  module resolution used by `funny run`.
- **Document symbols**: an outline of `fn`/`struct` declarations (struct fields
  nested underneath) and `plan` blocks (with `step`s nested underneath, as a
  lightweight tree view of the plan graph).
- **Formatting**: delegates to the same formatter as `funny fmt`.
- **Find references** (`textDocument/references`): every occurrence of the
  identifier under the cursor — a local variable/parameter search is scoped to its
  own function (matching the same over-approximation `documentSymbol`/hover use for
  shadowed names in different blocks), a top-level `let`/`fn`/`struct` search covers
  the whole document. Scoped to the requested document only; occurrences in other
  files (even ones that `import` this one) are not indexed.
- **Rename** (`textDocument/prepareRename` + `textDocument/rename`): validates the
  symbol under the cursor is a renameable identifier (not a keyword or builtin),
  then reuses the same reference search as above to build a `WorkspaceEdit` that
  updates every occurrence in the current document. `meta` block fields are plain
  free-form strings with no grammar-level link to a `fn`/`struct` name, so rename
  intentionally does not attempt to pattern-match and rewrite them.
- **`funny/planGraph`** (custom extension, not part of standard LSP): given a
  document URI, returns `{"plans": [...]}` — one node/edge graph per `plan` block,
  built to mirror `internal/agent/engine.go`'s actual execution semantics rather
  than grammar shape alone. Each `step` is a node (`id`, `label` = step name, `kind`
  = `tool`/`guard`/`transform`/`parallel`/`branch`/`delay`, `range`, and optional
  `retry`/`timeout`); consecutive top-level steps get a `"sequence"` edge. A
  `parallel` step's body statements each run concurrently at runtime (one goroutine
  per statement — there's no nested named sub-step syntax), so they become child
  `"task"` nodes (`parentId` set to the parallel step) connected by `"parallel"`
  edges, and the step *after* the parallel step is linked from the parallel step
  itself (matching where the engine rejoins after waiting for every task). Editors
  without custom-request support can fall back to the `documentSymbol` outline,
  which already nests `step`s under their `plan`.