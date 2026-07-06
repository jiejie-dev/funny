package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jiejie-dev/funny/v2/internal/cli"
	"github.com/jiejie-dev/funny/v2/internal/lsp"
	"github.com/jiejie-dev/funny/v2/internal/mcp"
)

// version is a fallback for non-release builds; `go build`/`go run` don't
// set it. Release builds should override it with
// `-ldflags "-X main.version=2.1.0"` so `funny --version` matches the tag
// actually released, instead of drifting from CHANGELOG.md/RELEASE_NOTES.md
// like the old hardcoded "0.1.0" did.
var version = "2.1.6"

var rootCmd = &cobra.Command{
	Use:     "funny",
	Short:   "funny v2 - AI-native scripting language",
	Version: version,
}

var runCmd = &cobra.Command{
	Use:   "run <script>",
	Short: "Execute a funny script",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := os.ReadFile(args[0])
		if err != nil {
			return err
		}
		if err := cli.Run(data, args[0]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return nil
	},
}

var astCmd = &cobra.Command{
	Use:   "ast <script>",
	Short: "Print JSON AST",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := os.ReadFile(args[0])
		if err != nil {
			return err
		}
		out, err := cli.Ast(data, args[0])
		if err != nil {
			return err
		}
		fmt.Println(string(out))
		return nil
	},
}

var fmtCmd = &cobra.Command{
	Use:   "fmt <script>",
	Short: "Format a funny script",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := os.ReadFile(args[0])
		if err != nil {
			return err
		}
		out, err := cli.Format(data, args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		write, _ := cmd.Flags().GetBool("write")
		if write {
			return os.WriteFile(args[0], []byte(out), 0o644)
		}
		fmt.Print(out)
		return nil
	},
}

var describeCmd = &cobra.Command{
	Use:   "describe <script>",
	Short: "Print JSON plan/metadata",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := os.ReadFile(args[0])
		if err != nil {
			return err
		}
		out, err := cli.Describe(data, args[0])
		if err != nil {
			return err
		}
		fmt.Println(string(out))
		return nil
	},
}

var debugCmd = &cobra.Command{
	Use:   "debug <script>",
	Short: "Debug a script (source map, breakpoints, single-step)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := os.ReadFile(args[0])
		if err != nil {
			return err
		}
		sourceMap, _ := cmd.Flags().GetBool("source-map")
		if sourceMap {
			out, err := cli.SourceMap(data, args[0])
			if err != nil {
				return err
			}
			fmt.Println(string(out))
			return nil
		}
		breaks, _ := cmd.Flags().GetStringArray("break")
		if err := cli.Debug(data, args[0], cli.DebugOptions{Breakpoints: breaks}, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return nil
	},
}

var disasmCmd = &cobra.Command{
	Use:   "disasm <script>",
	Short: "Print bytecode disassembly",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := os.ReadFile(args[0])
		if err != nil {
			return err
		}
		out, err := cli.Disasm(data, args[0])
		if err != nil {
			return err
		}
		fmt.Print(out)
		return nil
	},
}

var lspCmd = &cobra.Command{
	Use:   "lsp",
	Short: "Start the LSP server over stdio (for editors/IDEs)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return lsp.Run(context.Background())
	},
}

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start the MCP server over stdio (for LLM clients)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return mcp.Run(context.Background())
	},
}

func init() {
	fmtCmd.Flags().BoolP("write", "w", false, "write result to the source file instead of stdout")
	debugCmd.Flags().Bool("source-map", false, "emit JSON source map and exit")
	debugCmd.Flags().StringArrayP("break", "b", nil, "breakpoint at line or file:line (repeatable)")
	rootCmd.AddCommand(runCmd, astCmd, fmtCmd, describeCmd, disasmCmd, debugCmd, lspCmd, mcpCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
