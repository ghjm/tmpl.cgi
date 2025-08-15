package server

import (
	"html/template"
	"os"
)

// OSFileReader implements FileReader using os package
type OSFileReader struct{}

func (r *OSFileReader) ReadFile(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}

// OSTemplateLoader implements TemplateLoader using html/template package
type OSTemplateLoader struct{}

func (l *OSTemplateLoader) ParseFiles(filenames ...string) (*template.Template, error) {
	return template.ParseFiles(filenames...)
}
