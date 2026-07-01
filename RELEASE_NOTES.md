# Release Notes — v2.0.0

**Release date:** 2026-07-01
**Module:** `github.com/jiejie-dev/funny`
**License:** MIT
**Binaries:** `funny` (CLI), `funny-mcp` (MCP server)

---

## Overview

Funny v2 is a complete rewrite of the v1 scripting language, designed from the ground up to be AI-native. The release ships the full M1–M3 stack plus the M4/M4.5 agent protocol and tooling: an indentation-sensitive parser, a strong type system, a stack-based bytecode VM, eleven standard-library modules, a complete agent protocol with plans, retry, parallel, branch, and guard step kinds, and an MCP server exposing six tools for LLM clients.

## Quick start

```bash
# Build
go install ./cmd/funny ./cmd/funny-mcp

# Run a script
funny run script.funny

# Print JSON AST
funny ast script.funny

# Print plan + metadata (M3)
funny describe script.funny

# Start MCP server (for LLM clients)
funny-mcp
```

## What's new in v2.0.0

### M1 — Language core (28 tasks)

- **Lexer** with INDENT / DEDENT / NEWLINE tokens, 59 token types
- **Parser** with Pratt expressions, full control flow (if/while/for/match), function and struct declarations
- **Type system** with 7 type kinds (Primitive, List, Map, Struct, Func, Result, Optional)
- **Tree-walking evaluator** as the default execution path (use `FUNNY_INTERPRET=0` to bypass the VM)
- **LSP scaffolding** (kept from v1; not exercised in v2.0.0 but available)

### M2-A — Strong typing (13 tasks + 3 fixes)

- **Recursive-descent type annotation parser** for `list[T]`, `Result[T,E]`, `T?`, etc.
- **Type checker** for expressions, statements, function calls, return values
- **Errors** with structured `E2xxx` codes and unified error format
- **Bonus parser surfaces** added: compound type annotations (`let xs: list[int] = ...`) and struct literal expressions (`Point(x: 1, y: 2)`)
- **Bonus type-checker support**: `?` operator on `Result` types, `.val` / `.tag` field access on Results

### M2-B — Bytecode VM (10 tasks)

- **45 typed instructions** in the spec §5.4 set
- **Stack-based VM** with operand stack + frame stack
- **Bytecode compiler** emits typed instructions for every expression kind
- **CLI default** is the VM; set `FUNNY_INTERPRET=1` to use the tree-walking interpreter
- **End-to-end**: `funny run fib.fn` produces `fib(10) = 55`

### M2-B.5 — Functions and data structures (8 tasks)

- **CALL / RETURN** with frame push/pop, recursive calls work
- **CALL_BUILTIN** dispatches to native Go builtins
- **Data structure instructions**: `BUILD_LIST`, `INDEX`, `BUILD_MAP`, `GET_FIELD`, `NEW_STRUCT`
- **Compiler emits** function declarations, calls, list/field/index, struct literals, full `for-in` loops
- **Benchmark**: VM 3.5× faster than the tree-walking interpreter on recursive fib(20)

### M2-C — Result, `?`, stdlib (9 tasks)

- **`Result` type runtime** with `ok()` / `err()` constructors
- **`?` postfix operator** with `TRY_OR_RETURN` VM instruction
- **Type-checked `?`**: requires operand to be `Result[T,E]`
- **Stdlib base**: `json`, `time`, `math`, `str` modules

### M3 — Agent protocol (11 tasks)

- **AST Step node** with 6 step kinds (tool, guard, transform, parallel, branch, delay)
- **Plan engine** walks `plan` blocks step-by-step, supports retry, parallel (goroutines), branch, guard
- **meta block** type validation (name/version required strings)
- **Stdlib extensions**: `regex`, `env`, `file`
- **CLI `describe` command** for JSON plan/metadata visualization
- **Integration test**: `testdata/agent/plan.fn` runs through the engine

### M4 — MCP server and release (2 tasks so far)

- **Full MCP server** with 6 tools (ast, format, list_skills, describe_skill, run_skill, lint)
- **Stdlib extensions**: `http_get` (net/http), `md5` / `sha256` / `b64_encode` / `b64_decode` (crypto)
- **AI-friendliness benchmark harness**: 50 tasks covering all v2 syntax; community runs against real LLMs

### M4.5 — Completion (8 tasks)

- **Stdlib extensions**: `jwt_encode` / `jwt_decode` (golang-jwt/jwt/v5), `sql_open` (modernc.org/sqlite)
- **Language manual** at `docs/language-manual.md`
- **5 community tutorials** at `docs/tutorial-0[1-5].funny`
- **CHANGELOG.md** and updated `README.md` for v2.0.0
- **Performance benchmark** (recursive fib VM vs interpreter)

## Performance

