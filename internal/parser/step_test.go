package parser

import (
	"testing"

	"github.com/jiejie-dev/funny/v2/internal/ast"
	"github.com/stretchr/testify/require"
)

func parseStepFrom(t *testing.T, planSrc string) *ast.Step {
	t.Helper()
	p := New(planSrc, "test.fn")
	prog, err := p.Parse()
	require.NoError(t, err)
	require.Len(t, prog.Stmts, 1)
	plan, ok := prog.Stmts[0].(*ast.PlanBlock)
	require.True(t, ok)
	require.Len(t, plan.Body.Statements, 1)
	step, ok := plan.Body.Statements[0].(*ast.Step)
	require.True(t, ok)
	return step
}

func TestParseStep_RetryMax_BackwardCompatible(t *testing.T) {
	step := parseStepFrom(t, "plan \"p\":\n    step \"s\" -> tool with retry max=3:\n        pass\n")
	require.NotNil(t, step.Retry)
	require.Equal(t, 3, step.Retry.Max)
	require.Equal(t, "", step.Retry.Backoff)
	require.Equal(t, "", step.Timeout)
}

func TestParseStep_RetryWithBackoff(t *testing.T) {
	step := parseStepFrom(t, "plan \"p\":\n    step \"s\" -> tool with retry max=3 backoff=exp:\n        pass\n")
	require.NotNil(t, step.Retry)
	require.Equal(t, 3, step.Retry.Max)
	require.Equal(t, "exp", step.Retry.Backoff)
}

func TestParseStep_RetryOptionalKeyword_BackoffAlone(t *testing.T) {
	// "retry" is now an optional, purely cosmetic keyword.
	step := parseStepFrom(t, "plan \"p\":\n    step \"s\" -> tool with max=2 backoff=linear:\n        pass\n")
	require.NotNil(t, step.Retry)
	require.Equal(t, 2, step.Retry.Max)
	require.Equal(t, "linear", step.Retry.Backoff)
}

func TestParseStep_UnknownBackoff_Errors(t *testing.T) {
	p := New("plan \"p\":\n    step \"s\" -> tool with retry max=3 backoff=bogus:\n        pass\n", "test.fn")
	_, err := p.Parse()
	require.Error(t, err)
}

func TestParseStep_Timeout(t *testing.T) {
	step := parseStepFrom(t, "plan \"p\":\n    step \"s\" -> tool with timeout=\"5s\":\n        pass\n")
	require.Nil(t, step.Retry)
	require.Equal(t, "5s", step.Timeout)
}

func TestParseStep_RetryAndTimeoutTogether(t *testing.T) {
	step := parseStepFrom(t, "plan \"p\":\n    step \"s\" -> tool with retry max=3 backoff=constant timeout=\"2s\":\n        pass\n")
	require.NotNil(t, step.Retry)
	require.Equal(t, 3, step.Retry.Max)
	require.Equal(t, "constant", step.Retry.Backoff)
	require.Equal(t, "2s", step.Timeout)
}

func TestParseStep_InvalidTimeoutDuration_Errors(t *testing.T) {
	p := New("plan \"p\":\n    step \"s\" -> tool with timeout=\"not-a-duration\":\n        pass\n", "test.fn")
	_, err := p.Parse()
	require.Error(t, err)
}

func TestParseStep_TimeoutRequiresString_Errors(t *testing.T) {
	p := New("plan \"p\":\n    step \"s\" -> tool with timeout=5:\n        pass\n", "test.fn")
	_, err := p.Parse()
	require.Error(t, err)
}

func TestParseStep_UnknownOption_Errors(t *testing.T) {
	p := New("plan \"p\":\n    step \"s\" -> tool with bogus=1:\n        pass\n", "test.fn")
	_, err := p.Parse()
	require.Error(t, err)
}
