package config

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"

	"github.com/Masterminds/sprig/v3"
	"gopkg.in/yaml.v3"
)

type Template struct {
	Pattern  string `yaml:"pattern"`
	Template string `yaml:"template"`
	TestURI  string `yaml:"test_uri,omitempty"`
}

// Config represents the configuration structure
type Config struct {
	ConfigFilePath  string     `yaml:"-"`
	DefaultTemplate string     `yaml:"default_template"`
	Templates       []Template `yaml:"templates"`
	Data            any        `yaml:"data"`
}

// TemplateData holds data passed to templates
type TemplateData struct {
	RequestURI string
	Request    interface{} // Using interface{} to avoid http import in tests
	Data       any
}

// ParseConfigFile parses YAML configuration data from a file
func ParseConfigFile(filePath string) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}
	var config Config
	if err = yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}
	config.ConfigFilePath = filePath
	return &config, nil
}

// FindTemplate loads the appropriate template for a given URI
func (c *Config) FindTemplate(uri string) (*template.Template, error) {
	for _, t := range c.Templates {
		re, err := regexp.Compile(t.Pattern)
		if err != nil {
			return nil, fmt.Errorf("compiling regexp: %w", err)
		}
		if re.MatchString(uri) {
			return c.LoadTemplate(t.Template)
		}
	}
	return c.LoadTemplate(c.DefaultTemplate)
}

// LoadTemplate reads and parses a template file
func (c *Config) LoadTemplate(filename string) (*template.Template, error) {
	if !filepath.IsAbs(filename) {
		filename = filepath.Join(path.Dir(c.ConfigFilePath), filename)
	}
	tmpl, err := template.New(path.Base(filename)).Funcs(sprig.FuncMap()).ParseFiles(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to parse: %w", err)
	}
	return tmpl, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {

	// Validate that all regexes compile
	for _, t := range c.Templates {
		_, err := regexp.Compile(t.Pattern)
		if err != nil {
			return fmt.Errorf("compiling regex: %w", err)
		}
	}

	// Validate default template
	if err := c.validateTemplate(&Template{
		Template: c.DefaultTemplate,
		TestURI:  "/test/path",
	}); err != nil {
		return fmt.Errorf("default template '%s': %w", c.DefaultTemplate, err)
	}

	// Validate pattern-specific templates
	for _, t := range c.Templates {
		if err := c.validateTemplate(&t); err != nil {
			return fmt.Errorf("template '%s': %w", t.Template, err)
		}
	}

	return nil
}

// validateTemplate validates a single template file
func (c *Config) validateTemplate(t *Template) error {
	tmpl, err := c.LoadTemplate(t.Template)
	if err != nil {
		return fmt.Errorf("loading template: %w", err)
	}

	sampleData := &TemplateData{
		RequestURI: "/test/path",
		Data:       c.Data,
	}
	if t.TestURI != "" {
		sampleData.RequestURI = t.TestURI
	}
	sampleData.Request = createSampleRequest(sampleData.RequestURI)

	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, sampleData); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}

	return nil
}

// createSampleRequest creates a minimal HTTP request for template testing
func createSampleRequest(uri string) *http.Request {
	req, _ := http.NewRequest("GET", uri, nil)
	req.Host = "example.com"
	req.Header.Set("User-Agent", "Template-Validator/1.0")
	return req
}
