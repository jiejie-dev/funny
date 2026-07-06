package parser

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jiejie-dev/funny/v2/internal/ast"
	"github.com/jiejie-dev/funny/v2/internal/errs"
	"github.com/jiejie-dev/funny/v2/internal/lexer"
)

// tokenLiteral returns the source-level text representation of a token,
// reconstructing punctuation that the lexer leaves as empty Data.
func tokenLiteral(k lexer.Kind, data string) string {
	switch k {
	case lexer.LBRACK:
		return "["
	case lexer.RBRACK:
		return "]"
	case lexer.LPAREN:
		return "("
	case lexer.RPAREN:
		return ")"
	case lexer.COMMA:
		return ","
	case lexer.QUESTION:
		return "?"
	case lexer.ARROW:
		return "->"
	}
	return data
}

// consumeTypeAnn consumes tokens until it hits one of the stop kinds (or EOF)
// and builds a type annotation string suitable for types.ParseType.
func (p *Parser) consumeTypeAnn(stopKinds ...lexer.Kind) string {
	var parts []string
	for {
		stop := false
		for _, k := range stopKinds {
			if p.cur.Kind == k {
				stop = true
				break
			}
		}
		if stop || p.cur.Kind == lexer.EOF {
			break
		}
		parts = append(parts, tokenLiteral(p.cur.Kind, p.cur.Data))
		p.advance()
	}
	var b strings.Builder
	for i, part := range parts {
		if i > 0 && parts[i-1] == "," {
			b.WriteString(" ")
		}
		b.WriteString(part)
	}
	return b.String()
}

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
	case lexer.STEP:
		return p.parseStep()
	case lexer.IMPORT:
		return p.parseImport()
	case lexer.PUB:
		return p.parsePub()
	case lexer.NAME:
		return p.parseAssignOrExpr()
	case lexer.COMMENT, lexer.DOC_COMMENT:
		text := p.cur.Data
		isDoc := p.cur.Kind == lexer.DOC_COMMENT
		pos := astPos(p.cur.Pos)
		p.advance()
		return &ast.CommentStmt{NodePos: pos, Text: text, Doc: isDoc}, nil
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
		lexer.NAME, lexer.LPAREN, lexer.LBRACK, lexer.LBRACE,
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
		typeAnn = p.consumeTypeAnn(lexer.EQ, lexer.NEWLINE)
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
	if p.cur.Kind == lexer.NEWLINE {
		p.advance()
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
func (p *Parser) parsePub() (ast.Statement, error) {
	p.advance()
	switch p.cur.Kind {
	case lexer.FN:
		fn, err := p.parseFnDecl()
		if err != nil {
			return nil, err
		}
		fn.(*ast.FnDecl).Pub = true
		return fn, nil
	case lexer.STRUCT:
		s, err := p.parseStructDecl()
		if err != nil {
			return nil, err
		}
		s.(*ast.StructDecl).Pub = true
		return s, nil
	}
	return nil, errs.New("E1030", "`pub` must precede `fn` or `struct`", errPos(p.cur.Pos), "")
}

func (p *Parser) parseFnDecl() (ast.Statement, error) {
	pos := astPos(p.cur.Pos)
	p.advance()
	if p.cur.Kind != lexer.NAME {
		return nil, errs.New("E1031", "expected function name after `fn`", errPos(p.cur.Pos), "")
	}
	name := p.cur.Data
	p.advance()
	if _, err := p.expect(lexer.LPAREN); err != nil {
		return nil, err
	}
	var params []ast.Param
	for p.cur.Kind != lexer.RPAREN && p.cur.Kind != lexer.EOF {
		if p.cur.Kind != lexer.NAME {
			return nil, errs.New("E1032", "expected parameter name", errPos(p.cur.Pos), "")
		}
		pname := p.cur.Data
		p.advance()
		var ptype string
		if p.cur.Kind == lexer.COLON {
			p.advance()
			ptype = p.consumeTypeAnn(lexer.COMMA, lexer.RPAREN)
		}
		params = append(params, ast.Param{Name: pname, TypeAnn: ptype})
		if p.cur.Kind == lexer.COMMA {
			p.advance()
		}
	}
	if _, err := p.expect(lexer.RPAREN); err != nil {
		return nil, err
	}
	var retType string
	if p.cur.Kind == lexer.ARROW {
		p.advance()
		retType = p.consumeTypeAnn(lexer.COLON)
	}
	if _, err := p.expect(lexer.COLON); err != nil {
		return nil, err
	}
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	return &ast.FnDecl{NodePos: pos, Name: name, Params: params, RetType: retType, Body: body}, nil
}

func (p *Parser) parseStructDecl() (ast.Statement, error) {
	pos := astPos(p.cur.Pos)
	p.advance()
	if p.cur.Kind != lexer.NAME {
		return nil, errs.New("E1033", "expected struct name", errPos(p.cur.Pos), "")
	}
	name := p.cur.Data
	p.advance()
	if _, err := p.expect(lexer.COLON); err != nil {
		return nil, err
	}
	if p.cur.Kind == lexer.NEWLINE {
		p.advance()
	}
	if _, err := p.expect(lexer.INDENT); err != nil {
		return nil, err
	}
	var fields []ast.Param
	for p.cur.Kind != lexer.DEDENT && p.cur.Kind != lexer.EOF {
		for p.cur.Kind == lexer.NEWLINE {
			p.advance()
		}
		if p.cur.Kind == lexer.DEDENT || p.cur.Kind == lexer.EOF {
			break
		}
		if p.cur.Kind != lexer.NAME {
			return nil, errs.New("E1034", "expected field name in struct", errPos(p.cur.Pos), "")
		}
		fname := p.cur.Data
		p.advance()
		var ftype string
		if p.cur.Kind == lexer.COLON {
			p.advance()
			ftype = p.consumeTypeAnn(lexer.NEWLINE)
		}
		fields = append(fields, ast.Param{Name: fname, TypeAnn: ftype})
	}
	if p.cur.Kind == lexer.DEDENT {
		p.advance()
	}
	return &ast.StructDecl{NodePos: pos, Name: name, Fields: fields}, nil
}
func (p *Parser) parseMeta() (ast.Statement, error) {
	pos := astPos(p.cur.Pos)
	p.advance()
	if _, err := p.expect(lexer.COLON); err != nil {
		return nil, err
	}
	block, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	fields := map[string]string{}
	for _, s := range block.Statements {
		assign, ok := s.(*ast.AssignStmt)
		if !ok {
			continue
		}
		varExpr, ok := assign.Target.(*ast.VariableExpr)
		if !ok {
			continue
		}
		if lit, ok := assign.Value.(*ast.LiteralExpr); ok {
			if s, ok := lit.Value.(string); ok {
				fields[varExpr.Name] = s
			}
		}
	}
	return &ast.MetaBlock{NodePos: pos, Fields: fields}, nil
}

func (p *Parser) parsePlan() (ast.Statement, error) {
	pos := astPos(p.cur.Pos)
	p.advance()
	if p.cur.Kind != lexer.STR {
		return nil, errs.New("E1040", "expected plan name as string", errPos(p.cur.Pos), "")
	}
	name := p.cur.Data
	p.advance()
	if _, err := p.expect(lexer.COLON); err != nil {
		return nil, err
	}
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	return &ast.PlanBlock{NodePos: pos, Name: name, Body: body}, nil
}

func (p *Parser) parseStep() (ast.Statement, error) {
	pos := astPos(p.cur.Pos)
	p.advance()
	if p.cur.Kind != lexer.STR {
		return nil, errs.New("E1043", "expected step name as string", errPos(p.cur.Pos), "")
	}
	name := p.cur.Data
	p.advance()
	step := &ast.Step{NodePos: pos, Name: name, Kind: ast.StepTool}
	if p.cur.Kind == lexer.ARROW {
		p.advance()
		if p.cur.Kind != lexer.NAME {
			return nil, errs.New("E1044", "expected step kind after ->", errPos(p.cur.Pos), "")
		}
		step.Kind = ast.StepKind(p.cur.Data)
		p.advance()
	}
	if p.cur.Kind == lexer.NAME && p.cur.Data == "with" {
		p.advance()
		// The literal `retry` keyword is optional and purely cosmetic now
		// that `with` accepts any mix of retry/timeout options — kept so
		// `with retry max=3:` (the only form that existed before timeout
		// and backoff were added) keeps parsing unchanged.
		if p.cur.Kind == lexer.NAME && p.cur.Data == "retry" {
			p.advance()
		}
		retry := &ast.Retry{}
		sawRetryOption := false
		for p.cur.Kind == lexer.NAME {
			key := p.cur.Data
			p.advance()
			if _, err := p.expect(lexer.EQ); err != nil {
				return nil, err
			}
			switch key {
			case "max":
				if p.cur.Kind != lexer.INT {
					return nil, errs.New("E1046", fmt.Sprintf("expected int value for %s", key), errPos(p.cur.Pos), "")
				}
				n, _ := strconv.Atoi(p.cur.Data)
				retry.Max = n
				sawRetryOption = true
				p.advance()
			case "backoff":
				if p.cur.Kind != lexer.NAME {
					return nil, errs.New("E1047", "expected backoff strategy name (constant, linear, or exp)", errPos(p.cur.Pos), "")
				}
				switch p.cur.Data {
				case "constant", "linear", "exp":
				default:
					return nil, errs.New("E1047", fmt.Sprintf("unknown backoff strategy %q (expected constant, linear, or exp)", p.cur.Data), errPos(p.cur.Pos), "")
				}
				retry.Backoff = p.cur.Data
				sawRetryOption = true
				p.advance()
			case "timeout":
				if p.cur.Kind != lexer.STR {
					return nil, errs.New("E1048", "expected quoted duration string for timeout (e.g. \"5s\")", errPos(p.cur.Pos), "")
				}
				if _, err := time.ParseDuration(p.cur.Data); err != nil {
					return nil, errs.New("E1048", fmt.Sprintf("invalid timeout duration %q: %s", p.cur.Data, err), errPos(p.cur.Pos), "")
				}
				step.Timeout = p.cur.Data
				p.advance()
			case "on":
				types, err := p.parseRetryOnList()
				if err != nil {
					return nil, err
				}
				retry.On = types
				sawRetryOption = true
			default:
				return nil, errs.New("E1049", fmt.Sprintf("unknown step option %q (expected max, backoff, timeout, or on)", key), errPos(p.cur.Pos), "")
			}
		}
		if sawRetryOption {
			step.Retry = retry
		}
	}
	if _, err := p.expect(lexer.COLON); err != nil {
		return nil, err
	}
	if step.Kind == ast.StepBranch {
		if err := p.parseBranchStepTail(step); err != nil {
			return nil, err
		}
		return step, nil
	}
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	step.Body = body
	return step, nil
}

// parseBranchStepTail parses either a case-list (`cond => "step"`) or a
// legacy statement body (`if`/`else` ...) after `-> branch:`.
func (p *Parser) parseBranchStepTail(step *ast.Step) error {
	if p.cur.Kind == lexer.NEWLINE {
		p.advance()
	}
	if _, err := p.expect(lexer.INDENT); err != nil {
		return err
	}
	if p.cur.Kind == lexer.IF {
		block, err := p.parseBlockFromIndentedStatements()
		if err != nil {
			return err
		}
		step.Body = block
		return nil
	}
	cases, err := p.parseBranchCasesBody()
	if err != nil {
		return err
	}
	step.BranchCases = cases
	return nil
}

func (p *Parser) parseBranchCasesBody() ([]ast.BranchCase, error) {
	var cases []ast.BranchCase
	for p.cur.Kind != lexer.DEDENT && p.cur.Kind != lexer.EOF {
		for p.cur.Kind == lexer.NEWLINE {
			p.advance()
		}
		if p.cur.Kind == lexer.DEDENT {
			break
		}
		cond, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(lexer.FATARROW); err != nil {
			return nil, err
		}
		if p.cur.Kind != lexer.STR {
			return nil, errs.New("E1051", "expected target step name as string after =>", errPos(p.cur.Pos), "")
		}
		target := p.cur.Data
		p.advance()
		cases = append(cases, ast.BranchCase{Cond: cond, Target: target})
	}
	if len(cases) == 0 {
		return nil, errs.New("E1051", "branch step requires at least one case (cond => \"step\")", errPos(p.cur.Pos), "")
	}
	if p.cur.Kind == lexer.DEDENT {
		p.advance()
	}
	return cases, nil
}

// parseBlockFromIndentedStatements parses statements until DEDENT when the
// INDENT token has already been consumed (used by branch legacy bodies).
func (p *Parser) parseBlockFromIndentedStatements() (*ast.Block, error) {
	pos := astPos(p.cur.Pos)
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

func (p *Parser) parseRetryOnList() ([]string, error) {
	if p.cur.Kind != lexer.NAME {
		return nil, errs.New("E1052", "expected error type name after on=", errPos(p.cur.Pos), "")
	}
	var types []string
	for {
		types = append(types, p.cur.Data)
		p.advance()
		if p.cur.Kind != lexer.COMMA {
			break
		}
		p.advance()
		if p.cur.Kind != lexer.NAME {
			return nil, errs.New("E1052", "expected error type name after comma in on=", errPos(p.cur.Pos), "")
		}
	}
	if len(types) == 0 {
		return nil, errs.New("E1052", "on= requires at least one error type name", errPos(p.cur.Pos), "")
	}
	return types, nil
}

func (p *Parser) parseImport() (ast.Statement, error) {
	pos := astPos(p.cur.Pos)
	p.advance()
	if p.cur.Kind != lexer.STR {
		return nil, errs.New("E1041", "expected import path as string", errPos(p.cur.Pos), "")
	}
	path := p.cur.Data
	p.advance()
	var alias string
	if p.cur.Kind == lexer.AS {
		p.advance()
		if p.cur.Kind != lexer.NAME {
			return nil, errs.New("E1042", "expected alias name", errPos(p.cur.Pos), "")
		}
		alias = p.cur.Data
		p.advance()
	}
	return &ast.ImportDecl{NodePos: pos, Path: path, Alias: alias}, nil
}
func (p *Parser) parseAssignOrExpr() (ast.Statement, error) {
	left, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if p.cur.Kind == lexer.EQ || p.cur.Kind == lexer.COLON {
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