```
BenchmarkFib_VM-12           ~2.31 ms/op   (recursive fib(20))
BenchmarkFib_Interpreter-12  ~8.07 ms/op
ratio: ~3.5x
```

The VM is ~3.5× faster than the tree-walking interpreter on the recursive fib(20) workload. The spec's 5× target is deferred to v2.0.x; expected wins come from inlining the dispatch loop and reducing per-instruction overhead.

## Standard library

| Module | Functions |
|---|---|
| core | `print`, `println`, `len`, `to_str`, `to_int`, `type_of` |
| result | `ok`, `err` |
| json | `to_json`, `parse_json` |
| time | `now`, `time_format` |
| math | `sqrt`, `pow`, `abs` |
| str | `str_upper`, `str_lower`, `str_contains`, `str_split` |
| regex | `regex_match`, `regex_replace` |
| env | `env_get` |
| file | `file_read`, `file_exists` |
| http | `http_get` |
| crypto | `md5`, `sha256`, `b64_encode`, `b64_decode` |
| jwt | `jwt_encode`, `jwt_decode` |
| sql | `sql_open` |

## CLI commands

```
funny run <script>          Execute a funny script
funny ast <script>          Print the JSON AST
funny describe <script>     Print the plan + metadata as JSON
funny-mcp                   Start the MCP server over stdio
```

`funny --help` lists the full set.

## MCP server

`funny-mcp` exposes 6 tools over stdio (per the Model Context Protocol):

- `ast(path)` — parse and return JSON AST
- `format(path)` — return formatted source (v2.0.0: no-op)
- `list_skills(dir)` — list all `.funny` files in `dir` with their meta blocks
- `describe_skill(path)` — meta + plan steps for one file
- `run_skill(path)` — execute a file via the CLI
- `lint(path)` — type-check only, report errors without executing

## Known limitations (v2.0.x follow-ups)

- 5× interpreter performance target not yet met (currently 3.5×)
- AI-friendliness benchmark requires community LLM evaluation; baseline harness is 50 tasks with a perfect-guesser scorer
- ~~Map literal AST parser syntax needs explicit braces~~ — fixed in v2.1: `{"k": v}` literals are now supported, including multi-line form with trailing commas (see CHANGELOG.md)
- `format` MCP tool is a no-op (real formatting lands in v2.1)
- Some stdlib functions return Result wrappers where plain values might be simpler
- `f"..."` string interpolation: M1 parser accepts the syntax; M2-A runtime substitution is deferred to v2.1
- ~~No LSP server in v2.0.0 (the v1 LSP scaffolding is gone in the v2 migration; v2.1 will re-add)~~ — fixed in v2.1: a from-scratch `funny-lsp` binary now provides diagnostics, hover, completion, signature help, go-to-definition (including across `import`s), document symbols, and formatting (see CHANGELOG.md)

## Upgrading from v1

v2 is a complete rewrite, not a backwards-compatible release. v1 source files (`.funny` extensions) will need their `=` assignment operator changed to `:` for type-annotated declarations, and the v1 `let` syntax for parameters with no type annotation is preserved in v2 but function parameters now require explicit types.

## Migration guide

1. Replace `module github.com/jerloo/funny` with `module github.com/jiejie-dev/funny` in your v1 code that imports v1 internal packages (v1 has been removed in the v2.0.0 release).
2. Rename any v1 keyword-only syntactic features you used (v2's `let` with explicit type annotation is now required for function parameters).
3. Re-test your scripts with `funny run`.
4. Optionally enable strict typing by adding `: Type` annotations to all `let` declarations.

## Acknowledgments

Built with the help of:
- Go 1.22+ standard library
- `github.com/stretchr/testify` for tests
- `github.com/modelcontextprotocol/go-sdk` for the MCP server
- `github.com/golang-jwt/jwt/v5` for JWT support
- `modernc.org/sqlite` for pure-Go SQLite

## Project layout

```
funny/
├── cmd/
│   └── funny/             # CLI entry (run, ast, describe)
├── docs/                  # language-manual + 5 tutorials
├── internal/              # core packages
│   ├── agent/             # plan engine
│   ├── ast/               # AST node types
│   ├── benchmark/         # AI-friendliness harness
│   ├── bytecode/          # 45 typed opcodes + module/function
│   ├── cli/               # CLI helpers
│   ├── compiler/          # typed-AST → bytecode compiler
│   ├── errs/              # unified error format
│   ├── evaluator/         # tree-walking interpreter (fallback)
│   ├── lexer/             # tokenizer
│   ├── mcp/               # MCP server (6 tools)
│   ├── parser/            # Pratt parser
│   ├── types/             # type system + checker
│   └── vm/                # stack-based VM
├── testdata/              # .funny source files
├── CHANGELOG.md
├── LICENSE                 (MIT)
├── README.md
└── go.mod
```

## License

MIT — see `LICENSE` for the full text.
