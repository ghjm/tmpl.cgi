package config

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseConfigFile(t *testing.T) {
	tests := []struct {
		name        string
		configYAML  string
		expectError bool
		expected    *Config
	}{
		{
			name: "Valid config",
			configYAML: `default_template: "default.html"
templates:
  - pattern: "^/api/.*"
    template: "api.html"
  - pattern: "^/admin/.*"
    template: "admin.html"
data:
  foo: bar
  number: 42`,
			expectError: false,
			expected: &Config{
				DefaultTemplate: "default.html",
				Templates: []Template{
					{Pattern: "^/api/.*", Template: "api.html"},
					{Pattern: "^/admin/.*", Template: "admin.html"},
				},
				Data: map[string]interface{}{
					"foo":    "bar",
					"number": 42,
				},
			},
		},
		{
			name:        "Minimal config",
			configYAML:  `default_template: "default.html"`,
			expectError: false,
			expected: &Config{
				DefaultTemplate: "default.html",
				Templates:       []Template{},
			},
		},
		{
			name: "Invalid YAML",
			configYAML: `default_template: "default.html"
invalid: yaml: content: [`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			tempFile, err := os.CreateTemp("", "config_test_*.yaml")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer func() { _ = os.Remove(tempFile.Name()) }()

			_, err = tempFile.WriteString(tt.configYAML)
			if err != nil {
				t.Fatalf("Failed to write config: %v", err)
			}
			_ = tempFile.Close()

			config, err := ParseConfigFile(tempFile.Name())

			if tt.expectError {
				if err == nil {
					t.Errorf("ParseConfigFile() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseConfigFile() unexpected error: %v", err)
				return
			}

			if config.ConfigFilePath != tempFile.Name() {
				t.Errorf("ConfigFilePath = %s, want %s", config.ConfigFilePath, tempFile.Name())
			}

			if config.DefaultTemplate != tt.expected.DefaultTemplate {
				t.Errorf("DefaultTemplate = %s, want %s", config.DefaultTemplate, tt.expected.DefaultTemplate)
			}

			if len(config.Templates) != len(tt.expected.Templates) {
				t.Errorf("Templates length = %d, want %d", len(config.Templates), len(tt.expected.Templates))
			}

			for i, tmpl := range config.Templates {
				if i >= len(tt.expected.Templates) {
					break
				}
				expected := tt.expected.Templates[i]
				if tmpl.Pattern != expected.Pattern {
					t.Errorf("Template[%d].Pattern = %s, want %s", i, tmpl.Pattern, expected.Pattern)
				}
				if tmpl.Template != expected.Template {
					t.Errorf("Template[%d].Template = %s, want %s", i, tmpl.Template, expected.Template)
				}
			}
		})
	}
}

func TestParseConfigFile_FileNotFound(t *testing.T) {
	_, err := ParseConfigFile("/nonexistent/config.yaml")
	if err == nil {
		t.Error("ParseConfigFile() with nonexistent file should return error")
	}
	if !strings.Contains(err.Error(), "reading config file") {
		t.Errorf("Error should mention 'reading config file', got: %v", err)
	}
}

func TestFindTemplate(t *testing.T) {
	// Create temporary directory and templates
	tempDir := t.TempDir()

	// Create test templates
	templates := map[string]string{
		"default.html": `<html><body>Default: {{.RequestURI}}</body></html>`,
		"api.html":     `<html><body>API: {{.RequestURI}}</body></html>`,
		"admin.html":   `<html><body>Admin: {{.RequestURI}}</body></html>`,
	}

	for name, content := range templates {
		path := filepath.Join(tempDir, name)
		err := os.WriteFile(path, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create template %s: %v", name, err)
		}
	}

	config := &Config{
		ConfigFilePath:  filepath.Join(tempDir, "config.yaml"),
		DefaultTemplate: "default.html",
		Templates: []Template{
			{Pattern: "^/api/.*", Template: "api.html"},
			{Pattern: "^/admin/.*", Template: "admin.html"},
			{Pattern: "^/blog/\\d+$", Template: "api.html"}, // Reuse api.html
		},
	}

	tests := []struct {
		name         string
		uri          string
		expectedName string
	}{
		{
			name:         "API pattern match",
			uri:          "/api/users",
			expectedName: "api.html",
		},
		{
			name:         "Admin pattern match",
			uri:          "/admin/dashboard",
			expectedName: "admin.html",
		},
		{
			name:         "Blog pattern match",
			uri:          "/blog/123",
			expectedName: "api.html",
		},
		{
			name:         "No pattern match - use default",
			uri:          "/home",
			expectedName: "default.html",
		},
		{
			name:         "Root path - use default",
			uri:          "/",
			expectedName: "default.html",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := config.FindTemplate(tt.uri)
			if err != nil {
				t.Errorf("FindTemplate() error: %v", err)
				return
			}

			if tmpl == nil {
				t.Error("FindTemplate() returned nil template")
				return
			}

			// Check that the correct template was loaded by checking its name
			if tmpl.Name() != tt.expectedName {
				t.Errorf("FindTemplate() template name = %s, want %s", tmpl.Name(), tt.expectedName)
			}
		})
	}
}

