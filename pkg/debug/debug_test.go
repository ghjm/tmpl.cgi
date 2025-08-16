package debug

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestIsDebugEnabled(t *testing.T) {
	// Save original state
	originalGlobal := debugGloballyEnabled
	originalEnv := os.Getenv("TMPL_CGI_DEBUG")

	// Restore state after test
	defer func() {
		debugGloballyEnabled = originalGlobal
		if originalEnv == "" {
			_ = os.Unsetenv("TMPL_CGI_DEBUG")
		} else {
			_ = os.Setenv("TMPL_CGI_DEBUG", originalEnv)
		}
	}()

	tests := []struct {
		name       string
		globalFlag bool
		envValue   string
		expected   bool
	}{
		{
			name:       "Global flag enabled",
			globalFlag: true,
			envValue:   "",
			expected:   true,
		},
		{
			name:       "Global flag enabled, env false",
			globalFlag: true,
			envValue:   "false",
			expected:   true, // Global flag takes precedence
		},
		{
			name:       "Env true",
			globalFlag: false,
			envValue:   "true",
			expected:   true,
		},
		{
			name:       "Env yes",
			globalFlag: false,
			envValue:   "yes",
			expected:   true,
		},
		{
			name:       "Env 1",
			globalFlag: false,
			envValue:   "1",
			expected:   true,
		},
		{
			name:       "Env TRUE (case insensitive)",
			globalFlag: false,
			envValue:   "TRUE",
			expected:   true,
		},
		{
			name:       "Env YES (case insensitive)",
			globalFlag: false,
			envValue:   "YES",
			expected:   true,
		},
		{
			name:       "Env false",
			globalFlag: false,
			envValue:   "false",
			expected:   false,
		},
		{
			name:       "Env no",
			globalFlag: false,
			envValue:   "no",
			expected:   false,
		},
		{
			name:       "Env 0",
			globalFlag: false,
			envValue:   "0",
			expected:   false,
		},
		{
			name:       "Env empty",
			globalFlag: false,
			envValue:   "",
			expected:   false,
		},
		{
			name:       "Env random value",
			globalFlag: false,
			envValue:   "random",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global state
			debugGloballyEnabled = tt.globalFlag

			// Set environment variable
			if tt.envValue == "" {
				_ = os.Unsetenv("TMPL_CGI_DEBUG")
			} else {
				_ = os.Setenv("TMPL_CGI_DEBUG", tt.envValue)
			}

			result := IsDebugEnabled()
			if result != tt.expected {
				t.Errorf("IsDebugEnabled() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSetDebugMode(t *testing.T) {
	// Save original state
	originalGlobal := debugGloballyEnabled
	defer func() {
		debugGloballyEnabled = originalGlobal
	}()

	// Initially should be false (or whatever the original state was)
	debugGloballyEnabled = false
	if IsDebugEnabled() && os.Getenv("TMPL_CGI_DEBUG") == "" {
		t.Error("Debug should be disabled initially")
	}

	// Set debug mode
	SetDebugMode()

	// Now should be enabled
	if !IsDebugEnabled() {
		t.Error("Debug should be enabled after SetDebugMode()")
	}

	// Should remain enabled even if env var is false
	originalEnv := os.Getenv("TMPL_CGI_DEBUG")
	_ = os.Setenv("TMPL_CGI_DEBUG", "false")
	defer func() {
		if originalEnv == "" {
			_ = os.Unsetenv("TMPL_CGI_DEBUG")
		} else {
			_ = os.Setenv("TMPL_CGI_DEBUG", originalEnv)
		}
	}()

	if !IsDebugEnabled() {
		t.Error("Debug should remain enabled even with env var false")
	}
}

func TestRenderDebugError(t *testing.T) {
	tests := []struct {
		name     string
		messages [][2]string
		expected []string // strings that should be present in output
	}{
		{
			name: "Single message",
			messages: [][2]string{
				{"Error", "Something went wrong"},
			},
			expected: []string{
				"Runtime Error",
				"Error:",
				"Something went wrong",
				"Debug Mode Enabled",
			},
		},
		{
			name: "Multiple messages",
			messages: [][2]string{
				{"Request URI", "/api/test"},
				{"Error", "Template not found"},
				{"Details", "File does not exist"},
			},
			expected: []string{
				"Runtime Error",
				"Request URI:",
				"/api/test",
				"Error:",
				"Template not found",
				"Details:",
				"File does not exist",
			},
		},
		{
			name:     "Empty messages",
			messages: [][2]string{},
			expected: []string{
				"Runtime Error",
				"Debug Mode Enabled",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			RenderDebugError(w, tt.messages)

			// Check status code
			if w.Code != http.StatusInternalServerError {
				t.Errorf("Status code = %d, want %d", w.Code, http.StatusInternalServerError)
			}

			// Check content type
			contentType := w.Header().Get("Content-Type")
			if contentType != "text/html; charset=utf-8" {
				t.Errorf("Content-Type = %s, want text/html; charset=utf-8", contentType)
			}

			// Check body content
			body := w.Body.String()
			for _, expected := range tt.expected {
				if !strings.Contains(body, expected) {
					t.Errorf("Body should contain %q, got: %s", expected, body)
				}
			}

			// Check that it's valid HTML
			if !strings.Contains(body, "<!DOCTYPE html>") {
				t.Error("Response should be valid HTML")
			}
		})
	}
}

func TestRenderDebugError_TemplateError(t *testing.T) {
	// This test is tricky because the template is hardcoded and should always work
	// But we can test the fallback behavior by testing with nil writer or similar edge cases

	// Test with valid input - should work normally
	w := httptest.NewRecorder()
	messages := [][2]string{{"Test", "Value"}}

	RenderDebugError(w, messages)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusInternalServerError)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Test:") || !strings.Contains(body, "Value") {
		t.Errorf("Body should contain test message, got: %s", body)
	}
}

func TestWriteDebugError_DebugEnabled(t *testing.T) {
	// Save original state
	originalGlobal := debugGloballyEnabled
	originalEnv := os.Getenv("TMPL_CGI_DEBUG")
	defer func() {
		debugGloballyEnabled = originalGlobal
		if originalEnv == "" {
			_ = os.Unsetenv("TMPL_CGI_DEBUG")
		} else {
			_ = os.Setenv("TMPL_CGI_DEBUG", originalEnv)
		}
	}()

	// Enable debug mode
	debugGloballyEnabled = true

	w := httptest.NewRecorder()
	messages := [][2]string{
		{"Error", "Test error"},
	}

	WriteDebugError(w, messages)

	// Should render debug error
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusInternalServerError)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Debug Mode Enabled") {
		t.Error("Should render debug error page when debug is enabled")
	}
	if !strings.Contains(body, "Test error") {
		t.Error("Should contain the error message")
	}
}

