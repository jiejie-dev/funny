package benchmark

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClassify_AllTasks(t *testing.T) {
	tasks, err := LoadTasks("tasks.json")
	require.NoError(t, err)
	for _, task := range tasks {
		actual := Classify(task.Prompt)
		if actual != task.Expect {
			t.Errorf("task %d: got %s, want %s for %q", task.ID, actual, task.Expect, task.Prompt)
		}
	}
}
