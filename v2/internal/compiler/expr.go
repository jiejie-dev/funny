// v2/internal/compiler/expr.go
package compiler

import (
	"fmt"

	"github.com/jerloo/funny/v2/internal/ast"
	"github.com/jerloo/funny/v2/internal/bytecode"
)

func (c *Compiler) compileExpr(e ast.Expression) (bytecode.OpCode, error) {
	switch n := e.(type) {
	case *ast.LiteralExpr:
		return c.compileLiteral(n)
	case *ast.VariableExpr:
		return c.compileVariable(n)
	case *ast.BinaryExpr:
		return c.compileBinary(n)
	case *ast.UnaryExpr:
		return c.compileUnary(n)
	}
	return "", fmt.Errorf("compileExpr: unsupported expression type %T", e)
}

func (c *Compiler) compileLiteral(n *ast.LiteralExpr) (bytecode.OpCode, error) {
	switch v := n.Value.(type) {
	case nil:
		c.fn.Emit(bytecode.PUSH_NIL, 0)
		return bytecode.PUSH_NIL, nil
	case int:
		idx := c.mod.AddConstant(v)
		c.fn.Emit(bytecode.PUSH_INT, idx)
		return bytecode.PUSH_INT, nil
	case float64:
		idx := c.mod.AddConstant(v)
		c.fn.Emit(bytecode.PUSH_FLOAT, idx)
		return bytecode.PUSH_FLOAT, nil
	case string:
		idx := c.mod.AddConstant(v)
		c.fn.Emit(bytecode.PUSH_STR, idx)
		return bytecode.PUSH_STR, nil
	case bool:
		idx := c.mod.AddConstant(v)
		c.fn.Emit(bytecode.PUSH_BOOL, idx)
		return bytecode.PUSH_BOOL, nil
	}
	return "", fmt.Errorf("compileLiteral: unsupported literal type %T", n.Value)
}

func (c *Compiler) compileVariable(n *ast.VariableExpr) (bytecode.OpCode, error) {
	if idx := c.lookupLocal(n.Name); idx >= 0 {
		c.fn.Emit(bytecode.LOAD_LOCAL, idx)
		return bytecode.LOAD_LOCAL, nil
	}
	idx := c.mod.AddConstant(n.Name)
	c.fn.Emit(bytecode.LOAD_GLOBAL, idx)
	return bytecode.LOAD_GLOBAL, nil
}

func (c *Compiler) compileBinary(n *ast.BinaryExpr) (bytecode.OpCode, error) {
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
	return op, nil
}

func pickBinaryOp(op string, lhs bytecode.OpCode) (bytecode.OpCode, error) {
	switch op {
	case "+":
		switch lhs {
		case bytecode.PUSH_INT:
			return bytecode.ADD_INT, nil
		case bytecode.PUSH_FLOAT:
			return bytecode.ADD_FLOAT, nil
		case bytecode.PUSH_STR:
			return bytecode.ADD_STR, nil
		}
	case "-":
		switch lhs {
		case bytecode.PUSH_INT:
			return bytecode.SUB_INT, nil
		case bytecode.PUSH_FLOAT:
			return bytecode.SUB_FLOAT, nil
		}
	case "*":
		switch lhs {
		case bytecode.PUSH_INT:
			return bytecode.MUL_INT, nil
		case bytecode.PUSH_FLOAT:
			return bytecode.MUL_FLOAT, nil
		}
	case "/":
		switch lhs {
		case bytecode.PUSH_INT:
			return bytecode.DIV_INT, nil
		case bytecode.PUSH_FLOAT:
			return bytecode.DIV_FLOAT, nil
		}
	case "%":
		if lhs == bytecode.PUSH_INT {
			return bytecode.MOD_INT, nil
		}
	case "==":
		switch lhs {
		case bytecode.PUSH_INT:
			return bytecode.EQ_INT, nil
		case bytecode.PUSH_STR:
			return bytecode.EQ_STR, nil
		case bytecode.PUSH_BOOL:
			return bytecode.EQ_BOOL, nil
		case bytecode.PUSH_NIL:
			return bytecode.EQ_NIL, nil
		}
	case "<":
		if lhs == bytecode.PUSH_INT {
			return bytecode.LT_INT, nil
		}
	case ">":
		if lhs == bytecode.PUSH_INT {
			return bytecode.GT_INT, nil
		}
	case "<=":
		if lhs == bytecode.PUSH_INT {
			return bytecode.LTE_INT, nil
		}
	case ">=":
		if lhs == bytecode.PUSH_INT {
			return bytecode.GTE_INT, nil
		}
	}
	return "", fmt.Errorf("pickBinaryOp: unsupported op %s for %s", op, lhs)
}

func (c *Compiler) compileUnary(n *ast.UnaryExpr) (bytecode.OpCode, error) {
	op, err := c.compileExpr(n.Expr)
	if err != nil {
		return "", err
	}
	switch n.Op {
	case "-":
		switch op {
		case bytecode.PUSH_INT:
			c.fn.Emit(bytecode.NEG_INT, 0)
			return bytecode.PUSH_INT, nil
		case bytecode.PUSH_FLOAT:
			c.fn.Emit(bytecode.NEG_FLOAT, 0)
			return bytecode.PUSH_FLOAT, nil
		}
	case "not":
		if op == bytecode.PUSH_BOOL {
			c.fn.Emit(bytecode.NOT_BOOL, 0)
			return bytecode.PUSH_BOOL, nil
		}
	}
	return "", fmt.Errorf("compileUnary: unsupported op %s for %s", n.Op, op)
}
