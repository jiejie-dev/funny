# Funny v2 Language Manual

Complete reference for Funny v2 (M1–M3).

## Lexical Elements

- **Indentation**: 4 spaces per level. Tabs are forbidden (runtime panic).
- **Identifiers**: `[a-zA-Z_][a-zA-Z0-9_]*`
- **Numbers**: `int` (decimal or `0x` hex), `float64`
- **Strings**: `"..."` or `'...'` with `\n \t \\ \" \'` escapes
- **F-strings**: `f"hello {name}"` (interpolation deferred to v2.1)
- **Comments**: `#` line comment, `##` doc comment (for agent metadata)
- **Operators**: `+ - * / % == != < > <= >= and or not in`
- **Punctuation**: `( ) [ ] { } , : . -> ?`

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
    name: "my_skill"
    version: "1.0"

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
funny describe script.fn    # JSON plan/metadata
funny-mcp                   # start MCP server
```

## MCP Server

The `funny-mcp` binary exposes 6 tools over stdio:
- `ast`: parse source, return JSON AST
- `format`: format source (M4.5: no-op)
- `list_skills`: list .fn files in a directory
- `describe_skill`: meta + plan info for one file
- `run_skill`: execute a .fn file
- `lint`: type-check only, no execution
```