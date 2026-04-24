package resource

// TranslateToClaudeMCPServer converts a unified MCPServer to a Claude-specific MCP server.
// Returns nil if the server is disabled for Claude platform.
func TranslateToClaudeMCPServer(mcp *MCPServer) *ClaudeMCPServer {
	// Check if server is enabled for Claude platform
	if !mcp.IsEnabledForPlatform("claude-code") {
		return nil
	}

	// Start with base configuration
	server := &ClaudeMCPServer{
		Name:        mcp.Name,
		Description: mcp.Description,
		Command:     mcp.Command,
		Args:        mcp.Args,
		Env:         mcp.Env,
		URL:         mcp.URL,
	}

	// Determine type based on transport
	if mcp.Command != "" {
		server.Type = "command"
	} else if mcp.URL != "" {
		server.Type = "http"
	}

	// Apply Claude-specific overrides if present
	if mcp.Claude != nil {
		if mcp.Claude.Disabled {
			return nil
		}
		applyClaudeOverride(server, mcp.Claude)
	}

	return server
}

// applyClaudeOverride applies platform-specific overrides to a Claude MCP server.
func applyClaudeOverride(server *ClaudeMCPServer, override *MCPServerPlatformOverride) {
	if override.Command != "" {
		server.Command = override.Command
		server.Type = "command"
	}
	if override.URL != "" {
		server.URL = override.URL
		server.Type = "http"
	}
	if len(override.Args) > 0 {
		server.Args = override.Args
	}
	if len(override.Env) > 0 {
		server.Env = override.Env
	}
}

// TranslateToCursorMCPServer converts a unified MCPServer to a Cursor-specific MCP server.
// Returns nil if the server is disabled for Cursor platform.
func TranslateToCursorMCPServer(mcp *MCPServer) *CursorMCPServer {
	// Check if server is enabled for Cursor platform
	if !mcp.IsEnabledForPlatform("cursor") {
		return nil
	}

	// Start with base configuration
	server := &CursorMCPServer{
		Name:        mcp.Name,
		Description: mcp.Description,
		Command:     mcp.Command,
		Args:        mcp.Args,
		Env:         mcp.Env,
		URL:         mcp.URL,
		EnvFile:     mcp.EnvFile,
		Headers:     mcp.Headers,
	}

	// Determine type based on transport
	if mcp.Command != "" {
		server.Type = "stdio"
	} else if mcp.URL != "" {
		server.Type = "http"
	}

	// Apply Cursor-specific overrides if present
	if mcp.Cursor != nil {
		if mcp.Cursor.Disabled {
			return nil
		}
		applyCursorOverride(server, mcp.Cursor)
	}

	return server
}

// applyCursorOverride applies platform-specific overrides to a Cursor MCP server.
func applyCursorOverride(server *CursorMCPServer, override *MCPServerPlatformOverride) {
	if override.Command != "" {
		server.Command = override.Command
		server.Type = "stdio"
	}
	if override.URL != "" {
		server.URL = override.URL
		server.Type = "http"
	}
	if len(override.Args) > 0 {
		server.Args = override.Args
	}
	if len(override.Env) > 0 {
		server.Env = override.Env
	}
	if override.EnvFile != "" {
		server.EnvFile = override.EnvFile
	}
	if len(override.Headers) > 0 {
		server.Headers = override.Headers
	}
}

// TranslateToCopilotMCPServer converts a unified MCPServer to a Copilot-specific MCP server.
// Returns nil if the server is disabled for Copilot platform.
func TranslateToCopilotMCPServer(mcp *MCPServer) *CopilotMCPServer {
	// Check if server is enabled for Copilot platform
	if !mcp.IsEnabledForPlatform("github-copilot") {
		return nil
	}

	// Start with base configuration
	server := &CopilotMCPServer{
		Name:        mcp.Name,
		Description: mcp.Description,
		Command:     mcp.Command,
		Args:        mcp.Args,
		Env:         mcp.Env,
		URL:         mcp.URL,
		EnvFile:     mcp.EnvFile,
		Headers:     mcp.Headers,
		Inputs:      mcp.Inputs,
	}

	// Determine type based on transport
	if mcp.Command != "" {
		server.Type = "stdio"
	} else if mcp.URL != "" {
		server.Type = "http"
	}

	// Apply Copilot-specific overrides if present
	if mcp.Copilot != nil {
		if mcp.Copilot.Disabled {
			return nil
		}
		applyCopilotOverride(server, mcp.Copilot)
	}

	return server
}

