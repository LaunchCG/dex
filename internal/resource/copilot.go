package resource

import "fmt"

// =============================================================================
// MERGED RESOURCES (multiple plugins contribute to same file)
// =============================================================================

// CopilotInstruction (singular) represents an instruction merged into .github/copilot-instructions.md.
// Similar to ClaudeRule which merges into CLAUDE.md, multiple plugins can contribute
// instructions which are combined together using marker-based sections.
type CopilotInstruction struct {
	// Name is the block label identifying this instruction
	Name string `hcl:"name,label"`

	// Description explains what this instruction provides
	Description string `hcl:"description,attr"`

	// Content is the instruction text to merge
	Content string `hcl:"content,optional"`

	// Files lists static files to copy alongside the instruction
	Files []FileBlock `hcl:"file,block"`

	// TemplateFiles lists template files to render and copy
	TemplateFiles []TemplateFileBlock `hcl:"template_file,block"`
}

// ResourceType returns the HCL block type for Copilot instructions (singular).
func (i *CopilotInstruction) ResourceType() string {
	return "copilot_instruction"
}

// ResourceName returns the instruction's name identifier.
func (i *CopilotInstruction) ResourceName() string {
	return i.Name
}

// Platform returns the target platform for Copilot instructions.
func (i *CopilotInstruction) Platform() string {
	return "github-copilot"
}

// GetContent returns the instruction's content.
func (i *CopilotInstruction) GetContent() string {
	return i.Content
}

// GetFiles returns the instruction's file blocks.
func (i *CopilotInstruction) GetFiles() []FileBlock {
	return i.Files
}

// GetTemplateFiles returns the instruction's template file blocks.
func (i *CopilotInstruction) GetTemplateFiles() []TemplateFileBlock {
	return i.TemplateFiles
}

// Validate checks that the instruction has all required fields.
func (i *CopilotInstruction) Validate() error {
	if i.Name == "" {
		return fmt.Errorf("copilot_instruction: name is required")
	}
	if i.Description == "" {
		return fmt.Errorf("copilot_instruction %q: description is required", i.Name)
	}
	if i.Content == "" {
		return fmt.Errorf("copilot_instruction %q: content is required", i.Name)
	}
	return nil
}

// CopilotMCPServer represents an MCP server configuration merged into .vscode/mcp.json.
// MCP servers provide additional tools and capabilities to GitHub Copilot.
type CopilotMCPServer struct {
	// Name is the unique identifier for this MCP server
	Name string `hcl:"name,label"`

	// Description explains what this MCP server provides
	Description string `hcl:"description,optional"`

	// Type specifies the server type: "stdio" or "http"/"sse"
	Type string `hcl:"type,attr"`

	// Command is the executable to run (required for type="stdio")
	Command string `hcl:"command,optional"`

	// Args are the command-line arguments for the command
	Args []string `hcl:"args,optional"`

	// Env contains environment variables for the server
	Env map[string]string `hcl:"env,optional"`

	// EnvFile is the path to an env file to load
	EnvFile string `hcl:"env_file,optional"`

	// URL is the HTTP/SSE endpoint (required for type="http" or type="sse")
	URL string `hcl:"url,optional"`

	// Headers contains HTTP headers for http/sse type servers
	Headers map[string]string `hcl:"headers,optional"`
}

// ResourceType returns the HCL block type for Copilot MCP servers.
func (m *CopilotMCPServer) ResourceType() string {
	return "copilot_mcp_server"
}

// ResourceName returns the MCP server's name identifier.
func (m *CopilotMCPServer) ResourceName() string {
	return m.Name
}

// Platform returns the target platform for Copilot MCP servers.
func (m *CopilotMCPServer) Platform() string {
	return "github-copilot"
}

// GetContent returns an empty string as MCP servers don't have content.
func (m *CopilotMCPServer) GetContent() string {
	return ""
}

// GetFiles returns nil as MCP servers don't have associated files.
func (m *CopilotMCPServer) GetFiles() []FileBlock {
	return nil
}

// GetTemplateFiles returns nil as MCP servers don't have associated template files.
func (m *CopilotMCPServer) GetTemplateFiles() []TemplateFileBlock {
	return nil
}

// Validate checks that the MCP server has all required fields.
func (m *CopilotMCPServer) Validate() error {
	if m.Name == "" {
		return fmt.Errorf("copilot_mcp_server: name is required")
	}
	if m.Type == "" {
		return fmt.Errorf("copilot_mcp_server %q: type is required", m.Name)
	}
	validTypes := map[string]bool{"stdio": true, "http": true, "sse": true}
	if !validTypes[m.Type] {
		return fmt.Errorf("copilot_mcp_server %q: type must be 'stdio', 'http', or 'sse', got %q", m.Name, m.Type)
	}
	if m.Type == "stdio" {
		if m.Command == "" {
			return fmt.Errorf("copilot_mcp_server %q: command is required for type 'stdio'", m.Name)
		}
	}
	if m.Type == "http" || m.Type == "sse" {
		if m.URL == "" {
			return fmt.Errorf("copilot_mcp_server %q: url is required for type %q", m.Name, m.Type)
		}
	}
	return nil
}

