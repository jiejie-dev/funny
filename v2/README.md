# Funny v2

AI-native scripting language. See `../docs/superpowers/specs/2026-07-01-funny-v2-ai-native-language-design.md` for the full design.

**Status: M1 (syntax skeleton)** — lexer + parser + tree-walking evaluator, no type checking.

## Build

```bash
cd v2
go build -o funny ./cmd/funny
./funny run testdata/integration/fib.fn
```

## Test

```bash
go test ./...
```