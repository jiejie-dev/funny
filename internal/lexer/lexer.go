package lexer

import "fmt"

type Lexer struct {
	src         string
	file        string
	pos         int
	line        int
	col         int
	savePos     int
	saveLine    int
	saveCol     int
	indentStack []int
	hasEmitted  bool
}

func New(src, file string) *Lexer {
	return &Lexer{src: src, file: file, indentStack: []int{0}}
}

func (l *Lexer) peek(n int) byte {
	p := l.pos + n
	if p >= len(l.src) {
		return 0
	}
	return l.src[p]
}

func (l *Lexer) advance() {
	if l.pos < len(l.src) {
		if l.src[l.pos] == '\n' {
			l.line++
			l.col = 0
		} else {
			l.col++
		}
		l.pos++
	}
}

func (l *Lexer) save() {
	l.savePos = l.pos
	l.saveLine = l.line
	l.saveCol = l.col
}

type LexerState struct {
	Pos         int
	Line        int
	Col         int
	IndentStack []int
	HasEmitted  bool
}

func (l *Lexer) Snapshot() LexerState {
	stack := make([]int, len(l.indentStack))
	copy(stack, l.indentStack)
	return LexerState{
		Pos:         l.pos,
		Line:        l.line,
		Col:         l.col,
		IndentStack: stack,
		HasEmitted:  l.hasEmitted,
	}
}

func (l *Lexer) Restore(s LexerState) {
	l.pos = s.Pos
	l.line = s.Line
	l.col = s.Col
	l.indentStack = make([]int, len(s.IndentStack))
	copy(l.indentStack, s.IndentStack)
	l.hasEmitted = s.HasEmitted
}

func (l *Lexer) emit(kind Kind, data string) Token {
	return Token{
		Kind: kind,
		Data: data,
		Pos: Position{
			File: l.file, Line: l.saveLine, Col: l.saveCol, Offset: l.savePos,
		},
	}
}

