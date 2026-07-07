package lsp

import (
	"errors"
	"sync"

	"github.com/jiejie-dev/funny/v2/internal/ast"
	"github.com/jiejie-dev/funny/v2/internal/docgen"
	"github.com/jiejie-dev/funny/v2/internal/errs"
	"github.com/jiejie-dev/funny/v2/internal/module"
	"github.com/jiejie-dev/funny/v2/internal/parser"
	"github.com/jiejie-dev/funny/v2/internal/types"
)

// document holds one open buffer plus the result of its most recent
// analysis pass (parse -> import resolution -> type check). Analysis is
// best-effort at every stage: a failure at one stage still keeps whatever
// was produced by earlier stages, so the editor keeps useful intelligence
// (diagnostics aside) even while the buffer is in a temporarily broken
// state, which is the common case while the user is actively typing.
type document struct {
	uri     string
	path    string
	text    string
	version int

	prog        *ast.Program // best available AST: resolved+rewritten if that succeeded, else the bare parse
	env         *types.Env   // best available type environment (populated up to the first error, if any)
	docIndex    map[string]docgen.SymbolDoc
	diagnostics []Diagnostic
}

// analyze runs the parse/resolve/type-check pipeline against the document's
// current text and stores the results (including diagnostics) on it.
func (d *document) analyze() {
	d.diagnostics = nil
	d.env = types.NewEnv(nil)
	d.prog = nil
	d.docIndex = nil

	p := parser.New(d.text, d.path)
	prog, err := p.Parse()
	if err != nil {
		d.diagnostics = append(d.diagnostics, errorToDiagnostic(err, d.path))
		return
	}
	d.prog = prog

	resolved, err := module.Resolve(prog, d.path)
	if err != nil {
		d.diagnostics = append(d.diagnostics, errorToDiagnostic(err, d.path))
		// Keep the unresolved prog: local (non-import) intelligence is
		// still valuable even when a dependency fails to resolve.
	} else {
		d.prog = resolved
	}

	if err := types.Check(d.prog, d.env); err != nil {
		d.diagnostics = append(d.diagnostics, errorToDiagnostic(err, d.path))
	}
	if d.prog != nil {
		d.docIndex = docgen.SymbolIndex(d.prog, d.env)
	}
}

// errorToDiagnostic converts any error produced by the parser, module
// resolver, or type checker into an LSP diagnostic. Structured errors
// (*errs.Error, *types.Error) carry a code and position and are anchored
// precisely; module-resolution errors additionally wrap the underlying
// structured error with fmt.Errorf("...: %w", ...) to add import-site
// context, so errors.As (not a plain type assertion) is required to find
// it. Anything else (should not normally happen) falls back to a
// document-start range so the message is still surfaced.
func errorToDiagnostic(err error, docPath string) Diagnostic {
	var typeErr *types.Error
	if errors.As(err, &typeErr) {
		if typeErr.Pos.File != "" && typeErr.Pos.File != docPath {
			return Diagnostic{
				Range: pointRange(Position{}), Severity: SeverityError, Code: typeErr.Code,
				Source: "funny", Message: "in imported module: " + err.Error(),
			}
		}
		msg := typeErr.Message
		if typeErr.Expected != nil && typeErr.Actual != nil {
			msg = typeErr.Message + ": expected " + typeErr.Expected.String() + ", got " + typeErr.Actual.String()
		}
		return Diagnostic{
			Range: pointRange(astPosToLSP(typeErr.Pos)), Severity: SeverityError, Code: typeErr.Code,
			Source: "funny", Message: msg,
		}
	}
	var structErr *errs.Error
	if errors.As(err, &structErr) {
		if structErr.Pos.File != "" && structErr.Pos.File != docPath {
			return Diagnostic{
				Range: pointRange(Position{}), Severity: SeverityError, Code: structErr.Code,
				Source: "funny", Message: "in imported module: " + err.Error(),
			}
		}
		return Diagnostic{
			Range: pointRange(errsPosToLSP(structErr.Pos)), Severity: SeverityError, Code: structErr.Code,
			Source: "funny", Message: structErr.Message,
		}
	}
	return Diagnostic{
		Range: pointRange(Position{}), Severity: SeverityError,
		Source: "funny", Message: err.Error(),
	}
}

// documentStore is a concurrency-safe map of open documents, keyed by URI.
type documentStore struct {
	mu   sync.RWMutex
	docs map[string]*document
}

func newDocumentStore() *documentStore {
	return &documentStore{docs: map[string]*document{}}
}

func (s *documentStore) open(uri, text string, version int) *document {
	d := &document{uri: uri, path: uriToPath(uri), text: text, version: version}
	d.analyze()
	s.mu.Lock()
	s.docs[uri] = d
	s.mu.Unlock()
	return d
}

func (s *documentStore) update(uri, text string, version int) *document {
	s.mu.Lock()
	d, ok := s.docs[uri]
	if !ok {
		d = &document{uri: uri, path: uriToPath(uri)}
		s.docs[uri] = d
	}
	d.text = text
	d.version = version
	s.mu.Unlock()
	d.analyze()
	return d
}

func (s *documentStore) close(uri string) {
	s.mu.Lock()
	delete(s.docs, uri)
	s.mu.Unlock()
}

func (s *documentStore) get(uri string) (*document, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.docs[uri]
	return d, ok
}
