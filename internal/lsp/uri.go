package lsp

import (
	"net/url"
	"runtime"
	"strings"
)

// uriToPath converts a `file://` URI (as sent by every LSP client for local
// files) into a plain filesystem path suitable for os.ReadFile / parser.New.
func uriToPath(uri string) string {
	if !strings.HasPrefix(uri, "file://") {
		return uri // already a bare path (used in tests / non-file schemes best-effort)
	}
	u, err := url.Parse(uri)
	if err != nil {
		return strings.TrimPrefix(uri, "file://")
	}
	p := u.Path
	if runtime.GOOS == "windows" {
		p = strings.TrimPrefix(p, "/")
	}
	return p
}

// pathToURI converts a plain filesystem path into a `file://` URI.
func pathToURI(path string) string {
	if strings.HasPrefix(path, "file://") {
		return path
	}
	p := path
	if runtime.GOOS == "windows" {
		p = "/" + strings.ReplaceAll(p, "\\", "/")
	}
	u := url.URL{Scheme: "file", Path: p}
	return u.String()
}
