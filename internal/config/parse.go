// Package config provides HCL configuration parsing for dex project and package files.
// It handles loading and validating dex.hcl (project configuration) and package.hcl
// (plugin package configuration) files using the HashiCorp HCL v2 library.
package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// Type aliases to avoid shadowing the template package
type textTemplate = template.Template

var textTemplateNew = template.New

// Parser wraps HCL parsing functionality and provides a reusable parser instance.
type Parser struct {
	parser *hclparse.Parser
}

// NewParser creates a new HCL parser instance.
func NewParser() *Parser {
	return &Parser{
		parser: hclparse.NewParser(),
	}
}

// ParseFile parses an HCL file and returns the parsed file and any diagnostics.
// The filename should be an absolute or relative path to the HCL file.
func (p *Parser) ParseFile(filename string) (*hcl.File, hcl.Diagnostics) {
	return p.parser.ParseHCLFile(filename)
}

// DecodeBody decodes an HCL body into the target struct using gohcl.
// The ctx parameter provides the evaluation context for expressions,
// and target should be a pointer to the struct to decode into.
func DecodeBody(body hcl.Body, ctx *hcl.EvalContext, target interface{}) hcl.Diagnostics {
	return gohcl.DecodeBody(body, ctx, target)
}

// NewEvalContext creates an HCL evaluation context with built-in functions.
// Currently provides the env() function for reading environment variables.
func NewEvalContext() *hcl.EvalContext {
	return &hcl.EvalContext{
		Functions: map[string]function.Function{
			"env": envFunction(),
		},
	}
}

// NewPackageEvalContext creates an HCL evaluation context for package.hcl files.
// It includes the file(), env(), and templatefile() functions for package configuration.
func NewPackageEvalContext(packageDir string) *hcl.EvalContext {
	return &hcl.EvalContext{
		Functions: map[string]function.Function{
			"env":          envFunction(),
			"file":         fileFunction(packageDir),
			"templatefile": templatefileFunction(packageDir),
		},
	}
}

// fileFunction returns an HCL function that reads file contents.
// Usage in HCL: file("relative/path/to/file.md")
// Paths are resolved relative to the package directory.
func fileFunction(baseDir string) function.Function {
	return function.New(&function.Spec{
		Description: "Reads the contents of a file relative to the package directory",
		Params: []function.Parameter{
			{
				Name:        "path",
				Type:        cty.String,
				Description: "The relative path to the file to read",
			},
		},
		Type: function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			relPath := args[0].AsString()
			fullPath := filepath.Join(baseDir, relPath)

			content, err := os.ReadFile(fullPath)
			if err != nil {
				return cty.StringVal(""), fmt.Errorf("failed to read file %s: %w", relPath, err)
			}

			return cty.StringVal(string(content)), nil
		},
	})
}

// envFunction returns an HCL function that reads environment variables.
// Usage in HCL: env("VAR_NAME") or env("VAR_NAME", "default_value")
// If the variable is not set and no default is provided, returns an empty string.
func envFunction() function.Function {
	return function.New(&function.Spec{
		Description: "Reads an environment variable, with an optional default value",
		Params: []function.Parameter{
			{
				Name:        "name",
				Type:        cty.String,
				Description: "The name of the environment variable to read",
			},
		},
		VarParam: &function.Parameter{
			Name:        "default",
			Type:        cty.String,
			Description: "Optional default value if the environment variable is not set",
		},
		Type: function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			envName := args[0].AsString()
			value := os.Getenv(envName)

			// If environment variable is not set and a default was provided, use it
			if value == "" && len(args) > 1 {
				value = args[1].AsString()
			}

			return cty.StringVal(value), nil
		},
	})
}

// templatefileFunction returns an HCL function that renders a template file.
// Usage in HCL: templatefile("path/to/template.tmpl", { var1 = "value1", var2 = "value2" })
// Templates use Go text/template syntax.
func templatefileFunction(baseDir string) function.Function {
	return function.New(&function.Spec{
		Description: "Reads and renders a template file with the provided variables",
		Params: []function.Parameter{
			{
				Name:        "path",
				Type:        cty.String,
				Description: "The relative path to the template file",
			},
			{
				Name:        "vars",
				Type:        cty.DynamicPseudoType,
				Description: "A map of variables to pass to the template",
			},
		},
		Type: function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			path := args[0].AsString()
			fullPath := filepath.Join(baseDir, path)

			content, err := os.ReadFile(fullPath)
			if err != nil {
				return cty.NilVal, fmt.Errorf("reading template %s: %w", path, err)
			}

			// Convert cty vars to Go map
			vars := make(map[string]any)
			if !args[1].IsNull() {
				for k, v := range args[1].AsValueMap() {
					vars[k] = ctyToGo(v)
				}
			}

			// Render the template using text/template
			tmpl, err := parseTemplate(string(content))
			if err != nil {
				return cty.NilVal, fmt.Errorf("parsing template %s: %w", path, err)
			}

			result, err := executeTemplate(tmpl, vars)
			if err != nil {
				return cty.NilVal, fmt.Errorf("rendering template %s: %w", path, err)
			}

			return cty.StringVal(result), nil
		},
	})
}

// ctyToGo converts a cty.Value to a Go value.
func ctyToGo(v cty.Value) any {
	if v.IsNull() {
		return nil
	}
	switch v.Type() {
	case cty.String:
		return v.AsString()
	case cty.Number:
		f, _ := v.AsBigFloat().Float64()
		return f
	case cty.Bool:
		return v.True()
	default:
		if v.Type().IsListType() || v.Type().IsTupleType() {
			var list []any
			for it := v.ElementIterator(); it.Next(); {
				_, val := it.Element()
				list = append(list, ctyToGo(val))
			}
			return list
		}
		if v.Type().IsMapType() || v.Type().IsObjectType() {
			m := make(map[string]any)
			for k, val := range v.AsValueMap() {
				m[k] = ctyToGo(val)
			}
			return m
		}
		return v.GoString()
	}
}

// parseTemplate creates a new template from content.
func parseTemplate(content string) (*textTemplate, error) {
	return textTemplateNew("hcl").Parse(content)
}

// executeTemplate executes a template with vars and returns the result.
func executeTemplate(tmpl *textTemplate, vars map[string]any) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", err
	}
	return buf.String(), nil
}
