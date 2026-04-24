package resource

import "fmt"

// Skill represents a universal skill resource that translates to platform-specific
// skill types (ClaudeSkill, CopilotSkill). Platforms that don't support skills
// (e.g., Cursor) skip them with a log warning.
type Skill struct {
	Name          string              `hcl:"name,label"`
	Description   string              `hcl:"description,attr"`
	Content       string              `hcl:"content,optional"`
	Files         []FileBlock         `hcl:"file,block"`
	TemplateFiles []TemplateFileBlock `hcl:"template_file,block"`
	Platforms     []string            `hcl:"platforms,optional"`

	Claude  *SkillClaudeOverride `hcl:"claude,block"`
	Copilot *PlatformOverride    `hcl:"copilot,block"`
	Cursor  *SkillCursorOverride `hcl:"cursor,block"`
}

// SkillCursorOverride contains Cursor-specific fields for skills.
// Cursor skills have a minimal frontmatter — no allowed-tools, model, or context fields.
type SkillCursorOverride struct {
	Disabled               bool              `hcl:"disabled,optional"`
	Content                string            `hcl:"content,optional"`
	License                string            `hcl:"license,optional"`
	Compatibility          string            `hcl:"compatibility,optional"`
	DisableModelInvocation bool              `hcl:"disable_model_invocation,optional"`
	Metadata               map[string]string `hcl:"metadata,optional"`
}

// SkillClaudeOverride contains Claude-specific fields for skills.
type SkillClaudeOverride struct {
	Disabled               bool                   `hcl:"disabled,optional"`
	Content                string                 `hcl:"content,optional"`
	ArgumentHint           string                 `hcl:"argument_hint,optional"`
	DisableModelInvocation bool                   `hcl:"disable_model_invocation,optional"`
	UserInvocable          *bool                  `hcl:"user_invocable,optional"`
	AllowedTools           []string               `hcl:"allowed_tools,optional"`
	Model                  string                 `hcl:"model,optional"`
	Context                string                 `hcl:"context,optional"`
	Agent                  string                 `hcl:"agent,optional"`
	Metadata               map[string]string      `hcl:"metadata,optional"`
	Hooks                  map[string]interface{} `hcl:"hooks,optional"`
}

func (s *Skill) ResourceType() string                  { return "skill" }
func (s *Skill) ResourceName() string                  { return s.Name }
func (s *Skill) Platform() string                      { return "universal" }
func (s *Skill) GetContent() string                    { return s.Content }
func (s *Skill) GetFiles() []FileBlock                 { return s.Files }
func (s *Skill) GetTemplateFiles() []TemplateFileBlock { return s.TemplateFiles }

func (s *Skill) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("skill: name is required")
	}
	if s.Description == "" {
		return fmt.Errorf("skill %q: description is required", s.Name)
	}
	if s.Content == "" {
		return fmt.Errorf("skill %q: content is required", s.Name)
	}
	return nil
}

// IsEnabledForPlatform checks if this skill should be installed for the given platform.
func (s *Skill) IsEnabledForPlatform(platform string) bool {
	if len(s.Platforms) == 0 {
		return true
	}
	for _, p := range s.Platforms {
		if p == platform {
			return true
		}
	}
	return false
}
