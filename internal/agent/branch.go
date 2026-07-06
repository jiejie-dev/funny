package agent

import (
	"fmt"

	"github.com/jiejie-dev/funny/v2/internal/ast"
)

type planContext struct {
	steps         map[string]*ast.Step
	branchTargets map[string]bool
	stmts         []ast.Statement
}

func buildPlanContext(plan *ast.PlanBlock) *planContext {
	pc := &planContext{
		steps:         map[string]*ast.Step{},
		branchTargets: map[string]bool{},
	}
	if plan.Body == nil {
		return pc
	}
	pc.stmts = plan.Body.Statements
	for _, stmt := range plan.Body.Statements {
		step, ok := stmt.(*ast.Step)
		if !ok {
			continue
		}
		pc.steps[step.Name] = step
		for _, c := range step.BranchCases {
			pc.branchTargets[c.Target] = true
		}
	}
	return pc
}

func (e *Engine) execPlanStatements(pc *planContext) error {
	for _, stmt := range pc.stmts {
		step, ok := stmt.(*ast.Step)
		if !ok {
			if _, _, err := e.execStmt(stmt); err != nil {
				return err
			}
			continue
		}
		if pc.branchTargets[step.Name] {
			continue
		}
		if err := e.execPlanStep(step, pc); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) execPlanStep(s *ast.Step, pc *planContext) error {
	if s.Kind == ast.StepBranch && len(s.BranchCases) > 0 {
		return e.execBranchCases(s, pc)
	}
	return e.execStep(s)
}

func (e *Engine) execBranchCases(s *ast.Step, pc *planContext) error {
	targetName, err := e.pickBranchTarget(s)
	if err != nil {
		return fmt.Errorf("step %q: %w", s.Name, err)
	}
	target, ok := pc.steps[targetName]
	if !ok {
		return fmt.Errorf("step %q: branch target %q not found", s.Name, targetName)
	}
	return e.execStep(target)
}

func (e *Engine) pickBranchTarget(s *ast.Step) (string, error) {
	for _, c := range s.BranchCases {
		if branchWildcard(c.Cond) {
			return c.Target, nil
		}
		v, err := e.eval.Eval(c.Cond)
		if err != nil {
			return "", err
		}
		if truthy(v) {
			return c.Target, nil
		}
	}
	return "", fmt.Errorf("no branch case matched")
}

func branchWildcard(expr ast.Expression) bool {
	v, ok := expr.(*ast.VariableExpr)
	return ok && v.Name == "_"
}
