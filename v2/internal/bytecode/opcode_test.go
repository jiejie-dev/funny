package bytecode

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOpCode_String(t *testing.T) {
	cases := []struct {
		op   OpCode
		want string
	}{
		{PUSH_INT, "PUSH_INT"},
		{PUSH_FLOAT, "PUSH_FLOAT"},
		{PUSH_STR, "PUSH_STR"},
		{PUSH_BOOL, "PUSH_BOOL"},
		{PUSH_NIL, "PUSH_NIL"},
		{POP, "POP"},
		{DUP, "DUP"},
		{LOAD_LOCAL, "LOAD_LOCAL"},
		{STORE_LOCAL, "STORE_LOCAL"},
		{LOAD_GLOBAL, "LOAD_GLOBAL"},
		{STORE_GLOBAL, "STORE_GLOBAL"},
		{ADD_INT, "ADD_INT"},
		{SUB_INT, "SUB_INT"},
		{MUL_INT, "MUL_INT"},
		{DIV_INT, "DIV_INT"},
		{MOD_INT, "MOD_INT"},
		{NEG_INT, "NEG_INT"},
		{ADD_FLOAT, "ADD_FLOAT"},
		{SUB_FLOAT, "SUB_FLOAT"},
		{MUL_FLOAT, "MUL_FLOAT"},
		{DIV_FLOAT, "DIV_FLOAT"},
		{NEG_FLOAT, "NEG_FLOAT"},
		{ADD_STR, "ADD_STR"},
		{EQ_INT, "EQ_INT"},
		{EQ_STR, "EQ_STR"},
		{EQ_BOOL, "EQ_BOOL"},
		{EQ_NIL, "EQ_NIL"},
		{LT_INT, "LT_INT"},
		{GT_INT, "GT_INT"},
		{LTE_INT, "LTE_INT"},
		{GTE_INT, "GTE_INT"},
		{NOT_BOOL, "NOT_BOOL"},
		{JUMP, "JUMP"},
		{JUMP_IF_FALSE, "JUMP_IF_FALSE"},
		{JUMP_IF_TRUE, "JUMP_IF_TRUE"},
		{CALL, "CALL"},
		{CALL_BUILTIN, "CALL_BUILTIN"},
		{RETURN, "RETURN"},
		{BUILD_LIST, "BUILD_LIST"},
		{INDEX, "INDEX"},
		{BUILD_MAP, "BUILD_MAP"},
		{GET_FIELD, "GET_FIELD"},
		{NEW_STRUCT, "NEW_STRUCT"},
		{HALT, "HALT"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, c.op.String(), "OpCode=%v", c.op)
	}
}
