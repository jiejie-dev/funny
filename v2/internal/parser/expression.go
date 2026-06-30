package parser

import (
	"fmt"
	"strconv"

	"github.com/jerloo/funny/v2/internal/ast"
	"github.com/jerloo/funny/v2/internal/errs"
	"github.com/jerloo/funny/v2/internal/lexer"
)

const (
	precLowest = iota
	precOr
	precAnd
	precNot
	precCmp
	precAdd
	precMul
	precUnary
	precCall
	precPrimary
)

func precedence(k lexer.Kind) int {
	switch k {
	case lexer.OR:
		return precOr
	case lexer.AND:
		return precAnd
	case lexer.NOT:
		return precNot
	case lexer.EQEQ, lexer.NEQ, lexer.LT, lexer.GT, lexer.LTE, lexer.GTE, lexer.IN:
		return precCmp
	case lexer.PLUS, lexer.MINUS:
		return precAdd
	case lexer.STAR, lexer.SLASH, lexer.PERCENT:
		return precMul
	case lexer.LPAREN, lexer.DOT, lexer.LBRACK:
		return precCall
	}
	return precLowest
}

func (p *Parser) parseExpression() (ast.Expression, error) {
	return p.parseBinary(precLowest)
}

func (p *Parser) parseBinary(minPrec int) (ast.Expression, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for {
		prec := precedence(p.cur.Kind)
		if prec <= minPrec {
			break
		}
		opStr := p.cur.Data
		pos := astPos(p.cur.Pos)
		p.advance()
		right, err := p.parseBinary(prec)
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpr{NodePos: pos, Left: left, Op: opStr, Right: right}
	}
	return left, nil
}

func (p *Parser) parseUnary() (ast.Expression, error) {
	if p.cur.Kind == lexer.MINUS || p.cur.Kind == lexer.NOT {
		op := p.cur.Data
		pos := astPos(p.cur.Pos)
		p.advance()
		inner, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &ast.UnaryExpr{NodePos: pos, Op: op, Expr: inner}, nil
	}
	return p.parsePostfix()
}

func (p *Parser) parsePostfix() (ast.Expression, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	for {
		// Struct literal: Name(field: val, ...) - detect before treating as a call.
		if varExpr, ok := left.(*ast.VariableExpr); ok {
			if p.cur.Kind == lexer.LPAREN {
				state := p.save()
				p.advance() // consume '('
				isStructLit := false
				if p.cur.Kind == lexer.NAME {
					p.advance()
					if p.cur.Kind == lexer.COLON {
						isStructLit = true
					}
				}
				p.restore(state)
				if isStructLit {
					lit, err := p.parseStructLiteral(varExpr.Name)
					if err != nil {
						return nil, err
					}
					left = lit
					continue
				}
			}
		}
		switch p.cur.Kind {
		case lexer.LPAREN:
			pos := astPos(p.cur.Pos)
			p.advance()
			var args []ast.Expression
			for p.cur.Kind != lexer.RPAREN && p.cur.Kind != lexer.EOF {
				e, err := p.parseExpression()
				if err != nil {
					return nil, err
				}
				args = append(args, e)
				if p.cur.Kind == lexer.COMMA {
					p.advance()
				}
			}
			if _, err := p.expect(lexer.RPAREN); err != nil {
				return nil, err
			}
			left = &ast.CallExpr{NodePos: pos, Func: left, Args: args}
		case lexer.DOT:
			p.advance()
			if p.cur.Kind != lexer.NAME {
				return nil, errs.New("E1010", "expected field name after `.`", errPos(p.cur.Pos), "")
			}
			left = &ast.FieldExpr{NodePos: left.Pos(), Object: left, Field: p.cur.Data}
			p.advance()
		case lexer.LBRACK:
			pos := astPos(p.cur.Pos)
			p.advance()
			idx, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			if _, err := p.expect(lexer.RBRACK); err != nil {
				return nil, err
			}
			left = &ast.IndexExpr{NodePos: pos, Object: left, Index: idx}
		default:
			return left, nil
		}
	}
}

func (p *Parser) parsePrimary() (ast.Expression, error) {
	pos := astPos(p.cur.Pos)
	switch p.cur.Kind {
	case lexer.INT:
		n, err := strconv.ParseInt(p.cur.Data, 0, 64)
		if err != nil {
			n2, err2 := strconv.ParseInt(p.cur.Data[2:], 16, 64)
			if err2 != nil {
				return nil, errs.New("E1011", fmt.Sprintf("invalid int %q", p.cur.Data), errPos(p.cur.Pos), "")
			}
			n = n2
		}
		p.advance()
		return &ast.LiteralExpr{NodePos: pos, Value: int(n)}, nil
	case lexer.FLOAT:
		f, err := strconv.ParseFloat(p.cur.Data, 64)
		if err != nil {
			return nil, errs.New("E1011", fmt.Sprintf("invalid float %q", p.cur.Data), errPos(p.cur.Pos), "")
		}
		p.advance()
		return &ast.LiteralExpr{NodePos: pos, Value: f}, nil
	case lexer.STR:
		s := p.cur.Data
		p.advance()
		return &ast.LiteralExpr{NodePos: pos, Value: s}, nil
	case lexer.TRUE:
		p.advance()
		return &ast.LiteralExpr{NodePos: pos, Value: true}, nil
	case lexer.FALSE:
		p.advance()
		return &ast.LiteralExpr{NodePos: pos, Value: false}, nil
	case lexer.NIL:
		p.advance()
		return &ast.LiteralExpr{NodePos: pos, Value: nil}, nil
	case lexer.FSTR:
		s := p.cur.Data
		p.advance()
		return &ast.FStringExpr{NodePos: pos, Raw: s}, nil
	case lexer.NAME:
		name := p.cur.Data
		p.advance()
		return &ast.VariableExpr{NodePos: pos, Name: name}, nil
	case lexer.LPAREN:
		p.advance()
		inner, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(lexer.RPAREN); err != nil {
			return nil, err
		}
		return &ast.SubExpr{NodePos: pos, Inner: inner}, nil
	case lexer.LBRACK:
		p.advance()
		var elems []ast.Expression
		for p.cur.Kind != lexer.RBRACK && p.cur.Kind != lexer.EOF {
			e, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			elems = append(elems, e)
			if p.cur.Kind == lexer.COMMA {
				p.advance()
			}
		}
		if _, err := p.expect(lexer.RBRACK); err != nil {
			return nil, err
		}
		return &ast.ListExpr{NodePos: pos, Elements: elems}, nil
	}
	return nil, errs.New("E1012",
		fmt.Sprintf("unexpected token %s in expression", p.cur.Kind),
		errPos(p.cur.Pos), "")
}

func (p *Parser) parseStructLiteral(typeName string) (ast.Expression, error) {
	pos := astPos(p.cur.Pos)
	p.advance() // consume '('
	fields := map[string]ast.Expression{}
	for p.cur.Kind != lexer.RPAREN && p.cur.Kind != lexer.EOF {
		if p.cur.Kind != lexer.NAME {
			return nil, errs.New("E1090", "expected field name in struct literal", errPos(p.cur.Pos), "")
		}
		fieldName := p.cur.Data
		p.advance()
		if _, err := p.expect(lexer.COLON); err != nil {
			return nil, err
		}
		val, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		fields[fieldName] = val
		if p.cur.Kind == lexer.COMMA {
			p.advance()
		}
	}
	if _, err := p.expect(lexer.RPAREN); err != nil {
		return nil, err
	}
	return &ast.StructLiteralExpr{
		NodePos:  pos,
		TypeName: typeName,
		Fields:   fields,
	}, nil
}
