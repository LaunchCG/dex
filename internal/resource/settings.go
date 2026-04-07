package resource

import "fmt"

// Settings represents a universal settings resource. Platform-specific settings
// go inside the platform override blocks (claude {}, copilot {}, cursor {}).
type Settings struct {
	Name      string   `hcl:"name,label"`
	Platforms []string `hcl:"platforms,optional"`

	Claude  *SettingsClaudeOverride `hcl:"claude,block"`
	Copilot *PlatformOverride       `hcl:"copilot,block"`
	Cursor  *PlatformOverride       `hcl:"cursor,block"`
}

// SettingsClaudeOverride contains all Claude-specific settings fields.
type SettingsClaudeOverride struct {
	Disabled                   bool              `hcl:"disabled,optional"`
	Allow                      []string          `hcl:"allow,optional"`
	Ask                        []string          `hcl:"ask,optional"`
	Deny                       []string          `hcl:"deny,optional"`
	Env                        map[string]string `hcl:"env,optional"`
	EnableAllProjectMCPServers bool              `hcl:"enable_all_project_mcp_servers,optional"`
	EnabledMCPServers          []string          `hcl:"enabled_mcp_servers,optional"`
	DisabledMCPServers         []string          `hcl:"disabled_mcp_servers,optional"`
	RespectGitignore           bool              `hcl:"respect_gitignore,optional"`
	IncludeCoAuthoredBy        bool              `hcl:"include_co_authored_by,optional"`
	Model                      string            `hcl:"model,optional"`
	OutputStyle                string            `hcl:"output_style,optional"`
	AlwaysThinkingEnabled      bool              `hcl:"always_thinking_enabled,optional"`
	PlansDirectory             string            `hcl:"plans_directory,optional"`
}

func (s *Settings) ResourceType() string                  { return "settings" }
func (s *Settings) ResourceName() string                  { return s.Name }
func (s *Settings) Platform() string                      { return "universal" }
func (s *Settings) GetContent() string                    { return "" }
func (s *Settings) GetFiles() []FileBlock                 { return nil }
func (s *Settings) GetTemplateFiles() []TemplateFileBlock { return nil }

func (s *Settings) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("settings: name is required")
	}
	return nil
}

func (s *Settings) IsEnabledForPlatform(platform string) bool {
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
