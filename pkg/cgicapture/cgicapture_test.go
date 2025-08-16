package cgicapture

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewResponseCapture(t *testing.T) {
	capture := newResponseCapture()

	if capture == nil {
		t.Fatal("newResponseCapture() returned nil")
	}

	if capture.statusCode != http.StatusOK {
		t.Errorf("Default status code = %d, want %d", capture.statusCode, http.StatusOK)
	}

	if capture.header == nil {
		t.Error("Header should be initialized")
	}

	if capture.buf.Len() != 0 {
		t.Error("Buffer should be empty initially")
	}
}

func TestResponseCapture_Header(t *testing.T) {
	capture := newResponseCapture()
	header := capture.Header()

	if header == nil {
		t.Fatal("Header() returned nil")
	}

	// Test setting and getting headers
	header.Set("Content-Type", "text/html")
	header.Set("X-Custom", "test-value")

	if header.Get("Content-Type") != "text/html" {
		t.Errorf("Content-Type = %s, want text/html", header.Get("Content-Type"))
	}

	if header.Get("X-Custom") != "test-value" {
		t.Errorf("X-Custom = %s, want test-value", header.Get("X-Custom"))
	}
}

func TestResponseCapture_Write(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{
			name:     "Simple write",
			data:     []byte("Hello, World!"),
			expected: "Hello, World!",
		},
		{
			name:     "Empty write",
			data:     []byte(""),
			expected: "",
		},
		{
			name:     "Binary data",
			data:     []byte{0x48, 0x65, 0x6c, 0x6c, 0x6f}, // "Hello"
			expected: "Hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capture := newResponseCapture()
			n, err := capture.Write(tt.data)

			if err != nil {
				t.Errorf("Write() error = %v", err)
			}

			if n != len(tt.data) {
				t.Errorf("Write() returned %d bytes, want %d", n, len(tt.data))
			}

			if capture.buf.String() != tt.expected {
				t.Errorf("Buffer content = %q, want %q", capture.buf.String(), tt.expected)
			}
		})
	}
}

func TestResponseCapture_WriteMultiple(t *testing.T) {
	capture := newResponseCapture()

	// Write multiple times
	data1 := []byte("Hello, ")
	data2 := []byte("World!")

	n1, err1 := capture.Write(data1)
	if err1 != nil {
		t.Errorf("First Write() error = %v", err1)
	}
	if n1 != len(data1) {
		t.Errorf("First Write() returned %d bytes, want %d", n1, len(data1))
	}

	n2, err2 := capture.Write(data2)
	if err2 != nil {
		t.Errorf("Second Write() error = %v", err2)
	}
	if n2 != len(data2) {
		t.Errorf("Second Write() returned %d bytes, want %d", n2, len(data2))
	}

	expected := "Hello, World!"
	if capture.buf.String() != expected {
		t.Errorf("Buffer content = %q, want %q", capture.buf.String(), expected)
	}
}

func TestResponseCapture_WriteHeader(t *testing.T) {
	capture := newResponseCapture()

	// Default status should be 200
	if capture.statusCode != http.StatusOK {
		t.Errorf("Default status = %d, want %d", capture.statusCode, http.StatusOK)
	}

	// Set different status codes
	testCodes := []int{
		http.StatusNotFound,
		http.StatusInternalServerError,
		http.StatusCreated,
		http.StatusBadRequest,
	}

	for _, code := range testCodes {
		capture := newResponseCapture()
		capture.WriteHeader(code)
		if capture.statusCode != code {
			t.Errorf("WriteHeader(%d): status = %d, want %d", code, capture.statusCode, code)
		}
	}
}

