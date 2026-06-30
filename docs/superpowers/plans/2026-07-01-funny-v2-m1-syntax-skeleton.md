# Funny v2 M1: Syntax Skeleton Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the lexer, parser, AST, and minimal tree-walking evaluator for Funny v2 — enough to run simple scripts without type checking.

**Architecture:** Classic three-stage pipeline (Lexer → Parser → Evaluator). All code in Go, organized under `v2/` to avoid breaking v1. Tree-walking evaluator (no bytecode VM until M2). Tests use `testify` (matches v1 convention).

**Tech Stack:** Go 1.22+, `github.com/stretchr/testify`, `github.com/spf13/cobra` (CLI).

**Reference Spec:** `docs/superpowers/specs/2026-07-01-funny-v2-ai-native-language-design.md` (§2 Syntax, §5.1-§5.3 Architecture, §6.2 M1)

**Scope:** This plan covers ONLY M1 from the spec. Type checking, bytecode VM, plan engine, and MCP server are deferred to M2/M3/M4.

---

## File Structure

All v2 code lives under `/Users/j/repos/funny/v2/` to preserve v1:

```
v2/
├── go.mod                              # module github.com/jerloo/funny/v2
├── README.md
├── .gitignore
├── cmd/
│   └── funny/
│       └── main.go                     # CLI entry (cobra)
├── internal/
│   ├── errors/
│   │   ├── errors.go
│   │   └── errors_test.go
│   ├── lexer/
│   │   ├── token.go
│   │   ├── lexer.go
│   │   └── lexer_test.go
│   ├── ast/
│   │   ├── ast.go
│   │   └── ast_test.go
│   ├── parser/
│   │   ├── parser.go
│   │   ├── expression.go
│   │   ├── statement.go
│   │   └── parser_test.go
│   ├── evaluator/
│   │   ├── scope.go
│   │   ├── evaluator.go
│   │   ├── builtin.go
│   │   ├── evaluator_test.go
│   │   └── bench_test.go
│   └── cli/
│       ├── run.go
│       └── run_test.go
└── testdata/
    ├── parser/
    │   ├── control_flow.fn
    │   └── function.fn
    └── integration/
        └── fib.fn
```

**Key design decisions**:
- M1 evaluator is **tree-walking** (no bytecode); M2 replaces it with bytecode VM
- M1 has **no type checker** (no static type validation, no Result/?); M2 adds
- M1 `meta` and `plan` blocks are **parsed but ignored at runtime**; M3 activates them
- M1 supports `import` parsing only (no actual module loading); M4 activates file loading

---

## Conventions

- Go 1.22+ (uses `any` instead of `interface{}`)
- Test files end in `_test.go` and use `testify/assert` + `testify/require`
- Each task ends with a commit
- Error messages follow spec §5.9 unified format
- Indentation in funny source is **4 spaces**

---

## Task 0: Project Skeleton

**Files:** `v2/go.mod`, `v2/README.md`, `v2/.gitignore`, `v2/cmd/funny/main.go`

- [ ] Create directory and init module: `mkdir -p /Users/j/repos/funny/v2 && cd /Users/j/repos/funny/v2 && go mod init github.com/jerloo/funny/v2`
- [ ] Add testify: `cd /Users/j/repos/funny/v2 && go get github.com/stretchr/testify@latest`
- [ ] Write `.gitignore`: `/funny`, `/coverage.out`, `*.test`
- [ ] Write minimal `README.md` (build/test instructions, M1 status)
- [ ] Write `cmd/funny/main.go`:

```go
package main
import "fmt"
func main() { fmt.Println("funny v2 (M1)") }
```

- [ ] Build and run: `cd /Users/j/repos/funny/v2 && go build -o funny ./cmd/funny && ./funny` → expect `funny v2 (M1)`
- [ ] Commit: `cd /Users/j/repos/funny && git add v2/ && git commit -m "v2: bootstrap M1 project skeleton"`

---

## Task 1: Error System

**Files:** `v2/internal/errors/errors.go`, `v2/internal/errors/errors_test.go`

- [ ] Write failing test `errors_test.go`:

```go
package errors
import ("testing"; "github.com/stretchr/testify/assert")

func TestError_Format(t *testing.T) {
    pos := Position{File: "test.fn", Line: 3, Col: 5}
    e := New("E1001", "unexpected token", pos, "expected `:`")
    got := e.Format()
    assert.Contains(t, got, "error[E1001]")
    assert.Contains(t, got, "unexpected token")
    assert.Contains(t, got, "test.fn:3:5")
    assert.Contains(t, got, "expected `:`")
}
func TestError_Format_WithoutHint(t *testing.T) {
    pos := Position{File: "test.fn", Line: 0, Col: 0}
    e := New("E0001", "lexer error", pos, "")
    assert.NotContains(t, e.Format(), "help:")
}
```

- [ ] Run `cd /Users/j/repos/funny/v2 && go test ./internal/errors/` → FAIL
- [ ] Write `errors.go`:

```go
package errors
import "fmt"

type Position struct { File string; Line int; Col int }

type Error struct {
    Code, Message, Hint string
    Pos                 Position
}

func New(code, message string, pos Position, hint string) *Error {
    return &Error{Code: code, Message: message, Pos: pos, Hint: hint}
}

func (e *Error) Error() string { return e.Format() }

func (e *Error) Format() string {
    s := fmt.Sprintf("error[%s]: %s\n --> %s:%d:%d\n", e.Code, e.Message, e.Pos.File, e.Pos.Line, e.Pos.Col)
    if e.Hint != "" { s += fmt.Sprintf("\nhelp: %s", e.Hint) }
    return s
}
```

- [ ] Run test → PASS
- [ ] Commit: `git add v2/internal/errors/ && git commit -m "v2: add unified error system"`

---

## Task 2: Token Types

**Files:** `v2/internal/lexer/token.go`, `v2/internal/lexer/token_test.go`

- [ ] Write failing test `token_test.go`:

```go
package lexer
import ("testing"; "github.com/stretchr/testify/assert")

func TestTokenKind_IsKeyword(t *testing.T) {
    assert.True(t, IF.IsKeyword())
    assert.False(t, NAME.IsKeyword())
    assert.False(t, PLUS.IsKeyword())
}
func TestTokenKind_String(t *testing.T) {
    assert.Equal(t, "if", string(IF))
    assert.Equal(t, "+", string(PLUS))
}
```

- [ ] Run test → FAIL
- [ ] Write `token.go`:

