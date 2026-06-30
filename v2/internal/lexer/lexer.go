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