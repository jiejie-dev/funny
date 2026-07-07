package testrunner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jiejie-dev/funny/v2/internal/ast"
	"github.com/jiejie-dev/funny/v2/internal/compiler"
	"github.com/jiejie-dev/funny/v2/internal/module"
	"github.com/jiejie-dev/funny/v2/internal/parser"
	"github.com/jiejie-dev/funny/v2/internal/types"
	"github.com/jiejie-dev/funny/v2/internal/vm"
)

// CaseResult is one executed test block.
type CaseResult struct {
	File     string        `json:"file"`
	Name     string        `json:"name"`
	Passed   bool          `json:"passed"`
	Error    string        `json:"error,omitempty"`
	Duration time.Duration `json:"duration_ns"`
}

// Report summarizes a test run.
type Report struct {
	Passed int          `json:"passed"`
	Failed int          `json:"failed"`
	Tests  []CaseResult `json:"tests"`
}

// Options configures test discovery and execution.
type Options struct {
	Path    string // file or directory (default ".")
	Verbose bool
}

// Run discovers *_test.fn files and executes `test "name":` blocks.
func Run(opts Options) (*Report, error) {
	root := opts.Path
	if root == "" {
		root = "."
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	files, err := discoverTestFiles(abs)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no *_test.fn files found under %s", abs)
	}
	report := &Report{}
	for _, file := range files {
		cases, err := runFile(file, opts.Verbose)
		if err != nil {
			return report, err
		}
		for _, c := range cases {
			report.Tests = append(report.Tests, c)
			if c.Passed {
				report.Passed++
			} else {
				report.Failed++
			}
		}
	}
	return report, nil
}

func discoverTestFiles(path string) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		if strings.HasSuffix(path, "_test.fn") {
			return []string{path}, nil
		}
		return nil, fmt.Errorf("%s: not a *_test.fn file", path)
	}
	var files []string
	err = filepath.Walk(path, func(p string, fi os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if fi.IsDir() {
			base := filepath.Base(p)
			if base == ".funny" || base == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(fi.Name(), "_test.fn") {
			files = append(files, p)
		}
		return nil
	})
	return files, err
}

func runFile(file string, verbose bool) ([]CaseResult, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	p := parser.New(string(data), file)
	prog, err := p.Parse()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	prog, err = module.Resolve(prog, file)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	env := types.NewEnv(nil)
	if err := types.Check(prog, env); err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	tests := collectTests(prog)
	if len(tests) == 0 {
		return nil, fmt.Errorf("%s: no test blocks found", file)
	}
	var results []CaseResult
	for _, tb := range tests {
		start := time.Now()
		runErr := runOne(prog, file, tb)
		cr := CaseResult{
			File:     file,
			Name:     tb.Name,
			Passed:   runErr == nil,
			Duration: time.Since(start),
		}
		if runErr != nil {
			cr.Error = runErr.Error()
		}
		if verbose {
			status := "PASS"
			if !cr.Passed {
				status = "FAIL"
			}
			fmt.Printf("%s  %s  %s", status, file, tb.Name)
			if cr.Error != "" {
				fmt.Printf("  (%s)", cr.Error)
			}
			fmt.Println()
		}
		results = append(results, cr)
	}
	return results, nil
}

func collectTests(prog *ast.Program) []*ast.TestBlock {
	var out []*ast.TestBlock
	seen := map[string]bool{}
	for _, s := range prog.Stmts {
		tb, ok := s.(*ast.TestBlock)
		if !ok {
			continue
		}
		if seen[tb.Name] {
			continue
		}
		seen[tb.Name] = true
		out = append(out, tb)
	}
	return out
}

func runOne(full *ast.Program, file string, tb *ast.TestBlock) error {
	harness := harnessProgram(full, tb)
	mod, err := compiler.Compile(harness, file)
	if err != nil {
		return fmt.Errorf("compile test %q: %w", tb.Name, err)
	}
	m := vm.New(mod)
	if _, err := m.Run(); err != nil {
		return err
	}
	return nil
}

func harnessProgram(full *ast.Program, tb *ast.TestBlock) *ast.Program {
	var stmts []ast.Statement
	for _, s := range full.Stmts {
		switch s.(type) {
		case *ast.TestBlock, *ast.PlanBlock, *ast.MetaBlock:
			continue
		}
		stmts = append(stmts, s)
	}
	if tb.Body != nil {
		stmts = append(stmts, tb.Body.Statements...)
	}
	return &ast.Program{Stmts: stmts}
}
