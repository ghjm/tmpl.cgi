package config

import (
	"testing"
)

func TestParseConfig(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		want    Config
		wantErr bool
	}{
		{
			name: "valid config with default template",
			yaml: `default_template: "default.html"
templates:
  - pattern: "^/api/.*"
    template: "api.html"
  - pattern: "^/admin/.*"
    template: "admin.html"`,
			want: Config{
				DefaultTemplate: "default.html",
				Templates: []struct {
					Pattern  string `yaml:"pattern"`
					Template string `yaml:"template"`
				}{
					{Pattern: "^/api/.*", Template: "api.html"},
					{Pattern: "^/admin/.*", Template: "admin.html"},
				},
			},
			wantErr: false,
		},
		{
			name: "config without default template",
			yaml: `templates:
  - pattern: "^/api/.*"
    template: "api.html"`,
			want: Config{
				DefaultTemplate: "",
				Templates: []struct {
					Pattern  string `yaml:"pattern"`
					Template string `yaml:"template"`
				}{
					{Pattern: "^/api/.*", Template: "api.html"},
				},
			},
			wantErr: false,
		},
		{
			name:    "invalid yaml",
			yaml:    `invalid: yaml: content: [`,
			want:    Config{},
			wantErr: true,
		},
		{
			name: "empty config",
			yaml: ``,
			want: Config{
				DefaultTemplate: "",
				Templates:       nil,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseConfig([]byte(tt.yaml))
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.DefaultTemplate != tt.want.DefaultTemplate {
					t.Errorf("ParseConfig() DefaultTemplate = %v, want %v", got.DefaultTemplate, tt.want.DefaultTemplate)
				}
				if len(got.Templates) != len(tt.want.Templates) {
					t.Errorf("ParseConfig() Templates length = %v, want %v", len(got.Templates), len(tt.want.Templates))
					return
				}
				for i, tmpl := range got.Templates {
					if tmpl.Pattern != tt.want.Templates[i].Pattern {
						t.Errorf("ParseConfig() Templates[%d].Pattern = %v, want %v", i, tmpl.Pattern, tt.want.Templates[i].Pattern)
					}
					if tmpl.Template != tt.want.Templates[i].Template {
						t.Errorf("ParseConfig() Templates[%d].Template = %v, want %v", i, tmpl.Template, tt.want.Templates[i].Template)
					}
				}
			}
		})
	}
}

func TestCompilePatterns(t *testing.T) {
	tests := []struct {
		name      string
		templates []struct {
			Pattern  string `yaml:"pattern"`
			Template string `yaml:"template"`
		}
		wantErr bool
	}{
		{
			name: "valid patterns",
			templates: []struct {
				Pattern  string `yaml:"pattern"`
				Template string `yaml:"template"`
			}{
				{Pattern: "^/api/.*", Template: "api.html"},
				{Pattern: "^/admin/.*", Template: "admin.html"},
				{Pattern: "^/blog/\\d+$", Template: "blog.html"},
			},
			wantErr: false,
		},
		{
			name: "invalid regex pattern",
			templates: []struct {
				Pattern  string `yaml:"pattern"`
				Template string `yaml:"template"`
			}{
				{Pattern: "^/api/.*", Template: "api.html"},
				{Pattern: "[invalid", Template: "invalid.html"},
			},
			wantErr: true,
		},
		{
			name:      "empty templates",
			templates: nil,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patterns, err := CompilePatterns(tt.templates)
			if (err != nil) != tt.wantErr {
				t.Errorf("CompilePatterns() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(patterns) != len(tt.templates) {
					t.Errorf("CompilePatterns() returned %d patterns, want %d", len(patterns), len(tt.templates))
				}
				// Test that patterns actually work
				for i, pattern := range patterns {
					if pattern == nil {
						t.Errorf("CompilePatterns() pattern[%d] is nil", i)
					}
				}
			}
		})
	}
}
