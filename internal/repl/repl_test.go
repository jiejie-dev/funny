package repl

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInputStatus_IncompleteBlock(t *testing.T) {
	complete, err := InputStatus("if true:")
	require.NoError(t, err)
	assert.False(t, complete)

	src := "if true:\n    1"
	complete, err = InputStatus(src)
	require.NoError(t, err)
	assert.True(t, complete)
}

func TestInputStatus_OpenParen(t *testing.T) {
	complete, err := InputStatus("println(1")
	require.NoError(t, err)
	assert.False(t, complete)
}

func TestSession_EvalLetAndExpr(t *testing.T) {
	s, err := NewSession(t.TempDir())
	require.NoError(t, err)

	_, showed, err := s.EvalCell("let x = 41")
	require.NoError(t, err)
	assert.False(t, showed)

	out, showed, err := s.EvalCell("x + 1")
	require.NoError(t, err)
	assert.True(t, showed)
	assert.Equal(t, "42", out)
}

func TestSession_MultiLineIf(t *testing.T) {
	s, err := NewSession(t.TempDir())
	require.NoError(t, err)
	src := strings.Join([]string{
		"let n = 3",
		"if n > 0:",
		"    n * 2",
	}, "\n")
	out, showed, err := s.EvalCell(src)
	require.NoError(t, err)
	assert.True(t, showed)
	assert.Equal(t, "6", out)
}

func TestSession_VMBackend_ListVars(t *testing.T) {
	s, err := NewSession(t.TempDir())
	require.NoError(t, err)
	require.False(t, s.interpret)
	_, _, err = s.EvalCell("let x = 41")
	require.NoError(t, err)
	lines := s.ListVars()
	require.Len(t, lines, 1)
	assert.Contains(t, lines[0], "x = 41")
}

func TestSession_Reset(t *testing.T) {
	s, err := NewSession(t.TempDir())
	require.NoError(t, err)
	_, _, err = s.EvalCell("let x = 1")
	require.NoError(t, err)
	s.Reset()
	assert.Empty(t, s.ListVars())
}

func TestRun_MetaCommands(t *testing.T) {
	in := strings.NewReader(":vars\n:quit\n")
	var out strings.Builder
	require.NoError(t, Run(t.TempDir(), in, &out))
	assert.Contains(t, out.String(), "no bindings")
}

func TestRun_LessonsMeta(t *testing.T) {
	dir := filepath.Join("..", "..", "docs")
	s, err := NewSessionWithOptions(Options{WorkDir: t.TempDir(), LessonsDir: dir})
	require.NoError(t, err)
	var out strings.Builder
	require.NoError(t, s.startLesson(1, &out))
	assert.Contains(t, out.String(), "Tutorial 1")
}