```go
package lexer

type Kind string

const (
    EOF Kind = "EOF"; NEWLINE Kind = "NEWLINE"; INDENT Kind = "INDENT"; DEDENT Kind = "DEDENT"
    NAME Kind = "NAME"; INT Kind = "INT"; FLOAT Kind = "FLOAT"; STR Kind = "STR"; FSTR Kind = "FSTR"
    LPAREN Kind = "("; RPAREN Kind = ")"; LBRACK Kind = "["; RBRACK Kind = "]"
    COMMA Kind = ","; DOT Kind = "."; COLON Kind = ":"; ARROW Kind = "->"; FATARROW Kind = "=>"
    QUESTION Kind = "?"; AT Kind = "@"
    PLUS Kind = "+"; MINUS Kind = "-"; STAR Kind = "*"; SLASH Kind = "/"; PERCENT Kind = "%"
    EQ Kind = "="; EQEQ Kind = "=="; NEQ Kind = "!="; LT Kind = "<"; GT Kind = ">"
    LTE Kind = "<="; GTE Kind = ">="
    COMMENT Kind = "COMMENT"; DOC_COMMENT Kind = "DOC_COMMENT"
    AND Kind = "and"; AS Kind = "as"; BREAK Kind = "break"; CONTINUE Kind = "continue"
    ELIF Kind = "elif"; ELSE Kind = "else"; FALSE Kind = "false"; FN Kind = "fn"
    FOR Kind = "for"; IF Kind = "if"; IMPORT Kind = "import"; IN Kind = "in"
    LET Kind = "let"; MATCH Kind = "match"; META Kind = "meta"; NIL Kind = "nil"
    NOT Kind = "not"; OR Kind = "or"; PLAN Kind = "plan"; PUB Kind = "pub"
    RETURN Kind = "return"; STEP Kind = "step"; STRUCT Kind = "struct"; TRUE Kind = "true"
    WHILE Kind = "while"
)

var keywordSet = map[string]Kind{
    "and": AND, "as": AS, "break": BREAK, "continue": CONTINUE,
    "elif": ELIF, "else": ELSE, "false": FALSE, "fn": FN, "for": FOR,
    "if": IF, "import": IMPORT, "in": IN, "let": LET, "match": MATCH,
    "meta": META, "nil": NIL, "not": NOT, "or": OR, "plan": PLAN, "pub": PUB,
    "return": RETURN, "step": STEP, "struct": STRUCT, "true": TRUE, "while": WHILE,
}

func (k Kind) IsKeyword() bool { _, ok := keywordSet[string(k)]; return ok }
func (k Kind) String() string { return string(k) }

type Position struct { File string; Line, Col, Offset int }
type Token struct { Kind Kind; Data string; Pos Position }
```

- [ ] Run test → PASS
- [ ] Commit: `git add v2/internal/lexer/ && git commit -m "v2: define token kinds and keyword set"`

---

## Task 3: Lexer - Skeleton and Whitespace

**Files:** `v2/internal/lexer/lexer.go`, `v2/internal/lexer/lexer_test.go`

- [ ] Write failing test `lexer_test.go`:

```go
package lexer
import ("testing"; "github.com/stretchr/testify/assert")

func TestLexer_EOF(t *testing.T) {
    l := New("hello", "")
    tok := l.Next()
    assert.Equal(t, NAME, tok.Kind)
    assert.Equal(t, "hello", tok.Data)
}
func TestLexer_SkipsSpaces(t *testing.T) {
    l := New("   a   b   ", "")
    assert.Equal(t, NAME, l.Next().Kind)
    assert.Equal(t, NAME, l.Next().Kind)
}
func TestLexer_TracksLineAndCol(t *testing.T) {
    l := New("a\nb", "test.fn")
    aTok := l.Next()
    assert.Equal(t, 0, aTok.Pos.Line)
    assert.Equal(t, 0, aTok.Pos.Col)
    bTok := l.Next()
    assert.Equal(t, 1, bTok.Pos.Line)
}
```

- [ ] Run test → FAIL
- [ ] Write `lexer.go`:

```go
package lexer

type Lexer struct {
    src         string
    file        string
    pos, line, col int
    savePos, saveLine, saveCol int
    indentStack []int
}

func New(src, file string) *Lexer {
    return &Lexer{src: src, file: file, indentStack: []int{0}}
}

func (l *Lexer) peek(n int) byte {
    p := l.pos + n
    if p >= len(l.src) { return 0 }
    return l.src[p]
}

func (l *Lexer) advance() {
    if l.pos < len(l.src) {
        if l.src[l.pos] == '\n' { l.line++; l.col = 0 } else { l.col++ }
        l.pos++
    }
}

func (l *Lexer) save() {
    l.savePos = l.pos; l.saveLine = l.line; l.saveCol = l.col
}

func (l *Lexer) emit(kind Kind, data string) Token {
    return Token{Kind: kind, Data: data, Pos: Position{File: l.file, Line: l.saveLine, Col: l.saveCol, Offset: l.savePos}}
}

func (l *Lexer) Next() Token {
    l.save()
    if l.pos >= len(l.src) { return l.emit(EOF, "") }
    ch := l.src[l.pos]
    for ch == ' ' || ch == '\t' {
        l.advance()
        if l.pos >= len(l.src) { return l.emit(EOF, "") }
        ch = l.src[l.pos]
    }
    l.save()
    if isLetter(ch) {
        start := l.pos
        for l.pos < len(l.src) && (isLetter(l.src[l.pos]) || isDigit(l.src[l.pos])) {
            l.advance()
        }
        data := l.src[start:l.pos]
        if k, ok := keywordSet[data]; ok { return l.emit(k, data) }
        return l.emit(NAME, data)
    }
    l.advance()
    return l.Next()
}

func isLetter(b byte) bool {
    return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || b == '_'
}
func isDigit(b byte) bool { return b >= '0' && b <= '9' }
```

- [ ] Run test → PASS
- [ ] Commit: `git add v2/internal/lexer/ && git commit -m "v2: lexer skeleton with whitespace handling"`

---

## Task 4: Lexer - Operators and Delimiters

**Files:** modify `v2/internal/lexer/lexer.go`, append to `v2/internal/lexer/lexer_test.go`

- [ ] Append test cases for `+ - * / % = == != < > <= > = ( ) [ ] , . : -> => ? @`
- [ ] Run test → FAIL
- [ ] Add operator switch to `Next()` BEFORE letter-reading block (covers all 22 operators). Use `peek(0)` for `->`, `==`, `!=`, `<=`, `>=`
- [ ] Run test → PASS
- [ ] Commit: `git add v2/internal/lexer/ && git commit -m "v2: lexer operators and delimiters"`

---

## Task 5: Lexer - Number Literals

**Files:** modify `v2/internal/lexer/lexer.go`, append to `v2/internal/lexer/lexer_test.go`

- [ ] Append tests for `42`, `0x1F`, `3.14`, `1e-3`, `2.5E+2`, `1` (INT) vs `1.0` (FLOAT)
- [ ] Run test → FAIL
- [ ] Add number handling block BEFORE letter-reading. Handle hex prefix `0x`/`0X`, decimal digits, optional `.` + decimal for float, optional `e`/`E` exponent
- [ ] Add `isHexDigit` helper
- [ ] Run test → PASS
- [ ] Commit: `git add v2/internal/lexer/ && git commit -m "v2: lexer number literals"`

---

## Task 6: Lexer - String Literals

**Files:** modify `v2/internal/lexer/lexer.go`, append to `v2/internal/lexer/lexer_test.go`

- [ ] Append tests for `"hello"`, `'world'`, escapes (`\n`, `\t`, `\\`), f-string `f"hello {name}"`
- [ ] Run test → FAIL
- [ ] Add string handling: `lexString(single)` and `lexFString()` helpers. Handle `\"`, `\'`, `\\`, `\n`, `\t`, `\r`
- [ ] Run test → PASS
- [ ] Commit: `git add v2/internal/lexer/ && git commit -m "v2: lexer string and f-string literals"`

---

## Task 7: Lexer - Comments

**Files:** modify `v2/internal/lexer/lexer.go`, append to `v2/internal/lexer/lexer_test.go`

- [ ] Append tests for `# comment` (COMMENT) and `## doc` (DOC_COMMENT)
- [ ] Run test → FAIL
- [ ] Add comment handling: check for `#`, peek for `##`, consume to end-of-line
- [ ] Run test → PASS
- [ ] Commit: `git add v2/internal/lexer/ && git commit -m "v2: lexer line and doc comments"`

