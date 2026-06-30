// v2/internal/evaluator/evaluator.go
package evaluator

import (
	"fmt"

	"github.com/jerloo/funny/v2/internal/ast"
	"github.com/jerloo/funny/v2/internal/errs"
)

type Evaluator struct {
	scope *Scope
}

func New(scope *Scope) *Evaluator {
	if scope == nil {
		scope = NewScope(nil)
	}
	return &Evaluator{scope: scope}
}

func (e *Evaluator) Eval(node ast.Expression) (any, error) {
	switch n := node.(type) {
	case *ast.LiteralExpr:
		return n.Value, nil
	case *ast.VariableExpr:
		v, ok := e.scope.Get(n.Name)
		if !ok {
			return nil, errs.New("E2001", fmt.Sprintf("undefined variable: %s", n.Name), toErrPos(n.NodePos), "")
		}
		return v, nil
	case *ast.BinaryExpr:
		left, err := e.Eval(n.Left)
		if err != nil {
			return nil, err
		}
		right, err := e.Eval(n.Right)
		if err != nil {
			return nil, err
		}
		return applyBinary(n.Op, left, right)
	case *ast.UnaryExpr:
		v, err := e.Eval(n.Expr)
		if err != nil {
			return nil, err
		}
		switch n.Op {
		case "-":
			if i, ok := v.(int); ok {
				return -i, nil
			}
			if f, ok := v.(float64); ok {
				return -f, nil
			}
		case "not":
			return !truthy(v), nil
		}
	case *ast.SubExpr:
		return e.Eval(n.Inner)
	case *ast.ListExpr:
		out := make([]any, 0, len(n.Elements))
		for _, el := range n.Elements {
			v, err := e.Eval(el)
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		return out, nil
	case *ast.IndexExpr:
		obj, err := e.Eval(n.Object)
		if err != nil {
			return nil, err
		}
		idx, err := e.Eval(n.Index)
		if err != nil {
			return nil, err
		}
		i, ok := idx.(int)
		if !ok {
			return nil, errs.New("E2050", "index must be int", toErrPos(n.NodePos), "")
		}
		if list, ok := obj.([]any); ok {
			if i < 0 || i >= len(list) {
				return nil, errs.New("E2051", "index out of bounds", toErrPos(n.NodePos), "")
			}
			return list[i], nil
		}
	case *ast.FieldExpr:
		obj, err := e.Eval(n.Object)
		if err != nil {
			return nil, err
		}
		if m, ok := obj.(map[string]any); ok {
			v, ok := m[n.Field]
			if !ok {
				return nil, errs.New("E2061", fmt.Sprintf("no field %q", n.Field), toErrPos(n.NodePos), "")
			}
			return v, nil
		}
		return nil, errs.New("E2060", "field access requires map/struct", toErrPos(n.NodePos), "")
	case *ast.CallExpr:
		return e.evalCall(n)
	case *ast.FStringExpr:
		return n.Raw, nil
	}
	return nil, errs.New("E2002", fmt.Sprintf("cannot eval %T", node), toErrPos(node.Pos()), "")
}

func applyBinary(op string, l, r any) (any, error) {
	switch op {
	case "+":
		switch lv := l.(type) {
		case int:
			if rv, ok := r.(int); ok {
				return lv + rv, nil
			}
			if rv, ok := r.(float64); ok {
				return float64(lv) + rv, nil
			}
		case float64:
			if rv, ok := r.(float64); ok {
				return lv + rv, nil
			}
			if rv, ok := r.(int); ok {
				return lv + float64(rv), nil
			}
		case string:
			if rv, ok := r.(string); ok {
				return lv + rv, nil
			}
		}
	case "-":
		if lv, ok := l.(int); ok {
			if rv, ok := r.(int); ok {
				return lv - rv, nil
			}
		}
	case "*":
		if lv, ok := l.(int); ok {
			if rv, ok := r.(int); ok {
				return lv * rv, nil
			}
		}
	case "/":
		if lv, ok := l.(int); ok {
			if rv, ok := r.(int); ok {
				if rv == 0 {
					return nil, errs.New("E2030", "division by zero", errs.Position{}, "")
				}
				return lv / rv, nil
			}
		}
	case "==":
		return l == r || equalsLoose(l, r), nil
	case "!=":
		return !(l == r || equalsLoose(l, r)), nil
	case "<":
		return compare(l, r) < 0, nil
	case ">":
		return compare(l, r) > 0, nil
	case "<=":
		return compare(l, r) <= 0, nil
	case ">=":
		return compare(l, r) >= 0, nil
	case "and":
		return truthy(l) && truthy(r), nil
	case "or":
		return truthy(l) || truthy(r), nil
	case "in":
		if list, ok := r.([]any); ok {
			for _, v := range list {
				if v == l {
					return true, nil
				}
			}
		}
		return false, nil
	}
	return nil, errs.New("E2031", fmt.Sprintf("unsupported binary op: %s", op), errs.Position{}, "")
}

func equalsLoose(l, r any) bool {
	if li, ok := l.(int); ok {
		if rf, ok := r.(float64); ok {
			return float64(li) == rf
		}
	}
	return false
}

func compare(l, r any) int {
	if li, ok := l.(int); ok {
		if ri, ok := r.(int); ok {
			if li < ri {
				return -1
			}
			if li > ri {
				return 1
			}
			return 0
		}
	}
	if ls, ok := l.(string); ok {
		if rs, ok := r.(string); ok {
			if ls < rs {
				return -1
			}
			if ls > rs {
				return 1
			}
			return 0
		}
	}
	return 0
}

func truthy(v any) bool {
	if v == nil {
		return false
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return true
}

func (e *Evaluator) evalCall(n *ast.CallExpr) (any, error) {
	fn, ok := n.Func.(*ast.VariableExpr)
	if !ok {
		return nil, errs.New("E2070", "only direct function calls supported in M1", toErrPos(n.NodePos), "")
	}
	if b, ok := builtins[fn.Name]; ok {
		args := make([]any, len(n.Args))
		for i, a := range n.Args {
			v, err := e.Eval(a)
			if err != nil {
				return nil, err
			}
			args[i] = v
		}
		return b.fn(e, args)
	}
	v, ok := e.scope.Get(fn.Name)
	if !ok {
		return nil, errs.New("E2071", fmt.Sprintf("undefined function: %s", fn.Name), toErrPos(n.NodePos), "")
	}
	userFn, ok := v.(*ast.FnDecl)
	if !ok {
		return nil, errs.New("E2072", fmt.Sprintf("%s is not a function", fn.Name), toErrPos(n.NodePos), "")
	}
	if len(n.Args) != len(userFn.Params) {
		return nil, errs.New("E2073",
			fmt.Sprintf("%s expects %d args, got %d", fn.Name, len(userFn.Params), len(n.Args)),
			toErrPos(n.NodePos), "")
	}
	callScope := NewScope(e.scope)
	for i, p := range userFn.Params {
		av, err := e.Eval(n.Args[i])
		if err != nil {
			return nil, err
		}
		callScope.Set(p.Name, av)
	}
	saved := e.scope
	e.scope = callScope
	defer func() { e.scope = saved }()
	ret, hasRet, err := e.execBlock(userFn.Body)
	if err != nil {
		return nil, err
	}
	if hasRet {
		return ret, nil
	}
	return nil, nil
}

func (e *Evaluator) execBlock(b *ast.Block) (any, bool, error) {
	for _, s := range b.Statements {
		v, has, err := e.execStmt(s)
		if err != nil {
			return nil, false, err
		}
		if has {
			return v, true, nil
		}
	}
	return nil, false, nil
}

// Exec runs a Program.
func (e *Evaluator) Exec(prog *ast.Program) error {
	for _, s := range prog.Stmts {
		if _, _, err := e.execStmt(s); err != nil {
			return err
		}
	}
	return nil
}

func (e *Evaluator) execStmt(s ast.Statement) (any, bool, error) {
	switch n := s.(type) {
	case *ast.LetStmt:
		v, err := e.Eval(n.Value)
		if err != nil {
			return nil, false, err
		}
		e.scope.Set(n.Name, v)
		return nil, false, nil
	case *ast.AssignStmt:
		v, err := e.Eval(n.Value)
		if err != nil {
			return nil, false, err
		}
		if !e.scope.Assign(n.Target.String(), v) {
			switch t := n.Target.(type) {
			case *ast.VariableExpr:
				e.scope.Set(t.Name, v)
			default:
				return nil, false, errs.New("E2010",
					fmt.Sprintf("cannot assign to %s", n.Target.String()),
					toErrPos(n.NodePos), "")
			}
		}
		return nil, false, nil
	case *ast.IfStmt:
		cond, err := e.Eval(n.Cond)
		if err != nil {
			return nil, false, err
		}
		if truthy(cond) {
			return e.execBlock(n.Then)
		}
		if n.ElseIf != nil {
			return e.execStmt(n.ElseIf)
		}
		if n.ElseBlock != nil {
			return e.execBlock(n.ElseBlock)
		}
		return nil, false, nil
	case *ast.ForStmt:
		iterable, err := e.Eval(n.Iterable)
		if err != nil {
			return nil, false, err
		}
		list, ok := iterable.([]any)
		if !ok {
			return nil, false, errs.New("E2011", "for-in requires list", toErrPos(n.NodePos), "")
		}
		for _, item := range list {
			saved := e.scope
			iterScope := NewScope(e.scope)
			iterScope.Set(n.Name, item)
			e.scope = iterScope
			_, has, err := e.execBlock(n.Body)
			e.scope = saved
			if err != nil {
				return nil, false, err
			}
			if has {
				return nil, true, nil
			}
		}
		return nil, false, nil
	case *ast.WhileStmt:
		for {
			cond, err := e.Eval(n.Cond)
			if err != nil {
				return nil, false, err
			}
			if !truthy(cond) {
				break
			}
			_, has, err := e.execBlock(n.Body)
			if err != nil {
				return nil, false, err
			}
			if has {
				return nil, true, nil
			}
		}
		return nil, false, nil
	case *ast.ReturnStmt:
		if n.Value == nil {
			return nil, true, nil
		}
		v, err := e.Eval(n.Value)
		if err != nil {
			return nil, false, err
		}
		return v, true, nil
	case *ast.ExprStmt:
		_, err := e.Eval(n.X)
		if err != nil {
			return nil, false, err
		}
		return nil, false, nil
	case *ast.BreakStmt:
		return nil, false, errs.New("E2012", "break outside for/while", toErrPos(n.NodePos), "")
	case *ast.ContinueStmt:
		return nil, false, errs.New("E2013", "continue outside for/while", toErrPos(n.NodePos), "")
	case *ast.FnDecl:
		e.scope.Set(n.Name, n)
		return nil, false, nil
	case *ast.StructDecl:
		e.scope.Set(n.Name, n)
		return nil, false, nil
	case *ast.MetaBlock:
		return nil, false, nil
	case *ast.PlanBlock:
		return nil, false, nil
	case *ast.ImportDecl:
		return nil, false, nil
	}
	return nil, false, errs.New("E2014", fmt.Sprintf("cannot exec %T", s), toErrPos(s.Pos()), "")
}

func toErrPos(p ast.Pos) errs.Position {
	return errs.Position{File: p.File, Line: p.Line, Col: p.Col}
}
