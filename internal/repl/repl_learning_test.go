package repl

import (
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscoverLessons_FromDocs(t *testing.T) {
	root := filepath.Join("..", "..")
	lessons, err := DiscoverLessons(filepath.Join(root, "docs"))
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(lessons), 6)
	assert.Equal(t, 1, lessons[0].Number)
	assert.Contains(t, lessons[0].Title, "Tutorial")
	assert.NotEmpty(t, lessons[0].Steps)
}

func TestSession_TypeAndDescribe(t *testing.T) {
	s, err := NewSession(t.TempDir())
	require.NoError(t, err)
	_, _, err = s.EvalCell("let x = 42")
	require.NoError(t, err)
	typ, err := s.TypeOfExpr("x + 1")
	require.NoError(t, err)
	assert.Equal(t, "int", typ)
	desc, err := s.DescribeName("x")
	require.NoError(t, err)
	assert.Contains(t, desc, "int")
}

func TestCompletions_IncludesBindings(t *testing.T) {
	s, err := NewSession(t.TempDir())
	require.NoError(t, err)
	_, _, err = s.EvalCell("let alpha = 1")
	require.NoError(t, err)
	comps := Completions(s, "al")
	assert.Contains(t, comps, "alpha")
	comps = Completions(s, "pr")
	assert.Contains(t, comps, "println")
}

func TestSession_LoadTutorialFile(t *testing.T) {
	s, err := NewSession(t.TempDir())
	require.NoError(t, err)
	path, err := filepath.Abs(filepath.Join("..", "..", "docs", "tutorial-01-hello.funny"))
	require.NoError(t, err)
	require.NoError(t, s.LoadFile(path))
	lines := s.ListVars()
	joined := strings.Join(lines, "\n")
	assert.Contains(t, joined, "greeting")
}

func TestSession_LessonStepDemo(t *testing.T) {
	dir := filepath.Join("..", "..", "docs")
	s, err := NewSessionWithOptions(Options{WorkDir: t.TempDir(), LessonsDir: dir})
	require.NoError(t, err)
	require.NoError(t, s.startLesson(1, io.Discard))
	step, ok := s.lesson.current()
	require.True(t, ok)
	assert.NotEmpty(t, step.Code)
	_, _, err = s.EvalCell(step.Code)
	require.NoError(t, err)
	assert.Contains(t, s.ListVars()[0], "=")
}

func TestHistory_SkipsMetaCommands(t *testing.T) {
	h := NewHistory(10)
	h.Add(":quit")
	h.Add("let x = 1")
	assert.Len(t, h.Lines(), 1)
	assert.Equal(t, "let x = 1", h.Lines()[0])
}
