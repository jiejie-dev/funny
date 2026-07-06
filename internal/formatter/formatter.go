// Package formatter implements a canonical AST pretty-printer for funny
// source code ("gofmt for funny"). It re-parses the source and re-emits it
// with consistent 4-space indentation and spacing, while preserving
// standalone and trailing `#`/`##` comments (see ast.CommentStmt).
//
// Known limitations (documented, not bugs):
//   - Blank-line runs are collapsed; exact blank-line counts are not preserved.
//   - Comments can only attach to statements, not to sub-expressions.
//   - map/struct-literal field order is not preserved (Go map has no order);
//     output is sorted by field name for deterministic, idempotent formatting.
package formatter

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jiejie-dev/funny/v2/internal/ast"
	"github.com/jiejie-dev/funny/v2/internal/parser"
)

const indentUnit = "    "

// Format parses src and re-emits it in canonical form. Parse errors are
// returned as-is (formatting requires syntactically valid input).
func Format(src []byte, file string) (string, error) {
	p := parser.New(string(src), file)
	prog, err := p.Parse()
	if err != nil {
		return "", err
	}
	pr := &printer{}
	pr.program(prog)
	return pr.String(), nil
}

type printer struct {
	b     strings.Builder
	depth int
}

func (p *printer) String() string { return p.b.String() }

func (p *printer) indent() string { return strings.Repeat(indentUnit, p.depth) }

func (p *printer) writeLine(s string) {
	p.b.WriteString(p.indent())
	p.b.WriteString(s)
	p.b.WriteByte('\n')
}

func (p *printer) program(prog *ast.Program) {
	p.stmts(prog.Stmts)
}

// stmts prints a statement list, inlining a CommentStmt onto the previous
// line when they share the same source line (a "trailing" comment).
func (p *printer) stmts(stmts []ast.Statement) {
	for i, s := range stmts {
		if c, ok := s.(*ast.CommentStmt); ok && i > 0 && c.Pos().Line == stmts[i-1].Pos().Line {
			out := p.b.String()
			out = strings.TrimSuffix(out, "\n")
			out += "  " + commentText(c) + "\n"
			p.b.Reset()
			p.b.WriteString(out)
			continue
		}
		p.stmt(s)
	}
}

func commentText(c *ast.CommentStmt) string {
	if c.Doc {
		return "##" + c.Text
	}
	return "#" + c.Text
}

func (p *printer) stmt(s ast.Statement) {
	switch n := s.(type) {
	case *ast.CommentStmt:
		p.writeLine(commentText(n))
	case *ast.ExprStmt:
		p.writeLine(p.expr(n.X))
	case *ast.LetStmt:
		if n.TypeAnn != "" {
			p.writeLine(fmt.Sprintf("let %s: %s = %s", n.Name, n.TypeAnn, p.expr(n.Value)))
		} else {
			p.writeLine(fmt.Sprintf("let %s = %s", n.Name, p.expr(n.Value)))
		}
	case *ast.AssignStmt:
		p.writeLine(fmt.Sprintf("%s = %s", p.expr(n.Target), p.expr(n.Value)))
	case *ast.ReturnStmt:
		if n.Value == nil {
			p.writeLine("return")
		} else {
			p.writeLine("return " + p.expr(n.Value))
		}
	case *ast.BreakStmt:
		p.writeLine("break")
	case *ast.ContinueStmt:
		p.writeLine("continue")
	case *ast.ImportDecl:
		if n.Alias != "" {
			p.writeLine(fmt.Sprintf("import %q as %s", n.Path, n.Alias))
		} else {
			p.writeLine(fmt.Sprintf("import %q", n.Path))
		}
	case *ast.IfStmt:
		p.ifStmt(n)
	case *ast.ForStmt:
		p.writeLine(fmt.Sprintf("for %s in %s:", n.Name, p.expr(n.Iterable)))
		p.block(n.Body)
	case *ast.WhileStmt:
		p.writeLine("while " + p.expr(n.Cond) + ":")
		p.block(n.Body)
	case *ast.MatchStmt:
		p.writeLine("match " + p.expr(n.Expr) + ":")
		p.depth++
		for _, arm := range n.Arms {
			p.writeLine(p.expr(arm.Pattern) + " =>")
			p.block(arm.Body)
		}
		p.depth--
	case *ast.FnDecl:
		p.fnDecl(n)
	case *ast.StructDecl:
		p.structDecl(n)
	case *ast.MetaBlock:
		p.metaBlock(n)
	case *ast.PlanBlock:
		p.writeLine(fmt.Sprintf("plan %q:", n.Name))
		p.block(n.Body)
	case *ast.Step:
		p.step(n)
	default:
		panic(fmt.Sprintf("formatter: unhandled statement %T", s))
	}
}

