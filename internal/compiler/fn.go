// v2/internal/compiler/fn.go
package compiler

import (
	"fmt"

	"github.com/jiejie-dev/funny/internal/ast"
	"github.com/jiejie-dev/funny/internal/bytecode"
)

// builtinNames lists functions that compile to CALL_BUILTIN instead of CALL.
var builtinNames = map[string]bool{
	"print":       true,
	"println":     true,
	"len":         true,
	"to_str":      true,
	"to_int":      true,
	"type_of":     true,
	"ok":          true,
	"err":         true,
	"to_json":     true,
	"parse_json":  true,
	"now":         true,
	"time_format": true,
	"sqrt":        true,
	"pow":         true,
	"abs":         true,
	"str_upper":   true,
	"str_lower":   true,
	"str_contains": true,
	"str_split":   true,
}

// compileFnDecl compiles a function declaration into a separate Function in the module.
func (c *Compiler) compileFnDecl(n *ast.FnDecl) error {
	if _, ok := c.functions[n.Name]; ok {
		return fmt.Errorf("function %s already declared", n.Name)
	}
	fn := &bytecode.Function{Name: n.Name, Arity: len(n.Params)}
	fnIdx := c.mod.AddFunction(fn)
	c.functions[n.Name] = fnIdx
	c.fnRetTypes[n.Name] = paramType(n.RetType)
	c.fn = fn
	c.scopes = []map[string]int{{}}
	for _, p := range n.Params {
		c.declareLocal(p.Name, paramType(p.TypeAnn))
	}
	if err := c.compileBlock(n.Body); err != nil {
		return err
	}
	c.fn.Emit(bytecode.RETURN, 0)
	c.fn = c.mod.Functions[c.functions["main"]]
	c.scopes = []map[string]int{{}}
	return nil
}

// paramType maps a parameter type annotation to a valueType so that
// subsequent variable lookups produce the correct operand type for
// type-sensitive operators like `+`.
func paramType(ann string) valueType {
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
		for _, arg := range n.Args {
			if _, err := c.compileExpr(arg); err != nil {
				return "", err
			}
		}
		nameIdx := c.mod.AddConstant(bytecode.BuiltinInfo{Name: name, Arity: len(n.Args)})
		c.fn.Emit(bytecode.CALL_BUILTIN, nameIdx)
		return valNil, nil
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