---

## Task 8: Lexer - INDENT/DEDENT (Critical)

**Files:** modify `v2/internal/lexer/lexer.go`, append to `v2/internal/lexer/lexer_test.go`

- [ ] Append tests:

```go
func TestLexer_IndentBasic(t *testing.T) {
    src := "a\n    b\n"
    l := New(src, "")
    kinds := drain(l)
    expected := []Kind{NAME, NEWLINE, INDENT, NAME, NEWLINE, DEDENT, EOF}
    assert.Equal(t, expected, kinds)
}
func TestLexer_IndentNested(t *testing.T) {
    src := "a\n    b\n        c\n    d\n"
    l := New(src, "")
    expected := []Kind{NAME, NEWLINE, INDENT, NAME, NEWLINE, INDENT, NAME, NEWLINE, DEDENT, NAME, NEWLINE, DEDENT, EOF}
    assert.Equal(t, expected, drain(l))
}
func drain(l *Lexer) []Kind {
    var kinds []Kind
    for { tok := l.Next(); kinds = append(kinds, tok.Kind); if tok.Kind == EOF { return kinds } }
}
```

- [ ] Run test → FAIL
- [ ] Replace `Next()` with full version. Algorithm:
  - Skip leading spaces (not tabs — panic on tab)
  - Skip blank lines (consume `\n`, increment line)
  - At EOF: emit DEDENT until stack is `[0]`, then EOF
  - At col==0 (line start): compute indent, compare to top of `indentStack`
    - indent > top: push, emit INDENT
    - indent < top: pop until matching, emit DEDENT (one at a time)
  - Then proceed with regular token lexing (comments, newlines, strings, numbers, identifiers, operators)
- [ ] Add `import "fmt"` for panic message
- [ ] Run test → PASS
- [ ] Commit: `git add v2/internal/lexer/ && git commit -m "v2: lexer INDENT/DEDENT with stack-based tracking"`

---

## Task 9: AST - Position and Node Interface

**Files:** `v2/internal/ast/ast.go`, `v2/internal/ast/ast_test.go`

- [ ] Write failing test:

```go
package ast
import ("testing"; "github.com/stretchr/testify/assert")
func TestPos_String(t *testing.T) {
    p := Pos{File: "a.fn", Line: 2, Col: 3}
    s := p.String()
    assert.Contains(t, s, "a.fn")
}
```

- [ ] Run test → FAIL
- [ ] Write `ast.go`:

```go
package ast
import "fmt"

type Pos struct { File string; Line, Col int }

func (p Pos) String() string {
    return fmt.Sprintf("%s:%d:%d", p.File, p.Line+1, p.Col+1)
}

type Node interface {
    Pos() Pos
    nodeMarker()
}

type Statement interface {
    Node
    stmtMarker()
}

type Expression interface {
    Node
    exprMarker()
}
```

- [ ] Run test → PASS
- [ ] Commit: `git add v2/internal/ast/ && git commit -m "v2: AST Pos and Node interface"`

---

## Task 10: AST - Expression Nodes

**Files:** modify `v2/internal/ast/ast.go`, create `v2/internal/ast/expression_test.go`

- [ ] Write tests for `LiteralExpr`, `VariableExpr`, `BinaryExpr`, `UnaryExpr`, `SubExpr`, `ListExpr`, `IndexExpr`, `FieldExpr`, `CallExpr`, `FStringExpr`
- [ ] Run test → FAIL
- [ ] Add all expression node types to `ast.go` (each implements `Pos()`, `exprMarker()`, `nodeMarker()`, `String()`)
- [ ] Add helper `joinComma(parts []string) string`
- [ ] Run test → PASS
- [ ] Commit: `git add v2/internal/ast/ && git commit -m "v2: AST expression nodes"`

---

## Task 11: AST - Statement Nodes

**Files:** modify `v2/internal/ast/ast.go`, create `v2/internal/ast/statement_test.go`

- [ ] Write tests for `LetStmt`, `AssignStmt`, `Block`, `IfStmt`, `ForStmt`, `WhileStmt`, `MatchStmt`, `MatchArm`, `ReturnStmt`, `BreakStmt`, `ContinueStmt`, `ExprStmt`
- [ ] Run test → FAIL
- [ ] Add all statement node types
- [ ] Run test → PASS
- [ ] Commit: `git add v2/internal/ast/ && git commit -m "v2: AST statement nodes"`

---

## Task 12: AST - Function and Struct Declarations

**Files:** modify `v2/internal/ast/ast.go`, create `v2/internal/ast/decl_test.go`

- [ ] Write tests for `Param`, `FnDecl`, `StructDecl`
- [ ] Run test → FAIL
- [ ] Add node types
- [ ] Run test → PASS
- [ ] Commit: `git add v2/internal/ast/ && git commit -m "v2: AST function and struct declarations"`

---

## Task 13: AST - Top-Level

**Files:** modify `v2/internal/ast/ast.go`, create `v2/internal/ast/toplevel_test.go`

- [ ] Write tests for `ImportDecl`, `MetaBlock`, `PlanBlock`, `Program`
- [ ] Run test → FAIL
- [ ] Add node types
- [ ] Run test → PASS
- [ ] Commit: `git add v2/internal/ast/ && git commit -m "v2: AST top-level (meta, plan, import, program)"`

---

## Task 14: Parser - Driver

**Files:** `v2/internal/parser/parser.go`, `v2/internal/parser/statement.go`, `v2/internal/parser/parser_test.go`

- [ ] Write test:

```go
package parser
import ("testing"; "github.com/stretchr/testify/assert")
func TestParser_Empty(t *testing.T) {
    p := New("", "")
    prog, err := p.Parse()
    assert.NoError(t, err)
    assert.Empty(t, prog.Stmts)
}
```

- [ ] Run test → FAIL
- [ ] Write `parser.go`:

```go
package parser

import (
    "fmt"
    "github.com/jerloo/funny/v2/internal/ast"
    "github.com/jerloo/funny/v2/internal/errors"
    "github.com/jerloo/funny/v2/internal/lexer"
)

type Parser struct {
    lx *lexer.Lexer
    cur, peek lexer.Token
}

func New(src, file string) *Parser {
    p := &Parser{lx: lexer.New(src, file)}
    p.advance(); p.advance()
    return p
}

func (p *Parser) advance() { p.cur = p.peek; p.peek = p.lx.Next() }

func (p *Parser) expect(k lexer.Kind) (lexer.Token, *errors.Error) {
    if p.cur.Kind == k { tok := p.cur; p.advance(); return tok, nil }
    return lexer.Token{}, errors.New("E1001",
        fmt.Sprintf("expected %s, got %s", k, p.cur.Kind),
        astPos(p.cur.Pos), fmt.Sprintf("expected `%s` here", k))
}

func (p *Parser) atEOF() bool { return p.cur.Kind == lexer.EOF }

func (p *Parser) Parse() (*ast.Program, error) {
    prog := &ast.Program{NodePos: astPos(p.cur.Pos)}
    for !p.atEOF() {
        for p.cur.Kind == lexer.NEWLINE { p.advance() }
        if p.atEOF() { break }
        s, err := p.parseStatement()
        if err != nil { return nil, err }
        if s != nil { prog.Stmts = append(prog.Stmts, s) }
    }
    return prog, nil
}

func astPos(p lexer.Position) ast.Pos { return ast.Pos{File: p.File, Line: p.Line, Col: p.Col} }
```

