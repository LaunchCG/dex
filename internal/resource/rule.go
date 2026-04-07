package resource

import "fmt"

// Rule represents a universal merged rule resource that translates to platform-specific
// types (ClaudeRule, CopilotInstruction, CursorRule). Multiple packages contribute rules
// that are merged into a single agent file (CLAUDE.md, copilot-instructions.md, AGENTS.md).
type Rule struct {
	Name          string              `hcl:"name,label"`
	Description   string              `hcl:"description,attr"`
	Content       string              `hcl:"content,optional"`
	Files         []FileBlock         `hcl:"file,block"`
	TemplateFiles []TemplateFileBlock `hcl:"template_file,block"`
	Platforms     []string            `hcl:"platforms,optional"`

	Claude  *RuleClaudeOverride `hcl:"claude,block"`
	Copilot *PlatformOverride   `hcl:"copilot,block"`
	Cursor  *PlatformOverride   `hcl:"cursor,block"`
}

// RuleClaudeOverride contains Claude-specific fields for rules.
type RuleClaudeOverride struct {
	Disabled bool     `hcl:"disabled,optional"`
	Content  string   `hcl:"content,optional"`
	Paths    []string `hcl:"paths,optional"`
}

func (r *Rule) ResourceType() string                  { return "rule" }
func (r *Rule) ResourceName() string                  { return r.Name }
func (r *Rule) Platform() string                      { return "universal" }
func (r *Rule) GetContent() string                    { return r.Content }
func (r *Rule) GetFiles() []FileBlock                 { return r.Files }
func (r *Rule) GetTemplateFiles() []TemplateFileBlock { return r.TemplateFiles }

func (r *Rule) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("rule: name is required")
	}
	if r.Description == "" {
		return fmt.Errorf("rule %q: description is required", r.Name)
	}
	if r.Content == "" {
		return fmt.Errorf("rule %q: content is required", r.Name)
	}
	return nil
}

func (r *Rule) IsEnabledForPlatform(platform string) bool {
	if len(r.Platforms) == 0 {
		return true
	}
	for _, p := range r.Platforms {
		if p == platform {
			return true
		}
	}
	return false
}
