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