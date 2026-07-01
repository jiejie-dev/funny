// v2/internal/compiler/data.go
package compiler

import (
	"fmt"

	"github.com/jiejie-dev/funny/internal/ast"
	"github.com/jiejie-dev/funny/internal/bytecode"
)

// compileList compiles a list literal into BUILD_LIST n.
// Returns the uniform element value type if all elements agree, otherwise valNil.
func (c *Compiler) compileList(n *ast.ListExpr) (valueType, error) {
	var elemType valueType = valNil
	for i, e := range n.Elements {
		vt, err := c.compileExpr(e)
		if err != nil {
			return "", err
		}
		if i == 0 {
			elemType = vt
		} else if vt != elemType {
			elemType = valNil
		}
	}
	c.fn.Emit(bytecode.BUILD_LIST, len(n.Elements))
	return elemType, nil
}

// compileIndex compiles a[b] (object on stack, then index, then INDEX).
func (c *Compiler) compileIndex(n *ast.IndexExpr) (valueType, error) {
	if _, err := c.compileExpr(n.Object); err != nil {
		return "", err
	}
	if _, err := c.compileExpr(n.Index); err != nil {
		return "", err
	}
	c.fn.Emit(bytecode.INDEX, 0)
	return valNil, nil
}

// compileField compiles a.b (push object, push field name as string, GET_FIELD).
// Without per-field static types we conservatively report valStr; this keeps
// downstream comparisons like `r.tag == "err"` well-typed while letting
// println accept the value (since println is untyped at compile time).
func (c *Compiler) compileField(n *ast.FieldExpr) (valueType, error) {
	if _, err := c.compileExpr(n.Object); err != nil {
		return "", err
	}
	nameIdx := c.mod.AddConstant(n.Field)
	c.fn.Emit(bytecode.PUSH_STR, nameIdx)
	c.fn.Emit(bytecode.GET_FIELD, 0)
	return valStr, nil
}

// compileMapLiteral compiles {k: v, ...} into BUILD_MAP n. Empty map literals
// are rejected by the type checker before compilation is reached.
func (c *Compiler) compileMapLiteral(n *ast.MapLiteralExpr) (valueType, error) {
	for i, k := range n.Keys {
		if _, err := c.compileExpr(k); err != nil {
			return "", err
		}
		if _, err := c.compileExpr(n.Values[i]); err != nil {
			return "", err
		}
	}
	c.fn.Emit(bytecode.BUILD_MAP, len(n.Keys))
	return valNil, nil
}

// compileStructLiteral compiles Point(x: 1, y: 2) into BUILD_MAP + NEW_STRUCT.
func (c *Compiler) compileStructLiteral(n *ast.StructLiteralExpr) (valueType, error) {
	if len(n.Fields) == 0 {
		return "", fmt.Errorf("empty struct literal")
	}
	for k, v := range n.Fields {
		nameIdx := c.mod.AddConstant(k)
		c.fn.Emit(bytecode.PUSH_STR, nameIdx)
		if _, err := c.compileExpr(v); err != nil {
			return "", err
		}
	}
	c.fn.Emit(bytecode.BUILD_MAP, len(n.Fields))
	typeIdx := c.mod.AddConstant(n.TypeName)
	c.fn.Emit(bytecode.NEW_STRUCT, typeIdx)
	return valNil, nil
}