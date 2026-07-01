// Package lsp implements a Language Server Protocol server for funny.
// The transport is JSON-RPC 2.0 framed with `Content-Length` headers over
// stdio, matching the standard LSP wire format used by every LSP client.
package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
)

// rpcMessage is the wire shape shared by requests, responses, and
// notifications. Requests/notifications differ only in whether ID is set;
// responses differ by having Result/Error instead of Method/Params.
type rpcMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Standard JSON-RPC / LSP error codes used by this server.
const (
	codeParseError     = -32700
	codeInvalidRequest = -32600
	codeMethodNotFound = -32601
	codeInvalidParams  = -32602
	codeInternalError  = -32603
)

// rpcReader reads Content-Length-framed JSON-RPC messages from r.
type rpcReader struct {
	br *bufio.Reader
}

func newRPCReader(r io.Reader) *rpcReader {
	return &rpcReader{br: bufio.NewReader(r)}
}

// readMessage reads one framed message. Returns io.EOF when the stream ends.
func (rr *rpcReader) readMessage() (*rpcMessage, error) {
	var contentLength int
	for {
		line, err := rr.br.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break // blank line ends the header block
		}
		name, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(name), "Content-Length") {
			n, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return nil, fmt.Errorf("lsp: invalid Content-Length: %w", err)
			}
			contentLength = n
		}
	}
	if contentLength <= 0 {
		return nil, fmt.Errorf("lsp: missing or zero Content-Length header")
	}
	body := make([]byte, contentLength)
	if _, err := io.ReadFull(rr.br, body); err != nil {
		return nil, err
	}
	var msg rpcMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		return nil, fmt.Errorf("lsp: invalid JSON-RPC message: %w", err)
	}
	return &msg, nil
}

// rpcWriter writes Content-Length-framed JSON-RPC messages to w. Safe for
// concurrent use since notifications (e.g. publishDiagnostics) can be sent
// from handlers running while another request is being processed.
type rpcWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func newRPCWriter(w io.Writer) *rpcWriter {
	return &rpcWriter{w: w}
}

func (rw *rpcWriter) write(msg any) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	rw.mu.Lock()
	defer rw.mu.Unlock()
	if _, err := fmt.Fprintf(rw.w, "Content-Length: %d\r\n\r\n", len(body)); err != nil {
		return err
	}
	_, err = rw.w.Write(body)
	return err
}

func (rw *rpcWriter) writeResult(id json.RawMessage, result any) error {
	payload, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return rw.write(rpcMessage{JSONRPC: "2.0", ID: id, Result: payload})
}

func (rw *rpcWriter) writeError(id json.RawMessage, code int, message string) error {
	return rw.write(rpcMessage{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: message}})
}

// notify sends a server-to-client notification (no ID, no response expected).
func (rw *rpcWriter) notify(method string, params any) error {
	payload, err := json.Marshal(params)
	if err != nil {
		return err
	}
	return rw.write(rpcMessage{JSONRPC: "2.0", Method: method, Params: payload})
}

func isNotification(msg *rpcMessage) bool { return len(msg.ID) == 0 }
