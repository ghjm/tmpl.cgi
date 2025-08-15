package main

import (
	"flag"
	"fmt"
	"gopkg.mhn.org/tmpl.cgi/pkg/debug"
	"log"
	"net"
	"net/http"
	"net/http/cgi"
	"os"
	"strings"

	"gopkg.mhn.org/tmpl.cgi/pkg/server"
)

func fatalErr(stage string, err error) {
	if debug.IsDebugEnabled() {
		s := debug.RenderDebugErrorAsCGIString([][2]string{
			{"Result", "Failed to start server"},
			{"Stage", stage},
			{"Error", err.Error()},
		})
		fmt.Print(s)
		os.Exit(0)
	} else {
		log.Fatalf("%s failed: %v", stage, err)
	}
}

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
		if strings.Contains(err.Error(), "failed to load config") && strings.Contains(err.Error(), "no such file or directory") && os.Getenv("TMPL_CGI_CONFIG") != "" && !debug.IsDebugEnabled() {
			log.Fatalf("Config file 'config.yaml' not found.  Set TMPL_CGI_CONFIG or use -config to specify the config file to load.")
		}
		fatalErr("Creating CGI server", err)
	}

	// Check if running as CGI
	if os.Getenv("GATEWAY_INTERFACE") != "" {
		// Running as CGI
		if err := cgi.Serve(srv); err != nil {
			fatalErr("Serving CGI server", err)
		}
	} else {
		// Running as standalone server for testing
		port := os.Getenv("TMPL_CGI_PORT")
		if port == "" {
			port = "8080"
		}

		ln, err := net.Listen("tcp", ":"+port)
		if err != nil {
			fatalErr(fmt.Sprintf("Listening on port %s", port), err)
		}

		log.Printf("Starting test server on port %s", port)

		if err := http.Serve(ln, srv); err != nil {
			fatalErr("Error in test server", err)
		}
	}
}
