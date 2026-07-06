package types

import (
	"fmt"

	"github.com/jiejie-dev/funny/v2/internal/ast"
	"github.com/jiejie-dev/funny/v2/internal/strfmt"
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
	case *ast.MapLiteralExpr:
		return checkMapLiteral(n, env)
	case *ast.StructLiteralExpr:
		return checkStructLiteral(n, env)
	case *ast.SubExpr:
		return CheckExpr(n.Inner, env)
	case *ast.FStringExpr:
		return checkFString(n, env)
	case *ast.TryExpr:
		return checkTry(n, env)
	}
	return nil, New("E2099", fmt.Sprintf("type checker: unsupported expression %T", expr), expr.Pos())
}

// checkTry type-checks `expr?`. The accepted operand is the inner expression's
// type. When the operand is a Result, `?` propagates Err (early-returns from
// the current function) but does NOT unwrap the Ok value: the type of `expr?`
// is still Result, and the user chains `.val` (or `.tag`) on it. For non-Result
// operands, `?` is treated as a transparent pass-through (the runtime leaves
// the value on the stack unchanged).
func checkTry(n *ast.TryExpr, env *Env) (Type, error) {
	innerT, err := CheckExpr(n.Inner, env)
	if err != nil {
		return nil, err
	}
	if _, ok := innerT.(Result); ok {
		return innerT, nil
	}
	if p, ok := innerT.(Primitive); ok && string(p) == "Result" {
		return innerT, nil
	}
	return innerT, nil
}

