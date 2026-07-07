package dap

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/jiejie-dev/funny/v2/internal/cli"
	"github.com/jiejie-dev/funny/v2/internal/vm"
)

const (
	refLocals = 1000
	refStack  = 1001
	threadID  = 1
	frameID   = 1
)

// Server implements a minimal DAP adapter for Funny bytecode debugging.
type Server struct {
	reader *bufio.Reader
	out    io.Writer
	seq    int
	mu     sync.Mutex

	program     string
	breakpoints []sourceBreakpoint

	lastEvent *vm.DebugEvent
	actionCh  chan vm.DebugAction
}

type sourceBreakpoint struct {
	file string
	line int // 1-based
}

type launchArgs struct {
	Program string `json:"program"`
	Cwd     string `json:"cwd,omitempty"`
}

type initializeArgs struct {
	ClientID string `json:"clientID"`
}

type setBreakpointsArgs struct {
	Source      sourceRef `json:"source"`
	Breakpoints []struct {
		Line int `json:"line"`
	} `json:"breakpoints"`
}

type sourceRef struct {
	Path string `json:"path"`
}

// Run serves DAP over stdio until disconnect.
func Run(in io.Reader, out io.Writer) error {
	if in == nil {
		in = os.Stdin
	}
	if out == nil {
		out = os.Stdout
	}
	s := &Server{
		reader:   bufio.NewReader(in),
		out:      out,
		actionCh: make(chan vm.DebugAction, 1),
	}
	return s.loop()
}

func (s *Server) loop() error {
	for {
		msg, err := readMessage(s.reader)
		if err != nil {
			return err
		}
		switch msg.Type {
		case "request":
			if err := s.handleRequest(msg); err != nil {
				return err
			}
		default:
			// ignore
		}
	}
}

func (s *Server) handleRequest(req Message) error {
	switch req.Command {
	case "initialize":
		var args initializeArgs
		_ = decodeBody(req.Arguments, &args)
		if err := s.respond(req, true, map[string]any{
			"supportsConfigurationDoneRequest": true,
		}); err != nil {
			return err
		}
		return s.event("initialized", nil)
	case "launch":
		var args launchArgs
		if err := decodeBody(req.Arguments, &args); err != nil {
			return s.respond(req, false, nil, err.Error())
		}
		s.program = args.Program
		if args.Cwd != "" {
			if !filepath.IsAbs(s.program) {
				s.program = filepath.Join(args.Cwd, s.program)
			}
		}
		s.program = filepath.Clean(s.program)
		return s.respond(req, true, nil)
	case "setBreakpoints":
		var args setBreakpointsArgs
		if err := decodeBody(req.Arguments, &args); err != nil {
			return s.respond(req, false, nil, err.Error())
		}
		s.breakpoints = nil
		for _, bp := range args.Breakpoints {
			s.breakpoints = append(s.breakpoints, sourceBreakpoint{
				file: args.Source.Path,
				line: bp.Line,
			})
		}
		bps := make([]map[string]any, len(args.Breakpoints))
		for i, bp := range args.Breakpoints {
			bps[i] = map[string]any{
				"id":       i + 1,
				"verified": true,
				"line":     bp.Line,
			}
		}
		return s.respond(req, true, map[string]any{"breakpoints": bps})
	case "configurationDone":
		if err := s.respond(req, true, nil); err != nil {
			return err
		}
		go s.runSession()
		return nil
	case "threads":
		return s.respond(req, true, map[string]any{
			"threads": []map[string]any{{"id": threadID, "name": "main"}},
		})
	case "stackTrace":
		ev := s.currentEvent()
		if ev == nil {
			return s.respond(req, true, map[string]any{"stackFrames": []any{}, "totalFrames": 0})
		}
		line := ev.Location.Line + 1
		col := ev.Location.Col + 1
		return s.respond(req, true, map[string]any{
			"stackFrames": []map[string]any{{
				"id":     frameID,
				"name":   ev.FnName,
				"line":   line,
				"column": col,
				"source": map[string]any{"path": ev.Location.File},
			}},
			"totalFrames": 1,
		})
	case "scopes":
		return s.respond(req, true, map[string]any{
			"scopes": []map[string]any{
				{"name": "Locals", "variablesReference": refLocals, "expensive": false},
				{"name": "Stack", "variablesReference": refStack, "expensive": false},
			},
		})
	case "variables":
		var args struct {
			VariablesReference int `json:"variablesReference"`
		}
		_ = decodeBody(req.Arguments, &args)
		ev := s.currentEvent()
		vars := []map[string]any{}
		if ev != nil {
			switch args.VariablesReference {
			case refLocals:
				id := 1
				for _, lv := range ev.Locals {
					if lv.Value == nil && len(lv.Name) > 0 && lv.Name[0] == '$' {
						continue
					}
					vars = append(vars, map[string]any{
						"name":               lv.Name,
						"value":              vm.FormatValue(lv.Value),
						"variablesReference": 0,
						"evaluateName":       lv.Name,
					})
					id++
				}
			case refStack:
				for i, val := range ev.Stack {
					vars = append(vars, map[string]any{
						"name":               fmt.Sprintf("[%d]", i),
						"value":              vm.FormatValue(val),
						"variablesReference": 0,
					})
				}
			}
		}
		return s.respond(req, true, map[string]any{"variables": vars})
	case "continue":
		s.actionCh <- vm.ActionContinue
		return s.respond(req, true, map[string]any{"allThreadsContinued": true})
	case "next":
		s.actionCh <- vm.ActionStep
		return s.respond(req, true, map[string]any{})
	case "disconnect":
		select {
		case s.actionCh <- vm.ActionQuit:
		default:
		}
		_ = s.respond(req, true, nil)
		return io.EOF
	default:
		return s.respond(req, true, nil)
	}
}

