package types

import (
	"fmt"

	"github.com/jerloo/funny/v2/internal/ast"
)

// CheckExpr type-checks an expression and returns its type.
func CheckExpr(expr ast.Expression, env *Env) (Type, error) {
	switch n := expr.(type) {
	case *ast.LiteralExpr:
		return literalType(n.Value), nil
	case *ast.VariableExpr:
		t, ok := env.LookupVar(n.Name)
		if !ok {
			return nil, New("E2001", fmt.Sprintf("undefined variable: %s", n.Name), n.NodePos)
		}
		return t, nil
	case *ast.BinaryExpr:
		return checkBinaryExpr(n, env)
	case *ast.UnaryExpr:
		return checkUnaryExpr(n, env)
	case *ast.CallExpr:
		return checkCallExpr(n, env)
	case *ast.IndexExpr:
		return checkIndexExpr(n, env)
	case *ast.FieldExpr:
		return checkFieldExpr(n, env)
	case *ast.ListExpr:
		return checkListLiteral(n, env)
	case *ast.SubExpr:
		return CheckExpr(n.Inner, env)
	case *ast.FStringExpr:
		return Primitive("str"), nil
	}
	return nil, New("E2099", fmt.Sprintf("type checker: unsupported expression %T", expr), expr.Pos())
}

// literalType infers a Type from a Go value.
func literalType(v any) Type {
	switch v.(type) {
	case int:
		return Primitive("int")
	case float64:
		return Primitive("float")
	case string:
		return Primitive("str")
	case bool:
		return Primitive("bool")
	case nil:
		return Primitive("nil")
	}
	return Primitive("unknown")
}

func checkBinaryExpr(n *ast.BinaryExpr, env *Env) (Type, error) {
	leftT, err := CheckExpr(n.Left, env)
	if err != nil {
		return nil, err
	}
	rightT, err := CheckExpr(n.Right, env)
	if err != nil {
		return nil, err
	}
	switch n.Op {
	case "+", "-", "*", "/", "%":
		if !Equal(leftT, rightT) {
			return nil, NewMismatch(n.NodePos, leftT, rightT)
		}
		return leftT, nil
	case "==", "!=", "<", ">", "<=", ">=":
		if !Equal(leftT, rightT) {
			return nil, NewMismatch(n.NodePos, leftT, rightT)
		}
		return Primitive("bool"), nil
	case "and", "or":
		if !Equal(leftT, Primitive("bool")) || !Equal(rightT, Primitive("bool")) {
			return nil, NewMismatch(n.NodePos, Primitive("bool"), leftT)
		}
		return Primitive("bool"), nil
	case "in":
		rightList, ok := rightT.(List)
		if !ok {
			return nil, New("E2050", fmt.Sprintf("'in' requires list on right side, got %s", rightT), n.NodePos)
		}
		if !Equal(leftT, rightList.Elem) {
			return nil, NewMismatch(n.NodePos, rightList.Elem, leftT)
		}
		return Primitive("bool"), nil
	}
	return nil, New("E2098", fmt.Sprintf("unsupported binary operator: %s", n.Op), n.NodePos)
}

func checkUnaryExpr(n *ast.UnaryExpr, env *Env) (Type, error) {
	inner, err := CheckExpr(n.Expr, env)
	if err != nil {
		return nil, err
	}
	switch n.Op {
	case "-":
		if !Equal(inner, Primitive("int")) && !Equal(inner, Primitive("float")) {
			return nil, NewMismatch(n.NodePos, Primitive("int"), inner)
		}
		return inner, nil
	case "not":
		if !Equal(inner, Primitive("bool")) {
			return nil, NewMismatch(n.NodePos, Primitive("bool"), inner)
		}
		return Primitive("bool"), nil
	}
	return nil, New("E2098", fmt.Sprintf("unsupported unary operator: %s", n.Op), n.NodePos)
}

func checkCallExpr(n *ast.CallExpr, env *Env) (Type, error) {
	varName, ok := n.Func.(*ast.VariableExpr)
	if !ok {
		return nil, New("E2070", "only direct function calls supported in M2-A", n.NodePos)
	}
	fn, ok := env.LookupFunc(varName.Name)
	if !ok {
		return nil, New("E2002", fmt.Sprintf("undefined function: %s", varName.Name), n.NodePos)
	}
	if len(n.Args) != fn.Arity() {
		return nil, New("E2020",
			fmt.Sprintf("%s expects %d args, got %d", varName.Name, fn.Arity(), len(n.Args)),
			n.NodePos)
	}
	for i, arg := range n.Args {
		argT, err := CheckExpr(arg, env)
		if err != nil {
			return nil, err
		}
		if !Equal(argT, fn.Params[i]) {
			return nil, NewMismatch(n.NodePos, fn.Params[i], argT)
		}
	}
	return fn.Return, nil
}

func checkIndexExpr(n *ast.IndexExpr, env *Env) (Type, error) {
	objT, err := CheckExpr(n.Object, env)
	if err != nil {
		return nil, err
	}
	idxT, err := CheckExpr(n.Index, env)
	if err != nil {
		return nil, err
	}
	if !Equal(idxT, Primitive("int")) {
		return nil, NewMismatch(n.NodePos, Primitive("int"), idxT)
	}
	switch t := objT.(type) {
	case List:
		return t.Elem, nil
	case Map:
		return t.Value, nil
	}
	return nil, New("E2050", fmt.Sprintf("cannot index into %s", objT), n.NodePos)
}

func checkFieldExpr(n *ast.FieldExpr, env *Env) (Type, error) {
	objT, err := CheckExpr(n.Object, env)
	if err != nil {
		return nil, err
	}
	s, ok := objT.(Struct)
	if !ok {
		return nil, New("E2051", fmt.Sprintf("field access requires struct, got %s", objT), n.NodePos)
	}
	f, ok := s.Field(n.Field)
	if !ok {
		return nil, New("E2052", fmt.Sprintf("struct %s has no field %q", s.Name, n.Field), n.NodePos)
	}
	return f, nil
}

func checkListLiteral(n *ast.ListExpr, env *Env) (Type, error) {
	if len(n.Elements) == 0 {
		return nil, New("E2011", "cannot infer type of empty list; add type annotation", n.NodePos)
	}
	first, err := CheckExpr(n.Elements[0], env)
	if err != nil {
		return nil, err
	}
	for i := 1; i < len(n.Elements); i++ {
		t, err := CheckExpr(n.Elements[i], env)
		if err != nil {
			return nil, err
		}
		if !Equal(t, first) {
			return nil, NewMismatch(n.Elements[i].Pos(), first, t)
		}
	}
	return List{Elem: first}, nil
}