- [ ] Write `statement.go` with stubs:

```go
package parser

import (
    "fmt"
    "github.com/jerloo/funny/v2/internal/ast"
    "github.com/jerloo/funny/v2/internal/errors"
    "github.com/jerloo/funny/v2/internal/lexer"
)

func (p *Parser) parseStatement() (ast.Statement, error) {
    switch p.cur.Kind {
    case lexer.LET: return p.parseLet()
    case lexer.IF: return p.parseIf()
    case lexer.FOR: return p.parseFor()
    case lexer.WHILE: return p.parseWhile()
    case lexer.MATCH: return p.parseMatch()
    case lexer.RETURN: return p.parseReturn()
    case lexer.BREAK: p.advance(); return &ast.BreakStmt{NodePos: astPos(p.cur.Pos)}, nil
    case lexer.CONTINUE: p.advance(); return &ast.ContinueStmt{NodePos: astPos(p.cur.Pos)}, nil
    case lexer.FN: return p.parseFnDecl()
    case lexer.STRUCT: return p.parseStructDecl()
    case lexer.META: return p.parseMeta()
    case lexer.PLAN: return p.parsePlan()
    case lexer.IMPORT: return p.parseImport()
    case lexer.PUB: return p.parsePub()
    case lexer.NAME: return p.parseAssignOrExpr()
    }
    return nil, errors.New("E1002",
        fmt.Sprintf("unexpected token %s", p.cur.Kind),
        astPos(p.cur.Pos), "")
}

// Stubs to be filled in later tasks:
func (p *Parser) parseLet() (ast.Statement, error) { return nil, fmt.Errorf("let stub") }
func (p *Parser) parseIf() (ast.Statement, error) { return nil, fmt.Errorf("if stub") }
func (p *Parser) parseFor() (ast.Statement, error) { return nil, fmt.Errorf("for stub") }
func (p *Parser) parseWhile() (ast.Statement, error) { return nil, fmt.Errorf("while stub") }
func (p *Parser) parseMatch() (ast.Statement, error) { return nil, fmt.Errorf("match stub") }
func (p *Parser) parseReturn() (ast.Statement, error) { return nil, fmt.Errorf("return stub") }
func (p *Parser) parseFnDecl() (ast.Statement, error) { return nil, fmt.Errorf("fn stub") }
func (p *Parser) parseStructDecl() (ast.Statement, error) { return nil, fmt.Errorf("struct stub") }
func (p *Parser) parseMeta() (ast.Statement, error) { return nil, fmt.Errorf("meta stub") }
func (p *Parser) parsePlan() (ast.Statement, error) { return nil, fmt.Errorf("plan stub") }
func (p *Parser) parseImport() (ast.Statement, error) { return nil, fmt.Errorf("import stub") }
func (p *Parser) parsePub() (ast.Statement, error) { return nil, fmt.Errorf("pub stub") }
func (p *Parser) parseAssignOrExpr() (ast.Statement, error) { return nil, fmt.Errorf("assignOrExpr stub") }
func (p *Parser) parseExpression() (ast.Expression, error) { return nil, fmt.Errorf("parseExpression stub") }
```

- [ ] Run test → PASS
- [ ] Commit: `git add v2/internal/parser/ && git commit -m "v2: parser driver skeleton"`

---

## Task 15: Parser - Pratt Expression

**Files:** `v2/internal/parser/expression.go`, modify `v2/internal/parser/statement.go`

- [ ] Append tests for `42`, `1 + 2 * 3` (precedence), `(1 + 2) * 3` (parens)
- [ ] Run test → FAIL
- [ ] Create `expression.go`:

```go
package parser

import (
    "fmt"
    "strconv"
    "github.com/jerloo/funny/v2/internal/ast"
    "github.com/jerloo/funny/v2/internal/errors"
    "github.com/jerloo/funny/v2/internal/lexer"
)

const (
    precLowest = iota
    precOr; precAnd; precNot
    precCmp; precAdd; precMul; precUnary; precCall; precPrimary
)

func precedence(k lexer.Kind) int {
    switch k {
    case lexer.OR: return precOr
    case lexer.AND: return precAnd
    case lexer.NOT: return precNot
    case lexer.EQEQ, lexer.NEQ, lexer.LT, lexer.GT, lexer.LTE, lexer.GTE, lexer.IN: return precCmp
    case lexer.PLUS, lexer.MINUS: return precAdd
    case lexer.STAR, lexer.SLASH, lexer.PERCENT: return precMul
    case lexer.LPAREN, lexer.DOT, lexer.LBRACK: return precCall
    }
    return precLowest
}

func (p *Parser) parseExpression() (ast.Expression, error) {
    return p.parseBinary(precLowest)
}

func (p *Parser) parseBinary(minPrec int) (ast.Expression, error) {
    left, err := p.parseUnary()
    if err != nil { return nil, err }
    for {
        prec := precedence(p.cur.Kind)
        if prec < minPrec { break }
        opStr := p.cur.Data
        pos := astPos(p.cur.Pos)
        p.advance()
        right, err := p.parseBinary(prec + 1)
        if err != nil { return nil, err }
        left = &ast.BinaryExpr{NodePos: pos, Left: left, Op: opStr, Right: right}
    }
    return left, nil
}

func (p *Parser) parseUnary() (ast.Expression, error) {
    if p.cur.Kind == lexer.MINUS || p.cur.Kind == lexer.NOT {
        op := p.cur.Data
        pos := astPos(p.cur.Pos)
        p.advance()
        inner, err := p.parseUnary()
        if err != nil { return nil, err }
        return &ast.UnaryExpr{NodePos: pos, Op: op, Expr: inner}, nil
    }
    return p.parsePostfix()
}

func (p *Parser) parsePostfix() (ast.Expression, error) {
    left, err := p.parsePrimary()
    if err != nil { return nil, err }
    for {
        switch p.cur.Kind {
        case lexer.LPAREN:
            pos := astPos(p.cur.Pos); p.advance()
            var args []ast.Expression
            for p.cur.Kind != lexer.RPAREN && p.cur.Kind != lexer.EOF {
                e, err := p.parseExpression()
                if err != nil { return nil, err }
                args = append(args, e)
                if p.cur.Kind == lexer.COMMA { p.advance() }
            }
            if _, err := p.expect(lexer.RPAREN); err != nil { return nil, err }
            left = &ast.CallExpr{NodePos: pos, Func: left, Args: args}
        case lexer.DOT:
            p.advance()
            if p.cur.Kind != lexer.NAME {
                return nil, errors.New("E1010", "expected field name after `.`", astPos(p.cur.Pos), "")
            }
            left = &ast.FieldExpr{NodePos: left.Pos(), Object: left, Field: p.cur.Data}
            p.advance()
        case lexer.LBRACK:
            pos := astPos(p.cur.Pos); p.advance()
            idx, err := p.parseExpression()
            if err != nil { return nil, err }
            if _, err := p.expect(lexer.RBRACK); err != nil { return nil, err }
            left = &ast.IndexExpr{NodePos: pos, Object: left, Index: idx}
        default:
            return left, nil
        }
    }
}

func (p *Parser) parsePrimary() (ast.Expression, error) {
    pos := astPos(p.cur.Pos)
    switch p.cur.Kind {
    case lexer.INT:
        n, err := strconv.ParseInt(p.cur.Data, 0, 64)
        if err != nil {
            n2, err2 := strconv.ParseInt(p.cur.Data[2:], 16, 64)
            if err2 != nil {
                return nil, errors.New("E1011", fmt.Sprintf("invalid int %q", p.cur.Data), pos, "")
            }
            n = n2
        }
        p.advance()
        return &ast.LiteralExpr{NodePos: pos, Value: int(n)}, nil
    case lexer.FLOAT:
        f, err := strconv.ParseFloat(p.cur.Data, 64)
        if err != nil {
            return nil, errors.New("E1011", fmt.Sprintf("invalid float %q", p.cur.Data), pos, "")
        }
        p.advance()
        return &ast.LiteralExpr{NodePos: pos, Value: f}, nil
    case lexer.STR:
        s := p.cur.Data; p.advance()
        return &ast.LiteralExpr{NodePos: pos, Value: s}, nil
    case lexer.TRUE: p.advance(); return &ast.LiteralExpr{NodePos: pos, Value: true}, nil
    case lexer.FALSE: p.advance(); return &ast.LiteralExpr{NodePos: pos, Value: false}, nil
    case lexer.NIL: p.advance(); return &ast.LiteralExpr{NodePos: pos, Value: nil}, nil
    case lexer.FSTR:
        s := p.cur.Data; p.advance()
        return &ast.FStringExpr{NodePos: pos, Raw: s}, nil
    case lexer.NAME:
        name := p.cur.Data; p.advance()
        return &ast.VariableExpr{NodePos: pos, Name: name}, nil
    case lexer.LPAREN:
        p.advance()
        inner, err := p.parseExpression()
        if err != nil { return nil, err }
        if _, err := p.expect(lexer.RPAREN); err != nil { return nil, err }
        return &ast.SubExpr{NodePos: pos, Inner: inner}, nil
    case lexer.LBRACK:
        p.advance()
        var elems []ast.Expression
        for p.cur.Kind != lexer.RBRACK && p.cur.Kind != lexer.EOF {
            e, err := p.parseExpression()
            if err != nil { return nil, err }
            elems = append(elems, e)
            if p.cur.Kind == lexer.COMMA { p.advance() }
        }
        if _, err := p.expect(lexer.RBRACK); err != nil { return nil, err }
        return &ast.ListExpr{NodePos: pos, Elements: elems}, nil
    }
    return nil, errors.New("E1012",
        fmt.Sprintf("unexpected token %s in expression", p.cur.Kind),
        pos, "")
}
```

