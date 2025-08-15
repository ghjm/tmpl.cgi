package server

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"github.com/Masterminds/sprig/v3"
	"gopkg.mhn.org/tmpl.cgi/pkg/debug"

	"gopkg.mhn.org/tmpl.cgi/pkg/config"
)

// CGIServer handles CGI requests
type CGIServer struct {
	config    config.Config
	templates map[string]*template.Template
	patterns  []*regexp.Regexp
	configDir string
}

// New creates a new CGI server instance
func New(configPath string) (*CGIServer, error) {
	server := &CGIServer{
		templates: make(map[string]*template.Template),
		configDir: filepath.Dir(configPath),
	}

	if err := server.loadConfig(configPath); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	if err := server.loadTemplates(); err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	return server, nil
}

// loadConfig loads the configuration from YAML file
func (s *CGIServer) loadConfig(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	cfg, err := config.ParseConfig(data)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	s.config = cfg

	// Compile regex patterns
	patterns, err := config.CompilePatterns(s.config.Templates)
	if err != nil {
		return err
	}
	s.patterns = patterns

	return nil
}

// loadTemplates loads all template files
func (s *CGIServer) loadTemplates() error {
	// Load default template
	if s.config.DefaultTemplate != "" {
		tmpl, err := s.loadTemplate(s.config.DefaultTemplate)
		if err != nil {
			return fmt.Errorf("failed to load default template: %w", err)
		}
		s.templates["default"] = tmpl
	}

	// Load pattern-specific templates
	for _, tmplConfig := range s.config.Templates {
		tmpl, err := s.loadTemplate(tmplConfig.Template)
		if err != nil {
			return fmt.Errorf("failed to load template '%s': %w", tmplConfig.Template, err)
		}
		s.templates[tmplConfig.Pattern] = tmpl
	}

	return nil
}

// loadTemplate loads a single template file
func (s *CGIServer) loadTemplate(templatePath string) (*template.Template, error) {
	// Check if path is absolute, if not make it relative to templates directory in config dir
	if !filepath.IsAbs(templatePath) {
		templatePath = filepath.Join(s.configDir, "templates", templatePath)
	}

	tmpl, err := template.New(filepath.Base(templatePath)).Funcs(sprig.FuncMap()).ParseFiles(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template file '%s': %w", templatePath, err)
	}

	return tmpl, nil
}

// ServeHTTP handles HTTP requests
func (s *CGIServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestURI := GetRequestURI(r)

	// Find matching template
	tmpl, templateName := s.FindTemplate(requestURI)

	// If no template found, return error
	if tmpl == nil {
		http.Error(w, "No template configured for this request", http.StatusNotFound)
		return
	}

	// Prepare template data
	data := config.TemplateData{
		RequestURI: requestURI,
		Request:    r,
	}

	// Execute template into a buffer first to catch debug before writing to response
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		log.Printf("Template execution error for '%s': %v", templateName, err)

		// Set content type for error responses
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		// Check if debug mode is enabled
		if debug.IsDebugEnabled() {
			debug.RenderDebugError(w, [][2]string{
				{"Template Name", templateName},
				{"Request URI", requestURI},
				{"Error", err.Error()},
			})
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Template execution failed"))
		}
		return
	}

	// Set content type and write successful response
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(buf.Bytes())
}

// FindTemplate finds the appropriate template for the given request URI
func (s *CGIServer) FindTemplate(requestURI string) (*template.Template, string) {
	// Check pattern-specific templates first
	for i, pattern := range s.patterns {
		if pattern.MatchString(requestURI) {
			tmpl := s.templates[s.config.Templates[i].Pattern]
			templateName := s.config.Templates[i].Template
			return tmpl, templateName
		}
	}

	// Fall back to default template if no pattern matches
	if defaultTmpl := s.templates["default"]; defaultTmpl != nil {
		return defaultTmpl, s.config.DefaultTemplate
	}

	return nil, ""
}

// GetRequestURI extracts the request URI from the HTTP request
func GetRequestURI(r *http.Request) string {
	requestURI := r.RequestURI
	if requestURI == "" {
		requestURI = r.URL.Path
	}
	return requestURI
}

// ValidateTemplates validates all templates in the configuration
func ValidateTemplates(configPath string) error {
	configDir := filepath.Dir(configPath)

	// Load configuration
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	cfg, err := config.ParseConfig(data)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate regex patterns
	_, err = config.CompilePatterns(cfg.Templates)
	if err != nil {
		return fmt.Errorf("invalid regex patterns: %w", err)
	}

	// Create sample template data for testing
	sampleData := config.TemplateData{
		RequestURI: "/test/path",
		Request:    createSampleRequest(),
	}

	// Validate default template if specified
	if cfg.DefaultTemplate != "" {
		if err := validateTemplate(cfg.DefaultTemplate, configDir, sampleData); err != nil {
			return fmt.Errorf("default template '%s': %w", cfg.DefaultTemplate, err)
		}
		log.Printf("✓ Default template '%s' is valid", cfg.DefaultTemplate)
	}

	// Validate pattern-specific templates
	for _, tmplConfig := range cfg.Templates {
		if err := validateTemplate(tmplConfig.Template, configDir, sampleData); err != nil {
			return fmt.Errorf("template '%s' (pattern: %s): %w", tmplConfig.Template, tmplConfig.Pattern, err)
		}
		log.Printf("✓ Template '%s' (pattern: %s) is valid", tmplConfig.Template, tmplConfig.Pattern)
	}

	return nil
}

// validateTemplate validates a single template file
func validateTemplate(templatePath string, configDir string, sampleData config.TemplateData) error {
	// Check if path is absolute, if not make it relative to templates directory in config dir
	if !filepath.IsAbs(templatePath) {
		templatePath = filepath.Join(configDir, "templates", templatePath)
	}

	// Parse the template
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return fmt.Errorf("failed to parse: %w", err)
	}

	// Test template execution with sample data
	var buf bytes.Buffer
	// Execute the template directly (same as production behavior)
	if err := tmpl.Execute(&buf, sampleData); err != nil {
		return fmt.Errorf("failed to execute with sample data: %w", err)
	}

	return nil
}

// createSampleRequest creates a minimal HTTP request for template testing
func createSampleRequest() *http.Request {
	req, _ := http.NewRequest("GET", "/test/path", nil)
	req.Host = "example.com"
	req.Header.Set("User-Agent", "Template-Validator/1.0")
	return req
}
