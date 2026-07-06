# Release Notes — v2.1.1

**Release date:** 2026-07-07
**Module:** `github.com/jiejie-dev/funny/v2`
**License:** MIT
**Binaries:** `funny` (CLI, MCP via `funny mcp`, LSP via `funny lsp`)

---

## Overview

Funny v2.1 is a major tooling and language-completeness release on top of the v2.0 stack. **v2.1.1** fixes the Go module path (`github.com/jiejie-dev/funny/v2`) so semver installs work; feature content matches v2.1.0.

## Quick start

```bash
# Install this release
go install github.com/jiejie-dev/funny/v2/cmd/funny@v2.1.1

# Run a script
funny run script.fn

# Format source
funny fmt script.fn
funny fmt script.fn -w

# Editor / LLM integration
funny lsp                   # LSP over stdio
funny mcp                   # MCP over stdio
```

## What's new in v2.1.0

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

### Notable fixes

- Struct field access and builtin return types no longer mis-codegen under the VM
- `for` loops no longer skip their first element on the default VM path
- Thirteen stdlib builtins (`regex_*`, `env_get`, `file_*`, `http_get`, crypto, jwt, sql) are now callable from `.fn` source
- Float comparisons, `!=`, and `and`/`or` compile on the VM path
- Lexer column positions fixed for LSP accuracy
- Many compiler crashes on `meta`/`plan` blocks, struct annotations, and mixed top-level decl order resolved

See `CHANGELOG.md` for the full itemized list.

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
- Struct fields are immutable after construction (`p.x = 99` is `E2010`)
- `branch` step kind is still `tool` + ordinary `if`/`else` (no case-list syntax)
- `retry.on` deferred until Funny has typed errors
- AI-friendliness benchmark harness is ready; community LLM runs are still needed

## Upgrading from v2.0.0

No breaking changes to the language surface shipped in v2.0. Install the new binary:

```bash
go install github.com/jiejie-dev/funny/v2/cmd/funny@v2.1.1
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
