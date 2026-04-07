package resource

import "fmt"

// Command represents a universal command resource that translates to platform-specific
// command types (ClaudeCommand, CopilotPrompt, CursorCommand).
type Command struct {
	Name          string              `hcl:"name,label"`
	Description   string              `hcl:"description,attr"`
	Content       string              `hcl:"content,optional"`
	Files         []FileBlock         `hcl:"file,block"`
	TemplateFiles []TemplateFileBlock `hcl:"template_file,block"`
	Platforms     []string            `hcl:"platforms,optional"`

	Claude  *CommandClaudeOverride  `hcl:"claude,block"`
	Copilot *CommandCopilotOverride `hcl:"copilot,block"`
	Cursor  *PlatformOverride       `hcl:"cursor,block"`
}

// CommandClaudeOverride contains Claude-specific fields for commands.
type CommandClaudeOverride struct {
	Disabled     bool     `hcl:"disabled,optional"`
	Content      string   `hcl:"content,optional"`
	ArgumentHint string   `hcl:"argument_hint,optional"`
	AllowedTools []string `hcl:"allowed_tools,optional"`
	Model        string   `hcl:"model,optional"`
}

// CommandCopilotOverride contains Copilot-specific fields for commands (maps to CopilotPrompt).
type CommandCopilotOverride struct {
	Disabled     bool     `hcl:"disabled,optional"`
	Content      string   `hcl:"content,optional"`
	ArgumentHint string   `hcl:"argument_hint,optional"`
	Agent        string   `hcl:"agent,optional"`
	Model        string   `hcl:"model,optional"`
	Tools        []string `hcl:"tools,optional"`
}

func (c *Command) ResourceType() string                  { return "command" }
func (c *Command) ResourceName() string                  { return c.Name }
func (c *Command) Platform() string                      { return "universal" }
func (c *Command) GetContent() string                    { return c.Content }
func (c *Command) GetFiles() []FileBlock                 { return c.Files }
func (c *Command) GetTemplateFiles() []TemplateFileBlock { return c.TemplateFiles }

func (c *Command) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("command: name is required")
	}
	if c.Description == "" {
		return fmt.Errorf("command %q: description is required", c.Name)
	}
	if c.Content == "" {
		return fmt.Errorf("command %q: content is required", c.Name)
	}
	return nil
}

func (c *Command) IsEnabledForPlatform(platform string) bool {
	if len(c.Platforms) == 0 {
		return true
	}
	for _, p := range c.Platforms {
		if p == platform {
			return true
		}
	}
	return false
}
