package docgen

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jiejie-dev/funny/v2/internal/ast"
	"github.com/jiejie-dev/funny/v2/internal/module"
	"github.com/jiejie-dev/funny/v2/internal/parser"
	"github.com/jiejie-dev/funny/v2/internal/types"
)

// SymbolDoc describes one documented declaration.
type SymbolDoc struct {
	Name      string            `json:"name"`
	Kind      string            `json:"kind"` // fn, struct
	Public    bool              `json:"public"`
	Summary   string            `json:"summary,omitempty"`
	Body      []string          `json:"body,omitempty"`
	Args      map[string]string `json:"args,omitempty"`
	Returns   string            `json:"returns,omitempty"`
	Signature string            `json:"signature"`
	File      string            `json:"file"`
	Line      int               `json:"line"`
}

// ModuleDoc is extracted documentation for one source file.
type ModuleDoc struct {
	File    string            `json:"file"`
	Meta    map[string]string `json:"meta,omitempty"`
	Symbols []SymbolDoc       `json:"symbols"`
}

// Extract parses and type-checks source, returning structured docs.
func Extract(src []byte, file string) (*ModuleDoc, error) {
	p := parser.New(string(src), file)
	prog, err := p.Parse()
	if err != nil {
		return nil, err
	}
	prog, err = module.Resolve(prog, file)
	if err != nil {
		return nil, err
	}
	env := types.NewEnv(nil)
	if err := types.Check(prog, env); err != nil {
		return nil, err
	}
	doc := &ModuleDoc{File: file}
	for _, s := range prog.Stmts {
		if meta, ok := s.(*ast.MetaBlock); ok {
			doc.Meta = meta.Fields
			break
		}
	}
	doc.Symbols = CollectSymbols(prog, env)
	return doc, nil
}

func fnSymbol(fn *ast.FnDecl, docLines []string, env *types.Env) SymbolDoc {
	sig := fnSignature(fn)
	sym := SymbolDoc{
		Name:      fn.Name,
		Kind:      "fn",
		Public:    fn.Pub,
		Signature: sig,
		File:      fn.NodePos.File,
		Line:      fn.NodePos.Line + 1,
	}
	parseDocLines(&sym, docLines)
	if fn.RetType != "" {
		if sym.Returns == "" {
			sym.Returns = fn.RetType
		}
	} else if f, ok := env.LookupFunc(fn.Name); ok && f.Return != nil {
		if sym.Returns == "" {
			sym.Returns = f.Return.String()
		}
	}
	return sym
}

func structSymbol(sd *ast.StructDecl, docLines []string) SymbolDoc {
	var fields []string
	for _, f := range sd.Fields {
		prefix := ""
		if f.Mut {
			prefix = "mut "
		}
		fields = append(fields, fmt.Sprintf("    %s%s: %s", prefix, f.Name, f.TypeAnn))
	}
	sig := fmt.Sprintf("struct %s:\n%s", sd.Name, strings.Join(fields, "\n"))
	sym := SymbolDoc{
		Name:      sd.Name,
		Kind:      "struct",
		Public:    sd.Pub,
		Signature: sig,
		File:      sd.NodePos.File,
		Line:      sd.NodePos.Line + 1,
	}
	parseDocLines(&sym, docLines)
	return sym
}

func fnSignature(fn *ast.FnDecl) string {
	parts := make([]string, len(fn.Params))
	for i, p := range fn.Params {
		if p.TypeAnn != "" {
			parts[i] = fmt.Sprintf("%s: %s", p.Name, p.TypeAnn)
		} else {
			parts[i] = p.Name
		}
	}
	ret := ""
	if fn.RetType != "" {
		ret = " -> " + fn.RetType
	}
	prefix := "fn "
	if fn.Pub {
		prefix = "pub fn "
	}
	return fmt.Sprintf("%s%s(%s)%s", prefix, fn.Name, strings.Join(parts, ", "), ret)
}

