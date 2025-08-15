# CGI Template Server

A Go-based CGI server that serves HTML templates based on request URI patterns. The server can be configured to serve different templates for different URL patterns using regular expressions.

## Features

- **Pattern-based template routing**: Configure different templates for different URL patterns using regex
- **Default template fallback**: Specify a default template for requests that don't match any patterns
- **Template data access**: Templates can access the request URI and full request object
- **CGI and standalone modes**: Can run as a CGI script or standalone HTTP server for testing
- **YAML configuration**: Easy-to-edit configuration file

## Configuration

Edit `config.yaml` to configure templates and URL patterns:

```yaml
# Default template when no patterns match
default_template: "default.html"

# Template patterns - first match wins
templates:
  - pattern: "^/api/.*"
    template: "api.html"
  
  - pattern: "^/user/[^/]+$"
    template: "user.html"
  
  - pattern: "^/admin/.*"
    template: "admin.html"
  
  - pattern: "^/(about|contact|help)$"
    template: "static.html"
```

### Configuration Options

- `default_template`: Template file to use when no patterns match
- `templates`: Array of pattern-template mappings
  - `pattern`: Regular expression to match against request URI
  - `template`: Template file to use for matching requests

## Template Data

Templates receive a data structure with the following fields:

```go
type TemplateData struct {
    RequestURI string        // The request URI (e.g., "/api/users")
    Request    *http.Request // Full HTTP request object
}
```

### Template Examples

Access request URI in templates:
```html
<p>Current path: {{.RequestURI}}</p>
```

Access request details:
```html
<p>Method: {{.Request.Method}}</p>
<p>Host: {{.Request.Host}}</p>
<p>User Agent: {{.Request.UserAgent}}</p>
```

Conditional content based on URI:
```html
{{if eq .RequestURI "/about"}}
<h1>About Page</h1>
{{else}}
<h1>Other Page</h1>
{{end}}
```

## Running the Server

### As a Standalone Server (for testing)

```bash
# Install dependencies
go mod tidy

# Run the server
go run main.go

# Or build and run
make tmpl.cgi
./tmpl.cgi
```

The server will start on port 8080 by default. You can set the `TMPL_CGI_PORT` environment variable to use a different port.

### As a CGI Script

1. Build the binary:
   ```bash
   make tmpl.cgi
   ```

2. Copy the binary and configuration to your web server's CGI directory

3. Configure your web server to execute the binary as a CGI script

4. Set the `TMPL_CGI_CONFIG` environment variable if your config file is not in the same directory as the binary

### Environment Variables

- `TMPL_CGI_PORT`: Port to use in standalone mode (default: 8080)
- `TMPL_CGI_CONFIG`: Path to configuration file (default: config.yaml)
- `GATEWAY_INTERFACE`: Automatically set by web servers when running as CGI

### Template Functions

The server uses Go's standard `html/template` package. You can use all built-in template functions and add custom ones by modifying the `loadTemplate` function in `main.go`.

