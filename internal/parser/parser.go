package parser

import (
	"fmt"

	"github.com/jiejie-dev/funny/internal/ast"
	"github.com/jiejie-dev/funny/internal/errs"
	"github.com/jiejie-dev/funny/internal/lexer"
)

type Parser struct {
	lx   *lexer.Lexer
	cur  lexer.Token
	peek lexer.Token
}

func New(src, file string) *Parser {
	p := &Parser{lx: lexer.New(src, file)}
	p.advance()
	p.advance()
	return p
}

func (p *Parser) advance() { p.cur = p.peek; p.peek = p.lx.Next() }

type parserState struct {
	cur  lexer.Token
	peek lexer.Token
	lx   lexer.LexerState
}

func (p *Parser) save() parserState {
	return parserState{cur: p.cur, peek: p.peek, lx: p.lx.Snapshot()}
}

func (p *Parser) restore(s parserState) {
	p.cur = s.cur
	p.peek = s.peek
	p.lx.Restore(s.lx)
}

func (p *Parser) expect(k lexer.Kind) (lexer.Token, *errs.Error) {
	if p.cur.Kind == k {
		tok := p.cur
		p.advance()
		return tok, nil
	}
	return lexer.Token{}, errs.New("E1001",
		fmt.Sprintf("expected %s, got %s", k, p.cur.Kind),
		errPos(p.cur.Pos),
		fmt.Sprintf("expected `%s` here", k))
}

func (p *Parser) atEOF() bool { return p.cur.Kind == lexer.EOF }

func (p *Parser) Parse() (*ast.Program, error) {
	prog := &ast.Program{NodePos: astPos(p.cur.Pos)}
	for !p.atEOF() {
		for p.cur.Kind == lexer.NEWLINE {
			p.advance()
		}
		if p.atEOF() {
			break
		}
		s, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		if s != nil {
			prog.Stmts = append(prog.Stmts, s)
		}
	}
	return prog, nil
}

func astPos(p lexer.Position) ast.Pos {
	return ast.Pos{File: p.File, Line: p.Line, Col: p.Col}
}

func errPos(p lexer.Position) errs.Position {
	return errs.Position{File: p.File, Line: p.Line, Col: p.Col}
}
