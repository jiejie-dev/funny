package lsp

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPlanGraph_BranchCaseListFansOut(t *testing.T) {
	src := "plan \"route\":\n" +
		"    let status = 200\n" +
		"    step \"pick\" -> branch:\n" +
		"        status == 200 => \"ok\"\n" +
		"        _ => \"fail\"\n" +
		"    step \"ok\" -> tool:\n" +
		"        println(1)\n" +
		"    step \"fail\" -> tool:\n" +
		"        println(2)\n" +
		"    step \"done\" -> tool:\n" +
		"        println(3)\n"
	d := analyzeDoc("/tmp/a.fn", src)
	require.Empty(t, d.diagnostics)
	g := d.planGraphs().Plans[0]
	require.Len(t, g.Nodes, 4)

	var branchEdges, sequenceEdges int
	for _, e := range g.Edges {
		switch e.Kind {
		case "branch":
			branchEdges++
		case "sequence":
			sequenceEdges++
		}
	}
	require.Equal(t, 2, branchEdges)
	require.Equal(t, 1, sequenceEdges)
}
