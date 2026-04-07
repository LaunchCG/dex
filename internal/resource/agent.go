package resource

import "fmt"

// Agent represents a universal agent resource that translates to platform-specific
// agent types (ClaudeSubagent, CopilotAgent). Platforms that don't support agents
// (e.g., Cursor) skip them with a log warning.
type Agent struct {
	Name          string              `hcl:"name,label"`
	Description   string              `hcl:"description,attr"`
	Content       string              `hcl:"content,optional"`
	Files         []FileBlock         `hcl:"file,block"`
	TemplateFiles []TemplateFileBlock `hcl:"template_file,block"`
	Platforms     []string            `hcl:"platforms,optional"`

	Claude  *AgentClaudeOverride  `hcl:"claude,block"`
	Copilot *AgentCopilotOverride `hcl:"copilot,block"`
	Cursor  *PlatformOverride     `hcl:"cursor,block"`
}

// AgentClaudeOverride contains Claude-specific fields for agents (maps to ClaudeSubagent).
type AgentClaudeOverride struct {
	Disabled bool                   `hcl:"disabled,optional"`
	Content  string                 `hcl:"content,optional"`
	Model    string                 `hcl:"model,optional"`
	Color    string                 `hcl:"color,optional"`
	Tools    []string               `hcl:"tools,optional"`
	Hooks    map[string]interface{} `hcl:"hooks,optional"`
}

// AgentCopilotOverride contains Copilot-specific fields for agents.
type AgentCopilotOverride struct {
	Disabled bool     `hcl:"disabled,optional"`
	Content  string   `hcl:"content,optional"`
	Model    string   `hcl:"model,optional"`
	Tools    []string `hcl:"tools,optional"`
	Handoffs []string `hcl:"handoffs,optional"`
	Infer    *bool    `hcl:"infer,optional"`
	Target   string   `hcl:"target,optional"`
}

func (a *Agent) ResourceType() string                  { return "agent" }
func (a *Agent) ResourceName() string                  { return a.Name }
func (a *Agent) Platform() string                      { return "universal" }
func (a *Agent) GetContent() string                    { return a.Content }
func (a *Agent) GetFiles() []FileBlock                 { return a.Files }
func (a *Agent) GetTemplateFiles() []TemplateFileBlock { return a.TemplateFiles }

func (a *Agent) Validate() error {
	if a.Name == "" {
		return fmt.Errorf("agent: name is required")
	}
	if a.Description == "" {
		return fmt.Errorf("agent %q: description is required", a.Name)
	}
	if a.Content == "" {
		return fmt.Errorf("agent %q: content is required", a.Name)
	}
	return nil
}

func (a *Agent) IsEnabledForPlatform(platform string) bool {
	if len(a.Platforms) == 0 {
		return true
	}
	for _, p := range a.Platforms {
		if p == platform {
			return true
		}
	}
	return false
}
