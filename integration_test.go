package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gopkg.mhn.org/tmpl.cgi/pkg/config"
	"gopkg.mhn.org/tmpl.cgi/pkg/server"
)

func TestIntegration_WithTestData(t *testing.T) {
	// Load the test configuration
	cfg, err := config.ParseConfigFile("testdata/config.yaml")
	if err != nil {
		t.Fatalf("Failed to load test config: %v", err)
	}

	// Validate the configuration
	err = cfg.Validate()
	if err != nil {
		t.Fatalf("Config validation failed: %v", err)
	}

	// Create server
	srv, err := server.New(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	tests := []struct {
		name         string
		path         string
		expectedText []string
	}{
		{
			name: "Default template",
			path: "/home",
			expectedText: []string{
				"Default Template",
				"Request URI: /home",
				"This is the default template",
			},
		},
		{
			name: "API template",
			path: "/api/users",
			expectedText: []string{
				"API Documentation",
				"Request URI: /api/users",
				"This is the API documentation template",
			},
		},
		{
			name: "Admin template",
			path: "/admin/dashboard",
			expectedText: []string{
				"Admin Panel",
				"Request URI: /admin/dashboard",
				"Welcome to the admin panel",
			},
		},
		{
			name: "Blog post template",
			path: "/blog/123",
			expectedText: []string{
				"Blog Post",
				"Request URI: /blog/123",
				"Blog Post #123",
			},
		},
		{
			name: "Data template",
			path: "/data/test",
			expectedText: []string{
				"Data Template",
				"Request URI: /data/test",
				"foo: bar", // From config data
				"bar: baz", // From config data
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://example.com"+tt.path, nil)
			req.RequestURI = tt.path
			w := httptest.NewRecorder()

			srv.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			body := w.Body.String()
			for _, expected := range tt.expectedText {
				if !strings.Contains(body, expected) {
					t.Errorf("Response should contain %q, got: %s", expected, body)
				}
			}

			// Check content type
			contentType := w.Header().Get("Content-Type")
			if contentType != "text/html; charset=utf-8" {
				t.Errorf("Expected Content-Type text/html; charset=utf-8, got %s", contentType)
			}
		})
	}
}

func TestIntegration_ConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		configFile  string
		expectError bool
	}{
		{
			name:        "Valid config",
			configFile:  "testdata/config.yaml",
			expectError: false,
		},
		{
			name:        "Invalid config",
			configFile:  "testdata/config_invalid.yaml",
			expectError: true,
		},
		{
			name:        "Config without default template",
			configFile:  "testdata/config_no_default.yaml",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := config.ParseConfigFile(tt.configFile)
			if err != nil {
				if !tt.expectError {
					t.Errorf("Unexpected error parsing config: %v", err)
				}
				return
			}

			err = cfg.Validate()
			if tt.expectError {
				if err == nil {
					t.Error("Expected validation error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected validation error: %v", err)
				}
			}
		})
	}
}
