// v2/internal/compiler/expr.go
package compiler

import (
	"fmt"

	"github.com/jiejie-dev/funny/v2/internal/ast"
	"github.com/jiejie-dev/funny/v2/internal/bytecode"
)

func (c *Compiler) compileExpr(e ast.Expression) (valueType, error) {
	c.pos = e.Pos()
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
	case *ast.ListExpr:
		return c.compileList(n)
	case *ast.MapLiteralExpr:
		return c.compileMapLiteral(n)
	case *ast.IndexExpr:
		return c.compileIndex(n)
	case *ast.FieldExpr:
		return c.compileField(n)
	case *ast.StructLiteralExpr:
		return c.compileStructLiteral(n)
	case *ast.TryExpr:
		return c.compileTry(n)
	case *ast.FStringExpr:
		return c.compileFString(n)
	}
	return "", fmt.Errorf("compileExpr: unsupported expression type %T", e)
}

// compileFString compiles an f-string into a sequence of PUSH_STR /
// <expr>+FORMAT_VALUE pieces, folded together with ADD_STR (reusing the
// existing string-concat opcode) — no new n-ary opcode is needed.
func (c *Compiler) compileFString(n *ast.FStringExpr) (valueType, error) {
	if len(n.Parts) == 0 {
		idx := c.mod.AddConstant("")
		c.emit(bytecode.PUSH_STR, idx)
		return valStr, nil
	}
	for i, part := range n.Parts {
		if part.Expr == nil {
			idx := c.mod.AddConstant(part.Text)
			c.emit(bytecode.PUSH_STR, idx)
		} else {
			if _, err := c.compileExpr(part.Expr); err != nil {
				return "", err
			}
			specIdx := c.mod.AddConstant(part.Spec)
			c.emit(bytecode.FORMAT_VALUE, specIdx)
		}
		if i > 0 {
			c.emit(bytecode.ADD_STR, 0)
		}
	}
	return valStr, nil
}

// compileTry compiles `expr?`. Emits the inner expression's code and, if
// the result is a Result, follows it with TRY_OR_RETURN to propagate Err
// or unwrap Ok. If the inner expression's type is not a Result, the `?` is
// a no-op (we still emit TRY_OR_RETURN but the runtime check is a no-op
// for non-Results).
func (c *Compiler) compileTry(n *ast.TryExpr) (valueType, error) {
	vt, err := c.compileExpr(n.Inner)
	if err != nil {
		return "", err
	}
	c.emit(bytecode.TRY_OR_RETURN, 0)
	if vt == valStr {
		return valStr, nil
	}
	return vt, nil
}

func (c *Compiler) compileLiteral(n *ast.LiteralExpr) (valueType, error) {
	switch v := n.Value.(type) {
	case nil:
		c.emit(bytecode.PUSH_NIL, 0)
		return valNil, nil
	case int:
		idx := c.mod.AddConstant(v)
		c.emit(bytecode.PUSH_INT, idx)
		return valInt, nil
	case float64:
		idx := c.mod.AddConstant(v)
		c.emit(bytecode.PUSH_FLOAT, idx)
		return valFloat, nil
	case string:
		idx := c.mod.AddConstant(v)
		c.emit(bytecode.PUSH_STR, idx)
		return valStr, nil
	case bool:
		idx := c.mod.AddConstant(v)
		c.emit(bytecode.PUSH_BOOL, idx)
		return valBool, nil
	}
	return "", fmt.Errorf("compileLiteral: unsupported literal type %T", n.Value)
}

func (c *Compiler) compileVariable(n *ast.VariableExpr) (valueType, error) {
	if slot, vt := c.lookupLocal(n.Name); slot >= 0 {
		c.emit(bytecode.LOAD_LOCAL, slot)
		return vt, nil
	}
	idx := c.mod.AddConstant(n.Name)
	c.emit(bytecode.LOAD_GLOBAL, idx)
	// Globals have no recorded type yet (M2-B.5 follow-up).
	return valNil, nil
}

