package types

import (
	"fmt"
	"strings"
	"unicode"
)

// ParseType parses a type annotation string into a Type.
// Grammar:
//
//	type      := primary
//	primary   := 'list' '[' type ']'
//	           | 'map' '[' type ',' type ']'
//	           | 'Result' '[' type ',' type ']'
//	           | func-param-list '->' type
//	           | IDENT '?'?
//	func-param-list := '(' (type (',' type)*)? ')'
func ParseType(src string) (Type, error) {
	p := &typeParser{src: src}
	t, err := p.parseType()
	if err != nil {
		return nil, err
	}
	p.skipSpace()
	if p.pos < len(p.src) {
		return nil, fmt.Errorf("unexpected trailing characters at position %d in %q", p.pos, p.src)
	}
	return t, nil
}

type typeParser struct {
	src string
	pos int
}

func (p *typeParser) peek() byte {
	if p.pos >= len(p.src) {
		return 0
	}
	return p.src[p.pos]
}

func (p *typeParser) skipSpace() {
	for p.pos < len(p.src) && unicode.IsSpace(rune(p.src[p.pos])) {
		p.pos++
	}
}

func (p *typeParser) readIdent() string {
	start := p.pos
	for p.pos < len(p.src) && isIdentByte(p.src[p.pos]) {
		p.pos++
	}
	return p.src[start:p.pos]
}

func isIdentByte(b byte) bool {
	return b == '_' || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

func (p *typeParser) expect(ch byte) error {
	p.skipSpace()
	if p.pos >= len(p.src) || p.src[p.pos] != ch {
		return fmt.Errorf("expected %q at position %d in %q, got %q", ch, p.pos, p.src, string(p.peek()))
	}
	p.pos++
	return nil
}

func (p *typeParser) parseType() (Type, error) {
	p.skipSpace()
	if p.pos >= len(p.src) {
		return nil, fmt.Errorf("unexpected end of type annotation")
	}
	ch := p.src[p.pos]

	switch {
	case ch == '(':
		return p.parseFuncType()
	case strings.HasPrefix(p.src[p.pos:], "list["):
		return p.parseListType()
	case strings.HasPrefix(p.src[p.pos:], "map["):
		return p.parseMapType()
	case strings.HasPrefix(p.src[p.pos:], "Result["):
		return p.parseResultType()
	}

	return p.parseNamedType()
}

func (p *typeParser) parseNamedType() (Type, error) {
	p.skipSpace()
	ident := p.readIdent()
	if ident == "" {
		return nil, fmt.Errorf("expected type name at position %d", p.pos)
	}
	base := Primitive(ident)
	p.skipSpace()
	if p.pos < len(p.src) && p.src[p.pos] == '?' {
		p.pos++
		return Optional{Inner: base}, nil
	}
	return base, nil
}

func (p *typeParser) parseListType() (Type, error) {
	p.pos += len("list[")
	elem, err := p.parseType()
	if err != nil {
		return nil, err
	}
	if err := p.expect(']'); err != nil {
		return nil, fmt.Errorf("malformed list type: %w", err)
	}
	return List{Elem: elem}, nil
}

func (p *typeParser) parseMapType() (Type, error) {
	p.pos += len("map[")
	key, err := p.parseType()
	if err != nil {
		return nil, err
	}
	if err := p.expect(','); err != nil {
		return nil, fmt.Errorf("malformed map type (expected ','): %w", err)
	}
	val, err := p.parseType()
	if err != nil {
		return nil, err
	}
	if err := p.expect(']'); err != nil {
		return nil, fmt.Errorf("malformed map type: %w", err)
	}
	return Map{Key: key, Value: val}, nil
}

func (p *typeParser) parseResultType() (Type, error) {
	p.pos += len("Result[")
	ok, err := p.parseType()
	if err != nil {
		return nil, err
	}
	if err := p.expect(','); err != nil {
		return nil, fmt.Errorf("malformed Result type (expected ','): %w", err)
	}
	errT, err := p.parseType()
	if err != nil {
		return nil, err
	}
	if err := p.expect(']'); err != nil {
		return nil, fmt.Errorf("malformed Result type: %w", err)
	}
	return Result{Ok: ok, Err: errT}, nil
}

func (p *typeParser) parseFuncType() (Type, error) {
	if err := p.expect('('); err != nil {
		return nil, err
	}
	p.skipSpace()
	var params []Type
	if p.pos < len(p.src) && p.src[p.pos] != ')' {
		for {
			t, err := p.parseType()
			if err != nil {
				return nil, err
			}
			params = append(params, t)
			p.skipSpace()
			if p.pos < len(p.src) && p.src[p.pos] == ',' {
				p.pos++
				continue
			}
			break
		}
	}
	if err := p.expect(')'); err != nil {
		return nil, fmt.Errorf("malformed func type: %w", err)
	}
	if err := p.expect('-'); err != nil {
		return nil, err
	}
	if p.pos >= len(p.src) || p.src[p.pos] != '>' {
		return nil, fmt.Errorf("expected '->' in func type")
	}
	p.pos++
	ret, err := p.parseType()
	if err != nil {
		return nil, err
	}
	return Func{Params: params, Return: ret}, nil
}