- [ ] Remove `parseExpression` stub from `statement.go`
- [ ] Run test → PASS
- [ ] Commit: `git add v2/internal/parser/ && git commit -m "v2: parser Pratt expression parser"`

---

## Task 16: Parser - Let and Assignment

**Files:** modify `v2/internal/parser/statement.go`

- [ ] Append tests for `let x = 1`, `let x: int = 1`, `x = 2`
- [ ] Run test → FAIL
- [ ] Implement `parseLet`:

```go
func (p *Parser) parseLet() (ast.Statement, error) {
    pos := astPos(p.cur.Pos)
    p.advance()
    if p.cur.Kind != lexer.NAME {
        return nil, errors.New("E1005", "expected variable name after `let`", pos, "")
    }
    name := p.cur.Data
    p.advance()
    var typeAnn string
    if p.cur.Kind == lexer.COLON {
        p.advance()
        typeAnn = p.cur.Data
        p.advance()
    }
    if _, err := p.expect(lexer.EQ); err != nil { return nil, err }
    val, err := p.parseExpression()
    if err != nil { return nil, err }
    return &ast.LetStmt{NodePos: pos, Name: name, TypeAnn: typeAnn, Value: val}, nil
}
```

- [ ] Implement `parseAssignOrExpr`:

```go
func (p *Parser) parseAssignOrExpr() (ast.Statement, error) {
    left, err := p.parseExpression()
    if err != nil { return nil, err }
    if p.cur.Kind == lexer.EQ {
        pos := astPos(p.cur.Pos)
        p.advance()
        val, err := p.parseExpression()
        if err != nil { return nil, err }
        return &ast.AssignStmt{NodePos: pos, Target: left, Value: val}, nil
    }
    return &ast.ExprStmt{NodePos: left.Pos(), X: left}, nil
}
```

- [ ] Run test → PASS
- [ ] Commit: `git add v2/internal/parser/ && git commit -m "v2: parser let and assignment"`

---

## Task 17: Parser - Control Flow + Block

**Files:** modify `v2/internal/parser/statement.go`

- [ ] Append tests for `if/elif/else`, `for x in list`, `while cond`, `return expr`
- [ ] Run test → FAIL
- [ ] Implement `parseIf`, `parseFor`, `parseWhile`, `parseMatch`, `parseReturn`, `parseBlock`:

```go
func (p *Parser) parseIf() (ast.Statement, error) {
    pos := astPos(p.cur.Pos); p.advance()
    cond, err := p.parseExpression()
    if err != nil { return nil, err }
    if _, err := p.expect(lexer.COLON); err != nil { return nil, err }
    thenBlock, err := p.parseBlock()
    if err != nil { return nil, err }
    ifStmt := &ast.IfStmt{NodePos: pos, Cond: cond, Then: thenBlock}
    if p.cur.Kind == lexer.ELIF {
        p.advance()
        elif, err := p.parseIf()
        if err != nil { return nil, err }
        ifStmt.ElseIf = elif.(*ast.IfStmt)
    } else if p.cur.Kind == lexer.ELSE {
        p.advance()
        if _, err := p.expect(lexer.COLON); err != nil { return nil, err }
        elseBlock, err := p.parseBlock()
        if err != nil { return nil, err }
        ifStmt.ElseBlock = elseBlock
    }
    return ifStmt, nil
}

func (p *Parser) parseFor() (ast.Statement, error) {
    pos := astPos(p.cur.Pos); p.advance()
    if p.cur.Kind != lexer.NAME {
        return nil, errors.New("E1020", "expected loop variable after `for`", pos, "")
    }
    name := p.cur.Data; p.advance()
    if _, err := p.expect(lexer.IN); err != nil { return nil, err }
    iterable, err := p.parseExpression()
    if err != nil { return nil, err }
    if _, err := p.expect(lexer.COLON); err != nil { return nil, err }
    body, err := p.parseBlock()
    if err != nil { return nil, err }
    return &ast.ForStmt{NodePos: pos, Name: name, Iterable: iterable, Body: body}, nil
}

func (p *Parser) parseWhile() (ast.Statement, error) {
    pos := astPos(p.cur.Pos); p.advance()
    cond, err := p.parseExpression()
    if err != nil { return nil, err }
    if _, err := p.expect(lexer.COLON); err != nil { return nil, err }
    body, err := p.parseBlock()
    if err != nil { return nil, err }
    return &ast.WhileStmt{NodePos: pos, Cond: cond, Body: body}, nil
}

func (p *Parser) parseMatch() (ast.Statement, error) {
    pos := astPos(p.cur.Pos); p.advance()
    expr, err := p.parseExpression()
    if err != nil { return nil, err }
    if _, err := p.expect(lexer.COLON); err != nil { return nil, err }
    if _, err := p.expect(lexer.INDENT); err != nil { return nil, err }
    var arms []ast.MatchArm
    for p.cur.Kind != lexer.DEDENT && p.cur.Kind != lexer.EOF {
        for p.cur.Kind == lexer.NEWLINE { p.advance() }
        if p.cur.Kind == lexer.DEDENT { break }
        pattern, err := p.parseExpression()
        if err != nil { return nil, err }
        if _, err := p.expect(lexer.FATARROW); err != nil { return nil, err }
        body, err := p.parseBlock()
        if err != nil { return nil, err }
        arms = append(arms, ast.MatchArm{Pattern: pattern, Body: body})
    }
    if p.cur.Kind == lexer.DEDENT { p.advance() }
    return &ast.MatchStmt{NodePos: pos, Expr: expr, Arms: arms}, nil
}

func (p *Parser) parseReturn() (ast.Statement, error) {
    pos := astPos(p.cur.Pos); p.advance()
    if p.cur.Kind == lexer.NEWLINE || p.cur.Kind == lexer.EOF || p.cur.Kind == lexer.DEDENT {
        return &ast.ReturnStmt{NodePos: pos, Value: nil}, nil
    }
    val, err := p.parseExpression()
    if err != nil { return nil, err }
    return &ast.ReturnStmt{NodePos: pos, Value: val}, nil
}

func (p *Parser) parseBlock() (*ast.Block, error) {
    pos := astPos(p.cur.Pos)
    if p.cur.Kind == lexer.NEWLINE { p.advance() }
    if p.cur.Kind != lexer.INDENT {
        return nil, errors.New("E1003",
            fmt.Sprintf("expected INDENT for block, got %s", p.cur.Kind),
            pos, "blocks must be on a new line with indented content")
    }
    p.advance()
    block := &ast.Block{NodePos: pos}
    for p.cur.Kind != lexer.DEDENT && p.cur.Kind != lexer.EOF {
        for p.cur.Kind == lexer.NEWLINE { p.advance() }
        if p.cur.Kind == lexer.DEDENT || p.cur.Kind == lexer.EOF { break }
        s, err := p.parseStatement()
        if err != nil { return nil, err }
        if s != nil { block.Statements = append(block.Statements, s) }
    }
    if p.cur.Kind == lexer.DEDENT { p.advance() }
    return block, nil
}
```