func (c *Compiler) compileBinary(n *ast.BinaryExpr) (valueType, error) {
	if n.Op == "in" {
		if _, err := c.compileExpr(n.Left); err != nil {
			return "", err
		}
		if _, err := c.compileExpr(n.Right); err != nil {
			return "", err
		}
		c.emit(bytecode.IN_LIST, 0)
		return valBool, nil
	}
	leftOp, err := c.compileExpr(n.Left)
	if err != nil {
		return "", err
	}
	rightOp, err := c.compileExpr(n.Right)
	if err != nil {
		return "", err
	}
	// valNil is this compiler's catch-all for "statically untracked"
	// (e.g. a `.val` field pulled off a Result, or anything else
	// compileField/compileIndex/builtinValueType couldn't resolve to a
	// concrete type) - it does *not* mean the runtime value is
	// guaranteed to be the nil literal. Before this, combining a
	// concretely-typed operand with an untracked one - e.g.
	// `"status: " + result.val` where result.val came from an http_get
	// Result typed str at the type-checker level but untracked here -
	// always hard-failed to compile with a "type mismatch X vs nil"
	// error, even though the actual runtime values are perfectly
	// compatible and the (separate, authoritative) type checker already
	// accepted the expression. Only a mismatch between two *concretely*
	// tracked types (e.g. int vs str) is still rejected here.
	opType := leftOp
	if leftOp != rightOp {
		switch {
		case leftOp == valNil && rightOp != valNil:
			opType = rightOp
		case rightOp == valNil && leftOp != valNil:
			opType = leftOp
		default:
			return "", fmt.Errorf("compileBinary: type mismatch %s vs %s for op %s", leftOp, rightOp, n.Op)
		}
	}
	// `!=` has no dedicated opcode family of its own; it reuses whichever
	// EQ_* opcode `==` would pick for this operand type and negates it.
	if n.Op == "!=" {
		op, err := pickBinaryOp("==", opType)
		if err != nil {
			return "", fmt.Errorf("compileBinary: unsupported op != for %s", opType)
		}
		c.emit(op, 0)
		c.emit(bytecode.NOT_BOOL, 0)
		return valBool, nil
	}
	op, err := pickBinaryOp(n.Op, opType)
	if err != nil {
		return "", err
	}
	c.emit(op, 0)

	// Comparison / equality / logical ops produce bool; arithmetic preserves
	// operand type.
	switch n.Op {
	case "+", "-", "*", "/", "%":
		return opType, nil
	case "==", "<", ">", "<=", ">=", "and", "or":
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
		case valFloat:
			return bytecode.EQ_FLOAT, nil
		}
	case "<":
		switch lhs {
		case valInt:
			return bytecode.LT_INT, nil
		case valFloat:
			return bytecode.LT_FLOAT, nil
		}
	case ">":
		switch lhs {
		case valInt:
			return bytecode.GT_INT, nil
		case valFloat:
			return bytecode.GT_FLOAT, nil
		}
	case "<=":
		switch lhs {
		case valInt:
			return bytecode.LTE_INT, nil
		case valFloat:
			return bytecode.LTE_FLOAT, nil
		}
	case ">=":
		switch lhs {
		case valInt:
			return bytecode.GTE_INT, nil
		case valFloat:
			return bytecode.GTE_FLOAT, nil
		}
	case "and":
		if lhs == valBool {
			return bytecode.AND_BOOL, nil
		}
	case "or":
		if lhs == valBool {
			return bytecode.OR_BOOL, nil
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
			c.emit(bytecode.NEG_INT, 0)
			return valInt, nil
		case valFloat:
			c.emit(bytecode.NEG_FLOAT, 0)
			return valFloat, nil
		}
	case "not":
		if op == valBool {
			c.emit(bytecode.NOT_BOOL, 0)
			return valBool, nil
		}
	}
	return "", fmt.Errorf("compileUnary: unsupported op %s for %s", n.Op, op)
}