// applyCopilotOverride applies platform-specific overrides to a Copilot MCP server.
func applyCopilotOverride(server *CopilotMCPServer, override *MCPServerPlatformOverride) {
	if override.Command != "" {
		server.Command = override.Command
		server.Type = "stdio"
	}
	if override.URL != "" {
		server.URL = override.URL
		server.Type = "http"
	}
	if len(override.Args) > 0 {
		server.Args = override.Args
	}
	if len(override.Env) > 0 {
		server.Env = override.Env
	}
	if override.EnvFile != "" {
		server.EnvFile = override.EnvFile
	}
	if len(override.Headers) > 0 {
		server.Headers = override.Headers
	}
	if len(override.Inputs) > 0 {
		server.Inputs = override.Inputs
	}
}

// =============================================================================
// Skill Translators
// =============================================================================

// TranslateToClaudeSkill converts a universal Skill to a Claude-specific skill.
// Returns nil if disabled for Claude.
func TranslateToClaudeSkill(s *Skill) *ClaudeSkill {
	if !s.IsEnabledForPlatform("claude-code") {
		return nil
	}

	cs := &ClaudeSkill{
		Name:          s.Name,
		Description:   s.Description,
		Content:       s.Content,
		Files:         s.Files,
		TemplateFiles: s.TemplateFiles,
	}

	if s.Claude != nil {
		if s.Claude.Disabled {
			return nil
		}
		if s.Claude.Content != "" {
			cs.Content = s.Claude.Content
		}
		cs.ArgumentHint = s.Claude.ArgumentHint
		cs.Arguments = s.Claude.Arguments
		cs.WhenToUse = s.Claude.WhenToUse
		cs.DisableModelInvocation = s.Claude.DisableModelInvocation
		cs.UserInvocable = s.Claude.UserInvocable
		cs.AllowedTools = s.Claude.AllowedTools
		cs.Model = s.Claude.Model
		cs.Effort = s.Claude.Effort
		cs.Context = s.Claude.Context
		cs.Agent = s.Claude.Agent
		cs.Paths = s.Claude.Paths
		cs.Shell = s.Claude.Shell
		cs.Metadata = s.Claude.Metadata
		cs.Hooks = s.Claude.Hooks
	}

	return cs
}

// TranslateToCopilotSkill converts a universal Skill to a Copilot-specific skill.
// Returns nil if disabled for Copilot.
func TranslateToCopilotSkill(s *Skill) *CopilotSkill {
	if !s.IsEnabledForPlatform("github-copilot") {
		return nil
	}

	cs := &CopilotSkill{
		Name:          s.Name,
		Description:   s.Description,
		Content:       s.Content,
		Files:         s.Files,
		TemplateFiles: s.TemplateFiles,
	}

	if s.Copilot != nil {
		if s.Copilot.Disabled {
			return nil
		}
		if s.Copilot.Content != "" {
			cs.Content = s.Copilot.Content
		}
	}

	return cs
}

// =============================================================================
// Command Translators
// =============================================================================

// TranslateToClaudeCommand converts a universal Command to a Claude-specific command.
// Returns nil if disabled for Claude.
//
// Claude Code merged custom commands into skills, so a Command translates
// using the same field set as a Skill. We reuse the skill translator and
// convert to ClaudeCommand (which is a named type of ClaudeSkill).
func TranslateToClaudeCommand(c *Command) *ClaudeCommand {
	skill := TranslateToClaudeSkill(&Skill{
		Name:          c.Name,
		Description:   c.Description,
		Content:       c.Content,
		Files:         c.Files,
		TemplateFiles: c.TemplateFiles,
		Platforms:     c.Platforms,
		Claude:        c.Claude, // CommandClaudeOverride is a type alias for SkillClaudeOverride
	})
	if skill == nil {
		return nil
	}
	return (*ClaudeCommand)(skill)
}

// TranslateToCopilotPrompt converts a universal Command to a Copilot prompt.
// Returns nil if disabled for Copilot.
func TranslateToCopilotPrompt(c *Command) *CopilotPrompt {
	if !c.IsEnabledForPlatform("github-copilot") {
		return nil
	}

	cp := &CopilotPrompt{
		Name:          c.Name,
		Description:   c.Description,
		Content:       c.Content,
		Files:         c.Files,
		TemplateFiles: c.TemplateFiles,
	}

	if c.Copilot != nil {
		if c.Copilot.Disabled {
			return nil
		}
		if c.Copilot.Content != "" {
			cp.Content = c.Copilot.Content
		}
		cp.ArgumentHint = c.Copilot.ArgumentHint
		cp.Agent = c.Copilot.Agent
		cp.Model = c.Copilot.Model
		cp.Tools = c.Copilot.Tools
	}

	return cp
}