func (s *Server) runSession() {
	data, err := os.ReadFile(s.program)
	if err != nil {
		s.sendTerminated(err)
		return
	}
	mod, err := cli.CompileModule(data, s.program)
	if err != nil {
		s.sendTerminated(err)
		return
	}

	dbg := vm.NewDebugger(func(ev vm.DebugEvent) (vm.DebugAction, error) {
		s.mu.Lock()
		cp := ev
		s.lastEvent = &cp
		s.mu.Unlock()

		_ = s.event("stopped", map[string]any{
			"reason":            "breakpoint",
			"threadId":          threadID,
			"allThreadsStopped": true,
		})

		action := <-s.actionCh
		return action, nil
	})

	for _, bp := range s.breakpoints {
		file := bp.file
		if file == "" {
			file = s.program
		}
		dbg.SetBreakpoint(file, bp.line)
	}

	m := vm.New(mod)
	_, err = m.RunDebug(dbg)
	if err != nil && err.Error() == "debug: stopped" {
		err = nil
	}
	s.sendTerminated(err)
}

func (s *Server) sendTerminated(err error) {
	body := map[string]any{}
	if err != nil {
		body["description"] = err.Error()
	}
	_ = s.event("terminated", body)
	close(s.actionCh)
}

func (s *Server) currentEvent() *vm.DebugEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastEvent
}

func (s *Server) nextSeq() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	return s.seq
}

func (s *Server) respond(req Message, ok bool, body any, errMsg ...string) error {
	var raw json.RawMessage
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		raw = b
	}
	msg := Message{
		Type:       "response",
		Seq:        s.nextSeq(),
		RequestSeq: req.Seq,
		Success:    ok,
		Command:    req.Command,
		Body:       raw,
	}
	if !ok && len(errMsg) > 0 {
		msg.Message = errMsg[0]
	}
	return writeMessage(s.out, msg)
}

func (s *Server) event(name string, body any) error {
	var raw json.RawMessage
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		raw = b
	}
	return writeMessage(s.out, Message{
		Type:  "event",
		Seq:   s.nextSeq(),
		Event: name,
		Body:  raw,
	})
}
