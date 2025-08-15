package main

import (
	"flag"
	"log"
	"net/http"
	"net/http/cgi"
	"os"
	"strings"

	"gopkg.mhn.org/tmpl.cgi/pkg/server"
)

func main() {
	// Parse command line flags
	var syntaxCheck = flag.Bool("syntax-check", false, "Check template syntax and exit")
	var configPath = flag.String("config", "", "Path to configuration file")
	flag.Parse()

	// Get config file path from flag, environment, or use default
	if *configPath == "" {
		*configPath = os.Getenv("TMPL_CGI_CONFIG")
		if *configPath == "" {
			*configPath = "config.yaml"
		}
	}

	// If syntax check mode, run validation and exit
	if *syntaxCheck {
		if err := server.ValidateTemplates(*configPath); err != nil {
			log.Fatalf("Template validation failed: %v", err)
		}
		log.Println("All templates are valid!")
		return
	}

	// Create CGI server
	srv, err := server.New(*configPath)
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
