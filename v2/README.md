# Funny v2 (M2-B)

AI-native scripting language. See `../docs/superpowers/specs/2026-07-01-funny-v2-ai-native-language-design.md` for the full design.

**Status: M3 (Agent Protocol) — RELEASED**

- ✅ M1–M2-C (lex, parse, types, VM, Result+?, stdlib)
- ✅ **Plan engine**: sequential/parallel/branch steps with retry
- ✅ **meta block** type validation (name/version required)
- ✅ **stdlib extensions**: regex, env, file
- ✅ **CLI `describe`**: JSON visualization of plan/metadata
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

## M3 Usage

Plans and metadata enable agent-driven execution. Demo:

```bash
$ ./funny describe ./testdata/agent/plan.fn
{
  "meta": {
    "name": "demo_plan",
    "version": "1.0"
  },
  "plan": {
    "name": "demo_plan",
    "steps": ["setup", "compute", "verify"]
  }
}
```

The plan engine executes steps in order, with support for parallel branches, retry, and guard steps. Available stdlib includes regex, env, and file operations on top of M2-C's json/time/math/str.

## Roadmap

| Version | Status | Highlights |
|---|---|---|
| v2.0.0-alpha (M1) | ✅ Done | Lexer + Parser + Evaluator (no types) |
| v2.0.0-beta (M2-A) | ✅ Done | Type system + type checker |
| v2.0.0-beta (M2-B) | ✅ Done | Bytecode VM (literals, arithmetic, control flow) |
| v2.0.0-beta (M2-B.5) | ✅ Done | VM function calls + data structures (~3.5× interpreter) |
| v2.0.0-beta (M2-C) | ✅ Done | Result + `?` + stdlib (json/time/math/str) |
| v2.0.0-rc (M3) | ✅ Done | Plan engine + agent protocol + extended stdlib |
| v2.0.0 (M4) | Planned | MCP server + full stdlib |

## Next: M2-B (Bytecode VM)

See `docs/superpowers/specs/2026-07-01-funny-v2-ai-native-language-design.md` §6.3.