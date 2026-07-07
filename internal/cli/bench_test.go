package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBenchAI_Mock(t *testing.T) {
	tasksPath := filepath.Join("..", "benchmark", "tasks.json")
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w
	err = BenchAI(BenchAIOptions{TasksPath: tasksPath, Mock: true})
	w.Close()
	os.Stdout = old
	require.NoError(t, err)
	var report struct {
		PassRate float64 `json:"pass_rate"`
		Results  []struct {
			Passed bool `json:"passed"`
		} `json:"results"`
	}
	require.NoError(t, json.NewDecoder(r).Decode(&report))
	assert.Equal(t, 1.0, report.PassRate)
	assert.Len(t, report.Results, 50)
}
