package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Server is a funny Language Server: it speaks LSP 3.17 (the subset
// implemented in protocol.go) over any io.Reader/io.Writer pair, normally
// process stdin/stdout.
type Server struct {
	docs *documentStore
	out  *rpcWriter
	log  io.Writer
}

// NewServer creates a Server. Diagnostic/debug logging (never protocol
// traffic) is written to log if non-nil.
func NewServer(log io.Writer) *Server {
	return &Server{docs: newDocumentStore(), log: log}
}

// Run starts the funny LSP server on stdio and blocks until the client
// disconnects (stdin EOF) or sends the `exit` notification. Logging goes to
// stderr, since stdout is reserved for the JSON-RPC stream.
func Run(ctx context.Context) error {
	return NewServer(os.Stderr).Serve(os.Stdin, os.Stdout)
}

func (s *Server) logf(format string, args ...any) {
	if s.log != nil {
		fmt.Fprintf(s.log, "[funny lsp] "+format+"\n", args...)
	}
}

// Serve reads JSON-RPC messages from r and writes responses/notifications
// to w until r is exhausted or an `exit` notification arrives.
func (s *Server) Serve(r io.Reader, w io.Writer) error {
	s.out = newRPCWriter(w)
	reader := newRPCReader(r)
	for {
		msg, err := reader.readMessage()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if msg.Method == "exit" {
			return nil
		}
		s.dispatch(msg)
	}
}

func (s *Server) dispatch(msg *rpcMessage) {
	defer func() {
		if r := recover(); r != nil {
			s.logf("panic handling %s: %v", msg.Method, r)
			if !isNotification(msg) {
				_ = s.out.writeError(msg.ID, codeInternalError, fmt.Sprintf("internal error: %v", r))
			}
		}
	}()

	switch msg.Method {
	case "initialize":
		s.handleInitialize(msg)
	case "initialized", "$/cancelRequest", "workspace/didChangeConfiguration":
		// no-op notifications
	case "shutdown":
		_ = s.out.writeResult(msg.ID, nil)
	case "textDocument/didOpen":
		s.handleDidOpen(msg)
	case "textDocument/didChange":
		s.handleDidChange(msg)
	case "textDocument/didClose":
		s.handleDidClose(msg)
	case "textDocument/hover":
		s.handleHover(msg)
	case "textDocument/completion":
		s.handleCompletion(msg)
	case "textDocument/signatureHelp":
		s.handleSignatureHelp(msg)
	case "textDocument/definition":
		s.handleDefinition(msg)
	case "textDocument/documentSymbol":
		s.handleDocumentSymbol(msg)
	case "textDocument/formatting":
		s.handleFormatting(msg)
	case "textDocument/references":
		s.handleReferences(msg)
	case "textDocument/prepareRename":
		s.handlePrepareRename(msg)
	case "textDocument/rename":
		s.handleRename(msg)
	case "funny/planGraph":
		s.handlePlanGraph(msg)
	default:
		if !isNotification(msg) {
			_ = s.out.writeError(msg.ID, codeMethodNotFound, "method not found: "+msg.Method)
		}
	}
}

func (s *Server) handleInitialize(msg *rpcMessage) {
	result := InitializeResult{
		ServerInfo: ServerInfo{Name: "funny", Version: "2.1.0"},
		Capabilities: ServerCapabilities{
			TextDocumentSync:           SyncFull,
			HoverProvider:              true,
			CompletionProvider:         &CompletionOptions{TriggerCharacters: []string{".", "("}},
			SignatureHelpProvider:      &SignatureHelpOptions{TriggerCharacters: []string{"(", ","}},
			DefinitionProvider:         true,
			DocumentSymbolProvider:     true,
			DocumentFormattingProvider: true,
			ReferencesProvider:         true,
			RenameProvider:             &RenameOptions{PrepareProvider: true},
		},
	}
	if err := s.out.writeResult(msg.ID, result); err != nil {
		s.logf("initialize: write result: %v", err)
	}
}

func (s *Server) publishDiagnostics(d *document) {
	diags := d.diagnostics
	if diags == nil {
		diags = []Diagnostic{}
	}
	if err := s.out.notify("textDocument/publishDiagnostics", PublishDiagnosticsParams{URI: d.uri, Diagnostics: diags}); err != nil {
		s.logf("publishDiagnostics: %v", err)
	}
}

func (s *Server) handleDidOpen(msg *rpcMessage) {
	var p DidOpenTextDocumentParams
	if err := json.Unmarshal(msg.Params, &p); err != nil {
		s.logf("didOpen: %v", err)
		return
	}
	d := s.docs.open(p.TextDocument.URI, p.TextDocument.Text, p.TextDocument.Version)
	s.publishDiagnostics(d)
}

func (s *Server) handleDidChange(msg *rpcMessage) {
	var p DidChangeTextDocumentParams
	if err := json.Unmarshal(msg.Params, &p); err != nil {
		s.logf("didChange: %v", err)
		return
	}
	if len(p.ContentChanges) == 0 {
		return
	}
	// Only full-document sync is advertised/supported, so the last change
	// event always carries the complete new text.
	text := p.ContentChanges[len(p.ContentChanges)-1].Text
	d := s.docs.update(p.TextDocument.URI, text, p.TextDocument.Version)
	s.publishDiagnostics(d)
}

func (s *Server) handleDidClose(msg *rpcMessage) {
	var p DidCloseTextDocumentParams
	if err := json.Unmarshal(msg.Params, &p); err != nil {
		s.logf("didClose: %v", err)
		return
	}
	s.docs.close(p.TextDocument.URI)
}

func (s *Server) withDocument(msg *rpcMessage, fn func(d *document, pos Position)) {
	var p TextDocumentPositionParams
	if err := json.Unmarshal(msg.Params, &p); err != nil {
		_ = s.out.writeError(msg.ID, codeInvalidParams, err.Error())
		return
	}
	d, ok := s.docs.get(p.TextDocument.URI)
	if !ok {
		_ = s.out.writeResult(msg.ID, nil)
		return
	}
	fn(d, p.Position)
}

func (s *Server) handleHover(msg *rpcMessage) {
	s.withDocument(msg, func(d *document, pos Position) {
		_ = s.out.writeResult(msg.ID, d.hover(pos))
	})
}

func (s *Server) handleCompletion(msg *rpcMessage) {
	s.withDocument(msg, func(d *document, pos Position) {
		_ = s.out.writeResult(msg.ID, d.completion(pos))
	})
}

func (s *Server) handleSignatureHelp(msg *rpcMessage) {
	s.withDocument(msg, func(d *document, pos Position) {
		_ = s.out.writeResult(msg.ID, d.signatureHelp(pos))
	})
}

func (s *Server) handleDefinition(msg *rpcMessage) {
	s.withDocument(msg, func(d *document, pos Position) {
		loc := d.definition(pos)
		if loc == nil {
			_ = s.out.writeResult(msg.ID, nil)
			return
		}
		_ = s.out.writeResult(msg.ID, loc)
	})
}

func (s *Server) handleDocumentSymbol(msg *rpcMessage) {
	var p DocumentSymbolParams
	if err := json.Unmarshal(msg.Params, &p); err != nil {
		_ = s.out.writeError(msg.ID, codeInvalidParams, err.Error())
		return
	}
	d, ok := s.docs.get(p.TextDocument.URI)
	if !ok {
		_ = s.out.writeResult(msg.ID, []DocumentSymbol{})
		return
	}
	syms := d.documentSymbols()
	if syms == nil {
		syms = []DocumentSymbol{}
	}
	_ = s.out.writeResult(msg.ID, syms)
}

func (s *Server) handleFormatting(msg *rpcMessage) {
	var p DocumentFormattingParams
	if err := json.Unmarshal(msg.Params, &p); err != nil {
		_ = s.out.writeError(msg.ID, codeInvalidParams, err.Error())
		return
	}
	d, ok := s.docs.get(p.TextDocument.URI)
	if !ok {
		_ = s.out.writeResult(msg.ID, []TextEdit{})
		return
	}
	edits, err := d.formatting()
	if err != nil {
		// Matches `funny fmt`'s behavior: refuse to format invalid source
		// rather than guessing; the diagnostics already explain why.
		_ = s.out.writeResult(msg.ID, []TextEdit{})
		return
	}
	if edits == nil {
		edits = []TextEdit{}
	}
	_ = s.out.writeResult(msg.ID, edits)
}

func (s *Server) handleReferences(msg *rpcMessage) {
	var p ReferenceParams
	if err := json.Unmarshal(msg.Params, &p); err != nil {
		_ = s.out.writeError(msg.ID, codeInvalidParams, err.Error())
		return
	}
	d, ok := s.docs.get(p.TextDocument.URI)
	if !ok {
		_ = s.out.writeResult(msg.ID, []Location{})
		return
	}
	locs := d.references(p.Position, p.Context.IncludeDeclaration)
	if locs == nil {
		locs = []Location{}
	}
	_ = s.out.writeResult(msg.ID, locs)
}

func (s *Server) handlePrepareRename(msg *rpcMessage) {
	var p PrepareRenameParams
	if err := json.Unmarshal(msg.Params, &p); err != nil {
		_ = s.out.writeError(msg.ID, codeInvalidParams, err.Error())
		return
	}
	d, ok := s.docs.get(p.TextDocument.URI)
	if !ok {
		_ = s.out.writeResult(msg.ID, nil)
		return
	}
	result, err := d.prepareRename(p.Position)
	if err != nil {
		_ = s.out.writeError(msg.ID, codeInvalidRequest, err.Error())
		return
	}
	_ = s.out.writeResult(msg.ID, result)
}

func (s *Server) handleRename(msg *rpcMessage) {
	var p RenameParams
	if err := json.Unmarshal(msg.Params, &p); err != nil {
		_ = s.out.writeError(msg.ID, codeInvalidParams, err.Error())
		return
	}
	d, ok := s.docs.get(p.TextDocument.URI)
	if !ok {
		_ = s.out.writeError(msg.ID, codeInvalidRequest, "document not open: "+p.TextDocument.URI)
		return
	}
	edit, err := d.rename(p.Position, p.NewName)
	if err != nil {
		_ = s.out.writeError(msg.ID, codeInvalidRequest, err.Error())
		return
	}
	_ = s.out.writeResult(msg.ID, edit)
}

func (s *Server) handlePlanGraph(msg *rpcMessage) {
	var p PlanGraphParams
	if err := json.Unmarshal(msg.Params, &p); err != nil {
		_ = s.out.writeError(msg.ID, codeInvalidParams, err.Error())
		return
	}
	d, ok := s.docs.get(p.TextDocument.URI)
	if !ok {
		_ = s.out.writeResult(msg.ID, PlanGraphResult{Plans: []PlanGraph{}})
		return
	}
	_ = s.out.writeResult(msg.ID, d.planGraphs())
}