// checkFString type-checks every interpolated expression inside an f-string
// and validates each part's format spec syntax. The overall type is always
// str, regardless of what's interpolated (interpolation always stringifies).
func checkFString(n *ast.FStringExpr, env *Env) (Type, error) {
	for _, part := range n.Parts {
		if part.Expr == nil {
			continue
		}
		if _, err := CheckExpr(part.Expr, env); err != nil {
			return nil, err
		}
		if _, err := strfmt.ParseSpec(part.Spec); err != nil {
			return nil, New("E2090", fmt.Sprintf("invalid format spec %q: %v", part.Spec, err), n.NodePos)
		}
	}
	return Primitive("str"), nil
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
	// Handle `expr?.field` first: `?` returns immediately from parsePostfix so
	// the parser produces BinaryExpr{TryExpr, ".", VariableExpr} instead of FieldExpr.
	// Reinterpret the "." as field access without checking the Right as a value expression.
	if n.Op == "." {
		varExpr, ok := n.Right.(*ast.VariableExpr)
		if !ok {
			return nil, New("E2098", "field name must be identifier", n.NodePos)
		}
		return checkFieldExpr(&ast.FieldExpr{NodePos: n.NodePos, Object: n.Left, Field: varExpr.Name}, env)
	}
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

// builtinTypeNames is the set of names recognized by the type checker as
// builtins (the compiler and VM also know these). Their return types are
// reported as Primitive("any") since the type checker does not narrow them.
//
// regex_match/regex_replace/env_get/file_read/file_exists/http_get/md5/
// sha256/b64_encode/b64_decode/jwt_encode/jwt_decode/sql_open were
// implemented in internal/vm/builtins.go (and documented in
// docs/language-manual.md) but never added here — since checkCallExpr
// rejects any name that's neither a builtin nor a user-declared function
// with E2002 "undefined function" *before* the compiler ever runs, every
// one of them was actually uncallable from a real .fn script; only
// Go-level VM tests that hand-build bytecode (bypassing the type checker)
// exercised them. See internal/compiler/fn.go's builtinNames for the
// matching compiler-side allowlist that had to be fixed alongside this one.
var builtinTypeNames = map[string]bool{
	"print":         true,
	"println":       true,
	"len":           true,
	"to_str":        true,
	"to_int":        true,
	"to_float":      true,
	"type_of":       true,
	"to_json":       true,
	"parse_json":    true,
	"now":           true,
	"time_format":   true,
	"sqrt":          true,
	"pow":           true,
	"abs":           true,
	"str_upper":     true,
	"str_lower":     true,
	"str_contains":  true,
	"str_split":     true,
	"regex_match":   true,
	"regex_replace": true,
	"env_get":       true,
	"file_read":     true,
	"file_exists":   true,
	"http_get":      true,
	"md5":           true,
	"sha256":        true,
	"b64_encode":    true,
	"b64_decode":    true,
	"jwt_encode":    true,
	"jwt_decode":    true,
	"sql_open":      true,
	"append":        true,
}

// builtinResultReturns lists builtins that return a Result, so the `?` operator
// can be applied to them at the type level. Only builtins whose
// internal/vm/builtins.go implementation *consistently* pushes a
// makeResult(...) on both the success and failure path qualify; jwt_encode
// and sql_open return a plain string on success and a Result only on
// failure, so they're deliberately left out (and just typed `any`) rather
// than claiming a Result shape their success value doesn't actually have.
var builtinResultReturns = map[string]bool{
	"file_read":  true,
	"http_get":   true,
	"b64_decode": true,
	"jwt_decode": true,
}

// BuiltinNames returns the sorted-by-declaration list of builtin function
// names the type checker (and therefore the compiler/VM/evaluator) knows
// about. Exposed for tooling (e.g. the LSP server's completion provider).
func BuiltinNames() []string {
	names := make([]string, 0, len(builtinTypeNames)+2)
	for name := range builtinTypeNames {
		names = append(names, name)
	}
	names = append(names, "ok", "err")
	return names
}

// builtinReturnType returns the type checker's view of a builtin call's
// result type. Concrete types are reported for the builtins whose result
// is always (or, for `abs`, argument-dependently) a single primitive -
// without this, e.g. `fn f() -> float: return sqrt(x)` failed with a
// spurious "expected float, got any" mismatch (Primitive("any") has no
// special-cased compatibility with concrete types in Equal), even though
// sqrt always returns a float. Falls back to Primitive("any") for
// builtins whose result isn't a single primitive (list/map/no value) or
// isn't narrowed here; mirrors internal/compiler.builtinValueType, which
// solves the same problem for the bytecode compiler's static value typing.
func builtinReturnType(name string, argTypes []Type) Type {
	if builtinResultReturns[name] {
		// file_read/http_get/b64_decode's Ok payload is always a string
		// in practice (file contents, response body, decoded bytes as
		// text) - reporting it as Primitive("any") made the extremely
		// common "unwrap and use the body as a string" pattern (e.g.
		// `result.val + " suffix"`, or passing it to a `-> str`
		// function) fail type-checking with "expected str, got any".
		// jwt_decode's Ok payload is a genuine claims map, so it's left
		// at Primitive("any") since there's no concrete Type for it here.
		okType := Type(Primitive("any"))
		switch name {
		case "file_read", "http_get", "b64_decode":
			okType = Primitive("str")
		}
		return Result{Ok: okType, Err: Primitive("str")}
	}
	switch name {
	case "len", "to_int", "now":
		return Primitive("int")
	case "sqrt", "pow", "to_float":
		return Primitive("float")
	case "abs":
		if len(argTypes) == 1 {
			if Equal(argTypes[0], Primitive("int")) || Equal(argTypes[0], Primitive("float")) {
				return argTypes[0]
			}
		}
	case "to_str", "type_of", "str_upper", "str_lower", "regex_replace",
		"env_get", "time_format", "md5", "sha256", "b64_encode":
		return Primitive("str")
	case "jwt_encode":
		// jwt_encode only fails (returning a Result instead) if HMAC
		// signing itself errors, which practically never happens for
		// HS256 with a valid secret - typing it as a plain str (its
		// actual success shape, like md5/sha256/b64_encode above) is far
		// more useful than Primitive("any"), which made it impossible to
		// use jwt_encode's result in a `-> str`-returning function or any
		// other typed string context. See builtinResultReturns' comment
		// for why it's *not* listed as a Result-returning builtin.
		return Primitive("str")
	case "str_contains", "regex_match", "file_exists":
		return Primitive("bool")
	case "str_split":
		// Without this, `str_split(s, sep)[i]` and `for part in
		// str_split(...)` both failed to type-check: checkIndexExpr and
		// the for-loop's iterable check both require a concrete List
		// type and reject Primitive("any"), even though the VM (see
		// internal/vm/builtins.go) always returns a list of strings.
		return List{Elem: Primitive("str")}
	case "append":
		// append(lst, item) returns the same list type it was given, so
		// e.g. `let xs: list[int] = []` then `xs = append(xs, 1)` keeps
		// its list[int] type instead of collapsing to Primitive("any").
		if len(argTypes) == 2 {
			if lt, ok := argTypes[0].(List); ok {
				return lt
			}
		}
	}
	return Primitive("any")
}

func checkCallExpr(n *ast.CallExpr, env *Env) (Type, error) {
	varName, ok := n.Func.(*ast.VariableExpr)
	if !ok {
		return nil, New("E2070", "only direct function calls supported in M2-A", n.NodePos)
	}
	// ok/err are polymorphic builtin Result constructors (M2-C).
	// `ok(x)` returns Result[T, str]; `err(x)` returns Result[str, T].
	// The Ok/Err types are inferred from the argument.
	if varName.Name == "ok" || varName.Name == "err" {
		if len(n.Args) != 1 {
			return nil, New("E2020",
				fmt.Sprintf("%s expects 1 arg, got %d", varName.Name, len(n.Args)),
				n.NodePos)
		}
		argT, err := CheckExpr(n.Args[0], env)
		if err != nil {
			return nil, err
		}
		if varName.Name == "ok" {
			return Result{Ok: argT, Err: Primitive("str")}, nil
		}
		return Result{Ok: Primitive("str"), Err: argT}, nil
	}
	if builtinTypeNames[varName.Name] {
		// Builtin call arguments used to go completely unchecked (this
		// branch returned before ever looking at n.Args), so something
		// like sqrt(undefined_var) type-checked fine and only surfaced as
		// a confusing "vm: unsupported op LOAD_GLOBAL" at runtime instead
		// of a proper E2001 here.
		argTypes := make([]Type, len(n.Args))
		for i, arg := range n.Args {
			t, err := CheckExpr(arg, env)
			if err != nil {
				return nil, err
			}
			argTypes[i] = t
		}
		return builtinReturnType(varName.Name, argTypes), nil
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
	switch t := objT.(type) {
	case List:
		if !Equal(idxT, Primitive("int")) {
			return nil, NewMismatch(n.NodePos, Primitive("int"), idxT)
		}
		return t.Elem, nil
	case Map:
		if !Equal(idxT, t.Key) {
			return nil, NewMismatch(n.NodePos, t.Key, idxT)
		}
		return t.Value, nil
	}
	return nil, New("E2050", fmt.Sprintf("cannot index into %s", objT), n.NodePos)
}

func checkFieldExpr(n *ast.FieldExpr, env *Env) (Type, error) {
	objT, err := CheckExpr(n.Object, env)
	if err != nil {
		return nil, err
	}
	if s, ok := objT.(Struct); ok {
		f, ok := s.Field(n.Field)
		if !ok {
			return nil, New("E2052", fmt.Sprintf("struct %s has no field %q", s.Name, n.Field), n.NodePos)
		}
		return f, nil
	}
	if r, ok := objT.(Result); ok {
		switch n.Field {
		case "tag":
			return Primitive("str"), nil
		case "val":
			return r.Ok, nil
		}
		return nil, New("E2052", fmt.Sprintf("Result has no field %q", n.Field), n.NodePos)
	}
	// Bare `Result` placeholder: also accept .tag/.val.
	if p, ok := objT.(Primitive); ok && string(p) == "Result" {
		switch n.Field {
		case "tag":
			return Primitive("str"), nil
		case "val":
			return Primitive("any"), nil
		}
		return nil, New("E2052", fmt.Sprintf("Result has no field %q", n.Field), n.NodePos)
	}
	return nil, New("E2051", fmt.Sprintf("field access requires struct or Result, got %s", objT), n.NodePos)
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

// checkMapLiteral infers a uniform Key/Value type from the first key/value
// pair and checks every other pair against it, mirroring checkListLiteral.
func checkMapLiteral(n *ast.MapLiteralExpr, env *Env) (Type, error) {
	if len(n.Keys) == 0 {
		return nil, New("E2011", "cannot infer type of empty map; add type annotation", n.NodePos)
	}
	keyT, err := CheckExpr(n.Keys[0], env)
	if err != nil {
		return nil, err
	}
	valT, err := CheckExpr(n.Values[0], env)
	if err != nil {
		return nil, err
	}
	for i := 1; i < len(n.Keys); i++ {
		kt, err := CheckExpr(n.Keys[i], env)
		if err != nil {
			return nil, err
		}
		if !Equal(kt, keyT) {
			return nil, NewMismatch(n.Keys[i].Pos(), keyT, kt)
		}
		vt, err := CheckExpr(n.Values[i], env)
		if err != nil {
			return nil, err
		}
		if !Equal(vt, valT) {
			return nil, NewMismatch(n.Values[i].Pos(), valT, vt)
		}
	}
	return Map{Key: keyT, Value: valT}, nil
}

func checkStructLiteral(n *ast.StructLiteralExpr, env *Env) (Type, error) {
	s, ok := env.LookupStruct(n.TypeName)
	if !ok {
		return nil, New("E2053", fmt.Sprintf("undefined struct type: %s", n.TypeName), n.NodePos)
	}
	for fname, expr := range n.Fields {
		expected, ok := s.Field(fname)
		if !ok {
			return nil, New("E2054", fmt.Sprintf("struct %s has no field %q", n.TypeName, fname), n.NodePos)
		}
		actual, err := CheckExpr(expr, env)
		if err != nil {
			return nil, err
		}
		if !Equal(actual, expected) {
			return nil, NewMismatch(expr.Pos(), expected, actual)
		}
	}
	return s, nil
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
	case *ast.MatchStmt:
		return checkMatch(n, env)
	case *ast.ReturnStmt:
		return checkReturn(n, env)
	case *ast.FnDecl:
		return checkFnDecl(n, env)
	case *ast.StructDecl:
		return checkStructDecl(n, env)
	case *ast.BreakStmt:
		return checkBreak(n, env)
	case *ast.ContinueStmt:
		return checkContinue(n, env)
	case *ast.ExprStmt:
		return checkExprStmt(n, env)
	case *ast.PlanBlock:
		return checkPlanBlock(n, env)
	case *ast.ImportDecl, *ast.CommentStmt:
		return nil
	case *ast.MetaBlock:
		return checkMeta(n, env)
	}
	return New("E2099", fmt.Sprintf("unsupported statement %T", s), s.Pos())
}

func checkExprStmt(n *ast.ExprStmt, env *Env) error {
	_, err := CheckExpr(n.X, env)
	return err
}

func checkLet(n *ast.LetStmt, env *Env) error {
	// `let xs: list[int] = []` (or `map[...]{}`) is the only way to seed an
	// accumulator that starts empty - e.g. collecting valid entries while
	// looping over parsed input with append(). checkListLiteral/
	// checkMapLiteral raise E2011 "cannot infer type of empty list/map" for
	// *any* empty literal, since they have no element to infer Elem/Value
	// from; but here a declared annotation already says what the container
	// should hold, so the empty literal can be trusted instead of rejected.
	if n.TypeAnn != "" && isEmptyContainerLiteral(n.Value) {
		declared, err := ParseType(n.TypeAnn)
		if err != nil {
			return New("E2012", fmt.Sprintf("invalid type annotation %q: %v", n.TypeAnn, err), n.NodePos)
		}
		declared = resolveNamedType(declared, env)
		switch declared.(type) {
		case List, Map:
			env.DeclareVar(n.Name, declared)
			return nil
		}
	}
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
		declared = resolveNamedType(declared, env)
		if !Equal(valT, declared) {
			return NewMismatch(n.NodePos, declared, valT)
		}
	} else {
		declared = valT
	}
	env.DeclareVar(n.Name, declared)
	return nil
}

// isEmptyContainerLiteral reports whether n is an empty list/map literal
// (`[]` or `{}`), which checkListLiteral/checkMapLiteral can't type-check
// on their own since they have no element to infer from.
func isEmptyContainerLiteral(n ast.Expression) bool {
	switch v := n.(type) {
	case *ast.ListExpr:
		return len(v.Elements) == 0
	case *ast.MapLiteralExpr:
		return len(v.Keys) == 0
	}
	return false
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
	return Check(n.Body.ToProgram(), bodyEnv.WithLoopBody())
}

func checkWhile(n *ast.WhileStmt, env *Env) error {
	condT, err := CheckExpr(n.Cond, env)
	if err != nil {
		return err
	}
	if !Equal(condT, Primitive("bool")) {
		return NewMismatch(n.NodePos, Primitive("bool"), condT)
	}
	return Check(n.Body.ToProgram(), env.WithLoopBody())
}

func checkMatch(n *ast.MatchStmt, env *Env) error {
	scrT, err := CheckExpr(n.Expr, env)
	if err != nil {
		return err
	}
	for _, arm := range n.Arms {
		if err := checkMatchPattern(arm.Pattern, scrT, env, n.NodePos); err != nil {
			return err
		}
		if err := Check(arm.Body.ToProgram(), env); err != nil {
			return err
		}
	}
	return nil
}

func checkMatchPattern(pattern ast.Expression, scrT Type, env *Env, pos ast.Pos) error {
	if v, ok := pattern.(*ast.VariableExpr); ok && v.Name == "_" {
		return nil
	}
	patT, err := CheckExpr(pattern, env)
	if err != nil {
		return err
	}
	if !Equal(patT, scrT) {
		return NewMismatch(pos, scrT, patT)
	}
	return nil
}

func checkBreak(n *ast.BreakStmt, env *Env) error {
	if !env.InLoop() {
		return New("E2012", "break outside for/while", n.NodePos)
	}
	return nil
}

func checkContinue(n *ast.ContinueStmt, env *Env) error {
	if !env.InLoop() {
		return New("E2013", "continue outside for/while", n.NodePos)
	}
	return nil
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
		retType = resolveNamedType(retType, env)
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
		paramTypes = append(paramTypes, resolveNamedType(pt, env))
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
		fields[f.Name] = resolveNamedType(ft, env)
	}
	env.DeclareStruct(n.Name, Struct{Name: n.Name, Fields: fields})
	return nil
}

