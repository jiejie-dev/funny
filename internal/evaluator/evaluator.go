// v2/internal/evaluator/evaluator.go
package evaluator

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jiejie-dev/funny/v2/internal/ast"
	"github.com/jiejie-dev/funny/v2/internal/errs"
	"github.com/jiejie-dev/funny/v2/internal/strfmt"
)

var (
	errLoopBreak    = errors.New("loop break")
	errLoopContinue = errors.New("loop continue")
)

type Evaluator struct {
	scope     *Scope
	loopDepth int
}

func New(scope *Scope) *Evaluator {
	if scope == nil {
		scope = NewScope(nil)
	}
	return &Evaluator{scope: scope}
}

// Scope returns the current execution scope.
func (e *Evaluator) Scope() *Scope {
	return e.scope
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
		if m, ok := obj.(map[string]any); ok {
			ks, ok := idx.(string)
			if !ok {
				ks = fmt.Sprintf("%v", idx)
			}
			v, ok := m[ks]
			if !ok {
				return nil, errs.New("E2051", fmt.Sprintf("key not found: %q", ks), toErrPos(n.NodePos), "")
			}
			return v, nil
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
		return e.evalFString(n)
	case *ast.StructLiteralExpr:
		fields := map[string]any{}
		for k, v := range n.Fields {
			val, err := e.Eval(v)
			if err != nil {
				return nil, err
			}
			fields[k] = val
		}
		return fields, nil
	case *ast.MapLiteralExpr:
		return e.evalMapLiteral(n)
	}
	return nil, errs.New("E2002", fmt.Sprintf("cannot eval %T", node), toErrPos(node.Pos()), "")
}

// assignIndex evaluates `obj[idx] = val`. Go lists ([]any) and maps
// (map[string]any) are both reference types, so mutating the element/entry
// after evaluating n.Object is visible through any other reference to the
// same underlying list/map (e.g. the variable it came from).
func (e *Evaluator) assignIndex(n *ast.IndexExpr, val any) error {
	obj, err := e.Eval(n.Object)
	if err != nil {
		return err
	}
	idx, err := e.Eval(n.Index)
	if err != nil {
		return err
	}
	if m, ok := obj.(map[string]any); ok {
		ks, ok := idx.(string)
		if !ok {
			ks = fmt.Sprintf("%v", idx)
		}
		m[ks] = val
		return nil
	}
	if list, ok := obj.([]any); ok {
		i, ok := idx.(int)
		if !ok {
			return errs.New("E2050", "index must be int", toErrPos(n.NodePos), "")
		}
		if i < 0 || i >= len(list) {
			return errs.New("E2051", "index out of bounds", toErrPos(n.NodePos), "")
		}
		list[i] = val
		return nil
	}
	return errs.New("E2050", "cannot index-assign into non-list/map", toErrPos(n.NodePos), "")
}

// evalMapLiteral evaluates a `{key: value, ...}` literal into a
// map[string]any, coercing non-string keys the same way the VM's BUILD_MAP
// instruction does, so both execution paths agree on runtime representation.
func (e *Evaluator) evalMapLiteral(n *ast.MapLiteralExpr) (any, error) {
	m := make(map[string]any, len(n.Keys))
	for i, k := range n.Keys {
		kv, err := e.Eval(k)
		if err != nil {
			return nil, err
		}
		vv, err := e.Eval(n.Values[i])
		if err != nil {
			return nil, err
		}
		ks, ok := kv.(string)
		if !ok {
			ks = fmt.Sprintf("%v", kv)
		}
		m[ks] = vv
	}
	return m, nil
}

