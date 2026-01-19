# Dex Documentation

Dex is a universal package manager for AI-augmented development tools. It manages skills, commands, sub-agents, and MCP servers across different AI coding assistants like Claude Code, Cursor, and Codex.

## Key Features

- **Universal Plugin Format**: Write plugins once, deploy to any supported AI assistant
- **Semantic Versioning**: Full semver support with version ranges (^, ~, >=, etc.)
- **Lock Files**: Deterministic installations with `sdlc.lock`
- **Template Rendering**: Platform and environment-aware context files using Jinja2
- **MCP Server Management**: Automatic configuration of Model Context Protocol servers
- **Local Registries**: File-based registries for private or offline use

## Quick Start

```bash
# Initialize a new project
dex init --agent claude-code

# Add a plugin to your project
dex install my-plugin@^1.0.0

# Install all plugins from sdlc.json
dex install

# List installed plugins
dex list
```

## Documentation

- [Installation](installation.md) - How to install Dex
- [Configuration](configuration.md) - sdlc.json and registry configuration
- [Plugins](plugins.md) - Creating and publishing plugins
- [CLI Reference](cli-reference.md) - Complete command reference

## Project Structure

After initialization, your project will have:

```
my-project/
├── sdlc.json           # Project configuration
├── sdlc.lock           # Lock file (after first install)
├── .mcp.json           # MCP server configuration (Claude Code)
└── .claude/            # Claude Code specific directory
    ├── settings.json   # Permissions/settings (NOT MCP config)
    └── skills/         # Installed skills
        └── plugin-name/
            └── skill-name/
                └── SKILL.md
```

## Example

```json
// sdlc.json
{
  "agent": "claude-code",
  "project_name": "my-project",
  "plugins": {
    "linting-skills": "^1.0.0",
    "git-helpers": "file:./local-plugins/git"
  },
  "registries": {
    "local": "file:./registry"
  },
  "default_registry": "local"
}
```

## License

MIT License - see [LICENSE](../LICENSE) for details.