func TestCaptureFuncCGI(t *testing.T) {
	tests := []struct {
		name     string
		handler  func(http.ResponseWriter)
		expected []string // strings that should be present in output
	}{
		{
			name: "Simple text response",
			handler: func(w http.ResponseWriter) {
				w.Header().Set("Content-Type", "text/plain")
				_, _ = w.Write([]byte("Hello, World!"))
			},
			expected: []string{
				"Content-Type: text/plain",
				"\r\n\r\n", // blank line between headers and body
				"Hello, World!",
			},
		},
		{
			name: "HTML response",
			handler: func(w http.ResponseWriter) {
				w.Header().Set("Content-Type", "text/html")
				_, _ = w.Write([]byte("<html><body>Test</body></html>"))
			},
			expected: []string{
				"Content-Type: text/html",
				"<html><body>Test</body></html>",
			},
		},
		{
			name: "No content type set",
			handler: func(w http.ResponseWriter) {
				_, _ = w.Write([]byte("Plain text"))
			},
			expected: []string{
				"Content-Type: text/plain", // default
				"Plain text",
			},
		},
		{
			name: "Empty response",
			handler: func(w http.ResponseWriter) {
				w.Header().Set("Content-Type", "text/html")
				// Don't write anything
			},
			expected: []string{
				"Content-Type: text/html",
				"\r\n\r\n", // should still have blank line
			},
		},
		{
			name: "Multiple writes",
			handler: func(w http.ResponseWriter) {
				w.Header().Set("Content-Type", "text/plain")
				_, _ = w.Write([]byte("Hello, "))
				_, _ = w.Write([]byte("World!"))
			},
			expected: []string{
				"Content-Type: text/plain",
				"Hello, World!",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CaptureFuncCGI(tt.handler)

			for _, expected := range tt.expected {
				if !strings.Contains(result, expected) {
					t.Errorf("Output should contain %q, got: %s", expected, result)
				}
			}

			// Check CGI format: should have headers, blank line, body
			if !strings.Contains(result, "\r\n\r\n") {
				t.Error("CGI output should have blank line between headers and body")
			}
		})
	}
}

func TestCaptureHandlerCGI(t *testing.T) {
	tests := []struct {
		name     string
		handler  http.Handler
		request  *http.Request
		expected []string
	}{
		{
			name: "Simple handler",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				_, _ = fmt.Fprintf(w, "Path: %s", r.URL.Path)
			}),
			request:  httptest.NewRequest("GET", "/test", nil),
			expected: []string{"Content-Type: text/plain", "Path: /test"},
		},
		{
			name: "Handler with status code",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error": "not found"}`))
			}),
			request:  httptest.NewRequest("GET", "/api/missing", nil),
			expected: []string{"Content-Type: application/json", `{"error": "not found"}`},
		},
		{
			name: "Handler reading request data",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				_, _ = fmt.Fprintf(w, "Method: %s, Host: %s", r.Method, r.Host)
			}),
			request:  httptest.NewRequest("POST", "http://example.com/submit", nil),
			expected: []string{"Method: POST", "Host: example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CaptureHandlerCGI(tt.handler, tt.request)

			for _, expected := range tt.expected {
				if !strings.Contains(result, expected) {
					t.Errorf("Output should contain %q, got: %s", expected, result)
				}
			}

			// Check CGI format
			if !strings.Contains(result, "\r\n\r\n") {
				t.Error("CGI output should have blank line between headers and body")
			}
		})
	}
}

