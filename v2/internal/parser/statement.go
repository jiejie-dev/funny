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
	return nil, errs.New("E1002",
		fmt.Sprintf("unexpected token %s at start of statement", p.cur.Kind),
		errPos(p.cur.Pos), "")
}

func (p *Parser) parseLet() (ast.Statement, error) { return nil, fmt.Errorf("parseLet stub (Task 16)") }
func (p *Parser) parseIf() (ast.Statement, error)  { return nil, fmt.Errorf("parseIf stub (Task 17)") }
func (p *Parser) parseFor() (ast.Statement, error) { return nil, fmt.Errorf("parseFor stub (Task 17)") }
func (p *Parser) parseWhile() (ast.Statement, error) {
	return nil, fmt.Errorf("parseWhile stub (Task 17)")
}
func (p *Parser) parseMatch() (ast.Statement, error) {
	return nil, fmt.Errorf("parseMatch stub (Task 17)")
}
func (p *Parser) parseReturn() (ast.Statement, error) {
	return nil, fmt.Errorf("parseReturn stub (Task 17)")
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
	return nil, fmt.Errorf("parseAssignOrExpr stub (Task 16)")
}

func (p *Parser) parseExpression() (ast.Expression, error) {
	return nil, fmt.Errorf("parseExpression stub (Task 15)")
}
