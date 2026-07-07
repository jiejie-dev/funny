package lsp

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jiejie-dev/funny/v2/internal/docgen"
)

func formatSymbolDoc(sym docgen.SymbolDoc, kindLabel string) string {
	var sb strings.Builder
	if sym.Summary != "" {
		sb.WriteString(sym.Summary)
		sb.WriteString("\n\n")
	}
	for _, line := range sym.Body {
		sb.WriteString(line)
		sb.WriteString("\n\n")
	}
	sb.WriteString("```funny\n")
	sb.WriteString(sym.Signature)
	sb.WriteString("\n```\n")
	sb.WriteString(kindLabel)
	if len(sym.Args) > 0 {
		sb.WriteString("\n\n**Arguments**\n\n")
		names := make([]string, 0, len(sym.Args))
		for n := range sym.Args {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			fmt.Fprintf(&sb, "- `%s`: %s\n", n, sym.Args[n])
		}
	}
	if sym.Returns != "" {
		fmt.Fprintf(&sb, "\n**Returns:** %s", sym.Returns)
	}
	return sb.String()
}
