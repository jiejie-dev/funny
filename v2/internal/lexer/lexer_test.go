package lexer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
	assert.Equal(t, 0, bTok.Pos.Col)
}

func TestLexer_Operators(t *testing.T) {
	cases := []struct {
		src  string
		kind Kind
		data string
	}{
		{"+", PLUS, "+"},
		{"-", MINUS, "-"},
		{"*", STAR, "*"},
		{"/", SLASH, "/"},
		{"%", PERCENT, "%"},
		{"=", EQ, "="},
		{"==", EQEQ, "=="},
		{"!=", NEQ, "!="},
		{"<", LT, "<"},
		{">", GT, ">"},
		{"<=", LTE, "<="},
		{">=", GTE, ">="},
		{"(", LPAREN, "("},
		{")", RPAREN, ")"},
		{"[", LBRACK, "["},
		{"]", RBRACK, "]"},
		{",", COMMA, ","},
		{".", DOT, "."},
		{":", COLON, ":"},
		{"->", ARROW, "->"},
		{"=>", FATARROW, "=>"},
		{"?", QUESTION, "?"},
		{"@", AT, "@"},
	}
	for _, c := range cases {
		l := New(c.src, "")
		tok := l.Next()
		assert.Equal(t, c.kind, tok.Kind, "src=%q", c.src)
		assert.Equal(t, c.data, tok.Data, "src=%q", c.src)
	}
}

func TestLexer_LoneBang_Placeholder(t *testing.T) {
	l := New("!", "")
	tok := l.Next()
	assert.Equal(t, EOF, tok.Kind)
}

func TestLexer_Int(t *testing.T) {
	l := New("42 0 7", "")
	assert.Equal(t, INT, l.Next().Kind)
	assert.Equal(t, INT, l.Next().Kind)
	assert.Equal(t, INT, l.Next().Kind)
}

func TestLexer_Hex(t *testing.T) {
	l := New("0x1F", "")
	tok := l.Next()
	assert.Equal(t, INT, tok.Kind)
	assert.Equal(t, "0x1F", tok.Data)
}

func TestLexer_Float(t *testing.T) {
	l := New("3.14 1e-3 2.5E+2", "")
	assert.Equal(t, FLOAT, l.Next().Kind)
	assert.Equal(t, FLOAT, l.Next().Kind)
	assert.Equal(t, FLOAT, l.Next().Kind)
}

func TestLexer_IntVsFloat(t *testing.T) {
	l := New("1 1.0", "")
	a := l.Next()
	b := l.Next()
	assert.Equal(t, INT, a.Kind)
	assert.Equal(t, FLOAT, b.Kind)
}

func TestLexer_String(t *testing.T) {
	l := New(`"hello" 'world'`, "")
	a := l.Next()
	assert.Equal(t, STR, a.Kind)
	assert.Equal(t, "hello", a.Data)
	b := l.Next()
	assert.Equal(t, STR, b.Kind)
	assert.Equal(t, "world", b.Data)
}

func TestLexer_StringEscapes(t *testing.T) {
	l := New(`"a\nb\tc\\d"`, "")
	tok := l.Next()
	assert.Equal(t, STR, tok.Kind)
	assert.Equal(t, "a\nb\tc\\d", tok.Data)
}

func TestLexer_FString(t *testing.T) {
	l := New(`f"hello {name}"`, "")
	tok := l.Next()
	assert.Equal(t, FSTR, tok.Kind)
	assert.Equal(t, "hello {name}", tok.Data)
}

func TestLexer_LineComment(t *testing.T) {
	l := New("# hello\na", "")
	c := l.Next()
	assert.Equal(t, COMMENT, c.Kind)
	assert.Equal(t, " hello", c.Data)
	a := l.Next()
	assert.Equal(t, NAME, a.Kind)
}

func TestLexer_DocComment(t *testing.T) {
	l := New("## doc\na", "")
	c := l.Next()
	assert.Equal(t, DOC_COMMENT, c.Kind)
	assert.Equal(t, " doc", c.Data)
}
