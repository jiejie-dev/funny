package lsp

import (
	"fmt"
	"strconv"

	"github.com/jiejie-dev/funny/internal/ast"
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
//   - All other kinds (tool/guard/transform/branch/delay) currently
//     execute identically at runtime (see execStep's default case) — the
//     graph still reports their declared Kind since that's the intended
//     semantic label, but does not invent branching/delay edges the
//     engine doesn't yet implement.
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

	var prevID string
	for i, stmt := range plan.Body.Statements {
		step, ok := stmt.(*ast.Step)
		if !ok {
			continue // a plan body may in principle hold other statements; only `step`s are graphed
		}
		id := fmt.Sprintf("step-%d", i)
		node := PlanNode{
			ID:      id,
			Label:   step.Name,
			Kind:    step.Kind.String(),
			Range:   stepRange(step),
			Timeout: step.Timeout,
			Retry:   retryInfo(step.Retry),
		}
		g.Nodes = append(g.Nodes, node)
		if prevID != "" {
			g.Edges = append(g.Edges, PlanEdge{From: prevID, To: id, Kind: "sequence"})
		}
		prevID = id

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