// evalFString evaluates an f-string by concatenating literal text with the
// formatted result of each interpolated expression.
func (e *Evaluator) evalFString(n *ast.FStringExpr) (any, error) {
	var b strings.Builder
	for _, part := range n.Parts {
		if part.Expr == nil {
			b.WriteString(part.Text)
			continue
		}
		v, err := e.Eval(part.Expr)
		if err != nil {
			return nil, err
		}
		s, ferr := strfmt.Format(v, part.Spec)
		if ferr != nil {
			return nil, errs.New("E2090", ferr.Error(), toErrPos(n.NodePos), "")
		}
		b.WriteString(s)
	}
	return b.String(), nil
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
	if isBuiltin(fn.Name) {
		args := make([]any, len(n.Args))
		for i, a := range n.Args {
			v, err := e.Eval(a)
			if err != nil {
				return nil, err
			}
			args[i] = v
		}
		return callBuiltin(fn.Name, args)
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
			if errors.Is(err, errLoopBreak) || errors.Is(err, errLoopContinue) {
				return nil, false, err
			}
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
			if errors.Is(err, errLoopBreak) {
				return errs.New("E2012", "break outside for/while", toErrPos(s.Pos()), "")
			}
			if errors.Is(err, errLoopContinue) {
				return errs.New("E2013", "continue outside for/while", toErrPos(s.Pos()), "")
			}
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
		if idx, ok := n.Target.(*ast.IndexExpr); ok {
			if err := e.assignIndex(idx, v); err != nil {
				return nil, false, err
			}
			return nil, false, nil
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
		e.loopDepth++
		defer func() { e.loopDepth-- }()
		for _, item := range list {
			saved := e.scope
			iterScope := NewScope(e.scope)
			iterScope.Set(n.Name, item)
			e.scope = iterScope
			_, has, err := e.execBlock(n.Body)
			e.scope = saved
			if err != nil {
				if errors.Is(err, errLoopBreak) {
					break
				}
				if errors.Is(err, errLoopContinue) {
					continue
				}
				return nil, false, err
			}
			if has {
				return nil, true, nil
			}
		}
		return nil, false, nil
	case *ast.WhileStmt:
		e.loopDepth++
		defer func() { e.loopDepth-- }()
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
				if errors.Is(err, errLoopBreak) {
					break
				}
				if errors.Is(err, errLoopContinue) {
					continue
				}
				return nil, false, err
			}
			if has {
				return nil, true, nil
			}
		}
		return nil, false, nil
	case *ast.MatchStmt:
		scrutinee, err := e.Eval(n.Expr)
		if err != nil {
			return nil, false, err
		}
		for _, arm := range n.Arms {
			matched, err := e.patternMatches(scrutinee, arm.Pattern)
			if err != nil {
				return nil, false, err
			}
			if !matched {
				continue
			}
			v, has, err := e.execBlock(arm.Body)
			if err != nil {
				if errors.Is(err, errLoopBreak) || errors.Is(err, errLoopContinue) {
					return nil, false, err
				}
				return nil, false, err
			}
			return v, has, nil
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
		if e.loopDepth == 0 {
			return nil, false, errs.New("E2012", "break outside for/while", toErrPos(n.NodePos), "")
		}
		return nil, false, errLoopBreak
	case *ast.ContinueStmt:
		if e.loopDepth == 0 {
			return nil, false, errs.New("E2013", "continue outside for/while", toErrPos(n.NodePos), "")
		}
		return nil, false, errLoopContinue
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
	case *ast.CommentStmt:
		return nil, false, nil
	}
	return nil, false, errs.New("E2014", fmt.Sprintf("cannot exec %T", s), toErrPos(s.Pos()), "")
}

func toErrPos(p ast.Pos) errs.Position {
	return errs.Position{File: p.File, Line: p.Line, Col: p.Col}
}

func (e *Evaluator) patternMatches(scrutinee any, pattern ast.Expression) (bool, error) {
	if v, ok := pattern.(*ast.VariableExpr); ok && v.Name == "_" {
		return true, nil
	}
	if v, ok := pattern.(*ast.VariableExpr); ok {
		other, ok := e.scope.Get(v.Name)
		if !ok {
			return false, errs.New("E2001", fmt.Sprintf("undefined variable: %s", v.Name), toErrPos(v.NodePos), "")
		}
		return valuesEqual(scrutinee, other), nil
	}
	pv, err := e.Eval(pattern)
	if err != nil {
		return false, err
	}
	return valuesEqual(scrutinee, pv), nil
}

func valuesEqual(a, b any) bool {
	if a == b {
		return true
	}
	eq, err := applyBinary("==", a, b)
	if err != nil {
		return false
	}
	if b, ok := eq.(bool); ok {
		return b
	}
	return false
}
