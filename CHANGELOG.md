# Changelog

## v2.1.0 (unreleased)

### Features
- **F-string interpolation**: full `f"...{expr:spec}..."` support (lexer/parser/type checker/evaluator/bytecode VM) with a Python/Rust-flavored format-spec mini-language (`internal/strfmt`)
- **Real formatter**: AST-based `funny fmt` / MCP `format` tool (previously a no-op), preserving comments
- **Map literals**: `{key: value, ...}` construction, previously impossible (the type existed only as a `map[K, V]` annotation with no way to build a value)
- **Bracket line-continuation**: any `(...)`, `[...]`, or `{...}` may now span multiple lines; a newline inside an open bracket is insignificant whitespace, enabling the conventional one-entry-per-line-with-trailing-comma style for map/list literals and call arguments
- **`m[key]` indexing**: map values can now be read and written with bracket indexing (`m["a"]`, `m["a"] = 1`), in addition to `.field` access; index assignment also now works for lists (`xs[0] = 1`), via a new `SET_INDEX` bytecode instruction
- **Real module loading**: `import "path.fn"` (previously a pure no-op) now actually reads, parses, and merges the target file's `pub fn`/`struct` declarations, relative to the importing file's directory (`internal/module`). Unaliased imports merge `pub` symbols under their bare name (`add(1, 2)`); `import "path.fn" as m` keeps the module's own names unchanged and instead teaches call sites that `m.add(...)` means "call `add` from that module" (à la Python's `import x as y`). Private (non-`pub`) functions remain reachable from within their own module but are hygienically renamed and inaccessible from outside it. Diamond dependencies are merged once; circular imports and cross-file symbol collisions are compile errors (`E1101`-`E1105`).
- **LSP server** (`cmd/funny-lsp`, `internal/lsp`): a from-scratch LSP 3.17 implementation (no third-party protocol dependency), replacing the v1 scaffolding that was dropped in the v2 migration. Diagnostics (parser/module/type-checker errors, precisely positioned, with error codes), hover (types, function signatures, struct layouts, builtin/keyword docs), type-aware completion (locals/functions/structs/builtins/keywords generally, struct-fields-only right after `<expr>.`), signature help, go-to-definition (including across `import`ed files), document symbols (`fn`/`struct`/`plan`+`step` outline), formatting (delegates to the `funny fmt` formatter), find-references and rename (`textDocument/references`, `textDocument/prepareRename`/`rename`, scoped to the requested document, with function-level scoping for locals), and a custom `funny/planGraph` request that renders a `plan` block as a node/edge graph mirroring the plan engine's actual sequential/parallel execution semantics. See "LSP Server" in `docs/language-manual.md` for the full capability list and known scoping trade-offs.

### Fixes
- `funny describe` was documented (README/language-manual/CLI help) and its underlying implementation (`cli.Describe`) existed and was unit-tested, but no `cobra.Command` in `cmd/funny/main.go` ever called it — the subcommand didn't exist on the CLI at all. Wired it up alongside `run`/`ast`/`fmt`
- Parser crash on standalone `#` comments (introduces `ast.CommentStmt`)
- Lexer bug where dedenting across multiple nesting levels to a non-zero column only emitted one DEDENT instead of all required levels
- Bytecode compiler crash (`unsupported statement type`) on any script containing a `meta:` or `plan "...":` block when run via the default VM path (`funny run` without `FUNNY_INTERPRET=1`) — these are now no-ops in the compiler, matching the tree-walking evaluator
- Type checker rejected every struct-typed annotation (`let p: Point = ...`, `fn f(p: Point)`, `fn f() -> Point`, struct fields, `list[Point]`, ...) with a spurious type mismatch, because `ParseType` has no environment access and could only produce an opaque `Primitive("Point")` for a bare struct name instead of the real `Struct` type; type annotations are now resolved against the environment so struct names compare correctly everywhere
- Bytecode compiler crash (`vm: unsupported op LOAD_GLOBAL`) for any script that declares a top-level `fn` in between declaring and later referencing a top-level variable (e.g. `let p = ...` / `fn foo(): ...` / `println(p)`). `compileFnDecl` was resetting the compiler's local-scope table to a brand-new empty map after compiling each function body instead of saving/restoring the enclosing scope, permanently losing track of every local declared before that point; it also never isolated `varTypes` (local slot → value type) per function, so a variable and an unrelated function parameter sharing the same slot number could silently corrupt each other's recorded type and mis-codegen type-sensitive operators like `+`
- Lexer reported the wrong column (and byte offset) for every token except the first on a given line: `Position` was only captured once per line, during indentation handling, and silently reused for every later token on that line instead of being refreshed per-token. Every consumer that only ever looked at line numbers (error messages, `ast` output) was unaffected, but this made column-accurate tooling like the new LSP server impossible until fixed; caught while implementing hover/completion/go-to-definition, which need real per-token columns

## v2.0.0 (2026-07-XX)

### Highlights
- Complete v2 stack: lexer, parser, type checker, bytecode VM, stdlib
- AI-native design: indentation-based syntax, strong typing, agent protocol
- `Result` + `?` operator for error propagation
- Plan engine with retry, parallel, branch, guard step kinds
- MCP server for LLM integration
- Standard library: json, time, math, str, regex, env, file, http, crypto, jwt, sql

### Features
- **Lexer**: INDENT/DEDENT/NEWLINE, 59 token types, escapes
- **Parser**: Pratt expressions, control flow, function/struct declarations
- **Type System**: 7 type kinds, recursive-descent annotation parser, type checker
- **VM**: typed bytecode, stack-based, frame support, 45 instructions
- **Bytecode Compiler**: 3.5× faster than tree-walking interpreter
- **Stdlib**: json, time, math, str, regex, env, file, http, crypto, jwt, sql
- **Agent Protocol**: meta block, plan block, 6 step kinds, plan engine
- **MCP Server**: 6 tools (ast, format, list_skills, describe_skill, run_skill, lint)

### Limitations (v2.0.x follow-ups)
- 5× interpreter target not yet met (currently 3.5×)
- AI-friendliness benchmark requires community LLM evaluation
- Map literal AST (`{"k": v}`) parser syntax needs explicit braces
- Formatting tool is a no-op (v2.1 will add real formatting)