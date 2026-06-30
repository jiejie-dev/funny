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

// Check type-checks a full program.
func Check(prog *ast.Program, env *Env) error {
	for _, s := range prog.Stmts {
		if err := checkStmt(s, env); err != nil {
			return err
		}
	}
	return nil
}

func checkStmt(s ast.Statement, env *Env) error {
	switch n := s.(type) {
	case *ast.LetStmt:
		return checkLet(n, env)
	case *ast.AssignStmt:
		return checkAssign(n, env)
	case *ast.IfStmt:
		return checkIf(n, env)
	case *ast.ForStmt:
		return checkFor(n, env)
	case *ast.WhileStmt:
		return checkWhile(n, env)
	case *ast.ReturnStmt:
		return checkReturn(n, env)
	case *ast.FnDecl:
		return checkFnDecl(n, env)
	case *ast.StructDecl:
		return checkStructDecl(n, env)
	case *ast.ExprStmt, *ast.BreakStmt, *ast.ContinueStmt, *ast.MetaBlock, *ast.PlanBlock, *ast.ImportDecl:
		return nil // M2-A doesn't type-check these
	}
	return New("E2099", fmt.Sprintf("unsupported statement %T", s), s.Pos())
}

func checkLet(n *ast.LetStmt, env *Env) error {
	valT, err := CheckExpr(n.Value, env)
	if err != nil {
		return err
	}
	var declared Type
	if n.TypeAnn != "" {
		declared, err = ParseType(n.TypeAnn)
		if err != nil {
			return New("E2012", fmt.Sprintf("invalid type annotation %q: %v", n.TypeAnn, err), n.NodePos)
		}
		if !Equal(valT, declared) {
			return NewMismatch(n.NodePos, declared, valT)
		}
	} else {
		declared = valT
	}
	env.DeclareVar(n.Name, declared)
	return nil
}

func checkAssign(n *ast.AssignStmt, env *Env) error {
	valT, err := CheckExpr(n.Value, env)
	if err != nil {
		return err
	}
	targetT, err := CheckExpr(n.Target, env)
	if err != nil {
		return err
	}
	if !Equal(valT, targetT) {
		return NewMismatch(n.NodePos, targetT, valT)
	}
	return nil
}

func checkIf(n *ast.IfStmt, env *Env) error {
	condT, err := CheckExpr(n.Cond, env)
	if err != nil {
		return err
	}
	if !Equal(condT, Primitive("bool")) {
		return NewMismatch(n.NodePos, Primitive("bool"), condT)
	}
	if err := Check(n.Then.ToProgram(), env); err != nil {
		return err
	}
	if n.ElseIf != nil {
		return checkIf(n.ElseIf, env)
	}
	if n.ElseBlock != nil {
		return Check(n.ElseBlock.ToProgram(), env)
	}
	return nil
}

func checkFor(n *ast.ForStmt, env *Env) error {
	iterT, err := CheckExpr(n.Iterable, env)
	if err != nil {
		return err
	}
	listT, ok := iterT.(List)
	if !ok {
		return New("E2050", fmt.Sprintf("for-in requires list, got %s", iterT), n.NodePos)
	}
	bodyEnv := NewEnv(env)
	bodyEnv.DeclareVar(n.Name, listT.Elem)
	return Check(n.Body.ToProgram(), bodyEnv)
}

func checkWhile(n *ast.WhileStmt, env *Env) error {
	condT, err := CheckExpr(n.Cond, env)
	if err != nil {
		return err
	}
	if !Equal(condT, Primitive("bool")) {
		return NewMismatch(n.NodePos, Primitive("bool"), condT)
	}
	return Check(n.Body.ToProgram(), env)
}

func checkReturn(n *ast.ReturnStmt, env *Env) error {
	if n.Value == nil {
		return nil
	}
	valT, err := CheckExpr(n.Value, env)
	if err != nil {
		return err
	}
	retT, ok := env.LookupVar("__return_type__")
	if !ok {
		return nil
	}
	expected, ok := retT.(Type)
	if !ok {
		return nil
	}
	if !Equal(valT, expected) {
		return NewMismatch(n.NodePos, expected, valT)
	}
	return nil
}

func checkFnDecl(n *ast.FnDecl, env *Env) error {
	var retType Type = Primitive("nil")
	if n.RetType != "" {
		var err error
		retType, err = ParseType(n.RetType)
		if err != nil {
			return New("E2012", fmt.Sprintf("invalid return type %q: %v", n.RetType, err), n.NodePos)
		}
	}
	var paramTypes []Type
	for _, p := range n.Params {
		if p.TypeAnn == "" {
			return New("E2013", fmt.Sprintf("parameter %q missing type annotation", p.Name), n.NodePos)
		}
		pt, err := ParseType(p.TypeAnn)
		if err != nil {
			return New("E2012", fmt.Sprintf("invalid type for parameter %q: %v", p.Name, err), n.NodePos)
		}
		paramTypes = append(paramTypes, pt)
	}
	env.DeclareFunc(n.Name, Func{Params: paramTypes, Return: retType})
	bodyEnv := NewEnv(env)
	bodyEnv.DeclareVar("__return_type__", retType)
	for i, p := range n.Params {
		bodyEnv.DeclareVar(p.Name, paramTypes[i])
	}
	return Check(n.Body.ToProgram(), bodyEnv)
}

func checkStructDecl(n *ast.StructDecl, env *Env) error {
	fields := map[string]Type{}
	for _, f := range n.Fields {
		if f.TypeAnn == "" {
			return New("E2013", fmt.Sprintf("struct field %q missing type annotation", f.Name), n.NodePos)
		}
		ft, err := ParseType(f.TypeAnn)
		if err != nil {
			return New("E2012", fmt.Sprintf("invalid type for field %q: %v", f.Name, err), n.NodePos)
		}
		fields[f.Name] = ft
	}
	env.DeclareStruct(n.Name, Struct{Name: n.Name, Fields: fields})
	return nil
}
