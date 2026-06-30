# Funny v2 (M2-B)

AI-native scripting language. See `../docs/superpowers/specs/2026-07-01-funny-v2-ai-native-language-design.md` for the full design.

**Status: M2-B (Bytecode VM) — RELEASED**

- ✅ Lexer, Parser, Type checker (M1, M2-A)
- ✅ Tree-walking evaluator (fallback via `FUNNY_INTERPRET=1`)
- ✅ **Bytecode compiler**: typed instructions per spec §5.4
- ✅ **Stack-based VM**: operand stack + frame stack
- ✅ **VM instructions**: arithmetic, comparison, logical, control flow
- ⏳ Function calls (CALL/RETURN) → M2-B.5 follow-up
- ⏳ Data structure ops (BUILD_LIST, GET_FIELD, NEW_STRUCT) → M2-B.5 follow-up
- ⏳ Result + `?` operator → M2-C
- ⏳ stdlib (json/time/math/str) → M2-C

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

## M2-B Performance

```
BenchmarkFib_VM-12           5287 ns/op
BenchmarkFib_Interpreter-12  6408 ns/op
```

VM is currently ~1.2× faster than the tree-walking interpreter on iterative workloads. Recursive performance benefits will materialize with M2-B.5 (function calls). Target ≥ 5× will be re-evaluated once CALL/RETURN land.

Run benchmarks locally:
```bash
go test -bench=BenchmarkFib -benchtime=2s -run=^$ ./internal/vm/
```

## Roadmap

| Version | Status | Highlights |
|---|---|---|
| v2.0.0-alpha (M1) | ✅ Done | Lexer + Parser + Evaluator (no types) |
| v2.0.0-beta (M2-A) | ✅ Done | Type system + type checker |
| v2.0.0-beta (M2-B) | ✅ Done | Bytecode VM (~1.2× interpreter; recursion deferred) |
| v2.0.0-beta (M2-B.5) | Planned | VM function calls + data ops |
| v2.0.0-beta (M2-C) | Planned | Result + `?` + stdlib |
| v2.0.0-rc (M3) | Planned | meta/plan engine + LSP |
| v2.0.0 (M4) | Planned | MCP server + full stdlib |

## Next: M2-B (Bytecode VM)

See `docs/superpowers/specs/2026-07-01-funny-v2-ai-native-language-design.md` §6.3.