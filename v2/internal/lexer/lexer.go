package lexer

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
	l.save()
	if l.pos >= len(l.src) {
		return l.emit(EOF, "")
	}
	ch := l.src[l.pos]

	for ch == ' ' || ch == '\t' {
		l.advance()
		if l.pos >= len(l.src) {
			return l.emit(EOF, "")
		}
		ch = l.src[l.pos]
	}

	l.save()

	switch ch {
	case '+':
		l.advance()
		return l.emit(PLUS, "+")
	case '-':
		l.advance()
		if l.peek(0) == '>' {
			l.advance()
			return l.emit(ARROW, "->")
		}
		return l.emit(MINUS, "-")
	case '*':
		l.advance()
		return l.emit(STAR, "*")
	case '/':
		l.advance()
		return l.emit(SLASH, "/")
	case '%':
		l.advance()
		return l.emit(PERCENT, "%")
	case '=':
		l.advance()
		if l.peek(0) == '=' {
			l.advance()
			return l.emit(EQEQ, "==")
		}
		if l.peek(0) == '>' {
			l.advance()
			return l.emit(FATARROW, "=>")
		}
		return l.emit(EQ, "=")
	case '!':
		l.advance()
		if l.peek(0) == '=' {
			l.advance()
			return l.emit(NEQ, "!=")
		}
		return Token{Kind: EOF}
	case '<':
		l.advance()
		if l.peek(0) == '=' {
			l.advance()
			return l.emit(LTE, "<=")
		}
		return l.emit(LT, "<")
	case '>':
		l.advance()
		if l.peek(0) == '=' {
			l.advance()
			return l.emit(GTE, ">=")
		}
		return l.emit(GT, ">")
	case '(':
		l.advance()
		return l.emit(LPAREN, "(")
	case ')':
		l.advance()
		return l.emit(RPAREN, ")")
	case '[':
		l.advance()
		return l.emit(LBRACK, "[")
	case ']':
		l.advance()
		return l.emit(RBRACK, "]")
	case ',':
		l.advance()
		return l.emit(COMMA, ",")
	case '.':
		l.advance()
		return l.emit(DOT, ".")
	case ':':
		l.advance()
		return l.emit(COLON, ":")
	case '?':
		l.advance()
		return l.emit(QUESTION, "?")
	case '@':
		l.advance()
		return l.emit(AT, "@")
	}

	if isLetter(ch) {
		start := l.pos
		for l.pos < len(l.src) && (isLetter(l.src[l.pos]) || isDigit(l.src[l.pos])) {
			l.advance()
		}
		data := l.src[start:l.pos]
		if k, ok := keywordSet[data]; ok {
			return l.emit(k, data)
		}
		return l.emit(NAME, data)
	}

	l.advance()
	return l.Next()
}

func isLetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || b == '_'
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}