// ifStmt prints an if/elif/else chain. The parser hoists a trailing `else`
// block up to the outermost IfStmt (see parseIf), so ElseBlock must be read
// from n before descending into the ElseIf chain, not at each nested level.
func (p *printer) ifStmt(n *ast.IfStmt) {
	elseBlock := n.ElseBlock
	keyword := "if"
	for cur := n; ; cur = cur.ElseIf {
		p.writeLine(keyword + " " + p.expr(cur.Cond) + ":")
		p.block(cur.Then)
		if cur.ElseIf == nil {
			break
		}
		keyword = "elif"
	}
	if elseBlock != nil {
		p.writeLine("else:")
		p.block(elseBlock)
	}
}

func (p *printer) block(b *ast.Block) {
	p.depth++
	p.stmts(b.Statements)
	p.depth--
}

func (p *printer) fnDecl(n *ast.FnDecl) {
	parts := make([]string, len(n.Params))
	for i, param := range n.Params {
		parts[i] = param.String()
	}
	prefix := ""
	if n.Pub {
		prefix = "pub "
	}
	sig := fmt.Sprintf("%sfn %s(%s)", prefix, n.Name, strings.Join(parts, ", "))
	if n.RetType != "" {
		sig += " -> " + n.RetType
	}
	p.writeLine(sig + ":")
	p.block(n.Body)
}

func (p *printer) structDecl(n *ast.StructDecl) {
	prefix := ""
	if n.Pub {
		prefix = "pub "
	}
	p.writeLine(fmt.Sprintf("%sstruct %s:", prefix, n.Name))
	p.depth++
	for _, f := range n.Fields {
		p.writeLine(f.String())
	}
	p.depth--
}

// metaBlock prints `meta:` fields as `key = "value"`, matching the syntax
// parseMeta actually accepts (AssignStmt-based, not colon-based).
func (p *printer) metaBlock(n *ast.MetaBlock) {
	p.writeLine("meta:")
	p.depth++
	keys := make([]string, 0, len(n.Fields))
	for k := range n.Fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		p.writeLine(fmt.Sprintf("%s = %q", k, n.Fields[k]))
	}
	p.depth--
}

// step prints a `step "name" -> kind with ...:` header.
//
// Before this, `n.Retry.Backoff` and `n.Timeout` were silently dropped -
// `funny fmt` on `step "x" -> tool with retry max=2 backoff=exp:` (or any
// step with a `timeout="..."`) reprinted it as just `step "x" with retry
// max=2:`, permanently losing the backoff strategy, the timeout bound,
// and even the explicit `-> tool` step kind's co-occurrence with retry
// options the next time someone read the "formatted" file - a formatter
// changing program behavior, not just style, on round-trip.
func (p *printer) step(n *ast.Step) {
	head := fmt.Sprintf("step %q", n.Name)
	if n.Kind != "" && n.Kind != ast.StepTool {
		head += " -> " + string(n.Kind)
	}
	var with []string
	if n.Retry != nil {
		retry := fmt.Sprintf("retry max=%d", n.Retry.Max)
		if n.Retry.Backoff != "" {
			retry += " backoff=" + n.Retry.Backoff
		}
		with = append(with, retry)
	}
	if n.Timeout != "" {
		with = append(with, fmt.Sprintf("timeout=%q", n.Timeout))
	}
	if len(with) > 0 {
		head += " with " + strings.Join(with, " ")
	}
	p.writeLine(head + ":")
	if n.Body != nil {
		p.block(n.Body)
	}
}