// TranslateToCursorCommand converts a universal Command to a Cursor-specific command.
// Returns nil if disabled for Cursor.
func TranslateToCursorCommand(c *Command) *CursorCommand {
	if !c.IsEnabledForPlatform("cursor") {
		return nil
	}

	cc := &CursorCommand{
		Name:          c.Name,
		Description:   c.Description,
		Content:       c.Content,
		Files:         c.Files,
		TemplateFiles: c.TemplateFiles,
	}

	if c.Cursor != nil {
		if c.Cursor.Disabled {
			return nil
		}
		if c.Cursor.Content != "" {
			cc.Content = c.Cursor.Content
		}
	}

	return cc
}

// =============================================================================
// Rule Translators (merged into agent file)
// =============================================================================

// TranslateToClaudeRule converts a universal Rule to a Claude-specific rule.
// Returns nil if disabled for Claude.
func TranslateToClaudeRule(r *Rule) *ClaudeRule {
	if !r.IsEnabledForPlatform("claude-code") {
		return nil
	}

	cr := &ClaudeRule{
		Name:          r.Name,
		Description:   r.Description,
		Content:       r.Content,
		Files:         r.Files,
		TemplateFiles: r.TemplateFiles,
	}

	if r.Claude != nil {
		if r.Claude.Disabled {
			return nil
		}
		if r.Claude.Content != "" {
			cr.Content = r.Claude.Content
		}
		cr.Paths = r.Claude.Paths
	}

	return cr
}

// TranslateToCopilotInstruction converts a universal Rule to a Copilot instruction.
// Returns nil if disabled for Copilot.
func TranslateToCopilotInstruction(r *Rule) *CopilotInstruction {
	if !r.IsEnabledForPlatform("github-copilot") {
		return nil
	}

	ci := &CopilotInstruction{
		Name:          r.Name,
		Description:   r.Description,
		Content:       r.Content,
		Files:         r.Files,
		TemplateFiles: r.TemplateFiles,
	}

	if r.Copilot != nil {
		if r.Copilot.Disabled {
			return nil
		}
		if r.Copilot.Content != "" {
			ci.Content = r.Copilot.Content
		}
	}

	return ci
}

// TranslateToCursorRule converts a universal Rule to a Cursor-specific rule.
// Returns nil if disabled for Cursor.
func TranslateToCursorRule(r *Rule) *CursorRule {
	if !r.IsEnabledForPlatform("cursor") {
		return nil
	}

	cr := &CursorRule{
		Name:        r.Name,
		Description: r.Description,
		Content:     r.Content,
	}

	if r.Cursor != nil {
		if r.Cursor.Disabled {
			return nil
		}
		if r.Cursor.Content != "" {
			cr.Content = r.Cursor.Content
		}
	}

	return cr
}

// =============================================================================
// Rules Translators (standalone files)
// =============================================================================

// TranslateToClaudeRules converts a universal Rules to Claude-specific standalone rules.
// Returns nil if disabled for Claude.
func TranslateToClaudeRules(r *Rules) *ClaudeRules {
	if !r.IsEnabledForPlatform("claude-code") {
		return nil
	}

	cr := &ClaudeRules{
		Name:          r.Name,
		Description:   r.Description,
		Content:       r.Content,
		Files:         r.Files,
		TemplateFiles: r.TemplateFiles,
	}

	if r.Claude != nil {
		if r.Claude.Disabled {
			return nil
		}
		if r.Claude.Content != "" {
			cr.Content = r.Claude.Content
		}
		cr.Paths = r.Claude.Paths
	}

	return cr
}

// TranslateToCopilotInstructions converts a universal Rules to Copilot standalone instructions.
// Returns nil if disabled for Copilot.
func TranslateToCopilotInstructions(r *Rules) *CopilotInstructions {
	if !r.IsEnabledForPlatform("github-copilot") {
		return nil
	}

	ci := &CopilotInstructions{
		Name:          r.Name,
		Description:   r.Description,
		Content:       r.Content,
		Files:         r.Files,
		TemplateFiles: r.TemplateFiles,
	}

	if r.Copilot != nil {
		if r.Copilot.Disabled {
			return nil
		}
		if r.Copilot.Content != "" {
			ci.Content = r.Copilot.Content
		}
		ci.ApplyTo = r.Copilot.ApplyTo
	}

	return ci
}

