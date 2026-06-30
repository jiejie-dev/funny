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