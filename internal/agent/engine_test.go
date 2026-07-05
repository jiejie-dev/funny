package agent

import (
	"os"
	"testing"
	"time"

	"github.com/jiejie-dev/funny/internal/ast"
	"github.com/jiejie-dev/funny/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runPlanSrc parses src (expected to be a single `plan "...":` statement)
// and runs it, returning the engine so tests can inspect its scope
// (e.g. __result) afterwards.
func runPlanSrc(t *testing.T, src string) (*Engine, error) {
	t.Helper()
	p := parser.New(src, "test.fn")
	prog, err := p.Parse()
	require.NoError(t, err)
	require.Len(t, prog.Stmts, 1)
	plan, ok := prog.Stmts[0].(*ast.PlanBlock)
	require.True(t, ok)
	e := New()
	return e, e.RunPlan(plan, "test")
}

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

func TestEngine_Result_CapturedFromFinalExpression(t *testing.T) {
	e, err := runPlanSrc(t, `plan "demo":
    step "compute" -> tool:
        1 + 41
`)
	require.NoError(t, err)
	v, ok := e.eval.Scope().Get("__result")
	require.True(t, ok)
	require.Equal(t, 42, v)
}

func TestEngine_Result_NotClobberedByTrailingLet(t *testing.T) {
	// A step ending in `let`, not a bare expression, makes no assertion
	// about __result — the previous step's result should survive.
	e, err := runPlanSrc(t, `plan "demo":
    step "compute" -> tool:
        42
    step "setup" -> tool:
        let z = 0
`)
	require.NoError(t, err)
	v, ok := e.eval.Scope().Get("__result")
	require.True(t, ok)
	require.Equal(t, 42, v)
}

func TestEngine_Guard_PassesOnTruthyFinalExpression(t *testing.T) {
	_, err := runPlanSrc(t, `plan "demo":
    step "verify" -> guard:
        1 > 0
`)
	require.NoError(t, err)
}

func TestEngine_Guard_FailsOnFalsyFinalExpression(t *testing.T) {
	_, err := runPlanSrc(t, `plan "demo":
    step "verify" -> guard:
        1 > 2
`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "guard failed")
}

func TestEngine_Guard_FailsOnErrResult(t *testing.T) {
	_, err := runPlanSrc(t, `plan "demo":
    step "verify" -> guard:
        err("bad state")
`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad state")
}

func TestEngine_Guard_PassesOnOkResultRegardlessOfPayload(t *testing.T) {
	_, err := runPlanSrc(t, `plan "demo":
    step "verify" -> guard:
        ok(0)
`)
	require.NoError(t, err)
}

func TestEngine_Guard_NoAssertionWhenBodyEndsInLet(t *testing.T) {
	// Matches testdata/agent/plan.fn's "verify" step shape: an else-less
	// `if` whose taken branch ends in `let` makes no explicit assertion,
	// so the guard must not fail just because that's nil/falsy.
	_, err := runPlanSrc(t, `plan "demo":
    let r = 20
    step "verify" -> guard:
        if r > 0:
            let z = 0
`)
	require.NoError(t, err)
}

func TestEngine_RetryBackoff_ConstantAddsDelayBetweenAttempts(t *testing.T) {
	start := time.Now()
	_, err := runPlanSrc(t, `plan "demo":
    let tries = 0
    step "flaky" -> tool with retry max=3 backoff=constant:
        tries = tries + 1
        if tries < 3:
            return err("not yet")
        return 1
`)
	elapsed := time.Since(start)
	require.NoError(t, err)
	// 2 failed attempts each incur one retryBackoffBase (10ms) delay.
	require.GreaterOrEqual(t, elapsed, 2*retryBackoffBase)
}

func TestEngine_RetryBackoff_NoBackoffIsImmediate(t *testing.T) {
	// Backward compatibility: `with retry max=N` alone (no `backoff=`)
	// must not introduce any inter-attempt delay.
	start := time.Now()
	_, err := runPlanSrc(t, `plan "demo":
    let tries = 0
    step "flaky" -> tool with retry max=5:
        tries = tries + 1
        if tries < 5:
            return err("not yet")
        return 1
`)
	require.NoError(t, err)
	require.Less(t, time.Since(start), 50*time.Millisecond)
}

func TestEngine_Timeout_FailsFastWithoutHanging(t *testing.T) {
	start := time.Now()
	_, err := runPlanSrc(t, `plan "demo":
    step "slow" -> tool with timeout="20ms":
        while true:
            let x = 1
`)
	elapsed := time.Since(start)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
	require.Less(t, elapsed, 500*time.Millisecond)
}

func TestEngine_Timeout_RetriesAfterTimingOut(t *testing.T) {
	e, err := runPlanSrc(t, `plan "demo":
    let tries = 0
    step "flaky" -> tool with retry max=2 timeout="50ms":
        tries = tries + 1
        if tries < 2:
            while true:
                let x = 1
        return 1
`)
	require.NoError(t, err)
	v, _ := e.eval.Scope().Get("__result")
	require.Equal(t, 1, v)
}

func TestEngine_Delay_SleepsForConfiguredDuration(t *testing.T) {
	start := time.Now()
	_, err := runPlanSrc(t, `plan "demo":
    step "wait" -> delay with timeout="30ms":
        pass_through = 1
`)
	require.NoError(t, err)
	require.GreaterOrEqual(t, time.Since(start), 30*time.Millisecond)
}

func TestEngine_Delay_RequiresTimeout(t *testing.T) {
	_, err := runPlanSrc(t, `plan "demo":
    step "wait" -> delay:
        pass
`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}

// TestEngine_CommentsBetweenSteps_AreSkipped is a regression test: a
// `plan` body interleaves `# ...` comments between step declarations at
// the same indent level whenever the plan is documented (exactly what a
// readable plan looks like), which shows up as *ast.CommentStmt nodes
// directly in plan.Body.Statements alongside the *ast.Step nodes - not
// just at file scope. execStmt used to have no case for it at all, so
// any such plan failed outright with "agent: unsupported statement type
// *ast.CommentStmt" the moment RunPlan reached the comment, even though
// the compiler and evaluator have always treated comments as no-ops
// everywhere else.
func TestEngine_CommentsBetweenSteps_AreSkipped(t *testing.T) {
	src := `plan "demo":
    # explains step one
    step "one" -> tool:
        1
    # explains step two
    step "two" -> tool:
        __result + 1
    # trailing comment
`
	e, err := runPlanSrc(t, src)
	require.NoError(t, err)
	v, ok := e.eval.Scope().Get("__result")
	require.True(t, ok)
	assert.Equal(t, 2, v)
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
