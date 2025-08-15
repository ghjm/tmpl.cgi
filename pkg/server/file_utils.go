package server

import (
	"html/template"
	"os"

	"github.com/Masterminds/sprig/v3"
)

// OSFileReader implements FileReader using os package
type OSFileReader struct{}

func (r *OSFileReader) ReadFile(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}

// OSTemplateLoader implements TemplateLoader using html/template package with Hugo-style functions
type OSTemplateLoader struct{}

func (l *OSTemplateLoader) ParseFiles(filenames ...string) (*template.Template, error) {
	// Create a new template with Sprig functions (Hugo-style templating)
	tmpl := template.New("").Funcs(sprig.FuncMap())
	return tmpl.ParseFiles(filenames...)
}
