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

// Options configures an interactive REPL session.
type Options struct {
	WorkDir       string
	LessonsDir    string
	StartLesson   int // 1-based; 0 = none
}

// Session is a persistent REPL state (types + runtime bindings).
type Session struct {
	eval     *evaluator.Evaluator
	env      *types.Env
	workDir  string
	replPath string
	lessons  []Lesson
	lesson   *LessonProgress
	history  *History
}

// NewSession creates a REPL session rooted at workDir (for import/pkg resolution).
func NewSession(workDir string) (*Session, error) {
	return NewSessionWithOptions(Options{WorkDir: workDir})
}

// NewSessionWithOptions creates a session with lesson discovery and optional guided start.
func NewSessionWithOptions(opts Options) (*Session, error) {
	abs, err := filepath.Abs(opts.WorkDir)
	if err != nil {
		return nil, err
	}
	lessonsDir := opts.LessonsDir
	if lessonsDir == "" {
		lessonsDir = defaultLessonsDir(abs)
	}
	var lessons []Lesson
	if lessonsDir != "" {
		lessons, _ = DiscoverLessons(lessonsDir)
	}
	return &Session{
		eval:     evaluator.New(nil),
		env:      types.NewEnv(nil),
		workDir:  abs,
		replPath: filepath.Join(abs, replFile),
		lessons:  lessons,
		history:  NewHistory(100),
	}, nil
}

// Reset clears runtime and type bindings.
func (s *Session) Reset() {
	s.eval = evaluator.New(nil)
	s.env = types.NewEnv(nil)
	s.lesson = nil
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
	return RunWithOptions(Options{WorkDir: workDir}, in, out)
}

