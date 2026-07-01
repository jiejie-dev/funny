package agent

import (
	"os"
	"testing"

	"github.com/jiejie-dev/funny/internal/ast"
	"github.com/jiejie-dev/funny/internal/parser"
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

func TestEngine_Retry(t *testing.T) {
	// Step that succeeds on second attempt (uses a counter).
	src := `plan "demo":
    let tries = 0
    step "flaky" -> tool with retry max=3:
        tries = tries + 1
        if tries < 2:
            return err("not yet")
        return 42
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

func TestEngine_Parallel(t *testing.T) {
	src := `plan "demo":
    step "p1" -> parallel:
        let x = 1
    step "p2" -> parallel:
        let y = 2
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

func TestEngine_Branch(t *testing.T) {
	src := `plan "demo":
    let cond = true
    step "b" -> branch:
        if cond:
            let a = 1
        else:
            let a = 2
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

func TestEngine_PlanFromFile(t *testing.T) {
	data, err := os.ReadFile("../../testdata/agent/plan.fn")
	if err != nil {
		t.Fatal(err)
	}
	p := parser.New(string(data), "plan.fn")
	prog, err := p.Parse()
	require.NoError(t, err)
	var plan *ast.PlanBlock
	for _, s := range prog.Stmts {
		if pb, ok := s.(*ast.PlanBlock); ok {
			plan = pb
		}
	}
	require.NotNil(t, plan)
	e := New()
	err = e.RunPlan(plan, "plan.fn")
	assert.NoError(t, err)
}
