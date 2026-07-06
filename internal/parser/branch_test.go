package parser

import (
	"testing"

	"github.com/jiejie-dev/funny/v2/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseStep_BranchCaseList(t *testing.T) {
	src := `plan "demo":
    step "route" -> branch:
        status == 200 => "success"
        status == 404 => "not_found"
        _ => "fallback"
    step "success" -> tool:
        pass
`
	p := New(src, "test.fn")
	prog, err := p.Parse()
	require.NoError(t, err)
	plan := prog.Stmts[0].(*ast.PlanBlock)
	route := plan.Body.Statements[0].(*ast.Step)
	require.Equal(t, ast.StepBranch, route.Kind)
	require.Len(t, route.BranchCases, 3)
	assert.Equal(t, "success", route.BranchCases[0].Target)
	assert.Equal(t, "not_found", route.BranchCases[1].Target)
	assert.Equal(t, "fallback", route.BranchCases[2].Target)
	assert.Nil(t, route.Body)
}

func TestParseStep_BranchLegacyIfBody(t *testing.T) {
	src := `plan "demo":
    step "b" -> branch:
        if cond:
            let a = 1
        else:
            let a = 2
`
	p := New(src, "test.fn")
	prog, err := p.Parse()
	require.NoError(t, err)
	plan := prog.Stmts[0].(*ast.PlanBlock)
	step := plan.Body.Statements[0].(*ast.Step)
	require.Empty(t, step.BranchCases)
	require.NotNil(t, step.Body)
}

func TestParseStep_BranchCaseListRequiresStringTarget(t *testing.T) {
	src := `plan "demo":
    step "route" -> branch:
        true => success
`
	_, err := New(src, "test.fn").Parse()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "E1051")
}