func (p *printer) expr(e ast.Expression) string {
	switch n := e.(type) {
	case *ast.LiteralExpr:
		return n.String()
	case *ast.VariableExpr:
		return n.Name
	case *ast.BinaryExpr:
		return fmt.Sprintf("%s %s %s", p.expr(n.Left), n.Op, p.expr(n.Right))
	case *ast.UnaryExpr:
		if n.Op == "not" {
			return "not " + p.expr(n.Expr)
		}
		return n.Op + p.expr(n.Expr)
	case *ast.SubExpr:
		return "(" + p.expr(n.Inner) + ")"
	case *ast.ListExpr:
		parts := make([]string, len(n.Elements))
		for i, el := range n.Elements {
			parts[i] = p.expr(el)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case *ast.MapLiteralExpr:
		parts := make([]string, len(n.Keys))
		for i, k := range n.Keys {
			parts[i] = fmt.Sprintf("%s: %s", p.expr(k), p.expr(n.Values[i]))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	case *ast.IndexExpr:
		return fmt.Sprintf("%s[%s]", p.expr(n.Object), p.expr(n.Index))
	case *ast.FieldExpr:
		return fmt.Sprintf("%s.%s", p.expr(n.Object), n.Field)
	case *ast.CallExpr:
		parts := make([]string, len(n.Args))
		for i, a := range n.Args {
			parts[i] = p.expr(a)
		}
		return fmt.Sprintf("%s(%s)", p.expr(n.Func), strings.Join(parts, ", "))
	case *ast.StructLiteralExpr:
		names := make([]string, 0, len(n.Fields))
		for k := range n.Fields {
			names = append(names, k)
		}
		sort.Strings(names)
		parts := make([]string, len(names))
		for i, k := range names {
			parts[i] = fmt.Sprintf("%s: %s", k, p.expr(n.Fields[k]))
		}
		return fmt.Sprintf("%s(%s)", n.TypeName, strings.Join(parts, ", "))
	case *ast.FStringExpr:
		return p.fstring(n)
	case *ast.TryExpr:
		return p.expr(n.Inner) + "?"
	}
	panic(fmt.Sprintf("formatter: unhandled expression %T", e))
}

// fstring reconstructs canonical f-string source from parsed Parts (rather
// than dumping Raw verbatim), so re-formatting round-trips consistently
// through `{{`/`}}` escaping and format-spec normalization.
func (p *printer) fstring(n *ast.FStringExpr) string {
	var b strings.Builder
	b.WriteByte('f')
	b.WriteByte('"')
	for _, part := range n.Parts {
		if part.Expr == nil {
			b.WriteString(escapeFStringLiteral(part.Text))
			continue
		}
		b.WriteByte('{')
		b.WriteString(p.expr(part.Expr))
		if part.Spec != "" {
			b.WriteByte(':')
			b.WriteString(part.Spec)
		}
		b.WriteByte('}')
	}
	b.WriteByte('"')
	return b.String()
}

// escapeFStringLiteral re-escapes a literal f-string text segment so it
// round-trips back through the lexer/parser unchanged: backslash/quote/
// control-char escapes, plus `{`/`}` doubling to avoid being mistaken for
// interpolation delimiters.
func escapeFStringLiteral(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		case '\r':
			b.WriteString(`\r`)
		case '{':
			b.WriteString("{{")
		case '}':
			b.WriteString("}}")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
