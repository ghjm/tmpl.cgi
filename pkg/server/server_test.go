package server

import (
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// MockFileReader for testing
type MockFileReader struct {
	files map[string][]byte
	err   error
}

func (m *MockFileReader) ReadFile(filename string) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	if data, exists := m.files[filename]; exists {
		return data, nil
	}
	return nil, fmt.Errorf("file not found: %s", filename)
}

// MockTemplateLoader for testing
type MockTemplateLoader struct {
	templates map[string]*template.Template
	err       error
}

func (m *MockTemplateLoader) ParseFiles(filenames ...string) (*template.Template, error) {
	if m.err != nil {
		return nil, m.err
	}
	if len(filenames) == 0 {
		return nil, fmt.Errorf("no filenames provided")
	}
	filename := filenames[0]
	if tmpl, exists := m.templates[filename]; exists {
		return tmpl, nil
	}
	return nil, fmt.Errorf("template not found: %s", filename)
}

func TestNewWithDeps(t *testing.T) {
	validConfig := `default_template: "default.html"
templates:
  - pattern: "^/api/.*"
    template: "api.html"`

	defaultTemplate := template.Must(template.New("default.html").Parse("<h1>Default: {{.RequestURI}}</h1>"))
	apiTemplate := template.Must(template.New("api.html").Parse("<h1>API: {{.RequestURI}}</h1>"))

	tests := []struct {
		name           string
		configPath     string
		fileReader     *MockFileReader
		templateLoader *MockTemplateLoader
		wantErr        bool
	}{
		{
			name:       "successful creation",
			configPath: "config.yaml",
			fileReader: &MockFileReader{
				files: map[string][]byte{
					"config.yaml": []byte(validConfig),
				},
			},
			templateLoader: &MockTemplateLoader{
				templates: map[string]*template.Template{
					"templates/default.html": defaultTemplate,
					"templates/api.html":     apiTemplate,
				},
			},
			wantErr: false,
		},
		{
			name:       "config file not found",
			configPath: "nonexistent.yaml",
			fileReader: &MockFileReader{
				files: map[string][]byte{},
			},
			templateLoader: &MockTemplateLoader{},
			wantErr:        true,
		},
		{
			name:       "invalid config",
			configPath: "config.yaml",
			fileReader: &MockFileReader{
				files: map[string][]byte{
					"config.yaml": []byte("invalid: yaml: ["),
				},
			},
			templateLoader: &MockTemplateLoader{},
			wantErr:        true,
		},
		{
			name:       "template loading error",
			configPath: "config.yaml",
			fileReader: &MockFileReader{
				files: map[string][]byte{
					"config.yaml": []byte(validConfig),
				},
			},
			templateLoader: &MockTemplateLoader{
				err: fmt.Errorf("template loading failed"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := NewWithDeps(tt.configPath, tt.fileReader, tt.templateLoader)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewWithDeps() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && server == nil {
				t.Error("NewWithDeps() returned nil server without error")
			}
		})
	}
}

func TestCGIServer_FindTemplate(t *testing.T) {
	// Create test templates
	defaultTemplate := template.Must(template.New("default.html").Parse("<h1>Default</h1>"))
	apiTemplate := template.Must(template.New("api.html").Parse("<h1>API</h1>"))
	adminTemplate := template.Must(template.New("admin.html").Parse("<h1>Admin</h1>"))

	validConfig := `default_template: "default.html"
templates:
  - pattern: "^/api/.*"
    template: "api.html"
  - pattern: "^/admin/.*"
    template: "admin.html"`

	fileReader := &MockFileReader{
		files: map[string][]byte{
			"config.yaml": []byte(validConfig),
		},
	}

	templateLoader := &MockTemplateLoader{
		templates: map[string]*template.Template{
			"templates/default.html": defaultTemplate,
			"templates/api.html":     apiTemplate,
			"templates/admin.html":   adminTemplate,
		},
	}

	server, err := NewWithDeps("config.yaml", fileReader, templateLoader)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	tests := []struct {
		name         string
		requestURI   string
		wantTemplate *template.Template
		wantName     string
	}{
		{
			name:         "api pattern match",
			requestURI:   "/api/users",
			wantTemplate: apiTemplate,
			wantName:     "api.html",
		},
		{
			name:         "admin pattern match",
			requestURI:   "/admin/dashboard",
			wantTemplate: adminTemplate,
			wantName:     "admin.html",
		},
		{
			name:         "default template fallback",
			requestURI:   "/home",
			wantTemplate: defaultTemplate,
			wantName:     "default.html",
		},
		{
			name:         "root path uses default",
			requestURI:   "/",
			wantTemplate: defaultTemplate,
			wantName:     "default.html",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTemplate, gotName := server.FindTemplate(tt.requestURI)
			if gotTemplate != tt.wantTemplate {
				t.Errorf("FindTemplate() gotTemplate = %v, want %v", gotTemplate, tt.wantTemplate)
			}
			if gotName != tt.wantName {
				t.Errorf("FindTemplate() gotName = %v, want %v", gotName, tt.wantName)
			}
		})
	}
}

