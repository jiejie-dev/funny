# Funny v2

AI-native scripting language. See `docs/superpowers/plans/` for design and planning documents.

**Status: v2.0.0 — RELEASED; v2.1.0 — in progress on `main`**

The complete Funny v2 stack is shipping. See `CHANGELOG.md` for the full release notes (including unreleased v2.1 work).

### Install

Go module: `github.com/jiejie-dev/funny`

One binary covers the CLI, LSP server, and MCP server:

```bash
go install github.com/jiejie-dev/funny/cmd/funny@latest
```

Ensure `$HOME/go/bin` (or `$GOPATH/bin`) is on your `PATH`, then verify:

```bash
funny --version
```

### Quick start

```bash
funny run script.fn         # execute a script
funny ast script.fn         # print JSON AST
funny describe script.fn    # print plan/metadata
funny disasm script.fn      # print bytecode disassembly
funny mcp                   # start MCP server (for LLM clients)
funny lsp                   # start LSP server (for editors/IDEs)
```

### Features

- AI-native design: indentation-based syntax, strong typing, agent protocol
- Bytecode VM (default) + tree-walking evaluator fallback (VM is ~3.5× faster; set `FUNNY_INTERPRET=1` to use the interpreter instead)
- `Result` + `?` operator for error propagation
- Plan engine: `tool`/`transform`/`guard`/`delay`/`parallel` step kinds with real retry+backoff, timeout, and guard-assertion semantics (`branch` is currently `tool` plus ordinary `if`/`else` — no dedicated case-list syntax yet)
- MCP server with 6 tools for LLM integration
- LSP server: diagnostics, hover, completion, signature help, go-to-definition, document symbols, formatting, find-references, rename, and a custom `funny/planGraph` plan-visualization request
- VS Code extension (`editors/vscode/`): syntax highlighting, snippets, LSP integration, run/format commands, plan graph view
- Standard library: json, time, math, str, regex, env, file, http, crypto, jwt, sql