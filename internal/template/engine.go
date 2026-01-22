package template

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

// Engine renders templates using Go's text/template.
type Engine struct {
	pluginDir string
	ctx       *Context
	funcMap   template.FuncMap
}

// NewEngine creates a template engine for the given plugin directory.
func NewEngine(pluginDir string, ctx *Context) *Engine {
	e := &Engine{
		pluginDir: pluginDir,
		ctx:       ctx,
	}
	e.funcMap = e.builtinFunctions()
	return e
}

// Render processes a template string with the context.
func (e *Engine) Render(content string) (string, error) {
	tmpl, err := template.New("content").Funcs(e.funcMap).Parse(content)
	if err != nil {
		return "", fmt.Errorf("parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, e.ctx.ToMap()); err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}

	return buf.String(), nil
}

// RenderFile reads a file and renders it as a template.
func (e *Engine) RenderFile(relativePath string) (string, error) {
	fullPath := filepath.Join(e.pluginDir, relativePath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("reading file %s: %w", relativePath, err)
	}
	return e.Render(string(content))
}

// RenderFileWithVars renders a template file with additional variables.
// The additional vars are merged with the context's ExtraVars for this render only.
func (e *Engine) RenderFileWithVars(relativePath string, vars map[string]any) (string, error) {
	// Create a cloned context with merged vars
	cloned := e.ctx.Clone()
	for k, v := range vars {
		cloned.ExtraVars[k] = v
	}

	// Create a temporary engine with the cloned context
	tempEngine := &Engine{
		pluginDir: e.pluginDir,
		ctx:       cloned,
	}
	tempEngine.funcMap = tempEngine.builtinFunctions()

	return tempEngine.RenderFile(relativePath)
}

// RenderWithVars renders a template string with additional variables.
// The additional vars are merged with the context's ExtraVars for this render only.
func (e *Engine) RenderWithVars(content string, vars map[string]any) (string, error) {
	// Create a cloned context with merged vars
	cloned := e.ctx.Clone()
	for k, v := range vars {
		cloned.ExtraVars[k] = v
	}

	// Create a temporary engine with the cloned context
	tempEngine := &Engine{
		pluginDir: e.pluginDir,
		ctx:       cloned,
	}
	tempEngine.funcMap = tempEngine.builtinFunctions()

	return tempEngine.Render(content)
}

// builtinFunctions returns the built-in template functions.
func (e *Engine) builtinFunctions() template.FuncMap {
	return template.FuncMap{
		// file reads a file relative to the plugin directory
		"file": func(path string) (string, error) {
			fullPath := filepath.Join(e.pluginDir, path)
			content, err := os.ReadFile(fullPath)
			if err != nil {
				return "", fmt.Errorf("reading file %s: %w", path, err)
			}
			return string(content), nil
		},

		// env reads an environment variable with optional default
		"env": func(args ...string) string {
			if len(args) == 0 {
				return ""
			}
			name := args[0]
			value := os.Getenv(name)
			if value == "" && len(args) > 1 {
				value = args[1]
			}
			return value
		},

		// dict creates a map from key-value pairs
		// Usage: {{ dict "key1" "val1" "key2" "val2" }}
		"dict": func(values ...any) map[string]any {
			m := make(map[string]any)
			for i := 0; i < len(values)-1; i += 2 {
				key, ok := values[i].(string)
				if ok {
					m[key] = values[i+1]
				}
			}
			return m
		},

		// templatefile reads and renders a template file with vars
		"templatefile": func(path string, vars map[string]any) (string, error) {
			return e.RenderFileWithVars(path, vars)
		},
	}
}
