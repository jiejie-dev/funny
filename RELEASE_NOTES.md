# Release Notes — v2.2.2

**Release date:** 2026-07-07
**Module:** `github.com/jiejie-dev/funny/v2`
**License:** MIT
**Binaries:** `funny` (CLI, MCP via `funny mcp`, LSP via `funny lsp`)

---

## Overview

**v2.2.2** completes the v2.2 spec's *interactive learning environment*: guided tutorials, type exploration, script loading, and package install — all inside `funny repl`.

## Quick start

```bash
# Install this release
go install github.com/jiejie-dev/funny/v2/cmd/funny@v2.2.2

# Guided tutorial
funny repl --lesson 1

# Interactive REPL
funny repl
```

## What's new in v2.2.2

### REPL learning environment

- **Guided tutorials** — `:lessons`, `:lesson N`, `:step`, `:hint`, `:show`, `:skip` over `docs/tutorial-*.funny`
- **CLI flags** — `funny repl --lesson 1`, `--lessons-dir ./docs`
- **Exploration** — `:type EXPR`, `:desc NAME`, `:complete PREFIX`, `:history`
- **Workspace** — `:load path.fn`, `:install [pkg]` (wraps `funny.pkg install`)

See `CHANGELOG.md` for the full itemized list.

## What's new in v2.2.1

### VM performance (~7×)

- **Locals pooling** — `CALL`/`RETURN` reuse `[]bytecode.Value` slots across recursive frames
- **Value-type frame stack** — `[]Frame` avoids per-call heap allocations
- **Hot-path dispatch** — inlined `ADD_INT`, `SUB_INT`, `LT_INT`, `CALL`, `RETURN`, `LOAD/STORE_LOCAL`
- **5× gate** — `TestFib_SpeedupRatio` in `internal/vm/bench_exec_test.go` enforces ≥5× exec-only speedup

### AI benchmark CLI

- **`funny bench ai`** — runs 50 tasks from `internal/benchmark/tasks.json`, prints JSON report
- **Real classification** — parser + type checker (fragment tasks forgive unresolved names only)
- **Providers** — `--provider mock|openai|anthropic`, `--model`, env `OPENAI_API_KEY` / `ANTHROPIC_API_KEY`

See `CHANGELOG.md` for the full itemized list.

## What's new in v2.2.0

### Interactive REPL

- **`funny repl`** — read-eval-print loop with persistent bindings
- **Multi-line input** — open blocks/brackets continue on `...` prompt
- **Result printing** — top-level and block-tail expressions display values
- **Meta-commands** — `:help`, `:vars`, `:reset`, `:quit`
- **Type-checked** — same checker as `funny run`; interpreter backend for session state

See `CHANGELOG.md` for the full itemized list.

## What's new in v2.1.7

### Package manager (`funny pkg`)

- **`funny.pkg`** — JSON manifest declaring dependencies and `source` URLs/paths
- **`funny pkg install`** — installs into `.funny/packages/<name>/` and writes `funny.lock` with SHA-256 checksums
- **`funny pkg list`** — shows locked packages
- **`import "pkg:name"`** — module resolver maps pkg imports via `funny.lock`
- **Sources** — `path:` (local file/dir), `https://` (single `.fn`), `git+<url>@<ref>` (shallow clone)

See `CHANGELOG.md` for the full itemized list.

## What's new in v2.1.6

### Bytecode debugger

- **Source maps** — compiler records `SourceLoc` per instruction and `LocalNames` per function
- **`funny debug`** — interactive debugger: `step`, `continue`, `break`, `locals`, `stack`, `where`, `quit`
- **`--source-map`** — JSON export of instruction index → source position
- **`funny disasm`** — disassembly lines annotated with `; file:line:col`

Applies to the default bytecode VM path only (not `FUNNY_INTERPRET=1`).

See `CHANGELOG.md` for the full itemized list.

## What's new in v2.1.5