// resolveNamedType rewrites bare type names that refer to a known struct
// into their full Struct type (with fields populated), recursing into
// compound types (list/map/optional/Result/func). ParseType has no access
// to the environment, so a struct type annotation like `Point` initially
// comes back as an opaque Primitive("Point"); left as-is, it would never
// compare equal to the real Struct{Name: "Point", ...} type produced by
// struct literals/lookups, causing every struct-typed annotation to fail
// type-checking with a spurious mismatch.
// ResolveNamedType is the exported form of resolveNamedType, for tooling
// (e.g. the LSP server) that needs to resolve a bare type-annotation string
// (from ParseType) against an environment the same way the type checker
// does internally.
func ResolveNamedType(t Type, env *Env) Type {
	return resolveNamedType(t, env)
}

func resolveNamedType(t Type, env *Env) Type {
	switch tt := t.(type) {
	case Primitive:
		if s, ok := env.LookupStruct(string(tt)); ok {
			return s
		}
		return tt
	case List:
		return List{Elem: resolveNamedType(tt.Elem, env)}
	case Map:
		return Map{Key: resolveNamedType(tt.Key, env), Value: resolveNamedType(tt.Value, env)}
	case Optional:
		return Optional{Inner: resolveNamedType(tt.Inner, env)}
	case Result:
		return Result{Ok: resolveNamedType(tt.Ok, env), Err: resolveNamedType(tt.Err, env)}
	case Func:
		params := make([]Type, len(tt.Params))
		for i, p := range tt.Params {
			params[i] = resolveNamedType(p, env)
		}
		return Func{Params: params, Return: resolveNamedType(tt.Return, env)}
	default:
		return t
	}
}

func checkMeta(n *ast.MetaBlock, env *Env) error {
	// Spec requires "name" and "version" fields to be non-empty strings.
	for _, required := range []string{"name", "version"} {
		v, ok := n.Fields[required]
		if !ok || v == "" {
			return New("E2014", fmt.Sprintf("meta.%s must be a non-empty string", required), n.NodePos)
		}
	}
	return nil
}