// RunWithOptions starts the REPL with lesson/tutorial support.
func RunWithOptions(opts Options, in io.Reader, out io.Writer) error {
	if in == nil {
		in = os.Stdin
	}
	if out == nil {
		out = os.Stdout
	}
	sess, err := NewSessionWithOptions(opts)
	if err != nil {
		return err
	}
	fmt.Fprintln(out, "Funny REPL (v2.2) — type :help for commands, :lessons for tutorials")
	if opts.StartLesson > 0 {
		if err := sess.startLesson(opts.StartLesson, out); err != nil {
			fmt.Fprintln(out, err)
		}
	}
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
		sess.history.Add(line)
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

func (s *Session) startLesson(number int, out io.Writer) error {
	if number < 1 || number > len(s.lessons) {
		return fmt.Errorf("lesson %d not found (try :lessons)", number)
	}
	lesson := s.lessons[number-1]
	s.lesson = &LessonProgress{Lesson: lesson, StepIndex: 0}
	fmt.Fprintf(out, "\n=== %s ===\n", lesson.Title)
	if lesson.Summary != "" {
		fmt.Fprintln(out, lesson.Summary)
	}
	fmt.Fprintf(out, "%d steps — :step run demo, :hint show hint, :show reveal code, :skip advance\n\n",
		len(lesson.Steps))
	s.printLessonStep(out)
	return nil
}

func (s *Session) printLessonStep(out io.Writer) {
	if s.lesson == nil {
		return
	}
	step, ok := s.lesson.current()
	if !ok {
		fmt.Fprintln(out, "(lesson complete — try the next :lesson or experiment on your own)")
		s.lesson = nil
		return
	}
	fmt.Fprintf(out, "[step %d/%d]\n", s.lesson.StepIndex+1, len(s.lesson.Lesson.Steps))
	if step.Hint != "" {
		fmt.Fprintln(out, step.Hint)
	}
}

func (s *Session) runLessonStep(out io.Writer) error {
	if s.lesson == nil {
		return fmt.Errorf("no active lesson (try :lesson N)")
	}
	step, ok := s.lesson.current()
	if !ok {
		return fmt.Errorf("lesson already complete")
	}
	result, showed, err := s.EvalCell(step.Code)
	if err != nil {
		return err
	}
	if showed {
		fmt.Fprintln(out, result)
	}
	if s.lesson.advance() {
		s.printLessonStep(out)
	} else {
		fmt.Fprintln(out, "(lesson complete)")
		s.lesson = nil
	}
	return nil
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
	cmd := parts[0]
	args := parts[1:]
	switch cmd {
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
	case "lessons", "tutorials", "ls":
		if len(sess.lessons) == 0 {
			fmt.Fprintln(out, "(no tutorials found — set --lessons-dir or run from repo root)")
		} else {
			for _, l := range sess.lessons {
				fmt.Fprintf(out, "  %d  %s  (%d steps)\n", l.Number, l.Title, len(l.Steps))
			}
		}
	case "lesson", "tutorial":
		if len(args) == 0 {
			fmt.Fprintln(out, "usage: :lesson N")
		} else {
			var n int
			if _, err := fmt.Sscanf(args[0], "%d", &n); err != nil {
				fmt.Fprintln(out, "usage: :lesson N")
			} else if err := sess.startLesson(n, out); err != nil {
				fmt.Fprintln(out, err)
			}
		}
	case "step", "demo":
		if err := sess.runLessonStep(out); err != nil {
			fmt.Fprintln(out, err)
		}
	case "hint":
		if sess.lesson == nil {
			fmt.Fprintln(out, "no active lesson")
		} else if step, ok := sess.lesson.current(); ok && step.Hint != "" {
			fmt.Fprintln(out, step.Hint)
		} else {
			fmt.Fprintln(out, "(no hint for this step)")
		}
	case "show", "answer":
		if sess.lesson == nil {
			fmt.Fprintln(out, "no active lesson")
		} else if step, ok := sess.lesson.current(); ok {
			fmt.Fprintln(out, step.Code)
		} else {
			fmt.Fprintln(out, "(lesson complete)")
		}
	case "skip", "next":
		if sess.lesson == nil {
			fmt.Fprintln(out, "no active lesson")
		} else if sess.lesson.advance() {
			sess.printLessonStep(out)
		} else {
			fmt.Fprintln(out, "(lesson complete)")
			sess.lesson = nil
		}
	case "load":
		if len(args) == 0 {
			fmt.Fprintln(out, "usage: :load path.fn")
		} else if err := sess.LoadFile(args[0]); err != nil {
			fmt.Fprintln(out, err)
		} else {
			fmt.Fprintf(out, "(loaded %s)\n", args[0])
		}
	case "type":
		if len(args) == 0 {
			fmt.Fprintln(out, "usage: :type EXPR")
		} else {
			expr := strings.Join(args, " ")
			if typ, err := sess.TypeOfExpr(expr); err != nil {
				fmt.Fprintln(out, err)
			} else {
				fmt.Fprintln(out, typ)
			}
		}
	case "desc", "describe":
		if len(args) == 0 {
			fmt.Fprintln(out, "usage: :desc NAME")
		} else if desc, err := sess.DescribeName(args[0]); err != nil {
			fmt.Fprintln(out, err)
		} else {
			fmt.Fprintln(out, desc)
		}
	case "complete", "comp":
		prefix := ""
		if len(args) > 0 {
			prefix = args[len(args)-1]
		}
		comps := Completions(sess, prefix)
		if len(comps) == 0 {
			fmt.Fprintln(out, "(no completions)")
		} else {
			for _, c := range comps {
				fmt.Fprintln(out, c)
			}
		}
	case "history":
		lines := sess.history.Lines()
		if len(lines) == 0 {
			fmt.Fprintln(out, "(empty)")
		} else {
			for i, l := range lines {
				fmt.Fprintf(out, "%4d  %s\n", i+1, l)
			}
		}
	case "install":
		msg, err := sess.InstallPackages(args)
		if err != nil {
			fmt.Fprintln(out, err)
		} else if msg == "" {
			fmt.Fprintln(out, "(nothing to install)")
		} else {
			fmt.Fprintln(out, msg)
		}
	default:
		fmt.Fprintf(out, "unknown command %q (try :help)\n", cmd)
	}
	return "", true
}

func printHelp(out io.Writer) {
	fmt.Fprintln(out, `REPL commands:
  :help (:h)           Show this help
  :quit (:q)           Exit the REPL
  :vars (:v)           List current bindings
  :reset               Clear session state
  :history             Show recent inputs
  :type EXPR           Show expression type
  :desc NAME           Describe a binding
  :complete PREFIX     Suggest completions
  :load PATH           Load and run a .funny/.fn file
  :install [PKG...]    Run funny.pkg install
  :lessons             List interactive tutorials
  :lesson N            Start guided tutorial N
  :step                Run current tutorial step (demo)
  :hint                Show current step hint
  :show                Reveal current step code
  :skip                Skip to next tutorial step

Enter Funny statements or expressions. Blocks continue on ... prompt.`)
}
