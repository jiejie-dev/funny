# Funny v2 Language Manual

Complete reference for Funny v2 (M1–M3).

## Lexical Elements

- **Indentation**: 4 spaces per level. Tabs are forbidden (runtime panic).
- **Identifiers**: `[a-zA-Z_][a-zA-Z0-9_]*`
- **Numbers**: `int` (decimal or `0x` hex), `float64`
- **Strings**: `"..."` or `'...'` with `\n \t \\ \" \'` escapes
- **F-strings**: `f"hello {name}"` — full `{expr}` interpolation with optional Python/Rust-flavored format specs, e.g. `f"{price:.2f}"`, `f"{n:>10}"` (see [Format Strings](#format-strings))
- **Comments**: `#` line comment, `##` doc comment (for agent metadata)
- **Operators**: `+ - * / % == != < > <= >= and or not in`
- **Punctuation**: `( ) [ ] { } , : . -> ?`

## Format Strings

F-strings (`f"..."`) support `{expr}` interpolation: any expression may appear
inside `{}`, and its value is converted to a string and spliced into the
result.

```
let name = "world"
let price = 19.5
println(f"hello {name}! total: {price:.2f}")   # hello world! total: 19.50
```

Use `{{` and `}}` to embed a literal brace:

```
println(f"{{literal braces}}")   # {literal braces}
```

### Format spec

An optional `:spec` after the expression controls how the value is rendered,
following a Python/Rust-flavored mini-grammar:

```
{expr:[[fill]align][sign][0][width][.precision][type]}
```

| Field | Values | Meaning |
|---|---|---|
| `fill` | any single char | padding character (default: space); only valid with an explicit `align` |
| `align` | `<` `>` `^` | left / right / center within `width` (default: `<` for strings, `>` for numbers) |
| `sign` | `+` | force a leading `+` on non-negative numbers |
| `0` | `0` | zero-pad shorthand (equivalent to fill `0`, align `>`) |
| `width` | decimal digits | minimum field width |
| `.precision` | `.` + decimal digits | decimal places for `f`/`%`; max length for `s`/default |
| `type` | `d f x X o b s %` | integer, fixed-point float, hex (lower/upper), octal, binary, string, percent |

Examples:

```
f"{n:5d}"      # right-aligned int in a 5-wide field:  "   42"
f"{n:05d}"     # zero-padded:                          "00042"
f"{pi:.2f}"    # fixed-point, 2 decimals:               "3.14"
f"{x:>10}"     # right-align in a 10-wide field
f"{x:^10}"     # center in a 10-wide field
f"{255:X}"     # uppercase hex:                          "FF"
f"{0.5:%}"     # percent:                          "50.000000%"
```

Omitting the spec (`{expr}`) falls back to the same default stringification
used by `to_str`/`println` (`true`/`false` for bools, `nil` for nil).

## Types

- **Primitives**: `int float bool str nil`
- **Composite**: `list[T]`, `map[K, V]`, `Result[T, E]`
- **Nullable**: `T?`
- **Function**: `(P1, P2) -> R`
- **Struct**: declared via `struct Name: field: T, ...`

## Declarations

### Variables
```
let x = 42                       # type inferred as int
let name: str = "hello"          # explicit type
let items: list[int] = [1, 2, 3] # explicit type
```

### Collections

List literals use `[...]`; map literals use `{key: value, ...}`. Both infer
their element/key/value types from the first entry when there's no explicit
annotation, and require an annotation when empty (`let xs: list[int] = []`,
`let m: map[str, int] = {}`).

```
let xs = [1, 2, 3]
let m: map[str, int] = {"a": 1, "b": 2}
```

Any bracketed literal - `[...]`, `(...)`, and `{...}` - may span multiple
lines; a newline inside an open bracket is insignificant whitespace, so the
usual convention is one entry per line ending with a trailing comma:

```
let m: map[str, int] = {
    "a": 1,
    "b": 2,
    "c": 3,
}
```

Map values are accessed with `.field` (like a struct), not `m["key"]`
indexing:

```
println(m.a)  # 1
```

### Functions
```
fn add(a: int, b: int) -> int:
    return a + b

pub fn greet(name: str) -> str:
    return "hello " + name
```

### Structs
```
struct User:
    name: str
    age: int

let u = User(name: "alice", age: 30)
println(u.name)  # field access
```

## Control Flow

### If
```
if x > 0:
    print("positive")
elif x == 0:
    print("zero")
else:
    print("negative")
```

### Loops
```
for i in [1, 2, 3]:
    print(i)

while x > 0:
    x = x - 1
```

### Match
```
match status:
    200 => print("ok")
    404 => print("not found")
    _   => print("other")
```

## Result + `?` Operator

`Result[T, E]` is a tagged union: Ok(value) or Err(error). The `?` postfix unwraps Ok or returns Err from the enclosing function.

```
fn divide(a: int, b: int) -> Result:
    if b == 0:
        return err("divide by zero")?
    return ok(a / b)?

let r = divide(10, 2)?
if r.tag == "err":
    print("error: " + r.val)
else:
    print("result: " + r.val)
```

## Plans (Agent Protocol)

```
meta:
    name = "my_skill"
    version = "1.0"

plan "my_skill":
    step "setup":
        let x = 1
    step "compute" -> tool with retry max=3:
        let r = x * 2
    step "verify" -> guard:
        if r > 0:
            pass
```

## Builtin Functions

| Function | Description |
|---|---|
| `print(...)` | Print to stdout (no newline) |
| `println(...)` | Print with newline |
| `len(x)` | Length of string or list |
| `to_str(x)` | Convert to string |
| `to_int(x)` | Convert to int |
| `type_of(x)` | Type name as string |
| `ok(x)` / `err(x)` | Construct Result |
| `regex_match(p, t)` | Test regex |
| `regex_replace(p, t, r)` | Replace matches |
| `env_get(name)` | Read environment variable |
| `file_read(path)` | Read file (returns Result) |
| `file_exists(path)` | Test file existence |
| `http_get(url)` | HTTP GET (returns Result) |
| `md5(s)` / `sha256(s)` | Hash functions |
| `b64_encode(s)` / `b64_decode(s)` | Base64 encoding |
| `jwt_encode(h, c, s)` | Sign JWT (HS256) |
| `jwt_decode(t, s)` | Verify and decode JWT |
| `sql_open(path)` | Open SQLite database |

## CLI Usage

```bash
funny run script.fn         # execute
funny ast script.fn         # JSON AST
funny fmt script.fn         # print canonically-formatted source to stdout
funny fmt script.fn -w      # reformat the file in place
funny describe script.fn    # JSON plan/metadata
funny-mcp                   # start MCP server
```

## MCP Server

The `funny-mcp` binary exposes 6 tools over stdio:
- `ast`: parse source, return JSON AST
- `format`: format source code (canonical 4-space indentation, preserves comments)
- `list_skills`: list .fn files in a directory
- `describe_skill`: meta + plan info for one file
- `run_skill`: execute a .fn file
- `lint`: type-check only, no execution
```