package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jiejie-dev/funny/v2/internal/cli"
	"github.com/jiejie-dev/funny/v2/internal/lsp"
	"github.com/jiejie-dev/funny/v2/internal/mcp"
	"github.com/jiejie-dev/funny/v2/internal/repl"
)

// version is a fallback for non-release builds; `go build`/`go run` don't
// set it. Release builds should override it with
// `-ldflags "-X main.version=2.1.0"` so `funny --version` matches the tag
// actually released, instead of drifting from CHANGELOG.md/RELEASE_NOTES.md
// like the old hardcoded "0.1.0" did.
var version = "2.2.2"

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

var pkgCmd = &cobra.Command{
	Use:   "pkg",
	Short: "Manage project dependencies (funny.pkg / funny.lock)",
}

var pkgInstallCmd = &cobra.Command{
	Use:   "install [name...]",
	Short: "Install dependencies from funny.pkg into .funny/packages/",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, _ := cmd.Flags().GetString("project")
		if dir == "" {
			dir = "."
		}
		if err := cli.PkgInstall(dir, args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return nil
	},
}

var pkgListCmd = &cobra.Command{
	Use:   "list",
	Short: "List packages recorded in funny.lock",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, _ := cmd.Flags().GetString("project")
		if dir == "" {
			dir = "."
		}
		return cli.PkgList(dir)
	},
}

var benchCmd = &cobra.Command{
	Use:   "bench",
	Short: "Run benchmarks (VM perf, AI friendliness)",
}

var benchAICmd = &cobra.Command{
	Use:   "ai",
	Short: "Run AI-friendliness benchmark (50 compile_ok/compile_err tasks)",
	RunE: func(cmd *cobra.Command, args []string) error {
		tasks, _ := cmd.Flags().GetString("tasks")
		provider, _ := cmd.Flags().GetString("provider")
		model, _ := cmd.Flags().GetString("model")
		mock, _ := cmd.Flags().GetBool("mock")
		return cli.BenchAI(cli.BenchAIOptions{
			TasksPath: tasks,
			Provider:  provider,
			Model:     model,
			Mock:      mock,
		})
	},
}

var replCmd = &cobra.Command{
	Use:   "repl",
	Short: "Start an interactive REPL (read-eval-print loop)",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, _ := cmd.Flags().GetString("project")
		if dir == "" {
			dir = "."
		}
		lessonsDir, _ := cmd.Flags().GetString("lessons-dir")
		lesson, _ := cmd.Flags().GetInt("lesson")
		return repl.RunWithOptions(repl.Options{
			WorkDir:       dir,
			LessonsDir:    lessonsDir,
			StartLesson:   lesson,
		}, os.Stdin, os.Stdout)
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
	pkgCmd.PersistentFlags().String("project", ".", "project root containing funny.pkg")
	pkgCmd.AddCommand(pkgInstallCmd, pkgListCmd)
	replCmd.Flags().String("project", ".", "working directory for imports and pkg: resolution")
	replCmd.Flags().String("lessons-dir", "", "directory with tutorial-*.funny lessons (default: docs/)")
	replCmd.Flags().Int("lesson", 0, "start guided tutorial N (1-based)")
	benchAICmd.Flags().String("tasks", "", "path to tasks.json (default: internal/benchmark/tasks.json)")
	benchAICmd.Flags().String("provider", "mock", "LLM provider: mock, openai, anthropic")
	benchAICmd.Flags().String("model", "", "model override (provider default if empty)")
	benchAICmd.Flags().Bool("mock", false, "use mock provider (echo prompt); same as --provider mock")
	benchCmd.AddCommand(benchAICmd)
	rootCmd.AddCommand(runCmd, astCmd, fmtCmd, describeCmd, disasmCmd, debugCmd, pkgCmd, replCmd, benchCmd, lspCmd, mcpCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
