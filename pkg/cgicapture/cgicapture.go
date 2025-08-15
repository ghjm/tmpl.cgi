// Package cgicapture provides utilities for capturing the output of
// http.Handlers or handler-like functions in CGI-style format.
package cgicapture

import (
	"bytes"
	"fmt"
	"net/http"
)

// responseCapture implements http.ResponseWriter and buffers the output.
type responseCapture struct {
	header     http.Header
	statusCode int
	buf        bytes.Buffer
}

// newResponseCapture creates a new capture with default status 200.
func newResponseCapture() *responseCapture {
	return &responseCapture{
		header:     make(http.Header),
		statusCode: http.StatusOK,
	}
}

func (c *responseCapture) Header() http.Header {
	return c.header
}

func (c *responseCapture) Write(b []byte) (int, error) {
	return c.buf.Write(b)
}

func (c *responseCapture) WriteHeader(statusCode int) {
	c.statusCode = statusCode
}

// CaptureFuncCGI runs a function that takes an http.ResponseWriter
// and returns the CGI-style output (headers + blank line + body).
func CaptureFuncCGI(handler func(http.ResponseWriter)) string {
	crw := newResponseCapture()

	// Run the handler
	handler(crw)

	return formatCGIOutput(crw)
}

// CaptureHandlerCGI runs an http.Handler or http.HandlerFunc with a dummy
// *http.Request and returns the CGI-style output.
func CaptureHandlerCGI(h http.Handler, req *http.Request) string {
	crw := newResponseCapture()

	// Run the handler
	h.ServeHTTP(crw, req)

	return formatCGIOutput(crw)
}

// formatCGIOutput formats the captured headers and body in CGI style.
func formatCGIOutput(crw *responseCapture) string {
	var out bytes.Buffer

	// Print a content-type
	if ctype := crw.header.Get("Content-Type"); ctype != "" {
		out.WriteString(fmt.Sprintf("Content-Type: %s\r\n", ctype))
	} else {
		// Default to text/plain if not set
		out.WriteString("Content-Type: text/plain\r\n")
	}

	// Blank line between headers and body
	out.WriteString("\r\n")

	// Body
	out.Write(crw.buf.Bytes())

	return out.String()
}
