// Package template provides template rendering functionality for dex.
// It uses Go's text/template package to render templates with context variables.
package template

// Context holds all variables available during template rendering.
type Context struct {
	// Built-in variables per INITIAL_RESOURCE_SPEC.md
	ComponentDir  string // Installation directory for the component
	PluginName    string // Name of the plugin being installed
	PluginVersion string // Version of the plugin
	ProjectRoot   string // Root directory of the project
	Platform      string // Target platform (e.g., "claude-code")

	// User-defined variables from package.hcl variable blocks
	Variables map[string]string

	// Additional vars passed to templatefile() calls
	ExtraVars map[string]any
}

// NewContext creates a template context for rendering.
func NewContext(pluginName, pluginVersion, projectRoot, platform string) *Context {
	return &Context{
		PluginName:    pluginName,
		PluginVersion: pluginVersion,
		ProjectRoot:   projectRoot,
		Platform:      platform,
		Variables:     make(map[string]string),
		ExtraVars:     make(map[string]any),
	}
}

// WithComponentDir sets the component directory and returns the context.
func (c *Context) WithComponentDir(dir string) *Context {
	c.ComponentDir = dir
	return c
}

// WithVariables sets user-defined variables and returns the context.
func (c *Context) WithVariables(vars map[string]string) *Context {
	c.Variables = vars
	return c
}

// ToMap converts the context to a map for template execution.
func (c *Context) ToMap() map[string]any {
	m := map[string]any{
		"ComponentDir":  c.ComponentDir,
		"PluginName":    c.PluginName,
		"PluginVersion": c.PluginVersion,
		"ProjectRoot":   c.ProjectRoot,
		"Platform":      c.Platform,
	}

	// Add user variables
	for k, v := range c.Variables {
		m[k] = v
	}

	// Add extra vars (for templatefile calls)
	for k, v := range c.ExtraVars {
		m[k] = v
	}

	return m
}

// Clone creates a copy of the context with independent maps.
func (c *Context) Clone() *Context {
	clone := &Context{
		ComponentDir:  c.ComponentDir,
		PluginName:    c.PluginName,
		PluginVersion: c.PluginVersion,
		ProjectRoot:   c.ProjectRoot,
		Platform:      c.Platform,
		Variables:     make(map[string]string),
		ExtraVars:     make(map[string]any),
	}

	for k, v := range c.Variables {
		clone.Variables[k] = v
	}
	for k, v := range c.ExtraVars {
		clone.ExtraVars[k] = v
	}

	return clone
}
