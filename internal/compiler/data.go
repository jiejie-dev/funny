// v2/internal/compiler/data.go
package compiler

import (
	"fmt"

	"github.com/jiejie-dev/funny/v2/internal/ast"
	"github.com/jiejie-dev/funny/v2/internal/bytecode"
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
//
// Returns the same valueType as the indexed object itself. That looks
// backwards at first, but matches the convention set by
// compileList/annotationValueType: since this compiler has no distinct
// "list" valueType, a list-valued expression's tracked valueType is
// already its *element* type - so indexing into it naturally reproduces
// that same valueType. Before this returned a hardcoded valNil, so e.g.
// `entries[0].response_ms` or `xs[i] + 1` lost all type information the
// moment an index was taken, even though the compiler knew the element
// type going in. (String indexing also produces a same-type result: one
// character back as a string - see internal/vm/instructions.go's
// execIndex. Map values aren't tracked by this compiler at all, so
// map-typed objects report valNil either way, same as before.)
func (c *Compiler) compileIndex(n *ast.IndexExpr) (valueType, error) {
	objType, err := c.compileExpr(n.Object)
	if err != nil {
		return "", err
	}
	if _, err := c.compileExpr(n.Index); err != nil {
		return "", err
	}
	c.fn.Emit(bytecode.INDEX, 0)
	return objType, nil
}

// compileField compiles a.b (push object, push field name as string, GET_FIELD).
// If the object's value type is a recognized struct name (see
// annotationValueType/compileStructLiteral), looks up the field's real
// declared type from c.structFields so it participates correctly in typed
// arithmetic/comparisons (`item.price + tax`, `entry.count > 0`, ...) -
// this used to unconditionally report valStr for *every* field access
// regardless of the field's actual type, so any non-string struct field
// used in a typed operator failed with a confusing "unsupported op * for
// str" (or silently picked the wrong opcode for the rare cases where a
// mismatch happened not to be caught). For object types we have no
// schema for (a Result from ok()/err(), an untracked global, ...), `.tag`
// is still always a string by construction (kept as a special case so
// `r.tag == "err"` stays well-typed); anything else conservatively falls
// back to valNil ("unknown") rather than guessing.
func (c *Compiler) compileField(n *ast.FieldExpr) (valueType, error) {
	objType, err := c.compileExpr(n.Object)
	if err != nil {
		return "", err
	}
	nameIdx := c.mod.AddConstant(n.Field)
	c.fn.Emit(bytecode.PUSH_STR, nameIdx)
	c.fn.Emit(bytecode.GET_FIELD, 0)
	if fields, ok := c.structFields[string(objType)]; ok {
		if ft, ok := fields[n.Field]; ok {
			return ft, nil
		}
	}
	if n.Field == "tag" {
		return valStr, nil
	}
	return valNil, nil
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
// Returns the struct's own name as its valueType (see annotationValueType),
// so a `let p = Point(...)` local (or a struct-typed function
// param/return) carries enough static type info for compileField to look
// up its real field types later.
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
	return valueType(n.TypeName), nil
}