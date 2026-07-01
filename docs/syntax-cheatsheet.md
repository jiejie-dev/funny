# Funny v2 Syntax Cheatsheet

## Variables & Assignment

```
let x = 42                    # inferred type
let name: str = "hello"       # explicit type
x = 100                       # reassignment
```

## Control Flow

```
if x > 0:
    print("positive")
elif x == 0:
    print("zero")
else:
    print("negative")

for i in [1, 2, 3]:
    print(i)

while x > 0:
    x = x - 1

match status:
    200 => print("ok")
    404 => print("not found")
    _   => print("other")
```

## Functions

```
fn add(a: int, b: int) -> int:
    return a + b

pub fn greet(name: str) -> str:
    return "hello " + name
```

## Data Structures

```
struct User:
    name: str
    age: int

let u = User(name: "alice", age: 30)
```

## Lists & Maps

```
let xs = [1, 2, 3]
let m  = {"key": "value"}
let x  = xs[0]
let v  = m["key"]
```

## Strings & F-strings

```
let s = "hello"
let n = to_str(42)
let f = f"hello {name}"
```

## Comments

```
# single-line comment
## documentation comment
```

## Operators

| Category | Operators |
|---|---|
| Arithmetic | `+` `-` `*` `/` `%` |
| Comparison | `==` `!=` `<` `>` `<=` `>=` |
| Logical | `and` `or` `not` |
| Other | `in` (e.g. `x in [1,2,3]`) |

## Builtins

| Function | Description |
|---|---|
| `print(...)` | Print values (no newline) |
| `println(...)` | Print values + newline |
| `len(x)` | Length of string or list |
| `to_str(x)` | Convert to string |
| `to_int(x)` | Convert to int |
| `type_of(x)` | Type name as string |

## Indentation Rules

- **Spaces only** (4 spaces recommended, tabs are forbidden)
- **Same indent** = same block level
- **Increase indent** = new block (after `:`)
- **Decrease indent** = end of block

## Block-opening Syntax

```
if cond:
    ...
elif cond:
    ...
else:
    ...

for x in iter:
    ...

while cond:
    ...

fn name(params) -> ret:
    ...

struct Name:
    field: type
    ...

meta:
    key: value

plan "name":
    step "name":
        ...
```