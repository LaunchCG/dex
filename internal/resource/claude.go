package resource

import "fmt"

// ClaudeSkill represents a skill for Claude Code.
// Skills are installed to .claude/skills/{plugin}-{name}/SKILL.md
// and provide specialized knowledge or capabilities to the AI assistant.
type ClaudeSkill struct {
	// Name is the block label identifying this skill
	Name string `hcl:"name,label"`

	// Description explains when and how to use this skill
	Description string `hcl:"description,attr"`

	// Content is the main body/instructions of the skill
	Content string `hcl:"content,optional"`

	// Files lists static files to copy alongside the skill
	Files []FileBlock `hcl:"file,block"`

	// TemplateFiles lists template files to render and copy
	TemplateFiles []TemplateFileBlock `hcl:"template_file,block"`

	// ArgumentHint provides a hint shown during autocomplete (e.g., "[filename]")
	ArgumentHint string `hcl:"argument_hint,optional"`

	// DisableModelInvocation prevents Claude from auto-loading this skill; user must invoke manually
	DisableModelInvocation bool `hcl:"disable_model_invocation,optional"`

	// UserInvocable controls whether the skill appears in the / menu (default: true when nil)
	UserInvocable *bool `hcl:"user_invocable,optional"`

	// AllowedTools lists tools Claude can use without asking permission (e.g., ["Read", "Grep"])
	AllowedTools []string `hcl:"allowed_tools,optional"`

	// Model specifies the model to use when skill is active: "sonnet", "haiku", or "opus"
	Model string `hcl:"model,optional"`

	// Context controls execution context; set to "fork" to run in isolated subagent
	Context string `hcl:"context,optional"`

	// Agent specifies which subagent type to use when Context is "fork" (e.g., "Explore", "Plan")
	Agent string `hcl:"agent,optional"`

	// Metadata contains additional frontmatter fields for the skill
	Metadata map[string]string `hcl:"metadata,optional"`
}

// ResourceType returns the HCL block type for Claude skills.
func (s *ClaudeSkill) ResourceType() string {
	return "claude_skill"
}

// ResourceName returns the skill's name identifier.
func (s *ClaudeSkill) ResourceName() string {
	return s.Name
}

// Platform returns the target platform for Claude skills.
func (s *ClaudeSkill) Platform() string {
	return "claude-code"
}

// GetContent returns the skill's content.
func (s *ClaudeSkill) GetContent() string {
	return s.Content
}

// GetFiles returns the skill's file blocks.
func (s *ClaudeSkill) GetFiles() []FileBlock {
	return s.Files
}

// GetTemplateFiles returns the skill's template file blocks.
func (s *ClaudeSkill) GetTemplateFiles() []TemplateFileBlock {
	return s.TemplateFiles
}

// Validate checks that the skill has all required fields.
func (s *ClaudeSkill) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("claude_skill: name is required")
	}
	if s.Description == "" {
		return fmt.Errorf("claude_skill %q: description is required", s.Name)
	}
	if s.Content == "" {
		return fmt.Errorf("claude_skill %q: content is required", s.Name)
	}
	return nil
}

// ClaudeCommand represents a command for Claude Code.
// Commands are installed to .claude/commands/{plugin}-{name}.md
// and can be invoked by users with /{name} syntax.
type ClaudeCommand struct {
	// Name is the block label identifying this command
	Name string `hcl:"name,label"`

	// Description explains when and how to use this command
	Description string `hcl:"description,attr"`

	// Content is the main body/instructions of the command
	Content string `hcl:"content,optional"`

	// Files lists static files to copy alongside the command
	Files []FileBlock `hcl:"file,block"`

	// TemplateFiles lists template files to render and copy
	TemplateFiles []TemplateFileBlock `hcl:"template_file,block"`

	// ArgumentHint provides a hint for command arguments (e.g., "[environment]")
	ArgumentHint string `hcl:"argument_hint,optional"`

	// AllowedTools restricts which tools this command can use
	AllowedTools []string `hcl:"allowed_tools,optional"`

	// Model specifies the model to use: "sonnet", "haiku", or "opus"
	Model string `hcl:"model,optional"`
}

// ResourceType returns the HCL block type for Claude commands.
func (c *ClaudeCommand) ResourceType() string {
	return "claude_command"
}

