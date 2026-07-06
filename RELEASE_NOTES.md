# Release Notes — v2.1.5

**Release date:** 2026-07-07
**Module:** `github.com/jiejie-dev/funny/v2`
**License:** MIT
**Binaries:** `funny` (CLI, MCP via `funny mcp`, LSP via `funny lsp`)

---

## Overview

**v2.1.5** adds explicit `mut` struct fields: fields declared with `mut` can be assigned after construction (`c.count = c.count + 1`); all other fields stay immutable and produce `E2010` if assigned.

## Quick start

```bash
# Install this release
go install github.com/jiejie-dev/funny/v2/cmd/funny@v2.1.5

# Run a script
funny run script.fn

# Format source
funny fmt script.fn
funny fmt script.fn -w

# Editor / LLM integration
funny lsp                   # LSP over stdio
funny mcp                   # MCP over stdio
```

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
BenchmarkFib_VM-12           ~2.31 ms/op   (recursive fib(20))
BenchmarkFib_Interpreter-12  ~8.07 ms/op
ratio: ~3.5x
```

The VM remains ~3.5× faster than the tree-walking interpreter. The spec's 5× target is a v2.1.x follow-up.

## Known limitations (v2.1.x follow-ups)

- 5× interpreter performance target not yet met (currently 3.5×)
- AI-friendliness benchmark harness is ready; community LLM runs are still needed

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
