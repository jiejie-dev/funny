package agent

import (
	"testing"

	"github.com/jiejie-dev/funny/v2/internal/ast"
	"github.com/jiejie-dev/funny/v2/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngine_BranchCaseList_DispatchesToTarget(t *testing.T) {
	src := `plan "demo":
    let status = 200
    step "route" -> branch:
        status == 200 => "success"
        _ => "fail"
    step "success" -> tool:
        "ok"
    step "fail" -> tool:
        "bad"
    step "done" -> tool:
        let x = 1
`
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	plan := prog.Stmts[0].(*ast.PlanBlock)
	e := New()
	require.NoError(t, e.RunPlan(plan, "test"))
	v, ok := e.eval.Scope().Get("__result")
	require.True(t, ok)
	assert.Equal(t, "ok", v)
}

func TestEngine_BranchCaseList_WildcardFallback(t *testing.T) {
	src := `plan "demo":
    let status = 500
    step "route" -> branch:
        status == 200 => "success"
        _ => "fail"
    step "success" -> tool:
        "ok"
    step "fail" -> tool:
        "fallback"
    step "done" -> tool:
        let x = 1
`
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	plan := prog.Stmts[0].(*ast.PlanBlock)
	e := New()
	require.NoError(t, e.RunPlan(plan, "test"))
	v, ok := e.eval.Scope().Get("__result")
	require.True(t, ok)
	assert.Equal(t, "fallback", v)
}

func TestEngine_BranchCaseList_NoMatchErrors(t *testing.T) {
	src := `plan "demo":
    step "route" -> branch:
        false => "success"
    step "success" -> tool:
        pass
`
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	plan := prog.Stmts[0].(*ast.PlanBlock)
	e := New()
	err = e.RunPlan(plan, "test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no branch case matched")
}

func TestEngine_BranchCaseList_UnknownTargetErrors(t *testing.T) {
	src := `plan "demo":
    step "route" -> branch:
        true => "missing"
`
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	plan := prog.Stmts[0].(*ast.PlanBlock)
	e := New()
	err = e.RunPlan(plan, "test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "branch target \"missing\" not found")
}