// ResourceName returns the command's name identifier.
func (c *ClaudeCommand) ResourceName() string {
	return c.Name
}

// Platform returns the target platform for Claude commands.
func (c *ClaudeCommand) Platform() string {
	return "claude-code"
}

// GetContent returns the command's content.
func (c *ClaudeCommand) GetContent() string {
	return c.Content
}

// GetFiles returns the command's file blocks.
func (c *ClaudeCommand) GetFiles() []FileBlock {
	return c.Files
}

// GetTemplateFiles returns the command's template file blocks.
func (c *ClaudeCommand) GetTemplateFiles() []TemplateFileBlock {
	return c.TemplateFiles
}

// Validate checks that the command has all required fields.
func (c *ClaudeCommand) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("claude_command: name is required")
	}
	if c.Description == "" {
		return fmt.Errorf("claude_command %q: description is required", c.Name)
	}
	if c.Content == "" {
		return fmt.Errorf("claude_command %q: content is required", c.Name)
	}
	return nil
}

// ClaudeSubagent represents a subagent for Claude Code.
// Subagents are installed to .claude/agents/{plugin}-{name}.md
// and provide specialized agent behaviors.
type ClaudeSubagent struct {
	// Name is the block label identifying this subagent
	Name string `hcl:"name,label"`

	// Description explains when and how to use this subagent
	Description string `hcl:"description,attr"`

	// Content is the main body/instructions of the subagent
	Content string `hcl:"content,optional"`

	// Files lists static files to copy alongside the subagent
	Files []FileBlock `hcl:"file,block"`

	// TemplateFiles lists template files to render and copy
	TemplateFiles []TemplateFileBlock `hcl:"template_file,block"`

	// Model specifies the model: "inherit", "sonnet", "haiku", or "opus"
	Model string `hcl:"model,optional"`

	// Color sets the agent's display color: "blue", "green", "yellow", "red", "purple"
	Color string `hcl:"color,optional"`

	// Tools lists the tools this subagent is allowed to use
	Tools []string `hcl:"tools,optional"`
}

// ResourceType returns the HCL block type for Claude subagents.
func (a *ClaudeSubagent) ResourceType() string {
	return "claude_subagent"
}

// ResourceName returns the subagent's name identifier.
func (a *ClaudeSubagent) ResourceName() string {
	return a.Name
}

// Platform returns the target platform for Claude subagents.
func (a *ClaudeSubagent) Platform() string {
	return "claude-code"
}

// GetContent returns the subagent's content.
func (a *ClaudeSubagent) GetContent() string {
	return a.Content
}

// GetFiles returns the subagent's file blocks.
func (a *ClaudeSubagent) GetFiles() []FileBlock {
	return a.Files
}

// GetTemplateFiles returns the subagent's template file blocks.
func (a *ClaudeSubagent) GetTemplateFiles() []TemplateFileBlock {
	return a.TemplateFiles
}

// Validate checks that the subagent has all required fields.
func (a *ClaudeSubagent) Validate() error {
	if a.Name == "" {
		return fmt.Errorf("claude_subagent: name is required")
	}
	if a.Description == "" {
		return fmt.Errorf("claude_subagent %q: description is required", a.Name)
	}
	if a.Content == "" {
		return fmt.Errorf("claude_subagent %q: content is required", a.Name)
	}
	return nil
}

// ClaudeRule represents a singular rule that is merged into CLAUDE.md.
// Multiple plugins can contribute rules which are combined together.
type ClaudeRule struct {
	// Name is the block label identifying this rule
	Name string `hcl:"name,label"`

	// Description explains when and how to use this rule
	Description string `hcl:"description,attr"`

	// Content is the main body/instructions of the rule
	Content string `hcl:"content,optional"`

	// Files lists static files to copy alongside the rule
	Files []FileBlock `hcl:"file,block"`

	// TemplateFiles lists template files to render and copy
	TemplateFiles []TemplateFileBlock `hcl:"template_file,block"`

	// Paths contains file patterns to scope when this rule applies
	Paths []string `hcl:"paths,optional"`
}

// ResourceType returns the HCL block type for Claude rules (singular).
func (r *ClaudeRule) ResourceType() string {
	return "claude_rule"
}

