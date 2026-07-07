package dap

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_Initialize(t *testing.T) {
	in := strings.NewReader(encodeRequest(1, "initialize", `{"clientID":"test"}`))
	var out bytes.Buffer
	go func() { _ = Run(in, &out) }()

	require.Eventually(t, func() bool {
		s := out.String()
		return strings.Contains(s, `"command":"initialize"`) &&
			strings.Contains(s, `"event":"initialized"`)
	}, time.Second, 10*time.Millisecond)
}

func TestServer_LaunchMinimalScript(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "sample.fn")
	require.NoError(t, os.WriteFile(script, []byte("println(\"hi\")\n"), 0o644))
	abs, err := filepath.Abs(script)
	require.NoError(t, err)

	var inBuf bytes.Buffer
	seq := 1
	write := func(cmd string, args string) {
		inBuf.WriteString(encodeRequest(seq, cmd, args))
		seq++
	}
	prog, _ := json.Marshal(abs)
	write("initialize", `{"clientID":"test"}`)
	write("launch", `{"program":`+string(prog)+`}`)
	write("setBreakpoints", `{"source":{"path":`+string(prog)+`},"breakpoints":[{"line":1}]}`)
	write("configurationDone", `{}`)
	write("disconnect", `{}`)

	var out bytes.Buffer
	go func() { _ = Run(&inBuf, &out) }()

	require.Eventually(t, func() bool {
		s := out.String()
		return strings.Contains(s, `"event":"stopped"`) || strings.Contains(s, `"event":"terminated"`)
	}, 3*time.Second, 50*time.Millisecond)

	assert.Contains(t, out.String(), `"command":"initialize"`)
}

func encodeRequest(seq int, command, args string) string {
	body := `{"seq":` + strconv.Itoa(seq) + `,"type":"request","command":"` + command + `","arguments":` + args + `}`
	return "Content-Length: " + strconv.Itoa(len(body)) + "\r\n\r\n" + body
}
