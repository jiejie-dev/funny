package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jiejie-dev/funny/v2/internal/docgen"
)

// Doc generates API documentation from ## doc comments.
func Doc(path, format, outDir string, includeTests bool) error {
	if path == "" {
		path = "."
	}
	docs, err := docgen.GenerateAll(path, includeTests)
	if err != nil {
		return err
	}
	if len(docs) == 0 {
		return fmt.Errorf("no .fn files found under %s", path)
	}
	switch format {
	case "json":
		return writeJSONDocs(docs, outDir)
	default:
		return writeMarkdownDocs(docs, outDir)
	}
}

func writeMarkdownDocs(docs []*docgen.ModuleDoc, outDir string) error {
	if outDir == "" {
		for _, doc := range docs {
			fmt.Print(docgen.RenderMarkdown(doc))
			if len(docs) > 1 {
				fmt.Println("---")
			}
		}
		return nil
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	for _, doc := range docs {
		base := strings.TrimSuffix(filepath.Base(doc.File), ".fn") + ".md"
		path := filepath.Join(outDir, base)
		if err := os.WriteFile(path, []byte(docgen.RenderMarkdown(doc)), 0o644); err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "wrote %s\n", path)
	}
	return nil
}

func writeJSONDocs(docs []*docgen.ModuleDoc, outDir string) error {
	if outDir == "" {
		for i, doc := range docs {
			data, err := docgen.RenderJSON(doc)
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			if i < len(docs)-1 {
				fmt.Println("---")
			}
		}
		return nil
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	for _, doc := range docs {
		data, err := docgen.RenderJSON(doc)
		if err != nil {
			return err
		}
		base := strings.TrimSuffix(filepath.Base(doc.File), ".fn") + ".json"
		path := filepath.Join(outDir, base)
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "wrote %s\n", path)
	}
	return nil
}
