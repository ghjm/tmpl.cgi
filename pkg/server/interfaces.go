package server

import "html/template"

// FileReader interface for reading files (for testing)
type FileReader interface {
	ReadFile(filename string) ([]byte, error)
}

// TemplateLoader interface for loading templates with Hugo-style functions (for testing)
type TemplateLoader interface {
	ParseFiles(filenames ...string) (*template.Template, error)
}
