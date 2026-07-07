package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/jiejie-dev/funny/v2/internal/bytecode"
	"github.com/jiejie-dev/funny/v2/internal/compiler"
	"github.com/jiejie-dev/funny/v2/internal/module"
	"github.com/jiejie-dev/funny/v2/internal/parser"
	"github.com/jiejie-dev/funny/v2/internal/types"
	"github.com/jiejie-dev/funny/v2/internal/vm"
)

// SourceMap compiles src and returns JSON source map bytes.
func SourceMap(src []byte, file string) ([]byte, error) {
	mod, err := CompileModule(src, file)
	if err != nil {
		return nil, err
	}
	return mod.SourceMapJSON()
}

// DebugOptions configures the interactive bytecode debugger.
type DebugOptions struct {
	Breakpoints []string // "line" or "file:line" (1-based lines)
}

// Debug runs script under the bytecode debugger with an interactive REPL on stdin.
func Debug(src []byte, file string, opts DebugOptions, in io.Reader, out io.Writer) error {
	mod, err := CompileModule(src, file)
	if err != nil {
		return err
	}
	if in == nil {
		in = os.Stdin
	}
	if out == nil {
		out = os.Stdout
	}

	var dbg *vm.Debugger
	dbg = vm.NewDebugger(func(ev vm.DebugEvent) (vm.DebugAction, error) {
		printDebugEvent(out, ev)
		return readDebugCommand(in, out, dbg, file, &ev)
	})
	for _, bp := range opts.Breakpoints {
		bpFile, line, err := parseBreakpoint(bp, file)
		if err != nil {
			return err
		}
		dbg.SetBreakpoint(bpFile, line)
	}

	m := vm.New(mod)
	_, err = m.RunDebug(dbg)
	if err != nil && strings.Contains(err.Error(), "debug: stopped") {
		fmt.Fprintln(out, "(debug session ended)")
		return nil
	}
	return err
}

// CompileModule parses, type-checks, and compiles source for debug/disasm.
func CompileModule(src []byte, file string) (*bytecode.Module, error) {
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
	return compiler.Compile(prog, file)
}

func parseBreakpoint(spec, defaultFile string) (file string, line1 int, err error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return "", 0, fmt.Errorf("empty breakpoint")
	}
	if idx := strings.LastIndex(spec, ":"); idx > 0 {
		file = spec[:idx]
		line1, err = strconv.Atoi(spec[idx+1:])
	} else {
		file = defaultFile
		line1, err = strconv.Atoi(spec)
	}
	if err != nil || line1 < 1 {
		return "", 0, fmt.Errorf("invalid breakpoint %q (use line or file:line)", spec)
	}
	return file, line1, nil
}

func printDebugEvent(out io.Writer, ev vm.DebugEvent) {
	loc := ev.Location.Display()
	fmt.Fprintf(out, "\n[%s] %s %s\n", loc, ev.FnName, ev.Instruction.String())
}

func readDebugCommand(in io.Reader, out io.Writer, dbg *vm.Debugger, defaultFile string, ev *vm.DebugEvent) (vm.DebugAction, error) {
	fmt.Fprint(out, "(dbg) ")
	scanner := bufio.NewScanner(in)
	if !scanner.Scan() {
		return vm.ActionQuit, nil
	}
	line := strings.TrimSpace(scanner.Text())
	if line == "" {
		return vm.ActionStep, nil
	}
	parts := strings.Fields(line)
	switch parts[0] {
	case "s", "step", "n", "next":
		return vm.ActionStep, nil
	case "c", "cont", "continue":
		return vm.ActionContinue, nil
	case "q", "quit", "exit":
		return vm.ActionQuit, nil
	case "l", "locals":
		PrintDebugLocals(out, *ev)
		return readDebugCommand(in, out, dbg, defaultFile, ev)
	case "p", "stack":
		PrintDebugStack(out, *ev)
		return readDebugCommand(in, out, dbg, defaultFile, ev)
	case "w", "where":
		fmt.Fprintf(out, "  at %s in %s ip=%d\n", ev.Location.Display(), ev.FnName, ev.IP)
		return readDebugCommand(in, out, dbg, defaultFile, ev)
	case "b", "break":
		if len(parts) < 2 {
			fmt.Fprintln(out, "usage: break <line> or break <file:line>")
			return readDebugCommand(in, out, dbg, defaultFile, ev)
		}
		bpFile, line1, err := parseBreakpoint(parts[1], defaultFile)
		if err != nil {
			fmt.Fprintln(out, err.Error())
			return readDebugCommand(in, out, dbg, defaultFile, ev)
		}
		dbg.SetBreakpoint(bpFile, line1)
		fmt.Fprintf(out, "breakpoint set at %s:%d\n", bpFile, line1)
		return readDebugCommand(in, out, dbg, defaultFile, ev)
	case "h", "help", "?":
		printDebugHelp(out)
		return readDebugCommand(in, out, dbg, defaultFile, ev)
	default:
		fmt.Fprintf(out, "unknown command %q (try help)\n", parts[0])
		return readDebugCommand(in, out, dbg, defaultFile, ev)
	}
}

func printDebugHelp(out io.Writer) {
	fmt.Fprintln(out, `Debugger commands:
  step (s)       Execute one instruction, then pause
  continue (c)   Run until the next breakpoint
  break (b) N    Set breakpoint at line N (or file:line)
  locals (l)     Show local variables
  stack (p)      Show operand stack
  where (w)      Show current source location
  quit (q)       Stop debugging
  help (h)       Show this help`)
}

// PrintDebugLocals writes locals from the last event (helper for extensions).
func PrintDebugLocals(out io.Writer, ev vm.DebugEvent) {
	for _, lv := range ev.Locals {
		if lv.Value == nil && strings.HasPrefix(lv.Name, "$") {
			continue
		}
		fmt.Fprintf(out, "  %s = %s\n", lv.Name, vm.FormatValue(lv.Value))
	}
}

// PrintDebugStack writes the operand stack.
func PrintDebugStack(out io.Writer, ev vm.DebugEvent) {
	for i, val := range ev.Stack {
		fmt.Fprintf(out, "  [%d] %s\n", i, vm.FormatValue(val))
	}
}
