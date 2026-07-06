package agent

import (
	"testing"
	"time"

	"github.com/jiejie-dev/funny/v2/internal/ast"
	"github.com/jiejie-dev/funny/v2/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngine_RetryOn_RetriesMatchingTypedError(t *testing.T) {
	src := `struct NetworkError:
    message: str

plan "demo":
    let tries = 0
    step "flaky" -> tool with retry max=3 on=NetworkError:
        tries = tries + 1
        if tries < 3:
            return err(NetworkError(message: "transient"))
        return 1
`
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	plan := prog.Stmts[1].(*ast.PlanBlock)
	e := New()
	require.NoError(t, e.RunPlan(plan, "test"))
	v, ok := e.eval.Scope().Get("__result")
	require.True(t, ok)
	assert.Equal(t, 1, v)
}

func TestEngine_RetryOn_SkipsNonMatchingTypedError(t *testing.T) {
	src := `struct NetworkError:
    message: str

struct FatalError:
    message: str

plan "demo":
    step "flaky" -> tool with retry max=3 on=NetworkError:
        return err(FatalError(message: "fatal"))
`
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	plan := prog.Stmts[2].(*ast.PlanBlock)
	e := New()
	err = e.RunPlan(plan, "test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed")
	assert.NotContains(t, err.Error(), "after 3 attempts")
}

func TestEngine_RetryOn_StringErrorWithStrType(t *testing.T) {
	start := time.Now()
	src := `plan "demo":
    let tries = 0
    step "flaky" -> tool with retry max=3 on=str:
        tries = tries + 1
        if tries < 3:
            return err("transient")
        return 1
`
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	plan := prog.Stmts[0].(*ast.PlanBlock)
	e := New()
	require.NoError(t, e.RunPlan(plan, "test"))
	require.GreaterOrEqual(t, time.Since(start), 0*time.Millisecond)
}
