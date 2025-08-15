package server

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"regexp"

	"gopkg.mhn.org/tmpl.cgi/pkg/config"
)

// CGIServer handles CGI requests
type CGIServer struct {
	config         config.Config
	templates      map[string]*template.Template
	patterns       []*regexp.Regexp
	fileReader     FileReader
	templateLoader TemplateLoader
	configDir      string // Directory where config file is located
}

// New creates a new CGI server instance
func New(configPath string) (*CGIServer, error) {
	return NewWithDeps(configPath, &OSFileReader{}, &OSTemplateLoader{})
}

// NewWithDeps creates a new CGI server instance with injectable dependencies
func NewWithDeps(configPath string, fileReader FileReader, templateLoader TemplateLoader) (*CGIServer, error) {
	server := &CGIServer{
		templates:      make(map[string]*template.Template),
		fileReader:     fileReader,
		templateLoader: templateLoader,
		configDir:      filepath.Dir(configPath),
	}

	// Load configuration
	if err := server.loadConfig(configPath); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Load templates
	if err := server.loadTemplates(); err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	return server, nil
}

// loadConfig loads the configuration from YAML file
func (s *CGIServer) loadConfig(configPath string) error {
	data, err := s.fileReader.ReadFile(configPath)
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

	tmpl, err := s.templateLoader.ParseFiles(templatePath)
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

	// Set content type
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Execute template
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Template execution error for '%s': %v", templateName, err)
		http.Error(w, "Template execution failed", http.StatusInternalServerError)
		return
	}
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
