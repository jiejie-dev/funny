# Funny — VS Code Extension

Language support for [Funny](https://github.com/jiejie-dev/funny), an AI-native scripting language.

## Features

| Feature | Source |
|---|---|
| Syntax highlighting | TextMate grammar (`.fn`, `.funny`) |
| Diagnostics | `funny-lsp` |
| Hover / type info | `funny-lsp` |
| Autocomplete | `funny-lsp` |
| Signature help | `funny-lsp` |
| Go to definition | `funny-lsp` |
| Find references | `funny-lsp` |
| Rename | `funny-lsp` |
| Document symbols (outline) | `funny-lsp` |
| Format document | `funny-lsp` (same as `funny fmt`) |
| Code snippets | Built-in |
| Run current file | `funny run` command |
| Plan graph visualization | Custom `funny/planGraph` LSP request |

## Prerequisites

Install the Funny toolchain and ensure `funny-lsp` and `funny` are on your `PATH`:

```bash
go install github.com/jiejie-dev/funny/cmd/funny-lsp@latest
go install github.com/jiejie-dev/funny/cmd/funny@latest
```

Or build from this repository:

```bash
go build -o funny-lsp ./cmd/funny-lsp
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
| `funny.lsp.path` | `funny-lsp` | Path to the language server |
| `funny.executablePath` | `funny` | Path to the CLI for run commands |
| `funny.trace.server` | `off` | LSP trace level (`off` / `messages` / `verbose`) |

## Commands

- **Funny: Run Current File** — runs `funny run` on the active editor
- **Funny: Format Document** — triggers LSP formatting
- **Funny: Show Plan Graph** — renders `plan` blocks as a Mermaid flowchart
- **Funny: Restart Language Server**

## Snippets

Type prefixes like `fn`, `struct`, `plan`, `step-tool`, `fstr`, etc. for common constructs.

## License

MIT — see repository root `LICENSE`.
