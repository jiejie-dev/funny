package lexer

type Kind string

const (
	EOF     Kind = "EOF"
	NEWLINE Kind = "NEWLINE"
	INDENT  Kind = "INDENT"
	DEDENT  Kind = "DEDENT"

	NAME  Kind = "NAME"
	INT   Kind = "INT"
	FLOAT Kind = "FLOAT"
	STR   Kind = "STR"
	FSTR  Kind = "FSTR"

	LPAREN   Kind = "("
	RPAREN   Kind = ")"
	LBRACK   Kind = "["
	RBRACK   Kind = "]"
	LBRACE   Kind = "{"
	RBRACE   Kind = "}"
	COMMA    Kind = ","
	DOT      Kind = "."
	COLON    Kind = ":"
	ARROW    Kind = "->"
	FATARROW Kind = "=>"
	QUESTION Kind = "?"
	AT       Kind = "@"

	PLUS    Kind = "+"
	MINUS   Kind = "-"
	STAR    Kind = "*"
	SLASH   Kind = "/"
	PERCENT Kind = "%"

	EQ   Kind = "="
	EQEQ Kind = "=="
	NEQ  Kind = "!="
	LT   Kind = "<"
	GT   Kind = ">"
	LTE  Kind = "<="
	GTE  Kind = ">="

	COMMENT     Kind = "COMMENT"
	DOC_COMMENT Kind = "DOC_COMMENT"

	AND      Kind = "and"
	AS       Kind = "as"
	BREAK    Kind = "break"
	CONTINUE Kind = "continue"
	ELIF     Kind = "elif"
	ELSE     Kind = "else"
	FALSE    Kind = "false"
	FN       Kind = "fn"
	FOR      Kind = "for"
	IF       Kind = "if"
	IMPORT   Kind = "import"
	IN       Kind = "in"
	LET      Kind = "let"
	MATCH    Kind = "match"
	META     Kind = "meta"
	NIL      Kind = "nil"
	NOT      Kind = "not"
	OR       Kind = "or"
	PLAN     Kind = "plan"
	PUB      Kind = "pub"
	RETURN   Kind = "return"
	STEP     Kind = "step"
	STRUCT   Kind = "struct"
	TRUE     Kind = "true"
	WHILE    Kind = "while"
)

var keywordSet = map[string]Kind{
	"and": AND, "as": AS, "break": BREAK, "continue": CONTINUE,
	"elif": ELIF, "else": ELSE, "false": FALSE, "fn": FN,
	"for": FOR, "if": IF, "import": IMPORT, "in": IN, "let": LET,
	"match": MATCH, "meta": META, "nil": NIL, "not": NOT, "or": OR,
	"plan": PLAN, "pub": PUB, "return": RETURN, "step": STEP,
	"struct": STRUCT, "true": TRUE, "while": WHILE,
}

func (k Kind) IsKeyword() bool {
	_, ok := keywordSet[string(k)]
	return ok
}

func (k Kind) String() string {
	return string(k)
}

type Position struct {
	File   string
	Line   int
	Col    int
	Offset int
}

type Token struct {
	Kind Kind
	Data string
	Pos  Position
}
