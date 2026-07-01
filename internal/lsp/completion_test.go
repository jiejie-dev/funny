package lsp

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func hasLabel(items []CompletionItem, label string) bool {
	for _, it := range items {
		if it.Label == label {
			return true
		}
	}
	return false
}

func TestCompletion_General_IncludesKeywordsBuiltinsFuncsAndLocals(t *testing.T) {
	src := "fn add(a: int, b: int) -> int:\n    return a + b\nlet total = 0\n"
	d := analyzeDoc("/tmp/a.fn", src)
	items := d.completion(Position{Line: 2, Character: 14})
	require.True(t, hasLabel(items, "fn"), "should suggest keyword fn")
	require.True(t, hasLabel(items, "println"), "should suggest builtin println")
	require.True(t, hasLabel(items, "add"), "should suggest declared function add")
	require.True(t, hasLabel(items, "total"), "should suggest local variable total")
}

func TestCompletion_AfterDot_StructFields(t *testing.T) {
	src := "struct Point:\n    x: int\n    y: int\nlet p = Point(x: 1, y: 2)\nlet v = p.\n"
	d := analyzeDoc("/tmp/a.fn", src)
	items := d.completion(Position{Line: 4, Character: 10})
	require.True(t, hasLabel(items, "x"))
	require.True(t, hasLabel(items, "y"))
	require.False(t, hasLabel(items, "println"), "dot completion should be type-scoped, not general")
}

func TestCompletion_AfterDot_UnresolvedObject_ReturnsEmpty(t *testing.T) {
	src := "let v = nope.\n"
	d := analyzeDoc("/tmp/a.fn", src)
	items := d.completion(Position{Line: 0, Character: 13})
	require.Empty(t, items)
}

func TestDotContext_DetectsTrailingDot(t *testing.T) {
	name, ok := dotContext("let v = point.", Position{Line: 0, Character: 14})
	require.True(t, ok)
	require.Equal(t, "point", name)

	_, ok = dotContext("let v = point", Position{Line: 0, Character: 13})
	require.False(t, ok)
}
