package lsp

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReferences_LocalVariable_WithinFunction(t *testing.T) {
	src := "fn f() -> int:\n    let x: int = 1\n    println(x)\n    return x + x\n"
	d := analyzeDoc("/tmp/a.fn", src)
	require.Empty(t, d.diagnostics)
	// cursor on the declaration "x" (line 1)
	locs := d.references(Position{Line: 1, Character: 8}, true)
	require.Len(t, locs, 4, "declaration + 3 uses (println, return x, return x)")
}

func TestReferences_LocalVariable_ExcludeDeclaration(t *testing.T) {
	src := "fn f() -> int:\n    let x: int = 1\n    return x\n"
	d := analyzeDoc("/tmp/a.fn", src)
	locs := d.references(Position{Line: 2, Character: 11}, false)
	require.Len(t, locs, 1)
	require.Equal(t, 2, locs[0].Range.Start.Line)
}

func TestReferences_Function_AcrossCallSites(t *testing.T) {
	src := "fn add(a: int, b: int) -> int:\n    return a + b\nlet x = add(1, 2)\nlet y = add(3, 4)\n"
	d := analyzeDoc("/tmp/a.fn", src)
	locs := d.references(Position{Line: 0, Character: 4}, true)
	require.Len(t, locs, 3, "declaration + 2 call sites")
}

func TestReferences_Param_ScopedToOwnFunction(t *testing.T) {
	src := "fn f(n: int) -> int:\n    return n\nfn g(n: int) -> int:\n    return n * 2\n"
	d := analyzeDoc("/tmp/a.fn", src)
	// "n" inside f's body
	locs := d.references(Position{Line: 1, Character: 11}, true)
	for _, l := range locs {
		require.LessOrEqual(t, l.Range.Start.Line, 1, "must not include g's unrelated param 'n'")
	}
	require.NotEmpty(t, locs)
}

func TestReferences_StructName(t *testing.T) {
	src := "struct Point:\n    x: int\n    y: int\nlet p: Point = Point(x: 1, y: 2)\n"
	d := analyzeDoc("/tmp/a.fn", src)
	locs := d.references(Position{Line: 0, Character: 8}, true)
	require.GreaterOrEqual(t, len(locs), 2, "declaration + at least one usage")
}

func TestReferences_UnknownIdentifier_ReturnsNil(t *testing.T) {
	src := "println(mystery)\n"
	d := analyzeDoc("/tmp/a.fn", src)
	locs := d.references(Position{Line: 0, Character: 10}, true)
	require.Nil(t, locs)
}
