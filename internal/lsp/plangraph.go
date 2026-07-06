package lsp

import (
	"fmt"
	"strconv"

	"github.com/jiejie-dev/funny/v2/internal/ast"
)

// planGraphs implements the custom funny/planGraph request: it turns every
// `plan "..."` block in the document into a node/edge graph an editor can
// render directly, instead of asking the user to read step syntax.
//
// Graph shape mirrors internal/agent/engine.go's actual execution
// semantics rather than the grammar alone:
//   - Top-level steps run sequentially: step[i] -> step[i+1] ("sequence").
//   - A `parallel` step's body statements each run concurrently as
//     independent tasks (execParallel spawns one goroutine per statement,
//     not one per named sub-step — there's no nested `step` construct);
//     those are modeled as child "task" nodes connected to the parallel
//     step with "parallel" edges, and the *next* top-level step is
//     connected from the parallel step itself (matching wg.Wait()
//     rejoining before execution continues).
//   - `guard`, `delay`, and retry `backoff`/`timeout` now have real
//     engine semantics (see internal/agent/engine.go), but none of them
//     change the *graph shape* — a guard's pass/fail assertion and a
//     delay's sleep both happen inside a single node, they don't fan out
//     into separate nodes/edges. A `branch` step with a case-list fans out
//     to its target step nodes via "branch" edges; target steps are skipped
//     in the linear "sequence" chain (they only run when selected).
func (d *document) planGraphs() PlanGraphResult {
	result := PlanGraphResult{Plans: []PlanGraph{}}
	if d.prog == nil {
		return result
	}
	for _, s := range d.prog.Stmts {
		plan, ok := s.(*ast.PlanBlock)
		if !ok {
			continue
		}
		result.Plans = append(result.Plans, buildPlanGraph(plan))
	}
	return result
}

func buildPlanGraph(plan *ast.PlanBlock) PlanGraph {
	g := PlanGraph{
		Name:  plan.Name,
		Range: lineRange(plan.NodePos.Line, plan.NodePos.Line),
		Nodes: []PlanNode{},
		Edges: []PlanEdge{},
	}
	if plan.Body == nil {
		return g
	}

	branchTargets := planBranchTargets(plan)
	nameToID := map[string]string{}
	for i, stmt := range plan.Body.Statements {
		if step, ok := stmt.(*ast.Step); ok {
			nameToID[step.Name] = fmt.Sprintf("step-%d", i)
		}
	}

	var prevID string
	for i, stmt := range plan.Body.Statements {
		step, ok := stmt.(*ast.Step)
		if !ok {
			continue
		}
		id := fmt.Sprintf("step-%d", i)
		g.Nodes = append(g.Nodes, PlanNode{
			ID:      id,
			Label:   step.Name,
			Kind:    step.Kind.String(),
			Range:   stepRange(step),
			Timeout: step.Timeout,
			Retry:   retryInfo(step.Retry),
		})

		isTarget := branchTargets[step.Name]
		if !isTarget && prevID != "" {
			g.Edges = append(g.Edges, PlanEdge{From: prevID, To: id, Kind: "sequence"})
		}
		if step.Kind == ast.StepBranch && len(step.BranchCases) > 0 {
			for _, c := range step.BranchCases {
				if targetID, ok := nameToID[c.Target]; ok {
					g.Edges = append(g.Edges, PlanEdge{From: id, To: targetID, Kind: "branch"})
				}
			}
		}
		if !isTarget {
			prevID = id
		}

		if step.Kind == ast.StepParallel && step.Body != nil {
			for j, taskStmt := range step.Body.Statements {
				taskID := fmt.Sprintf("%s-task-%d", id, j)
				g.Nodes = append(g.Nodes, PlanNode{
					ID:       taskID,
					Label:    stmtSummary(taskStmt),
					Kind:     "task",
					Range:    lineRange(taskStmt.Pos().Line, taskStmt.Pos().Line),
					ParentID: id,
				})
				g.Edges = append(g.Edges, PlanEdge{From: id, To: taskID, Kind: "parallel"})
			}
		}
	}
	return g
}

func planBranchTargets(plan *ast.PlanBlock) map[string]bool {
	targets := map[string]bool{}
	if plan.Body == nil {
		return targets
	}
	for _, stmt := range plan.Body.Statements {
		step, ok := stmt.(*ast.Step)
		if !ok {
			continue
		}
		for _, c := range step.BranchCases {
			targets[c.Target] = true
		}
	}
	return targets
}

func retryInfo(r *ast.Retry) *RetryInfo {
	if r == nil {
		return nil
	}
	return &RetryInfo{Max: r.Max, Backoff: r.Backoff, On: r.On}
}

// stepRange highlights just the step's header line (its `step "name" ->
// kind:` line), matching planStepSymbols' SelectionRange convention in
// docsymbols.go — the body's statements get their own nodes only for
// `parallel` children, so there's no need for a full-body span here.
func stepRange(s *ast.Step) Range {
	return lineRange(s.NodePos.Line, s.NodePos.Line)
}

// stmtSummary renders a short, human-readable label for a plan statement
// that has no declared name of its own (used for a `parallel` step's
// concurrent body statements).
func stmtSummary(s ast.Statement) string {
	switch n := s.(type) {
	case *ast.ExprStmt:
		return exprSummary(n.X)
	case *ast.AssignStmt:
		return exprSummary(n.Target) + " = " + exprSummary(n.Value)
	case *ast.LetStmt:
		return "let " + n.Name
	default:
		return fmt.Sprintf("<stmt @ line %d>", s.Pos().Line+1)
	}
}

func exprSummary(e ast.Expression) string {
	switch n := e.(type) {
	case *ast.CallExpr:
		unit := "args"
		if len(n.Args) == 1 {
			unit = "arg"
		}
		return exprSummary(n.Func) + "(" + strconv.Itoa(len(n.Args)) + " " + unit + ")"
	case *ast.VariableExpr:
		return n.Name
	case *ast.FieldExpr:
		return exprSummary(n.Object) + "." + n.Field
	default:
		return "<expr>"
	}
}
