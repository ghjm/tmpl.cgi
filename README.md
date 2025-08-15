# CGI Template Server

A Go-based CGI server that serves HTML templates based on request URI patterns. The server can be configured to serve different templates for different URL patterns using regular expressions.

## Features

- **Hugo-style templating**: Full Sprig function library with 100+ template functions for advanced templating
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

### Template Syntax Validation

Before deploying, you can validate that all your templates are syntactically correct and will execute without errors:

```bash
# Build the binary first
make tmpl.cgi

# Check template syntax
./tmpl.cgi -syntax-check

# Or use the Makefile target
make syntax-check

# Check templates with a specific config file
./tmpl.cgi -syntax-check -config path/to/config.yaml
```

The syntax checker will:
- Verify the configuration file is valid YAML
- Validate all regex patterns compile correctly
- Parse all template files to check for syntax errors
- Execute each template with sample data to catch runtime errors
- Report which templates are valid or show detailed error messages

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

### Command Line Options

- `-syntax-check`: Validate all templates and exit (does not start server)
- `-config path`: Specify path to configuration file

### Environment Variables

- `TMPL_CGI_PORT`: Port to use in standalone mode (default: 8080)
- `TMPL_CGI_CONFIG`: Path to configuration file (default: config.yaml)
- `GATEWAY_INTERFACE`: Automatically set by web servers when running as CGI

### Template Functions

The server now uses **Hugo-style templating** with the full Sprig function library, providing over 100 additional template functions beyond Go's standard `html/template` package.

#### Available Function Categories

- **String Functions**: `upper`, `lower`, `title`, `camelcase`, `kebabcase`, `snakecase`, `trim`, `trunc`, `repeat`, `replace`, `regexFind`, `regexReplaceAll`, etc.
- **Math Functions**: `add`, `sub`, `mul`, `div`, `mod`, `max`, `min`, `ceil`, `floor`, `round`, etc.
- **Date Functions**: `now`, `date`, `dateInZone`, `duration`, `ago`, etc.
- **List Functions**: `list`, `first`, `last`, `rest`, `initial`, `reverse`, `sort`, `uniq`, `join`, `split`, etc.
- **Dict Functions**: `dict`, `get`, `set`, `keys`, `values`, `pick`, `omit`, etc.
- **Encoding Functions**: `b64enc`, `b64dec`, `urlquery`, `htmlEscape`, `jsEscape`, etc.
- **Crypto Functions**: `sha256sum`, `sha1sum`, `md5sum`, etc.
- **UUID Functions**: `uuidv4`, etc.
- **Default Functions**: `default`, `empty`, `coalesce`, etc.
- **Flow Control**: `if`, `else`, `range`, `with`, `eq`, `ne`, `lt`, `le`, `gt`, `ge`, `and`, `or`, `not`, etc.

#### Template Examples with Hugo/Sprig Functions

**String manipulation:**
```html
<h1>{{.RequestURI | upper}}</h1>
<p>Page title: {{"hello world" | title}}</p>
<p>Slug: {{.RequestURI | regexReplaceAll "^/" "" | kebabcase}}</p>
```

**Date and time:**
```html
<p>Current time: {{now | date "2006-01-02 15:04:05"}}</p>
<p>Published: {{now | date "January 2, 2006"}}</p>
```

**Math operations:**
```html
<p>Total items: {{add 5 3}}</p>
<p>Random reading time: {{randInt 1 10}} minutes</p>
```

**Lists and iteration:**
```html
{{$tags := list "golang" "templates" "hugo" "sprig"}}
<ul>
{{range $i, $tag := $tags}}
  <li>Tag {{add $i 1}}: {{$tag | title}}</li>
{{end}}
</ul>
```

**URL and regex operations:**
```html
{{$postId := .RequestURI | regexFind "\\d+" | default "unknown"}}
<p>Post ID: {{$postId}}</p>
<p>Share URL: https://twitter.com/intent/tweet?text={{urlquery "Check this out!"}}</p>
```

**Conditional logic with defaults:**
```html
<p>{{.Title | default "Untitled Page"}}</p>
{{if eq .RequestURI "/"}}
  <h1>Welcome to the home page!</h1>
{{else}}
  <h1>{{.RequestURI | regexReplaceAll "^/" "" | title}}</h1>
{{end}}
```

**Advanced string operations:**
```html
<p>Preview: {{"This is a very long content..." | trunc 50}}</p>
<p>Repeated: {{"★" | repeat 5}}</p>
```

For a complete list of available functions, see the [Sprig Function Documentation](http://masterminds.github.io/sprig/).

