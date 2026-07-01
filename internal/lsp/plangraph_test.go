package lsp

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPlanGraph_SequentialSteps(t *testing.T) {
	src := "plan \"deploy\":\n" +
		"    step \"build\" -> tool:\n" +
		"        println(1)\n" +
		"    step \"ship\" -> tool:\n" +
		"        println(2)\n"
	d := analyzeDoc("/tmp/a.fn", src)
	require.Empty(t, d.diagnostics)
	result := d.planGraphs()
	require.Len(t, result.Plans, 1)
	g := result.Plans[0]
	require.Equal(t, "deploy", g.Name)
	require.Len(t, g.Nodes, 2)
	require.Equal(t, "build", g.Nodes[0].Label)
	require.Equal(t, "tool", g.Nodes[0].Kind)
	require.Equal(t, "ship", g.Nodes[1].Label)
	require.Len(t, g.Edges, 1)
	require.Equal(t, "sequence", g.Edges[0].Kind)
	require.Equal(t, g.Nodes[0].ID, g.Edges[0].From)
	require.Equal(t, g.Nodes[1].ID, g.Edges[0].To)
}

func TestPlanGraph_ParallelStepGetsTaskChildren(t *testing.T) {
	src := "plan \"notify\":\n" +
		"    step \"fanout\" -> parallel:\n" +
		"        println(1)\n" +
		"        println(2)\n" +
		"    step \"done\" -> tool:\n" +
		"        println(3)\n"
	d := analyzeDoc("/tmp/a.fn", src)
	require.Empty(t, d.diagnostics)
	g := d.planGraphs().Plans[0]

	require.Len(t, g.Nodes, 4) // fanout, 2 tasks, done
	require.Equal(t, "fanout", g.Nodes[0].Label)
	require.Equal(t, "parallel", g.Nodes[0].Kind)
	require.Equal(t, "task", g.Nodes[1].Kind)
	require.Equal(t, g.Nodes[0].ID, g.Nodes[1].ParentID)
	require.Equal(t, "task", g.Nodes[2].Kind)
	require.Equal(t, g.Nodes[0].ID, g.Nodes[2].ParentID)

	var parallelEdges, sequenceEdges int
	for _, e := range g.Edges {
		switch e.Kind {
		case "parallel":
			parallelEdges++
			require.Equal(t, g.Nodes[0].ID, e.From)
		case "sequence":
			sequenceEdges++
			require.Equal(t, g.Nodes[0].ID, e.From, "next step follows the parallel step itself, not one of its tasks")
			require.Equal(t, g.Nodes[3].ID, e.To)
		}
	}
	require.Equal(t, 2, parallelEdges)
	require.Equal(t, 1, sequenceEdges)
}

func TestPlanGraph_RetryAndTimeoutSurface(t *testing.T) {
	src := "plan \"deploy\":\n" +
		"    step \"build\" -> tool with retry max=3:\n" +
		"        println(1)\n"
	d := analyzeDoc("/tmp/a.fn", src)
	require.Empty(t, d.diagnostics)
	g := d.planGraphs().Plans[0]
	require.Len(t, g.Nodes, 1)
	require.NotNil(t, g.Nodes[0].Retry)
	require.Equal(t, 3, g.Nodes[0].Retry.Max)
}

func TestPlanGraph_NoPlanBlock_ReturnsEmptyList(t *testing.T) {
	src := "let x: int = 1\n"
	d := analyzeDoc("/tmp/a.fn", src)
	result := d.planGraphs()
	require.Empty(t, result.Plans)
}