func parseDocLines(sym *SymbolDoc, lines []string) {
	if len(lines) == 0 {
		return
	}
	sym.Summary = lines[0]
	var body []string
	inArgs := false
	for _, line := range lines[1:] {
		lower := strings.ToLower(strings.TrimSpace(line))
		switch lower {
		case "args:", "arguments:":
			inArgs = true
			if sym.Args == nil {
				sym.Args = map[string]string{}
			}
			continue
		case "returns:", "return:":
			inArgs = false
			continue
		}
		if strings.HasPrefix(lower, "returns:") || strings.HasPrefix(lower, "return:") {
			inArgs = false
			sym.Returns = strings.TrimSpace(line[strings.Index(line, ":")+1:])
			continue
		}
		if inArgs {
			if sym.Args == nil {
				sym.Args = map[string]string{}
			}
			name, desc, ok := strings.Cut(line, ":")
			if ok {
				sym.Args[strings.TrimSpace(name)] = strings.TrimSpace(desc)
			} else {
				body = append(body, line)
			}
			continue
		}
		body = append(body, line)
	}
	sym.Body = body
}

// RenderMarkdown renders one module as Markdown API reference.
func RenderMarkdown(doc *ModuleDoc) string {
	var sb strings.Builder
	title := filepath.Base(doc.File)
	if doc.Meta != nil {
		if name, ok := doc.Meta["name"]; ok && name != "" {
			title = strings.Trim(name, `"`)
		}
	}
	sb.WriteString("# " + title + "\n\n")
	if doc.Meta != nil {
		sb.WriteString("## Metadata\n\n")
		keys := make([]string, 0, len(doc.Meta))
		for k := range doc.Meta {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			sb.WriteString(fmt.Sprintf("- **%s**: %s\n", k, doc.Meta[k]))
		}
		sb.WriteString("\n")
	}
	if len(doc.Symbols) == 0 {
		sb.WriteString("_No documented symbols._\n")
		return sb.String()
	}
	sb.WriteString("## Symbols\n\n")
	for _, sym := range doc.Symbols {
		vis := ""
		if sym.Public {
			vis = " (public)"
		}
		sb.WriteString(fmt.Sprintf("### `%s` — %s%s\n\n", sym.Name, sym.Kind, vis))
		if sym.Summary != "" {
			sb.WriteString(sym.Summary + "\n\n")
		}
		for _, line := range sym.Body {
			sb.WriteString(line + "\n\n")
		}
		sb.WriteString("```funny\n" + sym.Signature + "\n```\n\n")
		if len(sym.Args) > 0 {
			sb.WriteString("**Arguments**\n\n")
			names := make([]string, 0, len(sym.Args))
			for n := range sym.Args {
				names = append(names, n)
			}
			sort.Strings(names)
			for _, n := range names {
				sb.WriteString(fmt.Sprintf("- `%s`: %s\n", n, sym.Args[n]))
			}
			sb.WriteString("\n")
		}
		if sym.Returns != "" {
			sb.WriteString(fmt.Sprintf("**Returns:** %s\n\n", sym.Returns))
		}
	}
	return sb.String()
}

// RenderJSON returns pretty-printed JSON for one module doc.
func RenderJSON(doc *ModuleDoc) ([]byte, error) {
	return json.MarshalIndent(doc, "", "  ")
}

// DiscoverFiles lists .fn files under path.
func DiscoverFiles(path string, includeTests bool) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		if strings.HasSuffix(path, ".fn") {
			return []string{path}, nil
		}
		return nil, fmt.Errorf("%s: not a .fn file", path)
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			sub, err := DiscoverFiles(filepath.Join(path, e.Name()), includeTests)
			if err != nil {
				return nil, err
			}
			files = append(files, sub...)
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".fn") {
			continue
		}
		if !includeTests && strings.HasSuffix(name, "_test.fn") {
			continue
		}
		files = append(files, filepath.Join(path, name))
	}
	sort.Strings(files)
	return files, nil
}

// GenerateAll extracts docs for every discovered file.
func GenerateAll(path string, includeTests bool) ([]*ModuleDoc, error) {
	files, err := DiscoverFiles(path, includeTests)
	if err != nil {
		return nil, err
	}
	var docs []*ModuleDoc
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		doc, err := Extract(data, file)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		docs = append(docs, doc)
	}
	return docs, nil
}