func TestFormatCGIOutput(t *testing.T) {
	tests := []struct {
		name          string
		contentType   string
		statusCode    int
		body          string
		expectedParts []string
	}{
		{
			name:        "With content type",
			contentType: "text/html",
			statusCode:  http.StatusOK,
			body:        "<html><body>Test</body></html>",
			expectedParts: []string{
				"Content-Type: text/html\r\n",
				"\r\n",
				"<html><body>Test</body></html>",
			},
		},
		{
			name:        "Without content type",
			contentType: "",
			statusCode:  http.StatusOK,
			body:        "Plain text",
			expectedParts: []string{
				"Content-Type: text/plain\r\n", // default
				"\r\n",
				"Plain text",
			},
		},
		{
			name:        "Empty body",
			contentType: "application/json",
			statusCode:  http.StatusOK,
			body:        "",
			expectedParts: []string{
				"Content-Type: application/json\r\n",
				"\r\n",
			},
		},
		{
			name:        "Custom content type",
			contentType: "application/xml; charset=utf-8",
			statusCode:  http.StatusCreated,
			body:        "<xml><data>test</data></xml>",
			expectedParts: []string{
				"Content-Type: application/xml; charset=utf-8\r\n",
				"\r\n",
				"<xml><data>test</data></xml>",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capture := newResponseCapture()
			if tt.contentType != "" {
				capture.header.Set("Content-Type", tt.contentType)
			}
			capture.statusCode = tt.statusCode
			capture.buf.WriteString(tt.body)

			result := formatCGIOutput(capture)

			for _, expected := range tt.expectedParts {
				if !strings.Contains(result, expected) {
					t.Errorf("Output should contain %q, got: %s", expected, result)
				}
			}

			// Verify the structure: Content-Type line, blank line, body
			lines := strings.Split(result, "\r\n")
			if len(lines) < 2 {
				t.Error("CGI output should have at least Content-Type and blank line")
			}

			// First line should be Content-Type
			if !strings.HasPrefix(lines[0], "Content-Type:") {
				t.Errorf("First line should be Content-Type, got: %s", lines[0])
			}

			// Should have a blank line
			blankLineFound := false
			for i, line := range lines {
				if line == "" && i > 0 {
					blankLineFound = true
					break
				}
			}
			if !blankLineFound {
				t.Error("Should have blank line separating headers from body")
			}
		})
	}
}

func TestResponseCapture_InterfaceCompliance(t *testing.T) {
	// Test that responseCapture implements http.ResponseWriter
	var _ http.ResponseWriter = &responseCapture{}

	// Test that it can be used as http.ResponseWriter
	capture := newResponseCapture()
	var w http.ResponseWriter = capture

	// Test Header method
	header := w.Header()
	header.Set("Test", "Value")
	if capture.header.Get("Test") != "Value" {
		t.Error("Header method should work through interface")
	}

	// Test Write method
	data := []byte("test data")
	n, err := w.Write(data)
	if err != nil {
		t.Errorf("Write through interface failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("Write returned %d, want %d", n, len(data))
	}

	// Test WriteHeader method
	w.WriteHeader(http.StatusNotFound)
	if capture.statusCode != http.StatusNotFound {
		t.Errorf("WriteHeader through interface failed: got %d, want %d",
			capture.statusCode, http.StatusNotFound)
	}
}

func TestCGIOutputFormat(t *testing.T) {
	// Test that the CGI output format is correct according to CGI specification
	handler := func(w http.ResponseWriter) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<html><body>Hello</body></html>"))
	}

	result := CaptureFuncCGI(handler)

	// Split into lines
	lines := strings.Split(result, "\r\n")

	// Should have at least: Content-Type line, blank line, body
	if len(lines) < 3 {
		t.Errorf("CGI output should have at least 3 lines, got %d", len(lines))
	}

	// First line should be Content-Type header
	if !strings.HasPrefix(lines[0], "Content-Type: ") {
		t.Errorf("First line should be Content-Type header, got: %s", lines[0])
	}

	// Should have blank line (empty string in split result)
	blankLineIndex := -1
	for i, line := range lines {
		if line == "" {
			blankLineIndex = i
			break
		}
	}
	if blankLineIndex == -1 {
		t.Error("CGI output should have blank line separating headers from body")
	}

	// Everything after blank line should be body
	if blankLineIndex >= 0 && blankLineIndex < len(lines)-1 {
		bodyPart := strings.Join(lines[blankLineIndex+1:], "\r\n")
		if !strings.Contains(bodyPart, "<html><body>Hello</body></html>") {
			t.Errorf("Body part should contain HTML content, got: %s", bodyPart)
		}
	}
}
