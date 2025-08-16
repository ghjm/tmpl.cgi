package server

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/cgi"
	"os"

	"gopkg.mhn.org/tmpl.cgi/pkg/config"
	"gopkg.mhn.org/tmpl.cgi/pkg/debug"
)

// CGIServer handles CGI requests
type CGIServer struct {
	config config.Config
}

// New creates a new CGI server instance
func New(cfg *config.Config) (*CGIServer, error) {
	return &CGIServer{config: *cfg}, nil
}

func (s *CGIServer) Run() error {
	// Check if running as CGI
	if os.Getenv("GATEWAY_INTERFACE") != "" {
		// Running as CGI
		err := cgi.Serve(s)
		if err != nil {
			return fmt.Errorf("serving CGI server: %v", err)
		}
	} else {
		// Running as standalone server for testing
		debug.SetDebugMode()
		port := os.Getenv("TMPL_CGI_PORT")
		if port == "" {
			port = "8080"
		}

		ln, err := net.Listen("tcp", ":"+port)
		if err != nil {
			return fmt.Errorf("listening on port %s: %v", port, err)
		}

		log.Printf("Starting test server on port %s", port)

		err = http.Serve(ln, s)
		if err != nil {
			return fmt.Errorf("serving debug server: %v", err)
		}

	}
	return nil
}

// ServeHTTP handles HTTP requests
func (s *CGIServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestURI := getRequestURI(r)
	tmpl, err := s.config.FindTemplate(requestURI)
	if err != nil {
		log.Printf("loading template: %v", err)
		debug.WriteDebugError(w, [][2]string{{"Request URI", requestURI}, {"Error loading template", err.Error()}})
		return
	}
	data := config.TemplateData{
		RequestURI: requestURI,
		Request:    r,
		Data:       s.config.Data,
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		log.Printf("executing template: %v", err)
		debug.WriteDebugError(w, [][2]string{{"Request URI", requestURI}, {"Error executing template", err.Error()}})
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(buf.Bytes())
}

// getRequestURI extracts the request URI from the HTTP request
func getRequestURI(r *http.Request) string {
	requestURI := r.RequestURI
	if requestURI == "" {
		requestURI = r.URL.Path
	}
	return requestURI
}
