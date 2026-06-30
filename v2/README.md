# Funny v2 (M1)

AI-native scripting language. See `../docs/superpowers/specs/2026-07-01-funny-v2-ai-native-language-design.md` for the full design.

**Status: M1 (Syntax Skeleton) — RELEASED**

- ✅ Lexer (indent-sensitive, all operators, strings, f-strings)
- ✅ Parser (Pratt expressions, control flow, fn/struct/meta/plan)
- ✅ Tree-walking evaluator (no type checking yet)
- ⏳ Strong typing → M2
- ⏳ Bytecode VM → M2
- ⏳ Plan engine → M3
- ⏳ MCP server → M4

## Build

```bash
cd v2
go build -o funny ./cmd/funny
```

## Usage

```bash
./funny run script.fn        # execute script
./funny ast script.fn        # output JSON AST
./funny --help               # all commands
```

## Test

```bash
go test ./...
```

## End-to-end demo

```bash
$ ./funny run ./testdata/integration/fib.fn
fib(10) = 55
```

## Project Layout

```
v2/
├── cmd/funny/             # CLI entry (cobra)
├── internal/
│   ├── errs/              # Unified error system (E0xxx–E5xxx)
│   ├── lexer/             # Tokenizer with INDENT/DEDENT
│   ├── ast/               # AST node types
│   ├── parser/            # Pratt parser + control flow
│   ├── evaluator/         # Tree-walking interpreter
│   └── cli/               # CLI helpers (Run, Ast)
└── testdata/              # .fn source files
```

## Limitations (M1)

- **No type checking** — variables are dynamically typed
- **No `?` Result operator** (defer to M2)
- **No actual `import` resolution** (parsed only)
- **`meta` and `plan` blocks parsed but not executed** (M3)
- **Limited stdlib**: `print`, `println`, `len`, `to_str`, `to_int`, `type_of`

## Roadmap

| Version | Status | Highlights |
|---|---|---|
| v2.0.0-alpha (M1) | ✅ Done | Lexer + Parser + Evaluator (no types) |
| v2.0.0-beta (M2) | Planned | Strong typing + Result + bytecode VM (5×+ perf) |
| v2.0.0-rc (M3) | Planned | meta/plan engine + LSP |
| v2.0.0 (M4) | Planned | MCP server + full stdlib |

## Next: M2 (Strong Typing + Bytecode VM)

See `docs/superpowers/specs/2026-07-01-funny-v2-ai-native-language-design.md` §6.3.