func (l *Lexer) Next() Token {
	for {
		// At start of line (col == 0): compute indent, handle blank lines and EOF
		if l.col == 0 {
			// Tab is forbidden at any position
			if l.pos < len(l.src) && l.src[l.pos] == '\t' {
				panic(fmt.Sprintf("tab character not allowed at %s:%d:%d (use 4 spaces)", l.file, l.line+1, l.col+1))
			}

			// Skip leading spaces, counting indent
			indent := 0
			for l.pos < len(l.src) && l.src[l.pos] == ' ' {
				indent++
				l.pos++
				l.col++
			}

			// EOF at line start: emit DEDENTs until stack is [0], then EOF
			if l.pos >= len(l.src) {
				if len(l.indentStack) > 1 {
					l.indentStack = l.indentStack[:len(l.indentStack)-1]
					return l.emit(DEDENT, "")
				}
				return l.emit(EOF, "")
			}

			// Blank line: just whitespace + newline, skip
			if l.src[l.pos] == '\n' {
				l.line++
				l.col = 0
				l.pos++
				continue
			}

			// Only do indent check if we've already emitted at least one real token
			// (first line's leading spaces are silently skipped — no INDENT for the first line)
			if l.hasEmitted {
				top := l.indentStack[len(l.indentStack)-1]
				if indent > top {
					l.indentStack = append(l.indentStack, indent)
					return l.emit(INDENT, "")
				}
				if indent < top {
					for len(l.indentStack) > 1 && l.indentStack[len(l.indentStack)-1] > indent {
						l.indentStack = l.indentStack[:len(l.indentStack)-1]
						return l.emit(DEDENT, "")
					}
					if indent != l.indentStack[len(l.indentStack)-1] {
						panic(fmt.Sprintf("inconsistent dedent at %s:%d:%d", l.file, l.line+1, indent+1))
					}
				}
			}

			l.save()
		}

		// EOF check (covers EOF reached with col != 0, e.g., trailing spaces or no newline)
		if l.pos >= len(l.src) {
			if len(l.indentStack) > 1 {
				l.indentStack = l.indentStack[:len(l.indentStack)-1]
				return l.emit(DEDENT, "")
			}
			return l.emit(EOF, "")
		}

		ch := l.src[l.pos]

		// Comments
		if ch == '#' {
			isDoc := l.peek(1) == '#'
			l.advance()
			if isDoc {
				l.advance()
			}
			start := l.pos
			for l.pos < len(l.src) && l.src[l.pos] != '\n' {
				l.advance()
			}
			kind := COMMENT
			if isDoc {
				kind = DOC_COMMENT
			}
			l.hasEmitted = true
			return l.emit(kind, l.src[start:l.pos])
		}

		// Newline as token
		if ch == '\n' {
			l.line++
			l.col = 0
			l.pos++
			l.hasEmitted = true
			return l.emit(NEWLINE, "\n")
		}

		// Strings
		if ch == '"' || ch == '\'' || ch == '`' {
			tok := l.lexString(ch)
			l.hasEmitted = true
			return tok
		}
		if ch == 'f' && l.peek(1) == '"' {
			l.advance()
			tok := l.lexFString()
			l.hasEmitted = true
			return tok
		}

		// Numbers
		if isDigit(ch) {
			start := l.pos
			if ch == '0' && (l.peek(1) == 'x' || l.peek(1) == 'X') {
				l.advance()
				l.advance()
				for l.pos < len(l.src) && isHexDigit(l.src[l.pos]) {
					l.advance()
				}
				l.hasEmitted = true
				return l.emit(INT, l.src[start:l.pos])
			}
			for l.pos < len(l.src) && isDigit(l.src[l.pos]) {
				l.advance()
			}
			isFloat := false
			if l.pos < len(l.src) && l.src[l.pos] == '.' && l.pos+1 < len(l.src) && isDigit(l.src[l.pos+1]) {
				isFloat = true
				l.advance()
				for l.pos < len(l.src) && isDigit(l.src[l.pos]) {
					l.advance()
				}
			}
			if l.pos < len(l.src) && (l.src[l.pos] == 'e' || l.src[l.pos] == 'E') {
				isFloat = true
				l.advance()
				if l.pos < len(l.src) && (l.src[l.pos] == '+' || l.src[l.pos] == '-') {
					l.advance()
				}
				for l.pos < len(l.src) && isDigit(l.src[l.pos]) {
					l.advance()
				}
			}
			kind := INT
			if isFloat {
				kind = FLOAT
			}
			l.hasEmitted = true
			return l.emit(kind, l.src[start:l.pos])
		}

		// Identifiers / Keywords
		if isLetter(ch) {
			start := l.pos
			for l.pos < len(l.src) && (isLetter(l.src[l.pos]) || isDigit(l.src[l.pos])) {
				l.advance()
			}
			data := l.src[start:l.pos]
			if k, ok := keywordSet[data]; ok {
				l.hasEmitted = true
				return l.emit(k, data)
			}
			l.hasEmitted = true
			return l.emit(NAME, data)
		}

		// Operators
		switch ch {
		case '+':
			l.advance()
			l.hasEmitted = true
			return l.emit(PLUS, "+")
		case '-':
			l.advance()
			if l.peek(0) == '>' {
				l.advance()
				l.hasEmitted = true
				return l.emit(ARROW, "->")
			}
			l.hasEmitted = true
			return l.emit(MINUS, "-")
		case '*':
			l.advance()
			l.hasEmitted = true
			return l.emit(STAR, "*")
		case '/':
			l.advance()
			l.hasEmitted = true
			return l.emit(SLASH, "/")
		case '%':
			l.advance()
			l.hasEmitted = true
			return l.emit(PERCENT, "%")
		case '=':
			l.advance()
			if l.peek(0) == '=' {
				l.advance()
				l.hasEmitted = true
				return l.emit(EQEQ, "==")
			}
			if l.peek(0) == '>' {
				l.advance()
				l.hasEmitted = true
				return l.emit(FATARROW, "=>")
			}
			l.hasEmitted = true
			return l.emit(EQ, "=")
		case '!':
			l.advance()
			if l.peek(0) == '=' {
				l.advance()
				l.hasEmitted = true
				return l.emit(NEQ, "!=")
			}
			return Token{Kind: EOF}
		case '<':
			l.advance()
			if l.peek(0) == '=' {
				l.advance()
				l.hasEmitted = true
				return l.emit(LTE, "<=")
			}
			l.hasEmitted = true
			return l.emit(LT, "<")
		case '>':
			l.advance()
			if l.peek(0) == '=' {
				l.advance()
				l.hasEmitted = true
				return l.emit(GTE, ">=")
			}
			l.hasEmitted = true
			return l.emit(GT, ">")
		case '(':
			l.advance()
			l.hasEmitted = true
			return l.emit(LPAREN, "(")
		case ')':
			l.advance()
			l.hasEmitted = true
			return l.emit(RPAREN, ")")
		case '[':
			l.advance()
			l.hasEmitted = true
			return l.emit(LBRACK, "[")
		case ']':
			l.advance()
			l.hasEmitted = true
			return l.emit(RBRACK, "]")
		case ',':
			l.advance()
			l.hasEmitted = true
			return l.emit(COMMA, ",")
		case '.':
			l.advance()
			l.hasEmitted = true
			return l.emit(DOT, ".")
		case ':':
			l.advance()
			l.hasEmitted = true
			return l.emit(COLON, ":")
		case '?':
			l.advance()
			l.hasEmitted = true
			return l.emit(QUESTION, "?")
		case '@':
			l.advance()
			l.hasEmitted = true
			return l.emit(AT, "@")
		}

		// Unknown: skip one byte and continue
		l.advance()
	}
}

func isLetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || b == '_'
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func isHexDigit(b byte) bool {
	return isDigit(b) || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}

func (l *Lexer) lexString(quote byte) Token {
	l.advance()
	var buf []byte
	for l.pos < len(l.src) && l.src[l.pos] != quote {
		if quote != '`' && l.src[l.pos] == '\\' && l.pos+1 < len(l.src) {
			esc := l.src[l.pos+1]
			switch esc {
			case 'n':
				buf = append(buf, '\n')
			case 't':
				buf = append(buf, '\t')
			case 'r':
				buf = append(buf, '\r')
			case '\\':
				buf = append(buf, '\\')
			case '"':
				buf = append(buf, '"')
			case '\'':
				buf = append(buf, '\'')
			default:
				buf = append(buf, '\\', esc)
			}
			l.advance()
			l.advance()
			continue
		}
		if l.src[l.pos] == '\n' {
			l.advance()
			continue
		}
		buf = append(buf, l.src[l.pos])
		l.advance()
	}
	if l.pos < len(l.src) {
		l.advance()
	}
	return l.emit(STR, string(buf))
}

func (l *Lexer) lexFString() Token {
	l.advance()
	var buf []byte
	for l.pos < len(l.src) && l.src[l.pos] != '"' {
		buf = append(buf, l.src[l.pos])
		l.advance()
	}
	if l.pos < len(l.src) {
		l.advance()
	}
	return l.emit(FSTR, string(buf))
}
