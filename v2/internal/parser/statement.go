package parser

import (
	"fmt"

	"github.com/jerloo/funny/v2/internal/ast"
	"github.com/jerloo/funny/v2/internal/errs"
	"github.com/jerloo/funny/v2/internal/lexer"
)

func (p *Parser) parseStatement() (ast.Statement, error) {
	switch p.cur.Kind {
	case lexer.LET:
		return p.parseLet()
	case lexer.IF:
		return p.parseIf()
	case lexer.FOR:
		return p.parseFor()
	case lexer.WHILE:
		return p.parseWhile()
	case lexer.MATCH:
		return p.parseMatch()
	case lexer.RETURN:
		return p.parseReturn()
	case lexer.BREAK:
		p.advance()
		return &ast.BreakStmt{NodePos: astPos(p.cur.Pos)}, nil
	case lexer.CONTINUE:
		p.advance()
		return &ast.ContinueStmt{NodePos: astPos(p.cur.Pos)}, nil
	case lexer.FN:
		return p.parseFnDecl()
	case lexer.STRUCT:
		return p.parseStructDecl()
	case lexer.META:
		return p.parseMeta()
	case lexer.PLAN:
		return p.parsePlan()
	case lexer.IMPORT:
		return p.parseImport()
	case lexer.PUB:
		return p.parsePub()
	case lexer.NAME:
		return p.parseAssignOrExpr()
	}
	if isExpressionStart(p.cur.Kind) {
		pos := astPos(p.cur.Pos)
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		return &ast.ExprStmt{NodePos: pos, X: expr}, nil
	}
	return nil, errs.New("E1002",
		fmt.Sprintf("unexpected token %s at start of statement", p.cur.Kind),
		errPos(p.cur.Pos), "")
}

func isExpressionStart(k lexer.Kind) bool {
	switch k {
	case lexer.INT, lexer.FLOAT, lexer.STR, lexer.FSTR,
		lexer.TRUE, lexer.FALSE, lexer.NIL,
		lexer.NAME, lexer.LPAREN, lexer.LBRACK,
		lexer.MINUS, lexer.NOT:
		return true
	}
	return false
}

func (p *Parser) parseLet() (ast.Statement, error) {
	pos := astPos(p.cur.Pos)
	p.advance()
	if p.cur.Kind != lexer.NAME {
		return nil, errs.New("E1005", "expected variable name after `let`", errPos(p.cur.Pos), "")
	}
	name := p.cur.Data
	p.advance()
	var typeAnn string
	if p.cur.Kind == lexer.COLON {
		p.advance()
		typeAnn = p.cur.Data
		p.advance()
	}
	if _, err := p.expect(lexer.EQ); err != nil {
		return nil, err
	}
	val, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	return &ast.LetStmt{NodePos: pos, Name: name, TypeAnn: typeAnn, Value: val}, nil
}
func (p *Parser) parseBlock() (*ast.Block, error) {
	pos := astPos(p.cur.Pos)
	if p.cur.Kind == lexer.NEWLINE {
		p.advance()
	}
	if p.cur.Kind != lexer.INDENT {
		return nil, errs.New("E1003",
			fmt.Sprintf("expected INDENT for block, got %s", p.cur.Kind),
			errPos(p.cur.Pos), "blocks must be on a new line with indented content")
	}
	p.advance()
	block := &ast.Block{NodePos: pos}
	for p.cur.Kind != lexer.DEDENT && p.cur.Kind != lexer.EOF {
		for p.cur.Kind == lexer.NEWLINE {
			p.advance()
		}
		if p.cur.Kind == lexer.DEDENT || p.cur.Kind == lexer.EOF {
			break
		}
		s, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		if s != nil {
			block.Statements = append(block.Statements, s)
		}
	}
	if p.cur.Kind == lexer.DEDENT {
		p.advance()
	}
	return block, nil
}

