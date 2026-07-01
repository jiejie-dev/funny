# Funny v2

AI-native scripting language. See `docs/superpowers/plans/` for design and planning documents.

**Status: v2.0.0 — RELEASED**

The complete Funny v2 stack is shipping. See `CHANGELOG.md` for the full release notes.

### Quick start

```bash
funny run script.fn         # execute a script
funny ast script.fn         # print JSON AST
funny describe script.fn    # print plan/metadata
funny disasm script.fn      # print bytecode disassembly
funny-mcp                   # start MCP server (for LLM clients)
funny-lsp                   # start LSP server (for editors/IDEs)
```

### Features

- AI-native design: indentation-based syntax, strong typing, agent protocol
- Tree-walking evaluator (default) + bytecode VM (~3.5× faster, set `FUNNY_INTERPRET=1` to use interpreter)
- `Result` + `?` operator for error propagation
- Plan engine: `tool`/`transform`/`guard`/`delay`/`parallel` step kinds with real retry+backoff, timeout, and guard-assertion semantics (`branch` is currently `tool` plus ordinary `if`/`else` — no dedicated case-list syntax yet)
- MCP server with 6 tools for LLM integration
- LSP server: diagnostics, hover, completion, signature help, go-to-definition, document symbols, formatting, find-references, rename, and a custom `funny/planGraph` plan-visualization request
- Standard library: json, time, math, str, regex, env, file, http, crypto, jwt, sql