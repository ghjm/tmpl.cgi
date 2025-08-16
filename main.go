package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"gopkg.mhn.org/tmpl.cgi/pkg/config"
	"gopkg.mhn.org/tmpl.cgi/pkg/debug"

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
	var validate = flag.Bool("validate", false, "Validate configuration and exit")
	var configPath = flag.String("config", "", "Path to configuration file")
	flag.Parse()

	// Get config file path from flag, environment, or use default
	if *configPath == "" {
		*configPath = os.Getenv("TMPL_CGI_CONFIG")
		if *configPath == "" {
			*configPath = "config.yaml"
		}
	}

	cfg, err := config.ParseConfigFile(*configPath)
	if err != nil {
		fatalErr("Failed to parse configuration file: %v", err)
	}

	// If syntax check mode, run validation and exit
	if *validate {
		err = cfg.Validate()
		if err != nil {
			fatalErr("Config validation failed: %v", err)
		}
		log.Println("All templates are valid!")
		return
	}

	// Create CGI server
	srv, err := server.New(cfg)
	if err != nil {
		fatalErr("Creating CGI server", err)
	}

	err = srv.Run()
	if err != nil {
		fatalErr("Running CGI server", err)
	}
}
