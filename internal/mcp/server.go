// internal/mcp/server.go
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiejie-dev/funny/internal/ast"
	"github.com/jiejie-dev/funny/internal/cli"
	"github.com/jiejie-dev/funny/internal/module"
	"github.com/jiejie-dev/funny/internal/parser"
	"github.com/jiejie-dev/funny/internal/types"
)

// Run starts the funny MCP server on stdio and blocks until ctx is canceled
// or the client disconnects.
func Run(ctx context.Context) error {
	server := mcp.NewServer(&mcp.Implementation{Name: "funny", Version: "2.0.0"}, nil)

	mcp.AddTool(server, &mcp.Tool{Name: "ast", Description: "Parse funny source and return the JSON AST."}, astTool)
	mcp.AddTool(server, &mcp.Tool{Name: "format", Description: "Format funny source code."}, formatTool)
	mcp.AddTool(server, &mcp.Tool{Name: "list_skills", Description: "List all .fn files in a directory and their meta blocks."}, listSkillsTool)
	mcp.AddTool(server, &mcp.Tool{Name: "describe_skill", Description: "Describe a single .fn file: meta + plan steps."}, describeSkillTool)
	mcp.AddTool(server, &mcp.Tool{Name: "run_skill", Description: "Execute a .fn file and return the result."}, runSkillTool)
	mcp.AddTool(server, &mcp.Tool{Name: "lint", Description: "Run type-check only; report errors without executing."}, lintTool)

	return server.Run(ctx, &mcp.StdioTransport{})
}

type pathArg struct {
	Path string `json:"path" jsonschema:"absolute path to a .fn source file"`
}

type dirArg struct {
	Dir string `json:"dir" jsonschema:"absolute path to a directory of .fn skill files"`
}

func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func astTool(ctx context.Context, req *mcp.CallToolRequest, args pathArg) (*mcp.CallToolResult, any, error) {
	data, err := readFile(args.Path)
	if err != nil {
		return nil, nil, err
	}
	out, err := cli.Ast(data, args.Path)
	if err != nil {
		return nil, nil, err
	}
	var parsed any
	if err := json.Unmarshal(out, &parsed); err != nil {
		return nil, nil, err
	}
	return nil, parsed, nil
}

func formatTool(ctx context.Context, req *mcp.CallToolRequest, args pathArg) (*mcp.CallToolResult, any, error) {
	data, err := readFile(args.Path)
	if err != nil {
		return nil, nil, err
	}
	out, err := cli.Format(data, args.Path)
	if err != nil {
		return nil, nil, err
	}
	return nil, out, nil
}

func listSkillsTool(ctx context.Context, req *mcp.CallToolRequest, args dirArg) (*mcp.CallToolResult, any, error) {
	entries, err := os.ReadDir(args.Dir)
	if err != nil {
		return nil, nil, err
	}
	skills := []map[string]any{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".fn") {
			continue
		}
		if skill, ok := extractSkill(filepath.Join(args.Dir, e.Name())); ok {
			skills = append(skills, skill)
		}
	}
	return nil, skills, nil
}

func describeSkillTool(ctx context.Context, req *mcp.CallToolRequest, args pathArg) (*mcp.CallToolResult, any, error) {
	skill, ok := extractSkill(args.Path)
	if !ok {
		return nil, nil, fmt.Errorf("not a valid skill file: %s", args.Path)
	}
	return nil, skill, nil
}

func runSkillTool(ctx context.Context, req *mcp.CallToolRequest, args pathArg) (*mcp.CallToolResult, any, error) {
	data, err := readFile(args.Path)
	if err != nil {
		return nil, nil, err
	}
	if err := cli.Run(data, args.Path); err != nil {
		return nil, map[string]any{"error": err.Error()}, nil
	}
	return nil, map[string]any{"status": "ok"}, nil
}

func lintTool(ctx context.Context, req *mcp.CallToolRequest, args pathArg) (*mcp.CallToolResult, any, error) {
	data, err := readFile(args.Path)
	if err != nil {
		return nil, nil, err
	}
	p := parser.New(string(data), args.Path)
	prog, err := p.Parse()
	if err != nil {
		return nil, map[string]any{"errors": []string{err.Error()}}, nil
	}
	prog, err = module.Resolve(prog, args.Path)
	if err != nil {
		return nil, map[string]any{"errors": []string{err.Error()}}, nil
	}
	env := types.NewEnv(nil)
	if err := types.Check(prog, env); err != nil {
		return nil, map[string]any{"errors": []string{err.Error()}}, nil
	}
	return nil, map[string]any{"status": "ok"}, nil
}

func extractSkill(path string) (map[string]any, bool) {
	data, err := readFile(path)
	if err != nil {
		return nil, false
	}
	p := parser.New(string(data), path)
	prog, err := p.Parse()
	if err != nil {
		return nil, false
	}
	env := types.NewEnv(nil)
	if err := types.Check(prog, env); err != nil {
		return nil, false
	}
	out := map[string]any{"name": filepath.Base(path), "path": path}
	var planSteps []string
	for _, s := range prog.Stmts {
		switch n := s.(type) {
		case *ast.MetaBlock:
			out["meta"] = n.Fields
		case *ast.PlanBlock:
			if n.Body != nil {
				for _, stmt := range n.Body.Statements {
					if step, ok := stmt.(*ast.Step); ok {
						planSteps = append(planSteps, step.Name)
					}
				}
			}
			out["plan"] = map[string]any{"name": n.Name, "steps": planSteps}
		}
	}
	return out, true
}