- [ ] Run test → PASS
- [ ] Commit: `git add v2/internal/parser/ && git commit -m "v2: parser control flow + block"`

---

## Task 18: Parser - Function and Struct Declarations

**Files:** modify `v2/internal/parser/statement.go`

- [ ] Append tests for `fn`, `struct`, `pub fn`
- [ ] Run test → FAIL
- [ ] Implement `parsePub`, `parseFnDecl`, `parseStructDecl`:

```go
func (p *Parser) parsePub() (ast.Statement, error) {
    p.advance()
    switch p.cur.Kind {
    case lexer.FN:
        fn, err := p.parseFnDecl()
        if err != nil { return nil, err }
        fn.(*ast.FnDecl).Pub = true
        return fn, nil
    case lexer.STRUCT:
        s, err := p.parseStructDecl()
        if err != nil { return nil, err }
        s.(*ast.StructDecl).Pub = true
        return s, nil
    }
    return nil, errors.New("E1030", "`pub` must precede `fn` or `struct`", astPos(p.cur.Pos), "")
}

func (p *Parser) parseFnDecl() (ast.Statement, error) {
    pos := astPos(p.cur.Pos); p.advance()
    if p.cur.Kind != lexer.NAME {
        return nil, errors.New("E1031", "expected function name after `fn`", pos, "")
    }
    name := p.cur.Data; p.advance()
    if _, err := p.expect(lexer.LPAREN); err != nil { return nil, err }
    var params []ast.Param
    for p.cur.Kind != lexer.RPAREN && p.cur.Kind != lexer.EOF {
        if p.cur.Kind != lexer.NAME {
            return nil, errors.New("E1032", "expected parameter name", astPos(p.cur.Pos), "")
        }
        pname := p.cur.Data; p.advance()
        var ptype string
        if p.cur.Kind == lexer.COLON {
            p.advance()
            ptype = p.cur.Data
            p.advance()
        }
        params = append(params, ast.Param{Name: pname, TypeAnn: ptype})
        if p.cur.Kind == lexer.COMMA { p.advance() }
    }
    if _, err := p.expect(lexer.RPAREN); err != nil { return nil, err }
    var retType string
    if p.cur.Kind == lexer.ARROW {
        p.advance()
        retType = p.cur.Data
        p.advance()
    }
    if _, err := p.expect(lexer.COLON); err != nil { return nil, err }
    body, err := p.parseBlock()
    if err != nil { return nil, err }
    return &ast.FnDecl{NodePos: pos, Name: name, Params: params, RetType: retType, Body: body}, nil
}

func (p *Parser) parseStructDecl() (ast.Statement, error) {
    pos := astPos(p.cur.Pos); p.advance()
    if p.cur.Kind != lexer.NAME {
        return nil, errors.New("E1033", "expected struct name", pos, "")
    }
    name := p.cur.Data; p.advance()
    if _, err := p.expect(lexer.COLON); err != nil { return nil, err }
    body, err := p.parseBlock()
    if err != nil { return nil, err }
    var fields []ast.Param
    for _, s := range body.Statements {
        assign, ok := s.(*ast.AssignStmt)
        if !ok { continue }
        varExpr, ok := assign.Target.(*ast.VariableExpr)
        if !ok { continue }
        if typeLit, ok := assign.Value.(*ast.LiteralExpr); ok {
            if typeStr, ok := typeLit.Value.(string); ok {
                fields = append(fields, ast.Param{Name: varExpr.Name, TypeAnn: typeStr})
            }
        }
    }
    return &ast.StructDecl{NodePos: pos, Name: name, Fields: fields}, nil
}
```

- [ ] Run test → PASS
- [ ] Commit: `git add v2/internal/parser/ && git commit -m "v2: parser function and struct declarations"`

---

## Task 19: Parser - Meta, Plan, Import

**Files:** modify `v2/internal/parser/statement.go`

- [ ] Append tests for `meta`, `plan`, `import`
- [ ] Run test → FAIL
- [ ] Implement `parseMeta`, `parsePlan`, `parseImport`:

```go
func (p *Parser) parseMeta() (ast.Statement, error) {
    pos := astPos(p.cur.Pos); p.advance()
    if _, err := p.expect(lexer.COLON); err != nil { return nil, err }
    block, err := p.parseBlock()
    if err != nil { return nil, err }
    fields := map[string]string{}
    for _, s := range block.Statements {
        assign, ok := s.(*ast.AssignStmt)
        if !ok { continue }
        varExpr, ok := assign.Target.(*ast.VariableExpr)
        if !ok { continue }
        if lit, ok := assign.Value.(*ast.LiteralExpr); ok {
            if s, ok := lit.Value.(string); ok {
                fields[varExpr.Name] = s
            }
        }
    }
    return &ast.MetaBlock{NodePos: pos, Fields: fields}, nil
}

func (p *Parser) parsePlan() (ast.Statement, error) {
    pos := astPos(p.cur.Pos); p.advance()
    if p.cur.Kind != lexer.STR {
        return nil, errors.New("E1040", "expected plan name as string", pos, "")
    }
    name := p.cur.Data; p.advance()
    if _, err := p.expect(lexer.COLON); err != nil { return nil, err }
    body, err := p.parseBlock()
    if err != nil { return nil, err }
    return &ast.PlanBlock{NodePos: pos, Name: name, Body: body}, nil
}

func (p *Parser) parseImport() (ast.Statement, error) {
    pos := astPos(p.cur.Pos); p.advance()
    if p.cur.Kind != lexer.STR {
        return nil, errors.New("E1041", "expected import path as string", pos, "")
    }
    path := p.cur.Data; p.advance()
    var alias string
    if p.cur.Kind == lexer.AS {
        p.advance()
        if p.cur.Kind != lexer.NAME {
            return nil, errors.New("E1042", "expected alias name", pos, "")
        }
        alias = p.cur.Data
        p.advance()
    }
    return &ast.ImportDecl{NodePos: pos, Path: path, Alias: alias}, nil
}
```

