package resource

import "fmt"

// Rules represents a universal standalone rules file that translates to platform-specific
// types (ClaudeRules, CopilotInstructions, CursorRules). Each creates a standalone file
// owned by a single package.
type Rules struct {
	Name          string              `hcl:"name,label"`
	Description   string              `hcl:"description,attr"`
	Content       string              `hcl:"content,optional"`
	Files         []FileBlock         `hcl:"file,block"`
	TemplateFiles []TemplateFileBlock `hcl:"template_file,block"`
	Platforms     []string            `hcl:"platforms,optional"`

	Claude  *RulesClaudeOverride  `hcl:"claude,block"`
	Copilot *RulesCopilotOverride `hcl:"copilot,block"`
	Cursor  *RulesCursorOverride  `hcl:"cursor,block"`
}

// RulesClaudeOverride contains Claude-specific fields for standalone rules.
type RulesClaudeOverride struct {
	Disabled bool     `hcl:"disabled,optional"`
	Content  string   `hcl:"content,optional"`
	Paths    []string `hcl:"paths,optional"`
}

// RulesCopilotOverride contains Copilot-specific fields for standalone instructions.
type RulesCopilotOverride struct {
	Disabled bool   `hcl:"disabled,optional"`
	Content  string `hcl:"content,optional"`
	ApplyTo  string `hcl:"apply_to,optional"`
}

// RulesCursorOverride contains Cursor-specific fields for standalone rules.
type RulesCursorOverride struct {
	Disabled    bool     `hcl:"disabled,optional"`
	Content     string   `hcl:"content,optional"`
	Globs       []string `hcl:"globs,optional"`
	AlwaysApply *bool    `hcl:"always_apply,optional"`
}

func (r *Rules) ResourceType() string                  { return "rules" }
func (r *Rules) ResourceName() string                  { return r.Name }
func (r *Rules) Platform() string                      { return "universal" }
func (r *Rules) GetContent() string                    { return r.Content }
func (r *Rules) GetFiles() []FileBlock                 { return r.Files }
func (r *Rules) GetTemplateFiles() []TemplateFileBlock { return r.TemplateFiles }

func (r *Rules) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("rules: name is required")
	}
	if r.Description == "" {
		return fmt.Errorf("rules %q: description is required", r.Name)
	}
	if r.Content == "" && len(r.Files) == 0 && len(r.TemplateFiles) == 0 {
		return fmt.Errorf("rules %q: must specify either content or file blocks", r.Name)
	}
	return nil
}

func (r *Rules) IsEnabledForPlatform(platform string) bool {
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
