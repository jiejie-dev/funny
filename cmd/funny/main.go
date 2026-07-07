package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jiejie-dev/funny/v2/internal/cli"
	"github.com/jiejie-dev/funny/v2/internal/dap"
	"github.com/jiejie-dev/funny/v2/internal/lsp"
	"github.com/jiejie-dev/funny/v2/internal/mcp"
	"github.com/jiejie-dev/funny/v2/internal/repl"
)

// version is a fallback for non-release builds; `go build`/`go run` don't
// set it. Release builds should override it with
// `-ldflags "-X main.version=2.1.0"` so `funny --version` matches the tag
// actually released, instead of drifting from CHANGELOG.md/RELEASE_NOTES.md
// like the old hardcoded "0.1.0" did.
var version = "2.4.2"

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

var pkgAddCmd = &cobra.Command{
	Use:   "add <name> [source]",
	Short: "Add a dependency to funny.pkg and install it",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, _ := cmd.Flags().GetString("project")
		if dir == "" {
			dir = "."
		}
		source, _ := cmd.Flags().GetString("source")
		if source == "" && len(args) > 1 {
			source = args[1]
		}
		version, _ := cmd.Flags().GetString("version")
		entry, _ := cmd.Flags().GetString("entry")
		if err := cli.PkgAdd(dir, args[0], cli.NormalizePkgSource(source), version, entry); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return nil
	},
}

var pkgUpdateCmd = &cobra.Command{
	Use:   "update [name...]",
	Short: "Re-fetch dependencies and refresh funny.lock checksums",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, _ := cmd.Flags().GetString("project")
		if dir == "" {
			dir = "."
		}
		if err := cli.PkgUpdate(dir, args); err != nil {
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

var dapCmd = &cobra.Command{
	Use:   "dap",
	Short: "Start the Debug Adapter Protocol server over stdio (for VS Code)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return dap.Run(os.Stdin, os.Stdout)
	},
}

var testCmd = &cobra.Command{
	Use:   "test [path]",
	Short: "Run test blocks in *_test.fn files",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := "."
		if len(args) > 0 {
			path = args[0]
		}
		verbose, _ := cmd.Flags().GetBool("verbose")
		jsonOut, _ := cmd.Flags().GetBool("json")
		if err := cli.Test(path, verbose, jsonOut); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return nil
	},
}

var docCmd = &cobra.Command{
	Use:   "doc [path]",
	Short: "Generate API docs from ## doc comments",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := "."
		if len(args) > 0 {
			path = args[0]
		}
		format, _ := cmd.Flags().GetString("format")
		outDir, _ := cmd.Flags().GetString("out")
		includeTests, _ := cmd.Flags().GetBool("include-tests")
		if err := cli.Doc(path, format, outDir, includeTests); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
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
	pkgCmd.PersistentFlags().String("project", ".", "project root containing funny.pkg")
	pkgCmd.AddCommand(pkgInstallCmd, pkgAddCmd, pkgUpdateCmd, pkgListCmd)
	pkgAddCmd.Flags().String("source", "", "package source (path:, https://, git+url@ref)")
	pkgAddCmd.Flags().String("version", "", "version constraint (1.2.3, >=1.0.0, ^1.2.0, *)")
	pkgAddCmd.Flags().String("entry", "", "entry .fn file (default: <name>.fn)")
	replCmd.Flags().String("project", ".", "working directory for imports and pkg: resolution")
	replCmd.Flags().String("lessons-dir", "", "directory with tutorial-*.funny lessons (default: docs/)")
	replCmd.Flags().Int("lesson", 0, "start guided tutorial N (1-based)")
	benchAICmd.Flags().String("tasks", "", "path to tasks.json (default: internal/benchmark/tasks.json)")
	benchAICmd.Flags().String("provider", "mock", "LLM provider: mock, openai, anthropic")
	benchAICmd.Flags().String("model", "", "model override (provider default if empty)")
	benchAICmd.Flags().Bool("mock", false, "use mock provider (echo prompt); same as --provider mock")
	benchCmd.AddCommand(benchAICmd)
	testCmd.Flags().BoolP("verbose", "v", false, "print each test as it runs")
	testCmd.Flags().Bool("json", false, "emit JSON report")
	docCmd.Flags().String("format", "markdown", "output format: markdown or json")
	docCmd.Flags().String("out", "", "write docs to directory (default: stdout)")
	docCmd.Flags().Bool("include-tests", false, "include *_test.fn files")
	rootCmd.AddCommand(runCmd, astCmd, fmtCmd, describeCmd, disasmCmd, debugCmd, pkgCmd, replCmd, benchCmd, testCmd, docCmd, dapCmd, lspCmd, mcpCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
