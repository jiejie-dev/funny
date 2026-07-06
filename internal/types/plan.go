package types

import (
	"github.com/jiejie-dev/funny/v2/internal/ast"
)

func checkPlanBlock(n *ast.PlanBlock, env *Env) error {
	if n.Body == nil {
		return nil
	}
	stepNames := map[string]bool{}
	for _, stmt := range n.Body.Statements {
		step, ok := stmt.(*ast.Step)
		if !ok {
			continue
		}
		if stepNames[step.Name] {
			return New("E2110", "duplicate step name "+step.Name, step.NodePos)
		}
		stepNames[step.Name] = true
	}
	for _, stmt := range n.Body.Statements {
		switch s := stmt.(type) {
		case *ast.Step:
			for _, c := range s.BranchCases {
				if !stepNames[c.Target] {
					return New("E2111", "branch target "+c.Target+" not found in plan", s.NodePos)
				}
				if !branchCaseWildcard(c.Cond) {
					if _, err := CheckExpr(c.Cond, env); err != nil {
						return err
					}
				}
			}
			if s.Body != nil {
				if err := Check(s.Body.ToProgram(), env); err != nil {
					return err
				}
			}
		default:
			if err := checkStmt(stmt, env); err != nil {
				return err
			}
		}
	}
	return nil
}

func branchCaseWildcard(expr ast.Expression) bool {
	v, ok := expr.(*ast.VariableExpr)
	return ok && v.Name == "_"
}
