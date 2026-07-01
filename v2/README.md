# Funny v2 (M2-B)

AI-native scripting language. See `../docs/superpowers/specs/2026-07-01-funny-v2-ai-native-language-design.md` for the full design.

**Status: M2-B.5 (VM Functions + Data Ops) — RELEASED**

- ✅ Lexer, Parser, Type checker (M1, M2-A)
- ✅ Tree-walking evaluator (fallback via `FUNNY_INTERPRET=1`)
- ✅ **Bytecode VM**: stack + frames, typed instructions
- ✅ **VM function calls**: CALL/RETURN + frame push/pop
- ✅ **VM builtins**: print/println/len/to_str/to_int/type_of via CALL_BUILTIN
- ✅ **VM data structures**: BUILD_LIST/INDEX/BUILD_MAP/GET_FIELD/NEW_STRUCT
- ✅ **Compiler**: function declarations, calls, list/field/index, struct literals, for-in
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

## M2-B.5 Performance

Recursive `fib(20)` benchmark (Apple M2 Max, go1.25.1, `-benchtime=3s`):

```
BenchmarkFib_VM-12           2,109,322 ns/op  (1694 iterations)
BenchmarkFib_Interpreter-12  7,337,193 ns/op  (492 iterations)
```

VM is ~3.5× faster than the tree-walking interpreter on the recursive fib workload. The 5× target is not yet met; expected gains will come from inlining the dispatch loop and reducing per-instruction overhead in a follow-up. Run locally:

```bash
go test -bench=BenchmarkFib -benchtime=3s -run=^$ ./internal/vm/
```

End-to-end demo:

```bash
$ ./funny run ./testdata/vm/fib.fn
fib(20) = 6765
```

## Roadmap

| Version | Status | Highlights |
|---|---|---|
| v2.0.0-alpha (M1) | ✅ Done | Lexer + Parser + Evaluator (no types) |
| v2.0.0-beta (M2-A) | ✅ Done | Type system + type checker |
| v2.0.0-beta (M2-B) | ✅ Done | Bytecode VM (literals, arithmetic, control flow) |
| v2.0.0-beta (M2-B.5) | ✅ Done | VM function calls + data structures (~3.5× interpreter) |
| v2.0.0-beta (M2-C) | Planned | Result + `?` + stdlib |
| v2.0.0-rc (M3) | Planned | meta/plan engine + LSP |
| v2.0.0 (M4) | Planned | MCP server + full stdlib |

## Next: M2-B (Bytecode VM)

See `docs/superpowers/specs/2026-07-01-funny-v2-ai-native-language-design.md` §6.3.