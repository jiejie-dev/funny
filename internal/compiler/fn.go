// v2/internal/compiler/fn.go
package compiler

import (
	"fmt"
	"strings"

	"github.com/jiejie-dev/funny/internal/ast"
	"github.com/jiejie-dev/funny/internal/bytecode"
)

// builtinNames lists functions that compile to CALL_BUILTIN instead of CALL.
// Must stay in sync with internal/types.builtinTypeNames (checked first) and
// internal/vm/builtins.go's execCallBuiltin switch (what actually runs) —
// see the comment on builtinTypeNames for how far out of sync this list had
// drifted from the VM before regex/env/file/http/crypto/jwt/sql were added
// here.
var builtinNames = map[string]bool{
	"print":         true,
	"println":       true,
	"len":           true,
	"to_str":        true,
	"to_int":        true,
	"to_float":      true,
	"type_of":       true,
	"ok":            true,
	"err":           true,
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

// compileFnDecl compiles a function declaration into a separate Function in
// the module. Local slot numbering and slot->type tracking (c.scopes,
// c.varTypes) are per-function state, so both must be swapped out for the
// duration of the function body and restored afterward - never just reset
// to empty, or the enclosing scope's own locals (e.g. top-level `let`s
// declared before this `fn`) would become permanently unreachable by name,
// silently falling back to (unimplemented) LOAD_GLOBAL lookups.
func (c *Compiler) compileFnDecl(n *ast.FnDecl) error {
	if _, ok := c.functions[n.Name]; ok {
		return fmt.Errorf("function %s already declared", n.Name)
	}
	fn := &bytecode.Function{Name: n.Name, Arity: len(n.Params)}
	fnIdx := c.mod.AddFunction(fn)
	c.functions[n.Name] = fnIdx
	c.fnRetTypes[n.Name] = annotationValueType(n.RetType, c.structFields)

	outerFn := c.fn
	outerScopes := c.scopes
	outerVarTypes := c.varTypes

	c.fn = fn
	c.scopes = []map[string]int{{}}
	c.varTypes = nil
	for _, p := range n.Params {
		c.declareLocal(p.Name, annotationValueType(p.TypeAnn, c.structFields))
	}
	if err := c.compileBlock(n.Body); err != nil {
		return err
	}
	c.fn.Emit(bytecode.RETURN, 0)

	c.fn = outerFn
	c.scopes = outerScopes
	c.varTypes = outerVarTypes
	return nil
}

// annotationValueType maps a type annotation string to a valueType so that
// subsequent variable lookups produce the correct operand type for
// type-sensitive operators like `+`. Struct names resolve to a valueType
// equal to the struct's own name (see compileField/compileStructLiteral),
// so `p.x` on a `p: Point` parameter or `let p = Point(...)` can look up
// Point's real field types instead of guessing. Anything else structurally
// untracked (list[T], map[K,V], T?, an unrecognized name) falls back to
// valNil ("unknown").
func annotationValueType(ann string, structFields map[string]map[string]valueType) valueType {
	switch ann {
	case "int":
		return valInt
	case "float":
		return valFloat
	case "str":
		return valStr
	case "bool":
		return valBool
	}
	// compileList reports a list value's tracked type as its *element*
	// type (there's no distinct "list" valueType) so a `for` loop over it
	// types its loop variable correctly - without this, a `list[int]`
	// (or `list[Point]`, ...) parameter or return type fell back to
	// valNil, and looping over it produced an untyped loop variable that
	// failed to compile the moment it was used in a typed operator
	// (`if x > 0:`).
	if inner, ok := strings.CutPrefix(ann, "list["); ok {
		if elem, ok := strings.CutSuffix(inner, "]"); ok {
			return annotationValueType(elem, structFields)
		}
	}
	if _, ok := structFields[ann]; ok {
		return valueType(ann)
	}
	return valNil
}

// compileReturn compiles a return statement.
func (c *Compiler) compileReturn(n *ast.ReturnStmt) error {
	if n.Value != nil {
		if _, err := c.compileExpr(n.Value); err != nil {
			return err
		}
	}
	c.fn.Emit(bytecode.RETURN, 0)
	return nil
}

// compileCall compiles a function call expression.
func (c *Compiler) compileCall(n *ast.CallExpr) (valueType, error) {
	varName, ok := n.Func.(*ast.VariableExpr)
	if !ok {
		return "", fmt.Errorf("compileCall: only direct function calls supported (got %T)", n.Func)
	}
	name := varName.Name
	if builtinNames[name] {
		argTypes := make([]valueType, len(n.Args))
		for i, arg := range n.Args {
			vt, err := c.compileExpr(arg)
			if err != nil {
				return "", err
			}
			argTypes[i] = vt
		}
		nameIdx := c.mod.AddConstant(bytecode.BuiltinInfo{Name: name, Arity: len(n.Args)})
		c.fn.Emit(bytecode.CALL_BUILTIN, nameIdx)
		return builtinValueType(name, argTypes), nil
	}
	fnIdx, ok := c.functions[name]
	if !ok {
		return "", fmt.Errorf("undefined function: %s", name)
	}
	for _, arg := range n.Args {
		if _, err := c.compileExpr(arg); err != nil {
			return "", err
		}
	}
	c.fn.Emit(bytecode.CALL, fnIdx)
	return c.fnRetTypes[name], nil
}

// builtinValueType returns the concrete compiler-tracked value type a
// builtin call produces, so `len(x) > 0` / `sqrt(x) < 1.0` / etc. can pick
// a typed comparison opcode the same way a literal or `let`-bound local
// would — before this, every builtin call was opaquely typed valNil (see
// the comment on the old return statement this replaced), so using *any*
// builtin's result directly in an arithmetic or comparison expression
// failed to compile with "type mismatch nil vs X", even though the same
// code ran fine under the tree-walking evaluator (which has no static
// value-type tracking to trip over).
//
// Builtins that return something the compiler doesn't model as a
// valueType at all (list, map, Result, or no value) are left at valNil —
// that's correct/harmless since their results reach further code through
// compileField/compileIndex, not typed arithmetic.
func builtinValueType(name string, argTypes []valueType) valueType {
	switch name {
	case "len", "to_int", "now":
		return valInt
	case "sqrt", "pow", "to_float":
		return valFloat
	case "abs":
		// abs preserves its argument's numeric type (int stays int, float
		// stays float); only claim a type when the argument's was known.
		if len(argTypes) == 1 {
			switch argTypes[0] {
			case valInt, valFloat:
				return argTypes[0]
			}
		}
	case "to_str", "type_of", "str_upper", "str_lower", "regex_replace",
		"env_get", "time_format", "md5", "sha256", "b64_encode", "jwt_encode":
		return valStr
	case "str_contains", "regex_match", "file_exists":
		return valBool
	}
	return valNil
}