func TestWriteDebugError_DebugDisabled(t *testing.T) {
	// Save original state
	originalGlobal := debugGloballyEnabled
	originalEnv := os.Getenv("TMPL_CGI_DEBUG")
	defer func() {
		debugGloballyEnabled = originalGlobal
		if originalEnv == "" {
			_ = os.Unsetenv("TMPL_CGI_DEBUG")
		} else {
			_ = os.Setenv("TMPL_CGI_DEBUG", originalEnv)
		}
	}()

	// Disable debug mode
	debugGloballyEnabled = false
	_ = os.Setenv("TMPL_CGI_DEBUG", "false")

	w := httptest.NewRecorder()
	messages := [][2]string{
		{"Error", "Test error"},
	}

	WriteDebugError(w, messages)

	// Should render simple error
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusInternalServerError)
	}

	body := w.Body.String()
	if strings.Contains(body, "Debug Mode Enabled") {
		t.Error("Should not render debug error page when debug is disabled")
	}
	if strings.Contains(body, "Test error") {
		t.Error("Should not contain the detailed error message when debug is disabled")
	}
	if !strings.Contains(body, "Server Error") {
		t.Error("Should contain generic server error message")
	}
	if !strings.Contains(body, "500 Server Error") {
		t.Error("Should contain 500 error title")
	}
}

func TestRenderDebugErrorAsCGIString(t *testing.T) {
	messages := [][2]string{
		{"Request URI", "/test"},
		{"Error", "Template not found"},
	}

	result := RenderDebugErrorAsCGIString(messages)

	// Should contain CGI headers
	if !strings.Contains(result, "Content-Type:") {
		t.Error("Result should contain Content-Type header")
	}

	// Should contain the error content
	if !strings.Contains(result, "Runtime Error") {
		t.Error("Result should contain error content")
	}
	if !strings.Contains(result, "Request URI:") {
		t.Error("Result should contain request URI")
	}
	if !strings.Contains(result, "/test") {
		t.Error("Result should contain the URI value")
	}
	if !strings.Contains(result, "Template not found") {
		t.Error("Result should contain the error message")
	}

	// Should have proper CGI format (headers, blank line, body)
	lines := strings.Split(result, "\r\n")
	if len(lines) < 3 {
		t.Error("CGI output should have headers, blank line, and body")
	}

	// Find the blank line separating headers from body
	blankLineFound := false
	for _, line := range lines {
		if line == "" {
			blankLineFound = true
			break
		}
	}
	if !blankLineFound {
		t.Error("CGI output should have blank line between headers and body")
	}
}

func TestDebugGlobalState(t *testing.T) {
	// Test that the global debug state is properly managed
	originalGlobal := debugGloballyEnabled
	defer func() {
		debugGloballyEnabled = originalGlobal
	}()

	// Test initial state
	debugGloballyEnabled = false
	if debugGloballyEnabled {
		t.Error("Global debug should be false initially")
	}

	// Test setting to true
	debugGloballyEnabled = true
	if !debugGloballyEnabled {
		t.Error("Global debug should be true after setting")
	}

	// Test that SetDebugMode affects the global state
	debugGloballyEnabled = false
	SetDebugMode()
	if !debugGloballyEnabled {
		t.Error("SetDebugMode should set global debug to true")
	}
}
