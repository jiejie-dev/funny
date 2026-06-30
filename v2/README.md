# Funny v2 (M2-A)

AI-native scripting language. See `../docs/superpowers/specs/2026-07-01-funny-v2-ai-native-language-design.md` for the full design.

**Status: M2-A (Strong Typing Foundation) — RELEASED**

- ✅ Lexer (indent-sensitive, all operators, strings, f-strings)
- ✅ Parser (Pratt expressions, control flow, fn/struct/meta/plan)
- ✅ Tree-walking evaluator (no type checking at runtime — types are compile-time only)
- ✅ **Type system** (M2-A): primitive types, list, map, struct, func, Result, optional
- ✅ **Type checker**: validates expressions, statements, function calls, returns
- ✅ **Compile-time error reporting**: E2xxx with unified format
- ⏳ Bytecode VM → M2-B
- ⏳ Result + `?` operator runtime → M2-C
- ⏳ stdlib (json/time/math/str) → M2-C
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
│   ├── types/             # M2-A: type system + checker
│   └── cli/               # CLI helpers (Run, Ast)
└── testdata/              # .fn source files
    ├── integration/       # end-to-end scripts
    ├── parser/            # parser positive cases
    └── types/             # type checker fixtures (M2-A)
```

## Limitations (M2-A)

- **Types checked at compile-time only** — runtime evaluator remains dynamic (no tagged values)
- **No `?` Result operator** (defer to M2-C)
- **No actual `import` resolution** (parsed only)
- **`meta` and `plan` blocks parsed but not executed** (M3)
- **Limited stdlib**: `print`, `println`, `len`, `to_str`, `to_int`, `type_of`

## Roadmap

| Version | Status | Highlights |
|---|---|---|
| v2.0.0-alpha (M1) | ✅ Done | Lexer + Parser + Evaluator (no types) |
| v2.0.0-beta (M2-A) | ✅ Done | Type system + type checker |
| v2.0.0-beta (M2-B) | Planned | Bytecode VM (5×+ perf) |
| v2.0.0-beta (M2-C) | Planned | Result + `?` + stdlib |
| v2.0.0-rc (M3) | Planned | meta/plan engine + LSP |
| v2.0.0 (M4) | Planned | MCP server + full stdlib |

## Next: M2-B (Bytecode VM)

See `docs/superpowers/specs/2026-07-01-funny-v2-ai-native-language-design.md` §6.3.