func (p *Parser) parseIf() (ast.Statement, error) {
	pos := astPos(p.cur.Pos)
	p.advance()
	cond, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.COLON); err != nil {
		return nil, err
	}
	thenBlock, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	ifStmt := &ast.IfStmt{NodePos: pos, Cond: cond, Then: thenBlock}
	if p.cur.Kind == lexer.ELIF {
		elif, err := p.parseIf()
		if err != nil {
			return nil, err
		}
		inner := elif.(*ast.IfStmt)
		ifStmt.ElseIf = inner
		if inner.ElseBlock != nil {
			ifStmt.ElseBlock = inner.ElseBlock
			inner.ElseBlock = nil
		}
	} else if p.cur.Kind == lexer.ELSE {
		p.advance()
		if _, err := p.expect(lexer.COLON); err != nil {
			return nil, err
		}
		elseBlock, err := p.parseBlock()
		if err != nil {
			return nil, err
		}
		ifStmt.ElseBlock = elseBlock
	}
	return ifStmt, nil
}

func (p *Parser) parseFor() (ast.Statement, error) {
	pos := astPos(p.cur.Pos)
	p.advance()
	if p.cur.Kind != lexer.NAME {
		return nil, errs.New("E1020", "expected loop variable after `for`", errPos(p.cur.Pos), "")
	}
	name := p.cur.Data
	p.advance()
	if _, err := p.expect(lexer.IN); err != nil {
		return nil, err
	}
	iterable, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.COLON); err != nil {
		return nil, err
	}
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	return &ast.ForStmt{NodePos: pos, Name: name, Iterable: iterable, Body: body}, nil
}

func (p *Parser) parseWhile() (ast.Statement, error) {
	pos := astPos(p.cur.Pos)
	p.advance()
	cond, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.COLON); err != nil {
		return nil, err
	}
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	return &ast.WhileStmt{NodePos: pos, Cond: cond, Body: body}, nil
}

func (p *Parser) parseMatch() (ast.Statement, error) {
	pos := astPos(p.cur.Pos)
	p.advance()
	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.COLON); err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.INDENT); err != nil {
		return nil, err
	}
	var arms []ast.MatchArm
	for p.cur.Kind != lexer.DEDENT && p.cur.Kind != lexer.EOF {
		for p.cur.Kind == lexer.NEWLINE {
			p.advance()
		}
		if p.cur.Kind == lexer.DEDENT {
			break
		}
		pattern, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(lexer.FATARROW); err != nil {
			return nil, err
		}
		body, err := p.parseBlock()
		if err != nil {
			return nil, err
		}
		arms = append(arms, ast.MatchArm{Pattern: pattern, Body: body})
	}
	if p.cur.Kind == lexer.DEDENT {
		p.advance()
	}
	return &ast.MatchStmt{NodePos: pos, Expr: expr, Arms: arms}, nil
}

func (p *Parser) parseReturn() (ast.Statement, error) {
	pos := astPos(p.cur.Pos)
	p.advance()
	if p.cur.Kind == lexer.NEWLINE || p.cur.Kind == lexer.EOF || p.cur.Kind == lexer.DEDENT {
		return &ast.ReturnStmt{NodePos: pos, Value: nil}, nil
	}
	val, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	return &ast.ReturnStmt{NodePos: pos, Value: val}, nil
}
func (p *Parser) parseFnDecl() (ast.Statement, error) {
	return nil, fmt.Errorf("parseFnDecl stub (Task 18)")
}
func (p *Parser) parseStructDecl() (ast.Statement, error) {
	return nil, fmt.Errorf("parseStructDecl stub (Task 18)")
}
func (p *Parser) parseMeta() (ast.Statement, error) {
	return nil, fmt.Errorf("parseMeta stub (Task 19)")
}
func (p *Parser) parsePlan() (ast.Statement, error) {
	return nil, fmt.Errorf("parsePlan stub (Task 19)")
}
func (p *Parser) parseImport() (ast.Statement, error) {
	return nil, fmt.Errorf("parseImport stub (Task 19)")
}
func (p *Parser) parsePub() (ast.Statement, error) { return nil, fmt.Errorf("parsePub stub (Task 18)") }
func (p *Parser) parseAssignOrExpr() (ast.Statement, error) {
	left, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if p.cur.Kind == lexer.EQ {
		pos := astPos(p.cur.Pos)
		p.advance()
		val, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		return &ast.AssignStmt{NodePos: pos, Target: left, Value: val}, nil
	}
	return &ast.ExprStmt{NodePos: left.Pos(), X: left}, nil
}