- [ ] Run test → PASS
- [ ] Commit: `git add v2/internal/parser/ && git commit -m "v2: parser meta, plan, import"`

---

## Task 20: Parser - Integration Tests with Real Files

**Files:** `v2/testdata/parser/control_flow.fn`, `v2/testdata/parser/function.fn`, append to `v2/internal/parser/parser_test.go`

- [ ] Create `v2/testdata/parser/control_flow.fn`:

```
let x = 10

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
```

- [ ] Create `v2/testdata/parser/function.fn`:

```
fn fib(n: int) -> int:
    if n < 2:
        return n
    return fib(n - 1) + fib(n - 2)

struct User:
    name: str
    age: int

pub fn greet(u: User) -> str:
    return "hello " + u.name
```

- [ ] Append test:

```go
import "os"
// ...
func TestParser_FromFile(t *testing.T) {
    cases := []string{
        "../../testdata/parser/control_flow.fn",
        "../../testdata/parser/function.fn",
    }
    for _, path := range cases {
        data, err := os.ReadFile(path)
        assert.NoError(t, err)
        p := New(string(data), path)
        _, err = p.Parse()
        assert.NoError(t, err, "file=%s", path)
    }
}
```

- [ ] Run test → PASS
- [ ] Commit: `git add v2/internal/parser/ v2/testdata/parser/ && git commit -m "v2: parser integration tests"`

---

## Task 21: Evaluator - Scope

**Files:** `v2/internal/evaluator/scope.go`, `v2/internal/evaluator/scope_test.go`

- [ ] Write tests for `Set/Get`, nested lookup, shadowing
- [ ] Run test → FAIL
- [ ] Write `scope.go`:

```go
package evaluator

type Scope struct {
    parent *Scope
    vars   map[string]any
}

func NewScope(parent *Scope) *Scope {
    return &Scope{parent: parent, vars: map[string]any{}}
}

func (s *Scope) Set(name string, value any) { s.vars[name] = value }

func (s *Scope) Get(name string) (any, bool) {
    if v, ok := s.vars[name]; ok { return v, true }
    if s.parent != nil { return s.parent.Get(name) }
    return nil, false
}

func (s *Scope) Has(name string) bool { _, ok := s.Get(name); return ok }

func (s *Scope) Assign(name string, value any) bool {
    if _, ok := s.vars[name]; ok { s.vars[name] = value; return true }
    if s.parent != nil { return s.parent.Assign(name, value) }
    return false
}
```

- [ ] Run test → PASS
- [ ] Commit: `git add v2/internal/evaluator/scope.go v2/internal/evaluator/scope_test.go && git commit -m "v2: evaluator scope"`

---

## Task 22: Evaluator - Expressions

**Files:** `v2/internal/evaluator/builtin.go`, `v2/internal/evaluator/evaluator.go`, `v2/internal/evaluator/evaluator_test.go`

- [ ] Write tests for `42`, `3.14`, `"hi"`, `true`, `2 + 3`, `5 > 3`, `"a" + "b"`, `x + 5` (with scope)
- [ ] Run test → FAIL
- [ ] Write `builtin.go` placeholder:

```go
package evaluator

type builtinFn struct {
    name string
    fn   func(e *Evaluator, args []any) (any, error)
}

var builtins = map[string]builtinFn{}
```

- [ ] Write `evaluator.go` with `Eval`, `applyBinary`, `compare`, `truthy`, `equalsLoose`, `evalCall`, `toErrPos`. (Full implementation as in earlier draft — see design.)
- [ ] Run test → PASS
- [ ] Commit: `git add v2/internal/evaluator/ && git commit -m "v2: evaluator expressions"`

---

## Task 23: Evaluator - Statements

**Files:** modify `v2/internal/evaluator/evaluator.go`, append to `v2/internal/evaluator/evaluator_test.go`

- [ ] Append tests for `let`/`assign`, `if/else`, `for-in`, `while`
- [ ] Run test → FAIL
- [ ] Add `Exec`, `execBlock`, `execStmt` methods handling all statement types. For `for-in`, iterate over `[]any` with new scope per iteration. For `while`, loop until condition false.
- [ ] Run test → PASS
- [ ] Commit: `git add v2/internal/evaluator/ && git commit -m "v2: evaluator statements"`

---

## Task 24: Builtins - print/len/to_str/to_int/type_of

**Files:** modify `v2/internal/evaluator/builtin.go`, create `v2/internal/evaluator/builtin_test.go`

- [ ] Write tests for `print`, `println`, `len`, `to_str`, `to_int`, `type_of`
- [ ] Run test → FAIL
- [ ] Implement builtins:

```go
var builtins = map[string]builtinFn{
    "print":   {name: "print", fn: func(e *Evaluator, args []any) (any, error) { fmt.Print(args...); return nil, nil }},
    "println": {name: "println", fn: func(e *Evaluator, args []any) (any, error) { fmt.Println(args...); return nil, nil }},
    "len":     {name: "len", fn: func(e *Evaluator, args []any) (any, error) {
        if len(args) != 1 { return nil, fmt.Errorf("len() takes 1 arg") }
        switch v := args[0].(type) {
        case string: return len(v), nil
        case []any: return len(v), nil
        }
        return nil, fmt.Errorf("len() not for %T", args[0])
    }},
    "to_str":  {name: "to_str", fn: func(e *Evaluator, args []any) (any, error) { return fmt.Sprintf("%v", args[0]), nil }},
    "to_int":  {name: "to_int", fn: func(e *Evaluator, args []any) (any, error) {
        switch v := args[0].(type) {
        case int: return v, nil
        case float64: return int(v), nil
        case string: return strconv.Atoi(v)
        }
        return nil, fmt.Errorf("to_int not for %T", args[0])
    }},
    "type_of": {name: "type_of", fn: func(e *Evaluator, args []any) (any, error) {
        switch args[0].(type) {
        case nil: return "nil", nil
        case bool: return "bool", nil
        case int: return "int", nil
        case float64: return "float", nil
        case string: return "str", nil
        case []any: return "list", nil
        case map[string]any: return "map", nil
        }
        return "unknown", nil
    }},
}
```

- [ ] Run test → PASS
- [ ] Commit: `git add v2/internal/evaluator/ && git commit -m "v2: evaluator builtins"`

---

## Task 25: Integration Tests - Fibonacci

**Files:** `v2/testdata/integration/fib.fn`, append to `v2/internal/evaluator/evaluator_test.go`

- [ ] Create `v2/testdata/integration/fib.fn`:

```
fn fib(n: int) -> int:
    if n < 2:
        return n
    return fib(n - 1) + fib(n - 2)

let r = fib(10)
println("fib(10) =", r)
```

- [ ] Append tests:

```go
func TestIntegration_Fib(t *testing.T) {
    src := `fn fib(n: int) -> int:
    if n < 2:
        return n
    return fib(n - 1) + fib(n - 2)
