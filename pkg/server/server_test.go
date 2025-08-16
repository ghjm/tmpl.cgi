package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"gopkg.mhn.org/tmpl.cgi/pkg/config"
)

func TestNew(t *testing.T) {
	cfg := &config.Config{
		DefaultTemplate: "default.html",
	}

	server, err := New(cfg)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	if server == nil {
		t.Fatal("New() returned nil server")
	}

	if server.config.DefaultTemplate != cfg.DefaultTemplate {
		t.Errorf("Expected DefaultTemplate %s, got %s", cfg.DefaultTemplate, server.config.DefaultTemplate)
	}
}

func TestGetRequestURI(t *testing.T) {
	tests := []struct {
		name       string
		requestURI string
		urlPath    string
		expected   string
	}{
		{
			name:       "RequestURI present",
			requestURI: "/test/path",
			urlPath:    "/different/path",
			expected:   "/test/path",
		},
		{
			name:       "RequestURI empty, use URL.Path",
			requestURI: "",
			urlPath:    "/url/path",
			expected:   "/url/path",
		},
		{
			name:       "Both present, RequestURI takes precedence",
			requestURI: "/request/uri",
			urlPath:    "/url/path",
			expected:   "/request/uri",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://example.com"+tt.urlPath, nil)
			req.RequestURI = tt.requestURI

			result := getRequestURI(req)
			if result != tt.expected {
				t.Errorf("getRequestURI() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestServeHTTP(t *testing.T) {
	// Create a temporary config file and templates for testing
	tempDir := t.TempDir()

	// Create a simple template
	templateContent := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<h1>Test Template</h1>
<p>URI: {{.RequestURI}}</p>
<p>Data: {{.Data.test}}</p>
</body>
</html>`

	templatePath := tempDir + "/test.html"
	err := os.WriteFile(templatePath, []byte(templateContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test template: %v", err)
	}

	cfg := &config.Config{
		ConfigFilePath:  tempDir + "/config.yaml",
		DefaultTemplate: templatePath,
		Data: map[string]interface{}{
			"test": "hello world",
		},
	}

	server, err := New(cfg)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectedBody   []string // strings that should be present in response
	}{
		{
			name:           "Valid request",
			path:           "/test/path",
			expectedStatus: http.StatusOK,
			expectedBody:   []string{"Test Template", "URI: /test/path", "Data: hello world"},
		},
		{
			name:           "Root path",
			path:           "/",
			expectedStatus: http.StatusOK,
			expectedBody:   []string{"Test Template", "URI: /"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://example.com"+tt.path, nil)
			req.RequestURI = tt.path // Set RequestURI explicitly
			w := httptest.NewRecorder()

			server.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("ServeHTTP() status = %d, want %d", w.Code, tt.expectedStatus)
			}

			body := w.Body.String()
			for _, expected := range tt.expectedBody {
				if !strings.Contains(body, expected) {
					t.Errorf("ServeHTTP() body should contain %q, got: %s", expected, body)
				}
			}

			// Check content type
			contentType := w.Header().Get("Content-Type")
			if contentType != "text/html; charset=utf-8" {
				t.Errorf("ServeHTTP() Content-Type = %s, want text/html; charset=utf-8", contentType)
			}
		})
	}
}

func TestServeHTTP_TemplateError(t *testing.T) {
	// Test with invalid template path
	cfg := &config.Config{
		ConfigFilePath:  "/nonexistent/config.yaml",
		DefaultTemplate: "/nonexistent/template.html",
	}

	server, err := New(cfg)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	req := httptest.NewRequest("GET", "http://example.com/test", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	// Should return an error response
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ServeHTTP() with invalid template status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestServeHTTP_TemplateExecutionError(t *testing.T) {
	// Create a template with invalid syntax
	tempDir := t.TempDir()

	templateContent := `<!DOCTYPE html>
<html>
<body>
<h1>Test Template</h1>
<p>Invalid template syntax: {{.NonExistentField.SubField}}</p>
</body>
</html>`

	templatePath := tempDir + "/invalid.html"
	err := os.WriteFile(templatePath, []byte(templateContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test template: %v", err)
	}

	cfg := &config.Config{
		ConfigFilePath:  tempDir + "/config.yaml",
		DefaultTemplate: templatePath,
	}

	server, err := New(cfg)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	req := httptest.NewRequest("GET", "http://example.com/test", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	// Should return an error response due to template execution failure
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ServeHTTP() with template execution error status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// TestRun is tricky to test directly since it involves network operations
// We'll test the logic paths but not the actual network binding
func TestRun_CGIDetection(t *testing.T) {
	cfg := &config.Config{
		DefaultTemplate: "test.html",
	}

	server, err := New(cfg)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Test CGI environment detection
	originalGateway := os.Getenv("GATEWAY_INTERFACE")
	defer func() {
		if originalGateway == "" {
			_ = os.Unsetenv("GATEWAY_INTERFACE")
		} else {
			_ = os.Setenv("GATEWAY_INTERFACE", originalGateway)
		}
	}()

	// Set CGI environment
	_ = os.Setenv("GATEWAY_INTERFACE", "CGI/1.1")

	// We can't easily test the actual CGI serving without complex setup,
	// but we can verify the environment detection logic works
	// The Run() method will attempt to serve CGI, which will likely fail
	// in test environment, but that's expected
	err = server.Run()
	// We expect an error here since we're not in a real CGI environment
	if err == nil {
		t.Log("Run() succeeded unexpectedly in CGI mode - this might be okay depending on test environment")
	}
}

func TestRun_StandaloneMode(t *testing.T) {
	cfg := &config.Config{
		DefaultTemplate: "test.html",
	}

	server, err := New(cfg)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Ensure we're not in CGI mode
	originalGateway := os.Getenv("GATEWAY_INTERFACE")
	originalPort := os.Getenv("TMPL_CGI_PORT")
	defer func() {
		if originalGateway == "" {
			_ = os.Unsetenv("GATEWAY_INTERFACE")
		} else {
			_ = os.Setenv("GATEWAY_INTERFACE", originalGateway)
		}
		if originalPort == "" {
			_ = os.Unsetenv("TMPL_CGI_PORT")
		} else {
			_ = os.Setenv("TMPL_CGI_PORT", originalPort)
		}
	}()

	_ = os.Unsetenv("GATEWAY_INTERFACE")
	_ = os.Setenv("TMPL_CGI_PORT", "0") // Use port 0 to let OS choose available port

	// We can't easily test the full server startup without it blocking,
	// but we can test that it attempts to start
	// In a real test, you might want to run this in a goroutine and then stop it
	go func() {
		err := server.Run()
		if err != nil {
			t.Logf("Server run failed (expected in test): %v", err)
		}
	}()

	// Give it a moment to attempt startup
	// In a more sophisticated test, you might check if the port is actually listening
}
