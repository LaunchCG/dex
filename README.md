Dex
===
Dex is an AI Context Manager for AI-augmented development tools (e.g. Claude Code, Cursor, etc)
that provides a standardized way to define, distribute, and install capabilities across multiple AI
agent platforms.


### Core Principles

- **Write Once, Deploy Everywhere**: Plugins are defined in a platform-agnostic format
- **Non-Prescriptive**: Authors declare components; adapters handle platform specifics
- **Composable**: Plugins can depend on and reference other plugins
- **Extensible**: Support for scripts, configs, MCP servers, and arbitrary files
- **Context-Aware**: Templating engine for platform and environment-specific variations


### Supported Platforms

| Feature | Claude Code | Cursor | GitHub Copilot | Codex | Antigravity |
|---------|:-----------:|:------:|:--------------:|:-----:|:-----------:|
| Skills | ✓ | - | ✓ | ✓ | ✓ |
| Commands | ✓ | - | - | - | - |
| Sub-agents | ✓ | - | ✓ | - | - |
| Rules | ✓ | ✓ | ✓ | ✓ | ✓ |
| Instructions | - | - | ✓ | - | - |
| Prompts | - | - | ✓ | - | - |
| Agent File | ✓ | - | - | ✓ | - |
| MCP Servers | ✓ | ✓ | ✓ | ✓ | - |

**Notes:**
- Agent File: Content injected into `CLAUDE.md` (Claude Code) or `AGENTS.md` (Codex) with markers
- Antigravity MCP: Configured through UI only, not project-level config files

See [docs/plugins.md](docs/plugins.md) for detailed platform mapping and file locations


### Installation

Run dex directly using `uvx` without installing:

```bash
uvx --from git+https://github.com/launchcg/dex dex <command>
```

Or install globally:

```bash
uv tool install git+https://github.com/launchcg/dex
```


### Quick Start

```bash
# Initialize a project for a specific AI agent
uvx --from git+https://github.com/launchcg/dex dex init --agent claude-code

# Install a plugin from GitHub
uvx --from git+https://github.com/launchcg/dex dex install --source git+https://github.com/owner/my-plugin.git

# Install from a local directory (for development)
uvx --from git+https://github.com/launchcg/dex dex install --source /path/to/my-plugin

# Install and save to dex.yaml
uvx --from git+https://github.com/launchcg/dex dex install --source git+https://github.com/owner/my-plugin.git --save

# Remove a plugin
uvx --from git+https://github.com/launchcg/dex dex uninstall my-plugin

# Remove and delete from dex.yaml
uvx --from git+https://github.com/launchcg/dex dex uninstall my-plugin --remove

# List installed plugins
uvx --from git+https://github.com/launchcg/dex dex list
```


### Creating a Plugin

A plugin is a directory with a `package.json` manifest:

```
my-plugin/
├── package.json          # Plugin manifest
├── skills/               # Skill context files
│   └── code-review.md
├── rules/                # Rule context files
│   └── code-style.md
└── commands/             # Command context files
    └── lint.md
```

**package.json:**
```json
{
  "name": "my-plugin",
  "version": "1.0.0",
  "description": "A useful plugin",
  "skills": [
    {
      "name": "code-review",
      "description": "Automated code review",
      "context": "./skills/code-review.md"
    }
  ],
  "rules": [
    {
      "name": "code-style",
      "description": "Code style guidelines",
      "context": "./rules/code-style.md",
      "glob": "**/*.py",
      "paths": ["src/**/*.py"]
    }
  ]
}
```

### Installed Directory Structure

When installed for Claude Code, skills are directories containing `SKILL.md` and any associated files:

```
project/
└── .claude/
    ├── skills/
    │   └── my-plugin-code-review/
    │       ├── SKILL.md           # Skill context with frontmatter
    │       ├── scripts/           # Associated files preserved
    │       │   └── setup.sh
    │       └── configs/
    │           └── settings.json
    └── rules/
        └── my-rule.md             # Rule with optional paths frontmatter
```

Other platforms have different structures - see [docs/plugins.md](docs/plugins.md) for details.


### Platform-Specific Context Files

Use file naming conventions to provide different content for different platforms:

```
my-plugin/
├── context/
│   ├── skill.md                  # Default
│   ├── skill.claude_code.md      # Claude Code override
│   └── skill.cursor.md           # Cursor override
```

Dex automatically resolves to the platform-specific version during installation.


### Template Variables

Context files support Jinja2 templating, enabling shared content with client-specific customizations:

```markdown
# Code Review Skill

Plugin: {{ plugin.name }} v{{ plugin.version }}

## Common Guidelines

These guidelines apply to all platforms:
- Follow consistent naming conventions
- Write clear commit messages
- Include tests for new features

{% if agent.name == 'claude-code' %}
## Claude Code Instructions

Use the Task tool for complex multi-step operations.
Prefer the Edit tool over Bash for file modifications.
{% elif agent.name == 'cursor' %}
## Cursor Instructions

Use @codebase to search across the project.
Apply changes incrementally with Composer.
{% elif agent.name == 'codex' %}
## Codex Instructions

Reference the AGENTS.md file for project conventions.
{% endif %}

{% if platform.os == 'windows' %}
Use PowerShell for shell commands.
{% else %}
Use Bash for shell commands.
{% endif %}
```

This approach lets you:
- **De-duplicate** common content across all clients
- **Customize** instructions for each client's unique capabilities
- **Adapt** to the user's operating system

For completely different content per platform, use [platform-specific files](#platform-specific-context-files) instead.

See [docs/plugins.md](docs/plugins.md) for all available variables and platform-specific metadata.
