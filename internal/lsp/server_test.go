package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func frame(t *testing.T, v any) []byte {
	t.Helper()
	body, err := json.Marshal(v)
	require.NoError(t, err)
	return []byte(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body))
}

// runOne feeds a single message to the server's dispatch logic directly
// (bypassing the blocking Serve loop, which is exercised separately in
// TestServer_ServeEndToEnd) and returns the raw response bytes written.
func runOne(t *testing.T, s *Server, msg *rpcMessage) *bytes.Buffer {
	t.Helper()
	out := &bytes.Buffer{}
	s.out = newRPCWriter(out)
	s.dispatch(msg)
	return out
}

func rawParams(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

func TestServer_Initialize(t *testing.T) {
	s := NewServer(nil)
	out := runOne(t, s, &rpcMessage{JSONRPC: "2.0", ID: json.RawMessage("1"), Method: "initialize", Params: rawParams(t, InitializeParams{})})

	var resp rpcMessage
	decodeFramed(t, out, &resp)
	require.Nil(t, resp.Error)
	var result InitializeResult
	require.NoError(t, json.Unmarshal(resp.Result, &result))
	require.True(t, result.Capabilities.HoverProvider)
	require.True(t, result.Capabilities.DefinitionProvider)
	require.True(t, result.Capabilities.DocumentSymbolProvider)
	require.True(t, result.Capabilities.DocumentFormattingProvider)
	require.NotNil(t, result.Capabilities.CompletionProvider)
	require.NotNil(t, result.Capabilities.SignatureHelpProvider)
	require.Equal(t, SyncFull, result.Capabilities.TextDocumentSync)
}

func TestServer_UnknownMethod_ReturnsMethodNotFound(t *testing.T) {
	s := NewServer(nil)
	out := runOne(t, s, &rpcMessage{JSONRPC: "2.0", ID: json.RawMessage("1"), Method: "textDocument/bogus"})
	var resp rpcMessage
	decodeFramed(t, out, &resp)
	require.NotNil(t, resp.Error)
	require.Equal(t, codeMethodNotFound, resp.Error.Code)
}

func TestServer_DidOpen_PublishesDiagnosticsForTypeError(t *testing.T) {
	s := NewServer(nil)
	out := &bytes.Buffer{}
	s.out = newRPCWriter(out)
	s.dispatch(&rpcMessage{Method: "textDocument/didOpen", Params: rawParams(t, DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{URI: "file:///tmp/bad.fn", Text: "let x: int = \"oops\"\n", Version: 1},
	})})

	var note rpcMessage
	decodeFramed(t, out, &note)
	require.Equal(t, "textDocument/publishDiagnostics", note.Method)
	var params PublishDiagnosticsParams
	require.NoError(t, json.Unmarshal(note.Params, &params))
	require.Len(t, params.Diagnostics, 1)
	require.Equal(t, "E2010", params.Diagnostics[0].Code)
	require.Equal(t, 0, params.Diagnostics[0].Range.Start.Line)
}

func TestServer_DidOpen_NoErrors_PublishesEmptyDiagnostics(t *testing.T) {
	s := NewServer(nil)
	out := &bytes.Buffer{}
	s.out = newRPCWriter(out)
	s.dispatch(&rpcMessage{Method: "textDocument/didOpen", Params: rawParams(t, DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{URI: "file:///tmp/ok.fn", Text: "let x = 1\nprintln(x)\n", Version: 1},
	})})

	var note rpcMessage
	decodeFramed(t, out, &note)
	var params PublishDiagnosticsParams
	require.NoError(t, json.Unmarshal(note.Params, &params))
	require.Empty(t, params.Diagnostics)
}

func TestServer_DidChange_ReanalyzesAndRepublishes(t *testing.T) {
	s := NewServer(nil)
	drain := &bytes.Buffer{}
	s.out = newRPCWriter(drain)
	s.dispatch(&rpcMessage{Method: "textDocument/didOpen", Params: rawParams(t, DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{URI: "file:///tmp/x.fn", Text: "let x = 1\n", Version: 1},
	})})

	out := &bytes.Buffer{}
	s.out = newRPCWriter(out)
	s.dispatch(&rpcMessage{Method: "textDocument/didChange", Params: rawParams(t, DidChangeTextDocumentParams{
		TextDocument:   VersionedTextDocumentIdentifier{URI: "file:///tmp/x.fn", Version: 2},
		ContentChanges: []TextDocumentContentChangeEvent{{Text: "let x: int = \"bad\"\n"}},
	})})
	var note rpcMessage
	decodeFramed(t, out, &note)
	var params PublishDiagnosticsParams
	require.NoError(t, json.Unmarshal(note.Params, &params))
	require.Len(t, params.Diagnostics, 1)
}

func TestServer_ServeEndToEnd(t *testing.T) {
	clientToServer := &bytes.Buffer{}
	serverToClient := &bytes.Buffer{}

	clientToServer.Write(frame(t, rpcMessage{JSONRPC: "2.0", ID: json.RawMessage("1"), Method: "initialize", Params: rawParams(t, InitializeParams{})}))
	clientToServer.Write(frame(t, rpcMessage{JSONRPC: "2.0", Method: "initialized", Params: rawParams(t, struct{}{})}))
	clientToServer.Write(frame(t, rpcMessage{JSONRPC: "2.0", Method: "exit"}))

	s := NewServer(nil)
	err := s.Serve(clientToServer, serverToClient)
	require.NoError(t, err)

	r := bufio.NewReader(serverToClient)
	var resp rpcMessage
	decodeFramedReader(t, r, &resp)
	require.Equal(t, json.RawMessage("1"), resp.ID)
	require.Nil(t, resp.Error)
}

// decodeFramed reads exactly one Content-Length-framed JSON message from buf.
func decodeFramed(t *testing.T, buf *bytes.Buffer, v any) {
	t.Helper()
	decodeFramedReader(t, bufio.NewReader(buf), v)
}

func decodeFramedReader(t *testing.T, r *bufio.Reader, v any) {
	t.Helper()
	rr := &rpcReader{br: r}
	msg, err := rr.readMessage()
	require.NoError(t, err)
	b, err := json.Marshal(msg)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(b, v))
}

func TestRPCFraming_RoundTrip(t *testing.T) {
	buf := &bytes.Buffer{}
	w := newRPCWriter(buf)
	require.NoError(t, w.writeResult(json.RawMessage("7"), map[string]string{"ok": "yes"}))

	r := newRPCReader(strings.NewReader(buf.String()))
	msg, err := r.readMessage()
	require.NoError(t, err)
	require.Equal(t, json.RawMessage("7"), msg.ID)
	require.JSONEq(t, `{"ok":"yes"}`, string(msg.Result))
}
