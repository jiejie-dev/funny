package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jiejie-dev/funny/v2/internal/benchmark"
)

// BenchAIOptions configures `funny bench ai`.
type BenchAIOptions struct {
	TasksPath string
	Provider  string
	Model     string
	Mock      bool
}

// BenchAI runs the AI-friendliness benchmark and prints a JSON report to stdout.
func BenchAI(opts BenchAIOptions) error {
	tasksPath := opts.TasksPath
	if tasksPath == "" {
		tasksPath = defaultTasksPath()
	}
	tasks, err := benchmark.LoadTasks(tasksPath)
	if err != nil {
		return fmt.Errorf("load tasks: %w", err)
	}
	provider := opts.Provider
	if opts.Mock {
		provider = benchmark.ProviderMock
	}
	gen, prov, model, err := benchmark.GeneratorFromEnv(provider, opts.Model)
	if err != nil {
		return err
	}
	report := benchmark.RunReport(tasks, gen, prov, model)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "pass rate: %.1f%% (%d/%d) provider=%s model=%s\n",
		report.PassRate*100,
		countPassed(report.Results),
		len(report.Results),
		report.Provider,
		modelLabel(report),
	)
	return nil
}

func defaultTasksPath() string {
	// Prefer repo-bundled tasks when running from source tree.
	candidates := []string{
		"internal/benchmark/tasks.json",
		filepath.Join("..", "internal", "benchmark", "tasks.json"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "internal/benchmark/tasks.json"
}

func countPassed(results []benchmark.Result) int {
	n := 0
	for _, r := range results {
		if r.Passed {
			n++
		}
	}
	return n
}

func modelLabel(r benchmark.Report) string {
	if r.Model != "" {
		return r.Model
	}
	if r.Provider == benchmark.ProviderMock {
		return "echo-prompt"
	}
	return "(default)"
}
