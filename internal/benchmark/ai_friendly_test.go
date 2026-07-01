package benchmark

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBenchmark_Runs(t *testing.T) {
	tasks, err := LoadTasks("../../internal/benchmark/tasks.json")
	require.NoError(t, err)
	require.NotEmpty(t, tasks)
	require.Len(t, tasks, 50)

	results, passRate := Run(tasks)
	assert.NotEmpty(t, results)
	assert.Len(t, results, 50)
	assert.GreaterOrEqual(t, passRate, 0.0)
	assert.LessOrEqual(t, passRate, 1.0)
	t.Logf("AI-friendliness pass rate (baseline perfect guesser): %.2f%% (%d/%d)",
		passRate*100, len(results), len(tasks))
}

func TestBenchmark_AllClassifiedCorrectly(t *testing.T) {
	tasks, err := LoadTasks("../../internal/benchmark/tasks.json")
	require.NoError(t, err)

	results, passRate := Run(tasks)
	for _, r := range results {
		assert.Equal(t, r.Expect, r.Actual,
			"task %d: expected %s, got %s for prompt %q",
			r.ID, r.Expect, r.Actual, r.Prompt)
	}
	assert.Equal(t, 1.0, passRate, "perfect guesser must achieve 100%% pass rate")
}

func TestLoadTasks_InvalidPath(t *testing.T) {
	_, err := LoadTasks("/nonexistent/path.json")
	assert.Error(t, err)
}