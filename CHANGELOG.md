# Changelog

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