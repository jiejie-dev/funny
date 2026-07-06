package types

import (
	"testing"

	"github.com/jiejie-dev/funny/v2/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheck_PlanBranchCaseList_Ok(t *testing.T) {
	src := `plan "demo":
    let status = 200
    step "route" -> branch:
        status == 200 => "success"
        _ => "fail"
    step "success" -> tool:
        1
    step "fail" -> tool:
        1
`
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	env := NewEnv(nil)
	require.NoError(t, Check(prog, env))
}

func TestCheck_PlanBranchUnknownTargetErrors(t *testing.T) {
	src := `plan "demo":
    step "route" -> branch:
        true => "missing"
`
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	env := NewEnv(nil)
	err = Check(prog, env)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "E2111")
}

func TestCheck_PlanDuplicateStepNameErrors(t *testing.T) {
	src := `plan "demo":
    step "dup" -> tool:
        pass
    step "dup" -> tool:
        pass
`
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	env := NewEnv(nil)
	err = Check(prog, env)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "E2110")
}