// ResourceName returns the rule's name identifier.
func (r *ClaudeRule) ResourceName() string {
	return r.Name
}

// Platform returns the target platform for Claude rules.
func (r *ClaudeRule) Platform() string {
	return "claude-code"
}

// GetContent returns the rule's content.
func (r *ClaudeRule) GetContent() string {
	return r.Content
}

// GetFiles returns the rule's file blocks.
func (r *ClaudeRule) GetFiles() []FileBlock {
	return r.Files
}

// GetTemplateFiles returns the rule's template file blocks.
func (r *ClaudeRule) GetTemplateFiles() []TemplateFileBlock {
	return r.TemplateFiles
}

// Validate checks that the rule has all required fields.
func (r *ClaudeRule) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("claude_rule: name is required")
	}
	if r.Description == "" {
		return fmt.Errorf("claude_rule %q: description is required", r.Name)
	}
	if r.Content == "" {
		return fmt.Errorf("claude_rule %q: content is required", r.Name)
	}
	return nil
}

// ClaudeRules represents a standalone rules file.
// Installed to .claude/rules/{plugin}-{name}.md as a complete file
// owned by a single plugin.
type ClaudeRules struct {
	// Name is the block label identifying this rules file
	Name string `hcl:"name,label"`

	// Description explains when and how to use these rules
	Description string `hcl:"description,attr"`

	// Content is the main body/instructions of the rules
	Content string `hcl:"content,optional"`

	// Files lists static files to copy alongside the rules
	Files []FileBlock `hcl:"file,block"`

	// TemplateFiles lists template files to render and copy
	TemplateFiles []TemplateFileBlock `hcl:"template_file,block"`

	// Paths contains file patterns to scope when these rules apply
	Paths []string `hcl:"paths,optional"`
}

// ResourceType returns the HCL block type for Claude rules (plural).
func (r *ClaudeRules) ResourceType() string {
	return "claude_rules"
}

// ResourceName returns the rules file's name identifier.
func (r *ClaudeRules) ResourceName() string {
	return r.Name
}

// Platform returns the target platform for Claude rules files.
func (r *ClaudeRules) Platform() string {
	return "claude-code"
}

// GetContent returns the rules content.
func (r *ClaudeRules) GetContent() string {
	return r.Content
}

// GetFiles returns the rules file blocks.
func (r *ClaudeRules) GetFiles() []FileBlock {
	return r.Files
}

// GetTemplateFiles returns the rules template file blocks.
func (r *ClaudeRules) GetTemplateFiles() []TemplateFileBlock {
	return r.TemplateFiles
}

// Validate checks that the rules have all required fields.
func (r *ClaudeRules) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("claude_rules: name is required")
	}
	if r.Description == "" {
		return fmt.Errorf("claude_rules %q: description is required", r.Name)
	}
	// Content is optional - can provide rules via file blocks only
	if r.Content == "" && len(r.Files) == 0 && len(r.TemplateFiles) == 0 {
		return fmt.Errorf("claude_rules %q: must specify either content or file blocks", r.Name)
	}
	return nil
}

// ClaudeSettings represents settings merged into .claude/settings.json.
// Multiple plugins can contribute permissions and environment variables.
// Project-level settings override plugin settings.
type ClaudeSettings struct {
	// Name is the identifier for this settings block
	Name string `hcl:"name,label"`

	// Allow lists tool patterns that are automatically approved
	Allow []string `hcl:"allow,optional"`

	// Ask lists tool patterns that require user confirmation
	Ask []string `hcl:"ask,optional"`

	// Deny lists tool patterns that are blocked
	Deny []string `hcl:"deny,optional"`

	// Env contains environment variables to set
	Env map[string]string `hcl:"env,optional"`

	// EnableAllProjectMCPServers auto-approves all project MCP servers
	EnableAllProjectMCPServers bool `hcl:"enable_all_project_mcp_servers,optional"`

	// EnabledMCPServers lists specific approved MCP servers
	EnabledMCPServers []string `hcl:"enabled_mcp_servers,optional"`

	// DisabledMCPServers lists rejected MCP servers
	DisabledMCPServers []string `hcl:"disabled_mcp_servers,optional"`

	// RespectGitignore filters suggestions by gitignore rules
	RespectGitignore bool `hcl:"respect_gitignore,optional"`

	// IncludeCoAuthoredBy includes co-author in commits
	IncludeCoAuthoredBy bool `hcl:"include_co_authored_by,optional"`

	// Model overrides the default model
	Model string `hcl:"model,optional"`

	// OutputStyle sets the response style preference
	OutputStyle string `hcl:"output_style,optional"`

	// AlwaysThinkingEnabled enables extended thinking mode
	AlwaysThinkingEnabled bool `hcl:"always_thinking_enabled,optional"`

	// PlansDirectory sets a custom location for plan files
	PlansDirectory string `hcl:"plans_directory,optional"`
}