// =============================================================================
// STANDALONE RESOURCES (one file per resource)
// =============================================================================

// CopilotInstructions (plural) represents a standalone instruction file.
// Similar to ClaudeRules which creates standalone files in .claude/rules/,
// this creates standalone files in .github/instructions/.
type CopilotInstructions struct {
	// Name is the block label identifying this instructions file
	Name string `hcl:"name,label"`

	// Description explains what these instructions provide
	Description string `hcl:"description,attr"`

	// Content is the instruction text
	Content string `hcl:"content,optional"`

	// ApplyTo is an optional glob pattern for selective application
	ApplyTo string `hcl:"apply_to,optional"`

	// Files lists static files to copy alongside the instructions
	Files []FileBlock `hcl:"file,block"`

	// TemplateFiles lists template files to render and copy
	TemplateFiles []TemplateFileBlock `hcl:"template_file,block"`
}

// ResourceType returns the HCL block type for Copilot instructions (plural).
func (i *CopilotInstructions) ResourceType() string {
	return "copilot_instructions"
}

// ResourceName returns the instructions file's name identifier.
func (i *CopilotInstructions) ResourceName() string {
	return i.Name
}

// Platform returns the target platform for Copilot instructions files.
func (i *CopilotInstructions) Platform() string {
	return "github-copilot"
}

// GetContent returns the instructions content.
func (i *CopilotInstructions) GetContent() string {
	return i.Content
}

// GetFiles returns the instructions file blocks.
func (i *CopilotInstructions) GetFiles() []FileBlock {
	return i.Files
}

// GetTemplateFiles returns the instructions template file blocks.
func (i *CopilotInstructions) GetTemplateFiles() []TemplateFileBlock {
	return i.TemplateFiles
}

// Validate checks that the instructions have all required fields.
func (i *CopilotInstructions) Validate() error {
	if i.Name == "" {
		return fmt.Errorf("copilot_instructions: name is required")
	}
	if i.Description == "" {
		return fmt.Errorf("copilot_instructions %q: description is required", i.Name)
	}
	if i.Content == "" {
		return fmt.Errorf("copilot_instructions %q: content is required", i.Name)
	}
	return nil
}

// CopilotPrompt represents a prompt for GitHub Copilot.
// Prompts are installed to .github/prompts/{plugin}-{name}.prompt.md
// and can be invoked by users.
type CopilotPrompt struct {
	// Name is the block label identifying this prompt
	Name string `hcl:"name,label"`

	// Description explains what this prompt does
	Description string `hcl:"description,attr"`

	// Content is the prompt body
	Content string `hcl:"content,optional"`

	// ArgumentHint is an optional hint shown during autocomplete
	ArgumentHint string `hcl:"argument_hint,optional"`

	// Agent specifies the agent mode: "ask", "edit", "agent", or custom
	Agent string `hcl:"agent,optional"`

	// Model is an optional model selection
	Model string `hcl:"model,optional"`

	// Tools is an optional list of tools to enable
	Tools []string `hcl:"tools,optional"`

	// Files lists static files to copy alongside the prompt
	Files []FileBlock `hcl:"file,block"`

	// TemplateFiles lists template files to render and copy
	TemplateFiles []TemplateFileBlock `hcl:"template_file,block"`
}

// ResourceType returns the HCL block type for Copilot prompts.
func (p *CopilotPrompt) ResourceType() string {
	return "copilot_prompt"
}

// ResourceName returns the prompt's name identifier.
func (p *CopilotPrompt) ResourceName() string {
	return p.Name
}

// Platform returns the target platform for Copilot prompts.
func (p *CopilotPrompt) Platform() string {
	return "github-copilot"
}

// GetContent returns the prompt's content.
func (p *CopilotPrompt) GetContent() string {
	return p.Content
}

// GetFiles returns the prompt's file blocks.
func (p *CopilotPrompt) GetFiles() []FileBlock {
	return p.Files
}

// GetTemplateFiles returns the prompt's template file blocks.
func (p *CopilotPrompt) GetTemplateFiles() []TemplateFileBlock {
	return p.TemplateFiles
}

