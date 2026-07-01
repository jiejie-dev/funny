// cmd/funny-lsp/main.go
package main

import (
	"context"
	"os"

	"github.com/jiejie-dev/funny/internal/lsp"
)

func main() {
	if err := lsp.Run(context.Background()); err != nil {
		os.Stderr.WriteString("lsp server error: " + err.Error() + "\n")
		os.Exit(1)
	}
}
