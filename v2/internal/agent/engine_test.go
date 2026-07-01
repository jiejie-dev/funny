package agent

import (
	"testing"

	"github.com/jerloo/funny/v2/internal/ast"
	"github.com/jerloo/funny/v2/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngine_SequentialSteps(t *testing.T) {
	src := `plan "demo":
    step "s1":
        let x = 1
        println("s1", x)
    step "s2":
        let y = 2
        println("s2", y)
`
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	require.Len(t, prog.Stmts, 1)
	plan, ok := prog.Stmts[0].(*ast.PlanBlock)
	require.True(t, ok)
	e := New()
	err = e.RunPlan(plan, "test")
	assert.NoError(t, err)
}