// ResourceType returns the HCL block type for Claude settings.
func (s *ClaudeSettings) ResourceType() string {
	return "claude_settings"
}

// ResourceName returns the settings block's name identifier.
func (s *ClaudeSettings) ResourceName() string {
	return s.Name
}

// Platform returns the target platform for Claude settings.
func (s *ClaudeSettings) Platform() string {
	return "claude-code"
}

// GetContent returns an empty string as settings don't have content.
func (s *ClaudeSettings) GetContent() string {
	return ""
}

// GetFiles returns nil as settings don't have associated files.
func (s *ClaudeSettings) GetFiles() []FileBlock {
	return nil
}

// GetTemplateFiles returns nil as settings don't have associated template files.
func (s *ClaudeSettings) GetTemplateFiles() []TemplateFileBlock {
	return nil
}

// Validate checks that the settings block has a name.
func (s *ClaudeSettings) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("claude_settings: name is required")
	}
	return nil
}

// ClaudeMCPServer represents an MCP server configuration merged into .mcp.json.
// MCP servers provide additional tools and capabilities to Claude Code.
type ClaudeMCPServer struct {
	// Name is the unique identifier for this MCP server
	Name string `hcl:"name,label"`

	// Description explains what this MCP server provides
	Description string `hcl:"description,optional"`

	// Type specifies the server type: "command" or "http"
	Type string `hcl:"type,attr"`

	// Command is the executable to run (required for type="command")
	Command string `hcl:"command,optional"`

	// Args are the command-line arguments for the command
	Args []string `hcl:"args,optional"`

	// Env contains environment variables for the server
	Env map[string]string `hcl:"env,optional"`

	// Source is a shortcut for common package managers: "npm:", "uvx:", "pip:"
	Source string `hcl:"source,optional"`

	// URL is the HTTP endpoint (required for type="http")
	URL string `hcl:"url,optional"`
}

// ResourceType returns the HCL block type for Claude MCP servers.
func (m *ClaudeMCPServer) ResourceType() string {
	return "claude_mcp_server"
}

// ResourceName returns the MCP server's name identifier.
func (m *ClaudeMCPServer) ResourceName() string {
	return m.Name
}

// Platform returns the target platform for Claude MCP servers.
func (m *ClaudeMCPServer) Platform() string {
	return "claude-code"
}

// GetContent returns an empty string as MCP servers don't have content.
func (m *ClaudeMCPServer) GetContent() string {
	return ""
}

// GetFiles returns nil as MCP servers don't have associated files.
func (m *ClaudeMCPServer) GetFiles() []FileBlock {
	return nil
}

// GetTemplateFiles returns nil as MCP servers don't have associated template files.
func (m *ClaudeMCPServer) GetTemplateFiles() []TemplateFileBlock {
	return nil
}

// Validate checks that the MCP server has all required fields.
func (m *ClaudeMCPServer) Validate() error {
	if m.Name == "" {
		return fmt.Errorf("claude_mcp_server: name is required")
	}
	if m.Type == "" {
		return fmt.Errorf("claude_mcp_server %q: type is required", m.Name)
	}
	if m.Type != "command" && m.Type != "http" {
		return fmt.Errorf("claude_mcp_server %q: type must be 'command' or 'http', got %q", m.Name, m.Type)
	}
	if m.Type == "command" {
		if m.Command == "" && m.Source == "" {
			return fmt.Errorf("claude_mcp_server %q: command or source is required for type 'command'", m.Name)
		}
	}
	if m.Type == "http" {
		if m.URL == "" {
			return fmt.Errorf("claude_mcp_server %q: url is required for type 'http'", m.Name)
		}
	}
	return nil
}
