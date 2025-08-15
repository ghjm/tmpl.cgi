package config

import (
	"fmt"
	"regexp"

	"gopkg.in/yaml.v3"
)

// Config represents the configuration structure
type Config struct {
	DefaultTemplate string `yaml:"default_template"`
	Templates       []struct {
		Pattern  string `yaml:"pattern"`
		Template string `yaml:"template"`
	} `yaml:"templates"`
}

// TemplateData holds data passed to templates
type TemplateData struct {
	RequestURI string
	Request    interface{} // Using interface{} to avoid http import in tests
}

// ParseConfig parses YAML configuration data
func ParseConfig(data []byte) (Config, error) {
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return Config{}, fmt.Errorf("failed to parse config: %w", err)
	}
	return config, nil
}

// CompilePatterns compiles regex patterns from template configurations
func CompilePatterns(templates []struct {
	Pattern  string `yaml:"pattern"`
	Template string `yaml:"template"`
}) ([]*regexp.Regexp, error) {
	patterns := make([]*regexp.Regexp, len(templates))
	for i, tmpl := range templates {
		pattern, err := regexp.Compile(tmpl.Pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern '%s': %w", tmpl.Pattern, err)
		}
		patterns[i] = pattern
	}
	return patterns, nil
}
