package lsp

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func findSymbol(syms []DocumentSymbol, name string) *DocumentSymbol {
	for i := range syms {
		if syms[i].Name == name {
			return &syms[i]
		}
	}
	return nil
}

func TestDocumentSymbols_FnAndStruct(t *testing.T) {
	src := "struct Point:\n    x: int\n    y: int\n\nfn add(a: int, b: int) -> int:\n    return a + b\n"
	d := analyzeDoc("/tmp/a.fn", src)
	syms := d.documentSymbols()
	require.Len(t, syms, 2)

	point := findSymbol(syms, "Point")
	require.NotNil(t, point)
	require.Equal(t, SKStruct, point.Kind)
	require.Len(t, point.Children, 2)
	require.NotNil(t, findSymbol(point.Children, "x"))
	require.NotNil(t, findSymbol(point.Children, "y"))

	add := findSymbol(syms, "add")
	require.NotNil(t, add)
	require.Equal(t, SKFunction, add.Kind)
	require.Contains(t, add.Detail, "int")
}

func TestDocumentSymbols_PlanWithSteps(t *testing.T) {
	src := "plan \"deploy\":\n    step \"build\" -> tool:\n        println(1)\n    step \"ship\" -> tool:\n        println(2)\n"
	d := analyzeDoc("/tmp/a.fn", src)
	syms := d.documentSymbols()
	require.Len(t, syms, 1)
	require.Equal(t, "deploy", syms[0].Name)
	require.Len(t, syms[0].Children, 2)
	require.Equal(t, "build", syms[0].Children[0].Name)
	require.Equal(t, "ship", syms[0].Children[1].Name)
}

func TestDocumentSymbols_ExcludesImportedDecls(t *testing.T) {
	src := "fn local() -> int:\n    return 1\n"
	d := analyzeDoc("/tmp/a.fn", src)
	syms := d.documentSymbols()
	require.Len(t, syms, 1)
	require.Equal(t, "local", syms[0].Name)
}
