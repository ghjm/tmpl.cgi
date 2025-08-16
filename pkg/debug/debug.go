package debug

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strings"

	"gopkg.mhn.org/tmpl.cgi/pkg/cgicapture"
)

var debugGloballyEnabled bool

// IsDebugEnabled checks if debug mode is enabled via TMPL_CGI_DEBUG environment variable
func IsDebugEnabled() bool {
	if debugGloballyEnabled {
		return true
	}
	debug := strings.ToLower(os.Getenv("TMPL_CGI_DEBUG"))
	return debug == "true" || debug == "yes" || debug == "1"
}

// SetDebugMode turns on debug mode globally
func SetDebugMode() {
	debugGloballyEnabled = true
}

func RenderDebugErrorAsCGIString(messages [][2]string) string {
	return cgicapture.CaptureFuncCGI(func(writer http.ResponseWriter) {
		RenderDebugError(writer, messages)
	})
}

// RenderDebugError renders a detailed error page
func RenderDebugError(w http.ResponseWriter, messages [][2]string) {
	debugTemplate := `<!DOCTYPE html>
<html>
<head>
    <title>Runtime Error - Debug Mode</title>
    <style>
        body { font-family: monospace; margin: 20px; background-color: #f5f5f5; }
        .error-container { background-color: white; padding: 20px; border-left: 5px solid #d32f2f; }
        .error-title { color: #d32f2f; font-size: 24px; margin-bottom: 20px; }
        .error-section { margin-bottom: 20px; }
        .error-label { font-weight: bold; color: #333; }
        .error-value { background-color: #f8f8f8; padding: 10px; border: 1px solid #ddd; white-space: pre-wrap; }
        .warning { background-color: #fff3cd; border: 1px solid #ffeaa7; padding: 10px; margin-bottom: 20px; }
    </style>
</head>
<body>
    <div class="warning">
        <strong>Debug Mode Enabled:</strong> This detailed error information is shown because TMPL_CGI_DEBUG is set to a true-like value (true, yes, 1). 
        In production, disable debug mode by unsetting or setting TMPL_CGI_DEBUG to false.
    </div>
    <div class="error-container">
        <div class="error-title">Runtime Error</div>
		{{range .}}
        <div class="error-section">
            <div class="error-label">{{index . 0}}:</div>
            <div class="error-value">{{index . 1}}</div>
        </div>
		{{end}}
    </div>
</body>
</html>`
	var buf bytes.Buffer
	tmpl, err := template.New("debug-error").Parse(debugTemplate)
	if err == nil {
		err = tmpl.Execute(&buf, messages)
	}
	if err != nil {
		// Fallback to plain text if template parsing fails
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintf(w, "Debug template error: %v\n\nMessages:\n", err)
		for _, v := range messages {
			_, _ = fmt.Fprintf(w, "%s: %s\n", v[0], v[1])
		}
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)
	_, _ = buf.WriteTo(w)
}

func WriteDebugError(w http.ResponseWriter, messages [][2]string) {
	if IsDebugEnabled() {
		RenderDebugError(w, messages)
	} else {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`<!DOCTYPE HTML PUBLIC "-//IETF//DTD HTML 2.0//EN">
<html><head>
<title>500 Server Error</title>
</head><body>
<h1>Server Error</h1>
<p>The server encountered an error processing this request.</p>
</body></html>`))
	}
}
