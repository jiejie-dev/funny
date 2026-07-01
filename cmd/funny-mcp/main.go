// cmd/funny-mcp/main.go
package main

import (
	"context"
	"os"

	"github.com/jiejie-dev/funny/internal/mcp"
)

func main() {
	if err := mcp.Run(context.Background()); err != nil {
		os.Stderr.WriteString("mcp server error: " + err.Error() + "\n")
		os.Exit(1)
	}
}
