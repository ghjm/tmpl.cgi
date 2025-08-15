package main

import (
	"log"
	"net/http"
	"net/http/cgi"
	"os"
	"strings"

	"gopkg.mhn.org/tmpl.cgi/pkg/server"
)

func main() {
	// Get config file path from environment or use default
	configPath := os.Getenv("TMPL_CGI_CONFIG")
	if configPath == "" {
		configPath = "config.yaml"
	}

	// Create CGI server
	srv, err := server.New(configPath)
	if err != nil {
		// Check if it's a config file not found error and provide helpful message
		if strings.Contains(err.Error(), "failed to load config") && strings.Contains(err.Error(), "no such file or directory") {
			if os.Getenv("TMPL_CGI_CONFIG") == "" {
				log.Fatalf("Config file 'config.yaml' not found. Set TMPL_CGI_CONFIG to specify the config file to load.")
			}
		}
		log.Fatalf("Failed to create CGI server: %v", err)
	}

	// Check if running as CGI
	if os.Getenv("GATEWAY_INTERFACE") != "" {
		// Running as CGI
		if err := cgi.Serve(srv); err != nil {
			log.Fatalf("CGI server error: %v", err)
		}
	} else {
		// Running as standalone server for testing
		port := os.Getenv("TMPL_CGI_PORT")
		if port == "" {
			port = "8080"
		}

		log.Printf("Starting test server on port %s", port)

		if err := http.ListenAndServe(":"+port, srv); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}
}
