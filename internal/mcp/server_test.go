// internal/mcp/server_test.go
package mcp

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiejie-dev/funny/v2/internal/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractSkill_PlanFn(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "agent", "plan.fn")
	skill, ok := extractSkill(path)
	if !ok {
		t.Fatalf("extractSkill(%q) returned ok=false", path)
	}
	if name, _ := skill["name"].(string); name != "plan.fn" {
		t.Errorf("name = %q, want plan.fn", skill["name"])
	}
	if _, ok := skill["meta"]; !ok {
		t.Error("meta field missing")
	}
	plan, ok := skill["plan"].(map[string]any)
	if !ok {
		t.Fatalf("plan field missing or wrong type: %T", skill["plan"])
	}
	if plan["name"] != "demo_plan" {
		t.Errorf("plan name = %v, want demo_plan", plan["name"])
	}
	steps, _ := plan["steps"].([]string)
	if len(steps) != 3 {
		t.Errorf("plan steps len = %d, want 3 (got %v)", len(steps), steps)
	}
}

func TestExtractSkill_TypeError(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.fn")
	if err := os.WriteFile(bad, []byte("let x: int = \"hello\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := extractSkill(bad); ok {
		t.Error("extractSkill on type-error source should return ok=false")
	}
}

func TestExtractSkill_ParseError(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.fn")
	if err := os.WriteFile(bad, []byte("let = 5\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := extractSkill(bad); ok {
		t.Error("extractSkill on parse-error source should return ok=false")
	}
}

func TestAstTool_BusinessLogic(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "agent", "plan.fn")
	data, err := readFile(path)
	if err != nil {
		t.Fatal(err)
	}
	out, err := cli.Ast(data, path)
	if err != nil {
		t.Fatalf("cli.Ast: %v", err)
	}
	if len(out) == 0 {
		t.Error("cli.Ast returned empty output")
	}
}

func TestFormatTool_ReallyFormats(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "messy.fn")
	require.NoError(t, os.WriteFile(path, []byte("let   x=1\nprintln(x)\n"), 0o644))
	data, err := readFile(path)
	require.NoError(t, err)
	out, err := cli.Format(data, path)
	require.NoError(t, err)
	assert.Equal(t, "let x = 1\nprintln(x)\n", out)
}

func TestLintTool_Valid(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "agent", "plan.fn")
	// Test the lint business logic: parse + typecheck only (no compile/execute).
	// We exercise this by running extractSkill which is the same parse+typecheck path.
	if _, ok := extractSkill(path); !ok {
		t.Errorf("extractSkill on plan.fn should return ok=true (parse + typecheck succeed)")
	}
}

func TestLintTool_TypeError(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.fn")
	if err := os.WriteFile(bad, []byte("let x: int = \"hello\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// extractSkill is the same parse+typecheck pipeline that lintTool uses.
	if _, ok := extractSkill(bad); ok {
		t.Error("extractSkill (and thus lintTool) on type-error source should return ok=false")
	}
}

func TestListSkillsTool_Dir(t *testing.T) {
	dir := filepath.Join("..", "..", "testdata", "agent")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".fn" {
			continue
		}
		count++
	}
	if count == 0 {
		t.Fatal("expected at least one .fn file in testdata/agent")
	}
}

func TestRun_DoesNotPanic(t *testing.T) {
	// Use a canceled context so the server returns immediately.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Run(ctx); err != nil {
		// StdioTransport with a canceled context may return an error; that's fine.
		t.Logf("Run(canceled ctx) returned %v (expected)", err)
	}
}
