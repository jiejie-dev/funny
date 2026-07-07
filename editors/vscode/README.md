# Funny — VS Code Extension

Language support for [Funny](https://github.com/jiejie-dev/funny), an AI-native scripting language.

## Features

| Feature | Source |
|---|---|
| Syntax highlighting | TextMate grammar (`.fn`, `.funny`) |
| Diagnostics | `funny lsp` |
| Hover / type info | `funny lsp` |
| Autocomplete | `funny lsp` |
| Signature help | `funny lsp` |
| Go to definition | `funny lsp` |
| Find references | `funny lsp` |
| Rename | `funny lsp` |
| Document symbols (outline) | `funny lsp` |
| Format document | `funny lsp` (same as `funny fmt`) |
| Code snippets | Built-in |
| Run current file | `funny run` command |
| REPL in terminal | `funny repl` command |
| VS Code debugging | `funny dap` Debug Adapter |
| Plan graph visualization | Custom `funny/planGraph` LSP request |

## Prerequisites

Install the Funny toolchain — one binary covers the CLI, LSP, and MCP servers:

```bash
go install github.com/jiejie-dev/funny/cmd/funny@latest
```

Ensure `$HOME/go/bin` is on your `PATH`. Or build from this repository:

```bash
go build -o funny ./cmd/funny
```

## Installation

### From source (development)

```bash
cd editors/vscode
npm install
npm run compile
```

Then press **F5** in VS Code to launch an Extension Development Host, or package and install:

```bash
npm install -g @vscode/vsce
vsce package
code --install-extension funny-vscode-*.vsix
```

### Settings

| Setting | Default | Description |
|---|---|---|
| `funny.lsp.path` | `funny` | Path to the executable that starts the language server |
| `funny.lsp.args` | `["lsp"]` | Arguments passed to start the language server |
| `funny.executablePath` | `funny` | Path to the CLI for run commands |
| `funny.trace.server` | `off` | LSP trace level (`off` / `messages` / `verbose`) |

## Commands

- **Funny: Run Current File** — runs `funny run` on the active editor
- **Funny: Start REPL** — opens a terminal with `funny repl --project <workspace>`
- **Funny: Debug Current File (Terminal)** — `funny debug` with breakpoint at cursor line
- **Funny: Format Document** — triggers LSP formatting
- **Funny: Show Plan Graph** — renders `plan` blocks as a Mermaid flowchart
- **Funny: Restart Language Server**

## Debugging

Use the **Run and Debug** panel with the **Debug Funny File** launch configuration (requires `funny dap` in your toolchain). The adapter uses the bytecode VM debugger with editor breakpoints, locals, and stack inspection.

For a quick terminal session, use **Funny: Debug Current File (Terminal)**.

## REPL

**Funny: Start REPL** opens an integrated terminal running `funny repl`. Use `:lessons` and `:lesson 1` inside the REPL for guided tutorials.

## Snippets

Type prefixes like `fn`, `struct`, `plan`, `step-tool`, `fstr`, etc. for common constructs.

## License

MIT — see repository root `LICENSE`.