// Validate checks that the prompt has all required fields.
func (p *CopilotPrompt) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("copilot_prompt: name is required")
	}
	if p.Description == "" {
		return fmt.Errorf("copilot_prompt %q: description is required", p.Name)
	}
	if p.Content == "" {
		return fmt.Errorf("copilot_prompt %q: content is required", p.Name)
	}
	return nil
}

// CopilotAgent represents an agent for GitHub Copilot.
// Agents are installed to .github/agents/{plugin}-{name}.agent.md
// and provide specialized agent behaviors.
type CopilotAgent struct {
	// Name is the block label identifying this agent
	Name string `hcl:"name,label"`

	// Description explains what this agent does
	Description string `hcl:"description,attr"`

	// Content is the agent instructions
	Content string `hcl:"content,optional"`

	// Model is an optional model selection
	Model string `hcl:"model,optional"`

	// Tools lists available tools for this agent
	Tools []string `hcl:"tools,optional"`

	// Handoffs lists sequential workflow transitions to other agents
	Handoffs []string `hcl:"handoffs,optional"`

	// Infer enables subagent usage (default: true)
	Infer *bool `hcl:"infer,optional"`

	// Target specifies the target environment: "vscode" or "github-copilot"
	Target string `hcl:"target,optional"`

	// Files lists static files to copy alongside the agent
	Files []FileBlock `hcl:"file,block"`

	// TemplateFiles lists template files to render and copy
	TemplateFiles []TemplateFileBlock `hcl:"template_file,block"`
}

// ResourceType returns the HCL block type for Copilot agents.
func (a *CopilotAgent) ResourceType() string {
	return "copilot_agent"
}

// ResourceName returns the agent's name identifier.
func (a *CopilotAgent) ResourceName() string {
	return a.Name
}

// Platform returns the target platform for Copilot agents.
func (a *CopilotAgent) Platform() string {
	return "github-copilot"
}

// GetContent returns the agent's content.
func (a *CopilotAgent) GetContent() string {
	return a.Content
}

// GetFiles returns the agent's file blocks.
func (a *CopilotAgent) GetFiles() []FileBlock {
	return a.Files
}

// GetTemplateFiles returns the agent's template file blocks.
func (a *CopilotAgent) GetTemplateFiles() []TemplateFileBlock {
	return a.TemplateFiles
}

// Validate checks that the agent has all required fields.
func (a *CopilotAgent) Validate() error {
	if a.Name == "" {
		return fmt.Errorf("copilot_agent: name is required")
	}
	if a.Description == "" {
		return fmt.Errorf("copilot_agent %q: description is required", a.Name)
	}
	if a.Content == "" {
		return fmt.Errorf("copilot_agent %q: content is required", a.Name)
	}
	return nil
}

// CopilotSkill represents a skill for GitHub Copilot.
// Skills are installed to .github/skills/{plugin}-{name}/SKILL.md
// and provide specialized knowledge or capabilities.
type CopilotSkill struct {
	// Name is the block label identifying this skill (max 64 chars)
	Name string `hcl:"name,label"`

	// Description explains when and how to use this skill (max 1024 chars)
	Description string `hcl:"description,attr"`

	// Content is the skill instructions
	Content string `hcl:"content,optional"`

	// Files lists static files to copy alongside the skill
	Files []FileBlock `hcl:"file,block"`

	// TemplateFiles lists template files to render and copy
	TemplateFiles []TemplateFileBlock `hcl:"template_file,block"`
}

// ResourceType returns the HCL block type for Copilot skills.
func (s *CopilotSkill) ResourceType() string {
	return "copilot_skill"
}

// ResourceName returns the skill's name identifier.
func (s *CopilotSkill) ResourceName() string {
	return s.Name
}

// Platform returns the target platform for Copilot skills.
func (s *CopilotSkill) Platform() string {
	return "github-copilot"
}

// GetContent returns the skill's content.
func (s *CopilotSkill) GetContent() string {
	return s.Content
}

// GetFiles returns the skill's file blocks.
func (s *CopilotSkill) GetFiles() []FileBlock {
	return s.Files
}

// GetTemplateFiles returns the skill's template file blocks.
func (s *CopilotSkill) GetTemplateFiles() []TemplateFileBlock {
	return s.TemplateFiles
}

// Validate checks that the skill has all required fields.
func (s *CopilotSkill) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("copilot_skill: name is required")
	}
	if len(s.Name) > 64 {
		return fmt.Errorf("copilot_skill %q: name must be at most 64 characters", s.Name)
	}
	if s.Description == "" {
		return fmt.Errorf("copilot_skill %q: description is required", s.Name)
	}
	if len(s.Description) > 1024 {
		return fmt.Errorf("copilot_skill %q: description must be at most 1024 characters", s.Name)
	}
	if s.Content == "" {
		return fmt.Errorf("copilot_skill %q: content is required", s.Name)
	}
	return nil
}