### `mut` struct fields

- **`mut fname: T`** — struct field modifier parsed by lexer/parser and preserved by formatter
- **Type checking** — `checkFieldAssign` validates mutability and field type before compile
- **Bytecode `SET_FIELD`** — VM and compiler support `obj.field = value` on the default path
- **Evaluator** — interpreter path (`FUNNY_INTERPRET=1`) also supports field assignment

See `CHANGELOG.md` for the full itemized list.

## What's new in v2.1.4

### Typed errors and retry.on

- **Struct `__type` tagging** — struct literals record their type name at runtime (evaluator + VM)
- **`retry.on=NetworkError,str`** — comma-separated error type filter on plan step retry; non-matching errors fail immediately
- **Plan type checking** — unknown struct names in `on=` are rejected at compile time (E2112)

See `CHANGELOG.md` for the full itemized list.

## What's new in v2.1.3

### Agent protocol

- **`branch` case-list** — `step "route" -> branch:` with arms like `status == 200 => "success"` and `_ => "fallback"`; the engine runs exactly one named target step
- **Plan type checking** — branch targets must exist in the plan (E2111); duplicate step names are rejected (E2110)
- **LSP planGraph** — branch steps fan out to target step nodes via `branch` edges; targets are omitted from the linear sequence chain

See `CHANGELOG.md` for the full itemized list.

## What's new in v2.1.2

### VM path completeness

- **`match`** — value matching compiles and executes on the default VM path
- **`break` / `continue`** — loop control flow on the default VM path
- **`x in list`** — membership test via new `IN_LIST` bytecode instruction

### Stdlib and type checking

- **Shared stdlib** — `internal/stdlib.Call` backs both VM and interpreter; all 33 builtins work under `FUNNY_INTERPRET=1` as well as on the default path
- **Top-level expression statements** — `println(undefined_fn())` and similar bare calls at file scope are now type-checked at compile time

See `CHANGELOG.md` for the full itemized list.

## What's new in v2.1.0 (prior release)

### Language

- **F-string interpolation** — `f"hello {name}!"` and `{expr:spec}` format specs end-to-end (lexer, parser, type checker, VM, interpreter)
- **Map literals** — `{"a": 1, "b": 2}` including multi-line trailing-comma style
- **`m[key]` indexing** — read/write for maps; list index assignment via new `SET_INDEX` opcode
- **Module imports** — `import "path.fn"` and `import "path.fn" as m` with `pub` symbol merging (`internal/module`)
- **Bracket line-continuation** — `(...)`, `[...]`, `{...}` may span lines

### Tooling

- **Formatter** — `funny fmt` and MCP `format` tool (AST-based, preserves comments)
- **LSP server** — `funny lsp`: diagnostics, hover, completion, signature help, go-to-definition (cross-file), document symbols, formatting, find-references, rename, custom `funny/planGraph`
- **Unified CLI** — LSP and MCP are subcommands of the main `funny` binary (no separate `funny-mcp` binary required)
- **VS Code extension** — `editors/vscode/` with syntax highlighting, LSP wiring, run/format commands, plan graph view
- **CLI wiring** — `funny describe` and `funny disasm` subcommands now exposed

### Agent protocol

- **`guard`** — final expression/`return` is an assertion (`err(...)`/falsy fails)
- **`delay`** — sleeps for `with timeout="<duration>"` before running body
- **Retry backoff** — `with retry max=N backoff=constant|linear|exp`
- **Step timeout** — `with timeout="<duration>"` bounds a single attempt (best-effort; evaluator is not preemptible)
- **`__result`** — step bodies that end in a bare expression/`return` publish to plan scope

## CLI commands