func TestFindTemplate_InvalidRegex(t *testing.T) {
	config := &Config{
		ConfigFilePath:  "/tmp/config.yaml",
		DefaultTemplate: "default.html",
		Templates: []Template{
			{Pattern: "[invalid regex", Template: "api.html"},
		},
	}

	_, err := config.FindTemplate("/api/test")
	if err == nil {
		t.Error("FindTemplate() with invalid regex should return error")
	}
	if !strings.Contains(err.Error(), "compiling regexp") {
		t.Errorf("Error should mention 'compiling regexp', got: %v", err)
	}
}

func TestLoadTemplate(t *testing.T) {
	tempDir := t.TempDir()

	templateContent := `<html><body>Hello {{.RequestURI}}</body></html>`
	templatePath := filepath.Join(tempDir, "test.html")
	err := os.WriteFile(templatePath, []byte(templateContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test template: %v", err)
	}

	config := &Config{
		ConfigFilePath: filepath.Join(tempDir, "config.yaml"),
	}

	tests := []struct {
		name        string
		filename    string
		expectError bool
	}{
		{
			name:        "Relative path",
			filename:    "test.html",
			expectError: false,
		},
		{
			name:        "Absolute path",
			filename:    templatePath,
			expectError: false,
		},
		{
			name:        "Nonexistent file",
			filename:    "nonexistent.html",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := config.LoadTemplate(tt.filename)

			if tt.expectError {
				if err == nil {
					t.Error("LoadTemplate() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("LoadTemplate() unexpected error: %v", err)
				return
			}

			if tmpl == nil {
				t.Error("LoadTemplate() returned nil template")
				return
			}

			// Test that the template can be executed
			data := TemplateData{RequestURI: "/test"}
			var buf strings.Builder
			err = tmpl.Execute(&buf, data)
			if err != nil {
				t.Errorf("Template execution failed: %v", err)
			}

			result := buf.String()
			if !strings.Contains(result, "/test") {
				t.Errorf("Template output should contain '/test', got: %s", result)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tempDir := t.TempDir()

	// Create valid templates
	validTemplate := `<html><body>Valid: {{.RequestURI}}</body></html>`
	invalidTemplate := `<html><body>Invalid: {{.NonExistent.Field}}</body></html>`

	validPath := filepath.Join(tempDir, "valid.html")
	invalidPath := filepath.Join(tempDir, "invalid.html")

	err := os.WriteFile(validPath, []byte(validTemplate), 0644)
	if err != nil {
		t.Fatalf("Failed to create valid template: %v", err)
	}

	err = os.WriteFile(invalidPath, []byte(invalidTemplate), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid template: %v", err)
	}

	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorText   string
	}{
		{
			name: "Valid config",
			config: &Config{
				ConfigFilePath:  filepath.Join(tempDir, "config.yaml"),
				DefaultTemplate: "valid.html",
				Templates: []Template{
					{Pattern: "^/api/.*", Template: "valid.html"},
				},
			},
			expectError: false,
		},
		{
			name: "Invalid regex pattern",
			config: &Config{
				ConfigFilePath:  filepath.Join(tempDir, "config.yaml"),
				DefaultTemplate: "valid.html",
				Templates: []Template{
					{Pattern: "[invalid", Template: "valid.html"},
				},
			},
			expectError: true,
			errorText:   "compiling regex",
		},
		{
			name: "Invalid default template",
			config: &Config{
				ConfigFilePath:  filepath.Join(tempDir, "config.yaml"),
				DefaultTemplate: "nonexistent.html",
			},
			expectError: true,
			errorText:   "default template",
		},
		{
			name: "Template execution error",
			config: &Config{
				ConfigFilePath:  filepath.Join(tempDir, "config.yaml"),
				DefaultTemplate: "invalid.html",
			},
			expectError: true,
			errorText:   "default template",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				if err == nil {
					t.Error("Validate() expected error, got nil")
					return
				}
				if tt.errorText != "" && !strings.Contains(err.Error(), tt.errorText) {
					t.Errorf("Error should contain '%s', got: %v", tt.errorText, err)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateTemplate(t *testing.T) {
	tempDir := t.TempDir()

	validTemplate := `<html><body>Valid: {{.RequestURI}}</body></html>`
	validPath := filepath.Join(tempDir, "valid.html")
	err := os.WriteFile(validPath, []byte(validTemplate), 0644)
	if err != nil {
		t.Fatalf("Failed to create valid template: %v", err)
	}

	config := &Config{
		ConfigFilePath: filepath.Join(tempDir, "config.yaml"),
		Data: map[string]interface{}{
			"test": "value",
		},
	}

	tests := []struct {
		name        string
		template    *Template
		expectError bool
	}{
		{
			name: "Valid template",
			template: &Template{
				Template: "valid.html",
				TestURI:  "/test",
			},
			expectError: false,
		},
		{
			name: "Valid template with default test URI",
			template: &Template{
				Template: "valid.html",
			},
			expectError: false,
		},
		{
			name: "Nonexistent template",
			template: &Template{
				Template: "nonexistent.html",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := config.validateTemplate(tt.template)

			if tt.expectError {
				if err == nil {
					t.Error("validateTemplate() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("validateTemplate() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestCreateSampleRequest(t *testing.T) {
	tests := []struct {
		name string
		uri  string
	}{
		{
			name: "Root path",
			uri:  "/",
		},
		{
			name: "API path",
			uri:  "/api/users",
		},
		{
			name: "Complex path",
			uri:  "/blog/2023/12/25/christmas",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createSampleRequest(tt.uri)

			if req == nil {
				t.Error("createSampleRequest() returned nil")
				return
			}

			if req.Method != "GET" {
				t.Errorf("Method = %s, want GET", req.Method)
			}

			if req.URL.Path != tt.uri {
				t.Errorf("URL.Path = %s, want %s", req.URL.Path, tt.uri)
			}

			if req.Host != "example.com" {
				t.Errorf("Host = %s, want example.com", req.Host)
			}

			userAgent := req.Header.Get("User-Agent")
			if userAgent != "Template-Validator/1.0" {
				t.Errorf("User-Agent = %s, want Template-Validator/1.0", userAgent)
			}
		})
	}
}

func TestTemplateData(t *testing.T) {
	// Test that TemplateData struct works as expected
	req, _ := http.NewRequest("GET", "/test", nil)
	data := TemplateData{
		RequestURI: "/test/path",
		Request:    req,
		Data: map[string]interface{}{
			"key": "value",
		},
	}

	if data.RequestURI != "/test/path" {
		t.Errorf("RequestURI = %s, want /test/path", data.RequestURI)
	}

	if data.Request != req {
		t.Error("Request field not set correctly")
	}

	if dataMap, ok := data.Data.(map[string]interface{}); ok {
		if dataMap["key"] != "value" {
			t.Errorf("Data['key'] = %v, want 'value'", dataMap["key"])
		}
	} else {
		t.Error("Data field is not the expected type")
	}
}

func TestTemplate_Struct(t *testing.T) {
	// Test Template struct
	tmpl := Template{
		Pattern:  "^/api/.*",
		Template: "api.html",
		TestURI:  "/api/test",
	}

	if tmpl.Pattern != "^/api/.*" {
		t.Errorf("Pattern = %s, want ^/api/.*", tmpl.Pattern)
	}

	if tmpl.Template != "api.html" {
		t.Errorf("Template = %s, want api.html", tmpl.Template)
	}

	if tmpl.TestURI != "/api/test" {
		t.Errorf("TestURI = %s, want /api/test", tmpl.TestURI)
	}
}

func TestConfig_Struct(t *testing.T) {
	// Test Config struct
	config := Config{
		ConfigFilePath:  "/path/to/config.yaml",
		DefaultTemplate: "default.html",
		Templates: []Template{
			{Pattern: "^/api/.*", Template: "api.html"},
		},
		Data: map[string]interface{}{
			"key": "value",
		},
	}

	if config.ConfigFilePath != "/path/to/config.yaml" {
		t.Errorf("ConfigFilePath = %s, want /path/to/config.yaml", config.ConfigFilePath)
	}

	if config.DefaultTemplate != "default.html" {
		t.Errorf("DefaultTemplate = %s, want default.html", config.DefaultTemplate)
	}

	if len(config.Templates) != 1 {
		t.Errorf("Templates length = %d, want 1", len(config.Templates))
	}

	if dataMap, ok := config.Data.(map[string]interface{}); ok {
		if dataMap["key"] != "value" {
			t.Errorf("Data['key'] = %v, want 'value'", dataMap["key"])
		}
	} else {
		t.Error("Data field is not the expected type")
	}
}
