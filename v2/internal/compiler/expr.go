// v2/internal/compiler/expr.go
package compiler

import (
	"fmt"

	"github.com/jerloo/funny/v2/internal/ast"
	"github.com/jerloo/funny/v2/internal/bytecode"
)

func (c *Compiler) compileExpr(e ast.Expression) (valueType, error) {
	switch n := e.(type) {
	case *ast.LiteralExpr:
		return c.compileLiteral(n)
	case *ast.VariableExpr:
		return c.compileVariable(n)
	case *ast.BinaryExpr:
		return c.compileBinary(n)
	case *ast.UnaryExpr:
		return c.compileUnary(n)
	case *ast.CallExpr:
		return c.compileCall(n)
	}
	return "", fmt.Errorf("compileExpr: unsupported expression type %T", e)
}

func (c *Compiler) compileLiteral(n *ast.LiteralExpr) (valueType, error) {
	switch v := n.Value.(type) {
	case nil:
		c.fn.Emit(bytecode.PUSH_NIL, 0)
		return valNil, nil
	case int:
		idx := c.mod.AddConstant(v)
		c.fn.Emit(bytecode.PUSH_INT, idx)
		return valInt, nil
	case float64:
		idx := c.mod.AddConstant(v)
		c.fn.Emit(bytecode.PUSH_FLOAT, idx)
		return valFloat, nil
	case string:
		idx := c.mod.AddConstant(v)
		c.fn.Emit(bytecode.PUSH_STR, idx)
		return valStr, nil
	case bool:
		idx := c.mod.AddConstant(v)
		c.fn.Emit(bytecode.PUSH_BOOL, idx)
		return valBool, nil
	}
	return "", fmt.Errorf("compileLiteral: unsupported literal type %T", n.Value)
}

func (c *Compiler) compileVariable(n *ast.VariableExpr) (valueType, error) {
	if slot, vt := c.lookupLocal(n.Name); slot >= 0 {
		c.fn.Emit(bytecode.LOAD_LOCAL, slot)
		return vt, nil
	}
	idx := c.mod.AddConstant(n.Name)
	c.fn.Emit(bytecode.LOAD_GLOBAL, idx)
	// Globals have no recorded type yet (M2-B.5 follow-up).
	return valNil, nil
}

func (c *Compiler) compileBinary(n *ast.BinaryExpr) (valueType, error) {
	leftOp, err := c.compileExpr(n.Left)
	if err != nil {
		return "", err
	}
	rightOp, err := c.compileExpr(n.Right)
	if err != nil {
		return "", err
	}
	if leftOp != rightOp {
		return "", fmt.Errorf("compileBinary: type mismatch %s vs %s for op %s", leftOp, rightOp, n.Op)
	}
	op, err := pickBinaryOp(n.Op, leftOp)
	if err != nil {
		return "", err
	}
	c.fn.Emit(op, 0)

	// Comparison / equality ops produce bool; arithmetic preserves operand type.
	switch n.Op {
	case "+", "-", "*", "/", "%":
		return leftOp, nil
	case "==", "<", ">", "<=", ">=":
		return valBool, nil
	}
	return "", fmt.Errorf("compileBinary: unknown operator %s", n.Op)
}

func pickBinaryOp(op string, lhs valueType) (bytecode.OpCode, error) {
	switch op {
	case "+":
		switch lhs {
		case valInt:
			return bytecode.ADD_INT, nil
		case valFloat:
			return bytecode.ADD_FLOAT, nil
		case valStr:
			return bytecode.ADD_STR, nil
		}
	case "-":
		switch lhs {
		case valInt:
			return bytecode.SUB_INT, nil
		case valFloat:
			return bytecode.SUB_FLOAT, nil
		}
	case "*":
		switch lhs {
		case valInt:
			return bytecode.MUL_INT, nil
		case valFloat:
			return bytecode.MUL_FLOAT, nil
		}
	case "/":
		switch lhs {
		case valInt:
			return bytecode.DIV_INT, nil
		case valFloat:
			return bytecode.DIV_FLOAT, nil
		}
	case "%":
		if lhs == valInt {
			return bytecode.MOD_INT, nil
		}
	case "==":
		switch lhs {
		case valInt:
			return bytecode.EQ_INT, nil
		case valStr:
			return bytecode.EQ_STR, nil
		case valBool:
			return bytecode.EQ_BOOL, nil
		case valNil:
			return bytecode.EQ_NIL, nil
		}
	case "<":
		if lhs == valInt {
			return bytecode.LT_INT, nil
		}
	case ">":
		if lhs == valInt {
			return bytecode.GT_INT, nil
		}
	case "<=":
		if lhs == valInt {
			return bytecode.LTE_INT, nil
		}
	case ">=":
		if lhs == valInt {
			return bytecode.GTE_INT, nil
		}
	}
	return "", fmt.Errorf("pickBinaryOp: unsupported op %s for %s", op, lhs)
}

func (c *Compiler) compileUnary(n *ast.UnaryExpr) (valueType, error) {
	op, err := c.compileExpr(n.Expr)
	if err != nil {
		return "", err
	}
	switch n.Op {
	case "-":
		switch op {
		case valInt:
			c.fn.Emit(bytecode.NEG_INT, 0)
			return valInt, nil
		case valFloat:
			c.fn.Emit(bytecode.NEG_FLOAT, 0)
			return valFloat, nil
		}
	case "not":
		if op == valBool {
			c.fn.Emit(bytecode.NOT_BOOL, 0)
			return valBool, nil
		}
	}
	return "", fmt.Errorf("compileUnary: unsupported op %s for %s", n.Op, op)
}