```
funny run <script>          Execute a funny script
funny ast <script>          Print the JSON AST
funny fmt <script> [-w]     Format source (stdout, or in-place with -w)
funny describe <script>     Print plan + metadata as JSON
funny disasm <script>       Print bytecode disassembly
funny debug <script>        Interactive bytecode debugger (-b, --source-map)
funny pkg install [name]    Install dependencies from funny.pkg
funny pkg list              List packages in funny.lock
funny repl                  Interactive REPL (persistent session)
funny mcp                   Start the MCP server over stdio
funny lsp                   Start the LSP server over stdio
```

## MCP server

`funny mcp` exposes 6 tools over stdio:

- `ast(path)` — parse and return JSON AST
- `format(path)` — return formatted source
- `list_skills(dir)` — list `.fn` files with meta blocks
- `describe_skill(path)` — meta + plan steps for one file
- `run_skill(path)` — execute a file
- `lint(path)` — type-check only

## Performance

```
BenchmarkFib_VM_ExecOnly-12           ~1.3 ms/op   (recursive fib(20), pooled VM)
BenchmarkFib_Interpreter_ExecOnly-12  ~9.2 ms/op
exec-only ratio: ~7×
```

Full pipeline (parse + typecheck + compile + run) remains ~4×; exec-only isolates VM dispatch improvements.

## Known limitations (v2.2.x follow-ups)

- JIT compilation (v2.3 roadmap) not started
- AI benchmark community leaderboard / CI integration not yet published
- REPL uses tree-walking evaluator (differs from default VM path); no VS Code REPL panel or DAP yet

## Upgrading from v2.2.1

No breaking changes. Reinstall the binary:

```bash
go install github.com/jiejie-dev/funny/v2/cmd/funny@v2.2.2
```

## Upgrading from v2.2.0

No breaking changes. Reinstall the binary:

```bash
go install github.com/jiejie-dev/funny/v2/cmd/funny@v2.2.1
```

## Upgrading from v2.1.7

No breaking changes. Reinstall the binary:

```bash
go install github.com/jiejie-dev/funny/v2/cmd/funny@v2.2.0
```

## Upgrading from v2.1.6

No breaking changes. Reinstall the binary:

```bash
go install github.com/jiejie-dev/funny/v2/cmd/funny@v2.1.7
```

## Upgrading from v2.1.5

No breaking changes. Reinstall the binary:

```bash
go install github.com/jiejie-dev/funny/v2/cmd/funny@v2.1.6
```

## Upgrading from v2.1.4

No breaking changes. Reinstall the binary:

```bash
go install github.com/jiejie-dev/funny/v2/cmd/funny@v2.1.5
```

## Upgrading from v2.1.3

No breaking changes. Reinstall the binary:

```bash
go install github.com/jiejie-dev/funny/v2/cmd/funny@v2.1.4
```

## Upgrading from v2.1.2

No breaking changes. Reinstall the binary:

```bash
go install github.com/jiejie-dev/funny/v2/cmd/funny@v2.1.4
```

## Upgrading from v2.1.1

No breaking changes. Reinstall the binary:

```bash
go install github.com/jiejie-dev/funny/v2/cmd/funny@v2.1.4
```

## Upgrading from v2.0.0

No breaking changes to the language surface shipped in v2.0. Install the new binary:

```bash
go install github.com/jiejie-dev/funny/v2/cmd/funny@v2.1.4
```

If you previously used a standalone `funny-mcp` binary, switch to `funny mcp`. Editor configs should use `funny lsp` (see `editors/vscode/`).

## Upgrading from v1

v2 is a complete rewrite. See the v2.0.0 section in `CHANGELOG.md` for migration notes.

## Project layout

```
funny/
├── cmd/funny/              # CLI (run, ast, fmt, describe, disasm, mcp, lsp)
├── docs/                   # language-manual + 6 tutorials
├── editors/vscode/         # VS Code extension
├── examples/log-audit/     # full-language showcase
├── internal/               # lexer, parser, types, compiler, vm, lsp, mcp, agent, …
├── testdata/
├── CHANGELOG.md
└── go.mod
```

## License

MIT — see `LICENSE` for the full text.
