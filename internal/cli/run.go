// v2/internal/cli/run.go
package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jerloo/funny/internal/ast"
	"github.com/jerloo/funny/internal/compiler"
	"github.com/jerloo/funny/internal/evaluator"
	"github.com/jerloo/funny/internal/parser"
	"github.com/jerloo/funny/internal/types"
	"github.com/jerloo/funny/internal/vm"
)

// Run parses, type-checks, and executes the given source.
// By default uses the bytecode VM; set FUNNY_INTERPRET=1 to use the tree-walking evaluator.
func Run(src []byte, file string) error {
	p := parser.New(string(src), file)
	prog, err := p.Parse()
	if err != nil {
		return err
	}
	env := types.NewEnv(nil)
	if err := types.Check(prog, env); err != nil {
		return err
	}
	if os.Getenv("FUNNY_INTERPRET") != "" {
		e := evaluator.New(nil)
		return e.Exec(prog)
	}
	mod, err := compiler.Compile(prog, file)
	if err != nil {
		return fmt.Errorf("compile: %w", err)
	}
	m := vm.New(mod)
	if _, err := m.Run(); err != nil {
		return err
	}
	return nil
}

// Ast returns the JSON-serialized AST.
func Ast(src []byte, file string) ([]byte, error) {
	p := parser.New(string(src), file)
	prog, err := p.Parse()
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(prog, "", "  ")
}

// Disasm compiles and returns the human-readable bytecode disassembly.
func Disasm(src []byte, file string) (string, error) {
	p := parser.New(string(src), file)
	prog, err := p.Parse()
	if err != nil {
		return "", err
	}
	env := types.NewEnv(nil)
	if err := types.Check(prog, env); err != nil {
		return "", err
	}
	mod, err := compiler.Compile(prog, file)
	if err != nil {
		return "", err
	}
	return mod.Disassemble(), nil
}

// Describe returns a JSON representation of the plan/metadata for tools to consume.
func Describe(src []byte, file string) ([]byte, error) {
	p := parser.New(string(src), file)
	prog, err := p.Parse()
	if err != nil {
		return nil, err
	}
	var plan *ast.PlanBlock
	var meta *ast.MetaBlock
	for _, s := range prog.Stmts {
		switch n := s.(type) {
		case *ast.PlanBlock:
			plan = n
		case *ast.MetaBlock:
			meta = n
		}
	}
	out := map[string]any{}
	if meta != nil {
		out["meta"] = meta.Fields
	}
	if plan != nil {
		steps := []string{}
		if plan.Body != nil {
			for _, stmt := range plan.Body.Statements {
				if step, ok := stmt.(*ast.Step); ok {
					steps = append(steps, step.Name)
				}
			}
		}
		out["plan"] = map[string]any{
			"name":  plan.Name,
			"steps": steps,
		}
	}
	return json.MarshalIndent(out, "", "  ")
}
