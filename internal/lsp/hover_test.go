package lsp

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func analyzeDoc(path, text string) *document {
	d := &document{uri: pathToURI(path), path: path, text: text}
	d.analyze()
	return d
}

func TestHover_LocalVariable(t *testing.T) {
	src := "let x: int = 5\nprintln(x)\n"
	d := analyzeDoc("/tmp/a.fn", src)
	// "x" on line 1 (println(x)) starts at column 8
	h := d.hover(Position{Line: 1, Character: 9})
	require.NotNil(t, h)
	require.Contains(t, h.Contents.Value, "x: int")
	require.Contains(t, h.Contents.Value, "local variable")
}

func TestHover_FunctionSignature(t *testing.T) {
	src := "fn add(a: int, b: int) -> int:\n    return a + b\nlet r = add(1, 2)\n"
	d := analyzeDoc("/tmp/a.fn", src)
	h := d.hover(Position{Line: 2, Character: 9})
	require.NotNil(t, h)
	require.Contains(t, h.Contents.Value, "fn add")
	require.Contains(t, h.Contents.Value, "int")
}

func TestHover_StructType(t *testing.T) {
	src := "struct Point:\n    x: int\n    y: int\nlet p = Point(x: 1, y: 2)\n"
	d := analyzeDoc("/tmp/a.fn", src)
	h := d.hover(Position{Line: 3, Character: 9})
	require.NotNil(t, h)
	require.Contains(t, h.Contents.Value, "struct Point")
	require.Contains(t, h.Contents.Value, "x: int")
	require.Contains(t, h.Contents.Value, "y: int")
}

func TestHover_Builtin(t *testing.T) {
	src := "println(1)\n"
	d := analyzeDoc("/tmp/a.fn", src)
	h := d.hover(Position{Line: 0, Character: 2})
	require.NotNil(t, h)
	require.Contains(t, h.Contents.Value, "builtin function")
}

func TestHover_Keyword(t *testing.T) {
	src := "fn add(a: int) -> int:\n    return a\n"
	d := analyzeDoc("/tmp/a.fn", src)
	h := d.hover(Position{Line: 0, Character: 1})
	require.NotNil(t, h)
	require.Contains(t, h.Contents.Value, "keyword")
}

func TestHover_UnknownIdentifier_ReturnsNil(t *testing.T) {
	src := "let x = 1\nprintln(unknownvar)\n"
	d := analyzeDoc("/tmp/a.fn", src)
	h := d.hover(Position{Line: 1, Character: 10})
	require.Nil(t, h)
}

func TestHover_Param(t *testing.T) {
	src := "fn double(n: int) -> int:\n    return n * 2\n"
	d := analyzeDoc("/tmp/a.fn", src)
	h := d.hover(Position{Line: 1, Character: 11})
	require.NotNil(t, h)
	require.Contains(t, h.Contents.Value, "n: int")
	require.Contains(t, h.Contents.Value, "parameter")
}
