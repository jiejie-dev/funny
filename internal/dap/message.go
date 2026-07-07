package dap

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Message is a DAP protocol message (request, response, or event).
type Message struct {
	Seq        int             `json:"seq,omitempty"`
	Type       string          `json:"type"`
	Command    string          `json:"command,omitempty"`
	Event      string          `json:"event,omitempty"`
	RequestSeq int             `json:"request_seq,omitempty"`
	Success    bool            `json:"success,omitempty"`
	Message    string          `json:"message,omitempty"`
	Body       json.RawMessage `json:"body,omitempty"`
	Arguments  json.RawMessage `json:"arguments,omitempty"`
}

func readMessage(r *bufio.Reader) (Message, error) {
	var length int
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return Message{}, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "Content-Length:") {
			n, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:")))
			if err != nil {
				return Message{}, err
			}
			length = n
			break
		}
	}
	if _, err := r.ReadString('\n'); err != nil { // blank line after headers
		return Message{}, err
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return Message{}, err
	}
	var msg Message
	if err := json.Unmarshal(buf, &msg); err != nil {
		return Message{}, err
	}
	return msg, nil
}

func writeMessage(w io.Writer, msg Message) error {
	raw, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Content-Length: %d\r\n\r\n", len(raw)); err != nil {
		return err
	}
	_, err = w.Write(raw)
	return err
}

func decodeBody[T any](body json.RawMessage, out *T) error {
	if len(body) == 0 {
		return nil
	}
	return json.Unmarshal(body, out)
}