// TranslateToCursorRules converts a universal Rules to Cursor-specific standalone rules.
// Returns nil if disabled for Cursor.
func TranslateToCursorRules(r *Rules) *CursorRules {
	if !r.IsEnabledForPlatform("cursor") {
		return nil
	}

	cr := &CursorRules{
		Name:          r.Name,
		Description:   r.Description,
		Content:       r.Content,
		Files:         r.Files,
		TemplateFiles: r.TemplateFiles,
	}

	if r.Cursor != nil {
		if r.Cursor.Disabled {
			return nil
		}
		if r.Cursor.Content != "" {
			cr.Content = r.Cursor.Content
		}
		cr.Globs = r.Cursor.Globs
		cr.AlwaysApply = r.Cursor.AlwaysApply
	}

	return cr
}

// =============================================================================
// Agent Translators
// =============================================================================

// TranslateToClaudeSubagent converts a universal Agent to a Claude-specific subagent.
// Returns nil if disabled for Claude.
func TranslateToClaudeSubagent(a *Agent) *ClaudeSubagent {
	if !a.IsEnabledForPlatform("claude-code") {
		return nil
	}

	cs := &ClaudeSubagent{
		Name:          a.Name,
		Description:   a.Description,
		Content:       a.Content,
		Files:         a.Files,
		TemplateFiles: a.TemplateFiles,
	}

	if a.Claude != nil {
		if a.Claude.Disabled {
			return nil
		}
		if a.Claude.Content != "" {
			cs.Content = a.Claude.Content
		}
		cs.Model = a.Claude.Model
		cs.Color = a.Claude.Color
		cs.Tools = a.Claude.Tools
		cs.DisallowedTools = a.Claude.DisallowedTools
		cs.PermissionMode = a.Claude.PermissionMode
		cs.MaxTurns = a.Claude.MaxTurns
		cs.Skills = a.Claude.Skills
		cs.MCPServers = a.Claude.MCPServers
		cs.Memory = a.Claude.Memory
		cs.Background = a.Claude.Background
		cs.Effort = a.Claude.Effort
		cs.Isolation = a.Claude.Isolation
		cs.InitialPrompt = a.Claude.InitialPrompt
		cs.Hooks = a.Claude.Hooks
	}

	return cs
}

// TranslateToCopilotAgent converts a universal Agent to a Copilot-specific agent.
// Returns nil if disabled for Copilot.
func TranslateToCopilotAgent(a *Agent) *CopilotAgent {
	if !a.IsEnabledForPlatform("github-copilot") {
		return nil
	}

	ca := &CopilotAgent{
		Name:          a.Name,
		Description:   a.Description,
		Content:       a.Content,
		Files:         a.Files,
		TemplateFiles: a.TemplateFiles,
	}

	if a.Copilot != nil {
		if a.Copilot.Disabled {
			return nil
		}
		if a.Copilot.Content != "" {
			ca.Content = a.Copilot.Content
		}
		ca.Model = a.Copilot.Model
		ca.Tools = a.Copilot.Tools
		ca.Handoffs = a.Copilot.Handoffs
		ca.Infer = a.Copilot.Infer
		ca.Target = a.Copilot.Target
	}

	return ca
}

// =============================================================================
// Settings Translators
// =============================================================================

// TranslateToClaudeSettings converts a universal Settings to Claude-specific settings.
// Returns nil if disabled for Claude.
func TranslateToClaudeSettings(s *Settings) *ClaudeSettings {
	if !s.IsEnabledForPlatform("claude-code") {
		return nil
	}

	if s.Claude == nil {
		// No claude block — nothing to configure
		return &ClaudeSettings{Name: s.Name}
	}

	if s.Claude.Disabled {
		return nil
	}

	return &ClaudeSettings{
		Name:                       s.Name,
		Allow:                      s.Claude.Allow,
		Ask:                        s.Claude.Ask,
		Deny:                       s.Claude.Deny,
		Env:                        s.Claude.Env,
		EnableAllProjectMCPServers: s.Claude.EnableAllProjectMCPServers,
		EnabledMCPServers:          s.Claude.EnabledMCPServers,
		DisabledMCPServers:         s.Claude.DisabledMCPServers,
		RespectGitignore:           s.Claude.RespectGitignore,
		IncludeCoAuthoredBy:        s.Claude.IncludeCoAuthoredBy,
		Model:                      s.Claude.Model,
		OutputStyle:                s.Claude.OutputStyle,
		AlwaysThinkingEnabled:      s.Claude.AlwaysThinkingEnabled,
		PlansDirectory:             s.Claude.PlansDirectory,
		AdditionalDirectories:      s.Claude.AdditionalDirectories,
		AutoMemoryDirectory:        s.Claude.AutoMemoryDirectory,
		IncludeGitInstructions:     s.Claude.IncludeGitInstructions,
		Agent:                      s.Claude.Agent,
	}
}