func TestCGIServer_ServeHTTP(t *testing.T) {
	// Create test templates
	defaultTemplate := template.Must(template.New("default.html").Parse("<h1>Default: {{.RequestURI}}</h1>"))
	apiTemplate := template.Must(template.New("api.html").Parse("<h1>API: {{.RequestURI}}</h1>"))

	validConfig := `default_template: "default.html"
templates:
  - pattern: "^/api/.*"
    template: "api.html"`

	fileReader := &MockFileReader{
		files: map[string][]byte{
			"config.yaml": []byte(validConfig),
		},
	}

	templateLoader := &MockTemplateLoader{
		templates: map[string]*template.Template{
			"templates/default.html": defaultTemplate,
			"templates/api.html":     apiTemplate,
		},
	}

	server, err := NewWithDeps("config.yaml", fileReader, templateLoader)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	tests := []struct {
		name           string
		requestURI     string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "api endpoint",
			requestURI:     "/api/users",
			expectedStatus: http.StatusOK,
			expectedBody:   "<h1>API: /api/users</h1>",
		},
		{
			name:           "default template",
			requestURI:     "/home",
			expectedStatus: http.StatusOK,
			expectedBody:   "<h1>Default: /home</h1>",
		},
		{
			name:           "root path",
			requestURI:     "/",
			expectedStatus: http.StatusOK,
			expectedBody:   "<h1>Default: /</h1>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.requestURI, nil)
			w := httptest.NewRecorder()

			server.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("ServeHTTP() status = %v, want %v", w.Code, tt.expectedStatus)
			}

			body := strings.TrimSpace(w.Body.String())
			if body != tt.expectedBody {
				t.Errorf("ServeHTTP() body = %v, want %v", body, tt.expectedBody)
			}

			// Check content type
			if w.Header().Get("Content-Type") != "text/html; charset=utf-8" {
				t.Errorf("ServeHTTP() Content-Type = %v, want %v", w.Header().Get("Content-Type"), "text/html; charset=utf-8")
			}
		})
	}
}

func TestCGIServer_ServeHTTP_NoTemplate(t *testing.T) {
	// Server with no templates configured
	validConfig := `templates: []`

	fileReader := &MockFileReader{
		files: map[string][]byte{
			"config.yaml": []byte(validConfig),
		},
	}

	templateLoader := &MockTemplateLoader{
		templates: map[string]*template.Template{},
	}

	server, err := NewWithDeps("config.yaml", fileReader, templateLoader)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("ServeHTTP() status = %v, want %v", w.Code, http.StatusNotFound)
	}

	expectedBody := "No template configured for this request\n"
	if w.Body.String() != expectedBody {
		t.Errorf("ServeHTTP() body = %v, want %v", w.Body.String(), expectedBody)
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
			name:       "request URI present",
			requestURI: "/api/users?page=1",
			urlPath:    "/api/users",
			expected:   "/api/users?page=1",
		},
		{
			name:       "request URI empty, use URL path",
			requestURI: "",
			urlPath:    "/api/users",
			expected:   "/api/users",
		},
		{
			name:       "both present, prefer request URI",
			requestURI: "/api/users?page=1",
			urlPath:    "/api/users",
			expected:   "/api/users?page=1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://example.com"+tt.urlPath, nil)
			req.RequestURI = tt.requestURI

			result := GetRequestURI(req)
			if result != tt.expected {
				t.Errorf("GetRequestURI() = %v, want %v", result, tt.expected)
			}
		})
	}
}
