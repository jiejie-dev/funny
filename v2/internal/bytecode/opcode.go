// v2/internal/bytecode/opcode.go
package bytecode

// OpCode identifies a typed bytecode instruction.
type OpCode string

const (
	// Stack manipulation
	PUSH_INT   OpCode = "PUSH_INT"
	PUSH_FLOAT OpCode = "PUSH_FLOAT"
	PUSH_STR   OpCode = "PUSH_STR"
	PUSH_BOOL  OpCode = "PUSH_BOOL"
	PUSH_NIL   OpCode = "PUSH_NIL"
	POP        OpCode = "POP"
	DUP        OpCode = "DUP"

	// Variables
	LOAD_LOCAL   OpCode = "LOAD_LOCAL"
	STORE_LOCAL  OpCode = "STORE_LOCAL"
	LOAD_GLOBAL  OpCode = "LOAD_GLOBAL"
	STORE_GLOBAL OpCode = "STORE_GLOBAL"

	// Arithmetic (typed)
	ADD_INT   OpCode = "ADD_INT"
	SUB_INT   OpCode = "SUB_INT"
	MUL_INT   OpCode = "MUL_INT"
	DIV_INT   OpCode = "DIV_INT"
	MOD_INT   OpCode = "MOD_INT"
	NEG_INT   OpCode = "NEG_INT"
	ADD_FLOAT OpCode = "ADD_FLOAT"
	SUB_FLOAT OpCode = "SUB_FLOAT"
	MUL_FLOAT OpCode = "MUL_FLOAT"
	DIV_FLOAT OpCode = "DIV_FLOAT"
	NEG_FLOAT OpCode = "NEG_FLOAT"
	ADD_STR   OpCode = "ADD_STR"

	// Comparison (typed)
	EQ_INT  OpCode = "EQ_INT"
	EQ_STR  OpCode = "EQ_STR"
	EQ_BOOL OpCode = "EQ_BOOL"
	EQ_NIL  OpCode = "EQ_NIL"
	LT_INT  OpCode = "LT_INT"
	GT_INT  OpCode = "GT_INT"
	LTE_INT OpCode = "LTE_INT"
	GTE_INT OpCode = "GTE_INT"

	// Logical
	NOT_BOOL OpCode = "NOT_BOOL"

	// Control flow
	JUMP          OpCode = "JUMP"
	JUMP_IF_FALSE OpCode = "JUMP_IF_FALSE"
	JUMP_IF_TRUE  OpCode = "JUMP_IF_TRUE"

	// Functions
	CALL         OpCode = "CALL"
	CALL_BUILTIN OpCode = "CALL_BUILTIN"
	RETURN       OpCode = "RETURN"

	// Data structures
	BUILD_LIST OpCode = "BUILD_LIST"
	INDEX      OpCode = "INDEX"
	BUILD_MAP  OpCode = "BUILD_MAP"
	GET_FIELD  OpCode = "GET_FIELD"
	NEW_STRUCT OpCode = "NEW_STRUCT"

	// Halt
	HALT OpCode = "HALT"
)

// String returns the string representation of the OpCode.
func (op OpCode) String() string { return string(op) }
