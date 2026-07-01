# Funny v2 (M2-B)

AI-native scripting language. See `../docs/superpowers/specs/2026-07-01-funny-v2-ai-native-language-design.md` for the full design.

**Status: M2-C (Result + ? + Stdlib) — RELEASED**

- ✅ Lexer, Parser, Type checker, Bytecode VM, VM Functions + Data Ops (M1–M2-B.5)
- ✅ **Result type runtime**: `ok()` / `err()` constructors
- ✅ **`?` operator**: postfix try-propagation
- ✅ **stdlib**: json, time, math, str modules
- ⏳ meta/plan engine + LSP → M3
- ⏳ MCP server + full stdlib → M4

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

## M2-C Usage

The full M2 stack is now usable. End-to-end demos:

```bash
$ ./funny run ./testdata/types/result.fn
10 / 2 = 5
expected error: divide by zero

$ ./funny run ./testdata/types/json.fn
alice
30
{"age":30,"name":"alice"}

$ ./funny run ./testdata/types/stdlib.fn
unix: <current timestamp>
HELLO FUNNY
count: 3
sqrt(16): 4
pow(2, 10): 1024
abs(-7): 7
date: <formatted timestamp>
```

The `?` operator propagates errors: `expr?` returns the Result from the enclosing function if `expr` is `Err`, or unwraps the Ok value (no automatic unwrap in M2-C — use `r.val` to access).

## Roadmap

| Version | Status | Highlights |
|---|---|---|
| v2.0.0-alpha (M1) | ✅ Done | Lexer + Parser + Evaluator (no types) |
| v2.0.0-beta (M2-A) | ✅ Done | Type system + type checker |
| v2.0.0-beta (M2-B) | ✅ Done | Bytecode VM (literals, arithmetic, control flow) |
| v2.0.0-beta (M2-B.5) | ✅ Done | VM function calls + data structures (~3.5× interpreter) |
| v2.0.0-beta (M2-C) | ✅ Done | Result + `?` + stdlib (json/time/math/str) |
| v2.0.0-rc (M3) | Planned | meta/plan engine + LSP |
| v2.0.0 (M4) | Planned | MCP server + full stdlib |

## Next: M2-B (Bytecode VM)

See `docs/superpowers/specs/2026-07-01-funny-v2-ai-native-language-design.md` §6.3.