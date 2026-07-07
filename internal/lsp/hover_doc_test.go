package lsp

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHover_DocCommentOnFunction(t *testing.T) {
	src := `## Add two numbers
## Args:
## a: first summand
## b: second summand
## Returns: sum
fn add(a: int, b: int) -> int:
    return a + b

let r = add(1, 2)
`
	d := analyzeDoc("/tmp/a.fn", src)
	h := d.hover(Position{Line: 8, Character: 9})
	require.NotNil(t, h)
	require.Contains(t, h.Contents.Value, "Add two numbers")
	require.Contains(t, h.Contents.Value, "first summand")
	require.Contains(t, h.Contents.Value, "**Returns:** sum")
	require.Contains(t, h.Contents.Value, "fn add")
}

func TestHover_DocCommentOnStruct(t *testing.T) {
	src := `## A 2D point
struct Point:
    x: int
    y: int

let p = Point(x: 1, y: 2)
`
	d := analyzeDoc("/tmp/a.fn", src)
	h := d.hover(Position{Line: 5, Character: 9})
	require.NotNil(t, h)
	require.Contains(t, h.Contents.Value, "A 2D point")
	require.Contains(t, h.Contents.Value, "struct Point")
}