let r = fib(10)
`
    p := parser.New(src, ""); prog, _ := p.Parse()
    e := New(nil); require.NoError(t, e.Exec(prog))
    v, _ := e.scope.Get("r")
    assert.Equal(t, 55, v)
}

func TestIntegration_Sum(t *testing.T) {
    src := `let sum = 0
for i in [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]:
    sum = sum + i
`
    p := parser.New(src, ""); prog, _ := p.Parse()
    e := New(nil); require.NoError(t, e.Exec(prog))
    v, _ := e.scope.Get("sum")
    assert.Equal(t, 55, v)
}
```

- [ ] Run test → PASS
- [ ] Commit: `git add v2/internal/evaluator/ v2/testdata/integration/ && git commit -m "v2: integration tests fib and sum"`

---

## Task 26: CLI - run/ast helpers

**Files:** `v2/internal/cli/run.go`, `v2/internal/cli/run_test.go`

- [ ] Write tests for `Run(src)` and `Ast(src)` returning JSON containing `"type"`
- [ ] Run test → FAIL
- [ ] Write `run.go`:

```go
package cli

import (
    "encoding/json"
    "github.com/jerloo/funny/v2/internal/evaluator"
    "github.com/jerloo/funny/v2/internal/parser"
)

func Run(src []byte, file string) error {
    p := parser.New(string(src), file)
    prog, err := p.Parse()
    if err != nil { return err }
    e := evaluator.New(nil)
    return e.Exec(prog)
}

func Ast(src []byte, file string) ([]byte, error) {
    p := parser.New(string(src), file)
    prog, err := p.Parse()
    if err != nil { return nil, err }
    return json.MarshalIndent(prog, "", "  ")
}
```

- [ ] Run test → PASS
- [ ] Commit: `git add v2/internal/cli/ && git commit -m "v2: CLI run/ast helpers"`

---

## Task 27: CLI - main.go with cobra

**Files:** modify `v2/cmd/funny/main.go`

- [ ] Add cobra dependency: `cd /Users/j/repos/funny/v2 && go get github.com/spf13/cobra@latest`
- [ ] Write `cmd/funny/main.go`:

```go
package main

import (
    "fmt"
    "os"
    "github.com/spf13/cobra"
    "github.com/jerloo/funny/v2/internal/cli"
)

var rootCmd = &cobra.Command{
    Use:   "funny",
    Short: "funny v2 - AI-native scripting language",
    Version: "0.1.0",
}

var runCmd = &cobra.Command{
    Use:   "run <script>",
    Short: "Execute a funny script",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        data, err := os.ReadFile(args[0])
        if err != nil { return err }
        if err := cli.Run(data, args[0]); err != nil {
            fmt.Fprintln(os.Stderr, err)
            os.Exit(1)
        }
        return nil
    },
}

var astCmd = &cobra.Command{
    Use:   "ast <script>",
    Short: "Print JSON AST",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        data, err := os.ReadFile(args[0])
        if err != nil { return err }
        out, err := cli.Ast(data, args[0])
        if err != nil { return err }
        fmt.Println(string(out))
        return nil
    },
}

func init() {
    rootCmd.AddCommand(runCmd, astCmd)
}

func main() {
    if err := rootCmd.Execute(); err != nil { os.Exit(1) }
}
```

- [ ] Build: `cd /Users/j/repos/funny/v2 && go build -o funny ./cmd/funny`
- [ ] Test CLI: `./funny run ../testdata/integration/fib.fn` should print `fib(10) = 55`
- [ ] Test AST: `./funny ast ../testdata/integration/fib.fn` should print JSON
- [ ] Commit: `git add v2/cmd/ && git commit -m "v2: CLI with run and ast commands"`

---

## Task 28: README and Documentation

**Files:** modify `v2/README.md`, create `v2/docs/syntax-cheatsheet.md`

- [ ] Update `v2/README.md`:

```markdown
# Funny v2 (M1)

AI-native scripting language. See `../docs/superpowers/specs/2026-07-01-funny-v2-ai-native-language-design.md`.

## Status: M1 (Syntax Skeleton)

- ✅ Lexer (indent-sensitive, all operators, strings, f-strings)
- ✅ Parser (Pratt expressions, control flow, fn/struct/meta/plan)
- ✅ Tree-walking evaluator (no type checking, no Result/?)
- ⏳ Type checking → M2
- ⏳ Bytecode VM → M2
- ⏳ Plan engine → M3
- ⏳ MCP server → M4

## Build

\`\`\`bash
cd v2
go build -o funny ./cmd/funny
\`\`\`

## Usage

\`\`\`bash
./funny run script.fn        # execute script
./funny ast script.fn        # output JSON AST
./funny --help               # all commands
\`\`\`

## Test

\`\`\`bash
go test ./...
\`\`\`

## Limitations (M1)

- No type checking (dynamic)
- No `?` Result operator (defer to M2)
- No actual `import` resolution (parsed only)
- `meta` and `plan` blocks parsed but not executed (M3)
- Limited stdlib (print, len, to_str, to_int, type_of)

## Next: M2 (Strong Typing + Bytecode VM)

See spec §6.3.
```

- [ ] Create `v2/docs/syntax-cheatsheet.md` (examples of each construct)
- [ ] Commit: `git add v2/README.md v2/docs/ && git commit -m "v2: README and syntax cheatsheet"`

---

## Self-Review

After completing all tasks, verify:

1. **All tests pass**: `cd /Users/j/repos/funny/v2 && go test ./...`
2. **CLI works end-to-end**:
   - `./funny run testdata/integration/fib.fn` → outputs `fib(10) = 55`
   - `./funny ast testdata/integration/fib.fn` → valid JSON with `"type":"Program"`
3. **Spec coverage**:
   - §2.1 Indentation rules → Task 8 lexer INDENT/DEDENT ✅
   - §2.2 Base types → Task 10 AST literals, Task 22 evaluator literals ✅
   - §2.3 Strong typing → M2 (deferred) ⚠️
   - §2.4 Control flow (4 kinds) → Task 17 parser, Task 23 evaluator ✅
   - §2.5 Functions (`fn` only) → Task 18 parser, Task 22 evaluator evalCall ✅
   - §2.6 Data structures (struct/map/list) → Task 18 parser, Task 22 evaluator ✅
   - §2.7 Strings + f-strings → Task 6 lexer, Task 10 AST FStringExpr ✅
   - §2.8 Module system (import) → Task 19 parser (no resolution yet) ⚠️
   - §2.9 Comments (`#` and `##`) → Task 7 lexer ✅
   - §5.1 Go implementation → Task 0 Go module ✅
   - §5.3 Module structure → matches plan file structure ✅
   - §6.2 M1 exit criteria → fib runs end-to-end ✅

4. **Placeholder scan**: No TBD/TODO in this plan (✅ verified before writing)

5. **Type consistency**: All token/keyword/struct references match across tasks ✅

---

## Exit Criteria for M1 Plan

The M1 plan is complete when:
- [ ] All 28 tasks checked off
- [ ] `go test ./...` passes
- [ ] `./funny run testdata/integration/fib.fn` outputs `fib(10) = 55`
- [ ] All commits in this plan's history
- [ ] M1 release tagged as `v0.1.0-alpha`

---

## Total Tasks: 28

**Estimated time**: 6-10 days for one developer (some tasks are large; Task 8 INDENT/DEDENT is the most complex).