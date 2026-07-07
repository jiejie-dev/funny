package repl

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jiejie-dev/funny/v2/internal/ast"
	"github.com/jiejie-dev/funny/v2/internal/evaluator"
	"github.com/jiejie-dev/funny/v2/internal/module"
	"github.com/jiejie-dev/funny/v2/internal/parser"
	"github.com/jiejie-dev/funny/v2/internal/types"
)

const replFile = "<repl>"

// Session is a persistent REPL state (types + runtime bindings).
type Session struct {
	eval     *evaluator.Evaluator
	env      *types.Env
	replPath string
}

// NewSession creates a REPL session rooted at workDir (for import/pkg resolution).
func NewSession(workDir string) (*Session, error) {
	abs, err := filepath.Abs(workDir)
	if err != nil {
		return nil, err
	}
	return &Session{
		eval:     evaluator.New(nil),
		env:      types.NewEnv(nil),
		replPath: filepath.Join(abs, replFile),
	}, nil
}

// Reset clears runtime and type bindings.
func (s *Session) Reset() {
	s.eval = evaluator.New(nil)
	s.env = types.NewEnv(nil)
}

// EvalCell parses, type-checks, and runs one REPL cell.
func (s *Session) EvalCell(src string) (result string, showed bool, err error) {
	complete, parseErr := InputStatus(src)
	if !complete {
		return "", false, nil
	}
	if parseErr != nil {
		return "", false, parseErr
	}
	p := parser.New(src, s.replPath)
	prog, err := p.Parse()
	if err != nil {
		return "", false, err
	}
	prog, err = module.Resolve(prog, s.replPath)
	if err != nil {
		return "", false, err
	}
	if err := types.Check(prog, s.env); err != nil {
		return "", false, err
	}
	v, show, err := s.eval.ExecCell(prog)
	if err != nil {
		return "", false, err
	}
	if show {
		return FormatValue(v), true, nil
	}
	return "", false, nil
}

// ListVars returns visible binding lines sorted by name.
func (s *Session) ListVars() []string {
	bindings := s.eval.Scope().Bindings()
	names := make([]string, 0, len(bindings))
	for name := range bindings {
		names = append(names, name)
	}
	sort.Strings(names)
	lines := make([]string, 0, len(names))
	for _, name := range names {
		lines = append(lines, fmt.Sprintf("%s = %s", name, formatBinding(bindings[name])))
	}
	return lines
}

func formatBinding(v any) string {
	switch x := v.(type) {
	case *ast.FnDecl:
		return fmt.Sprintf("fn %s(...)", x.Name)
	case *ast.StructDecl:
		return fmt.Sprintf("struct %s", x.Name)
	default:
		return FormatValue(v)
	}
}

// Run starts the interactive REPL on in/out until :quit or EOF.
func Run(workDir string, in io.Reader, out io.Writer) error {
	if in == nil {
		in = os.Stdin
	}
	if out == nil {
		out = os.Stdout
	}
	sess, err := NewSession(workDir)
	if err != nil {
		return err
	}
	fmt.Fprintln(out, "Funny REPL (v2.2) — type :help for commands, :quit to exit")
	scanner := bufio.NewScanner(in)
	var buffer []string
	prompt := "funny> "
	for {
		fmt.Fprint(out, prompt)
		if !scanner.Scan() {
			fmt.Fprintln(out)
			return nil
		}
		line := scanner.Text()
		if len(buffer) == 0 {
			if cmd, ok := handleMetaCommand(line, sess, out); ok {
				if cmd == "quit" {
					return nil
				}
				continue
			}
		}
		buffer = append(buffer, line)
		src := strings.Join(buffer, "\n")
		complete, parseErr := InputStatus(src)
		if !complete {
			prompt = "... "
			continue
		}
		buffer = nil
		prompt = "funny> "
		if strings.TrimSpace(src) == "" {
			continue
		}
		if parseErr != nil {
			fmt.Fprintln(out, parseErr)
			continue
		}
		result, showed, err := sess.EvalCell(src)
		if err != nil {
			fmt.Fprintln(out, err)
			continue
		}
		if showed {
			fmt.Fprintln(out, result)
		}
	}
}

func handleMetaCommand(line string, sess *Session, out io.Writer) (action string, handled bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return "", false
	}
	if trimmed[0] != ':' {
		return "", false
	}
	parts := strings.Fields(trimmed[1:])
	if len(parts) == 0 {
		return "", true
	}
	switch parts[0] {
	case "help", "h", "?":
		printHelp(out)
	case "quit", "q", "exit":
		return "quit", true
	case "reset", "clear":
		sess.Reset()
		fmt.Fprintln(out, "(session reset)")
	case "vars", "v":
		lines := sess.ListVars()
		if len(lines) == 0 {
			fmt.Fprintln(out, "(no bindings)")
		} else {
			for _, l := range lines {
				fmt.Fprintln(out, l)
			}
		}
	default:
		fmt.Fprintf(out, "unknown command %q (try :help)\n", parts[0])
	}
	return "", true
}

func printHelp(out io.Writer) {
	fmt.Fprintln(out, `REPL commands:
  :help (:h)     Show this help
  :quit (:q)     Exit the REPL
  :vars (:v)     List current bindings
  :reset         Clear session state

Enter Funny statements or expressions. Blocks continue on ... prompt.`)
}
