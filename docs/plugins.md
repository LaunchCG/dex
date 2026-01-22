# Plugin Development

Plugins are the core of Dex. They contain skills, commands, rules, agents, and MCP server configurations that extend AI coding assistants.

## Plugin Structure

```
my-plugin/
├── package.hcl           # Plugin manifest (required)
├── content/              # Shared content files
│   └── code-review.md
├── skills/               # Skill-specific content
│   └── testing.md
├── commands/             # Command content
│   └── deploy.md.tmpl
├── rules/                # Rule content
│   └── security.md
└── scripts/              # Helper scripts
    └── setup.sh
```

## Package Manifest (package.hcl)

The `package.hcl` file defines your plugin's metadata and resources using HCL syntax.

```hcl
package {
  name        = "my-plugin"
  version     = "1.0.0"
  description = "A useful plugin for AI coding assistants"
  author      = "Your Name"
  license     = "MIT"
  repository  = "https://github.com/owner/my-plugin"
  platforms   = ["claude-code", "cursor", "github-copilot"]
}

variable "api_endpoint" {
  description = "API endpoint URL"
  default     = "https://api.example.com"
}

claude_skill "code-review" {
  description = "Performs thorough code reviews"
  content     = file("skills/code-review.md")
}

claude_command "deploy" {
  description = "Deploy the application"
  content     = templatefile("commands/deploy.md.tmpl", {
    environment = "production"
  })
}
```

### Package Block

| Attribute | Required | Description |
|-----------|----------|-------------|
| `name` | yes | Plugin name (lowercase, hyphens allowed) |
| `version` | yes | Semantic version (e.g., "1.0.0") |
| `description` | no | Human-readable description |
| `author` | no | Plugin author |
| `license` | no | License identifier (e.g., "MIT") |
| `repository` | no | Source repository URL |
| `platforms` | no | Supported platforms (empty = all) |

### Variable Block

Variables allow users to customize plugin behavior at installation time.

```hcl
variable "python_version" {
  description = "Python version to use"
  default     = "3.11"
  required    = false
  env         = "PYTHON_VERSION"
}
```

| Attribute | Required | Description |
|-----------|----------|-------------|
| `name` | yes | Variable identifier (block label) |
| `description` | no | Variable description |
| `default` | no | Default value |
| `required` | no | Whether user must provide a value |
| `env` | no | Environment variable to read from |

## HCL Functions

### file()

Reads the contents of a file relative to the plugin directory.

```hcl
content = file("skills/my-skill.md")
```

### templatefile()

Reads a template file and renders it with Go `text/template` syntax.

```hcl
content = templatefile("skills/my-skill.md.tmpl", {
  project_name = "MyProject"
  api_version  = "v2"
})
```

### env()

Reads an environment variable with an optional default value.

```hcl
env = {
  API_KEY = env("MY_API_KEY")
  DEBUG   = env("DEBUG", "false")
}
```

## Template Variables

These variables are available in templates rendered via `templatefile()`:

| Variable | Description |
|----------|-------------|
| `{{ .ComponentDir }}` | Absolute path to the installed component directory |
| `{{ .PluginName }}` | Name of the plugin being installed |
| `{{ .PluginVersion }}` | Version of the plugin |
| `{{ .ProjectRoot }}` | Absolute path to the project root |
| `{{ .Platform }}` | Target platform (e.g., `claude-code`) |

### Template Example

```markdown
# Setup for {{ .PluginName }}

Install dependencies:

```bash
cd {{ .ComponentDir }}
pip install -r requirements.txt
```

This skill is part of {{ .PluginName }} v{{ .PluginVersion }}.
```

## Platform Resource Mapping

Each resource type maps to different locations depending on the target platform:

### Skills

| Platform | Location |
|----------|----------|
| Claude Code | `.claude/skills/{plugin}-{name}/SKILL.md` |
| GitHub Copilot | `.github/skills/{plugin}-{name}/SKILL.md` |
| Cursor | Not supported |

### Commands

| Platform | Location |
|----------|----------|
| Claude Code | `.claude/commands/{plugin}-{name}.md` |
| Cursor | `.cursor/commands/{plugin}-{name}.md` |
| GitHub Copilot | Not supported (use prompts) |

### Prompts

| Platform | Location |
|----------|----------|
| GitHub Copilot | `.github/prompts/{plugin}-{name}.prompt.md` |
| Claude Code | Not supported (use commands) |
| Cursor | Not supported |

### Agents/Subagents

| Platform | Location |
|----------|----------|
| Claude Code | `.claude/agents/{plugin}-{name}.md` |
| GitHub Copilot | `.github/agents/{plugin}-{name}.agent.md` |
| Cursor | Not supported |

### Rules (Merged)

Content merged into a single file with markers.

| Platform | Location |
|----------|----------|
| Claude Code | `CLAUDE.md` |
| Cursor | `AGENTS.md` |
| GitHub Copilot | `.github/copilot-instructions.md` |

### Rules (Standalone)

Individual rule files per plugin.

| Platform | Location |
|----------|----------|
| Claude Code | `.claude/rules/{plugin}-{name}.md` |
| Cursor | `.cursor/rules/{plugin}-{name}.mdc` |
| GitHub Copilot | `.github/instructions/{plugin}-{name}.instructions.md` |

### MCP Servers

| Platform | Location |
|----------|----------|
| Claude Code | `.mcp.json` |
| Cursor | `.cursor/mcp.json` |
| GitHub Copilot | `.vscode/mcp.json` |

## Resource Types by Platform

See [resources.md](resources.md) for complete documentation of all resource types.

### Claude Code Resources

- `claude_skill` - Skills with specialized knowledge
- `claude_command` - User-invokable commands (`/{name}`)
- `claude_subagent` - Specialized agents
- `claude_rule` - Rules merged into CLAUDE.md
- `claude_rules` - Standalone rule files
- `claude_settings` - Settings merged into settings.json
- `claude_mcp_server` - MCP server configurations

### Cursor Resources

- `cursor_rule` - Rules merged into AGENTS.md
- `cursor_rules` - Standalone rule files
- `cursor_command` - User-invokable commands
- `cursor_mcp_server` - MCP server configurations

### GitHub Copilot Resources

- `copilot_instruction` - Instructions merged into copilot-instructions.md
- `copilot_instructions` - Standalone instruction files
- `copilot_prompt` - User-invokable prompts
- `copilot_agent` - Specialized agents
- `copilot_skill` - Skills with specialized knowledge
- `copilot_mcp_server` - MCP server configurations

## File and Template File Blocks

Resources can include additional files:

### file Block

Copies a static file alongside the resource.

```hcl
claude_skill "data-validation" {
  description = "Validates JSON data against schemas"
  content     = file("skills/data-validation.md")

  file {
    src   = "schemas/user.schema.json"
    dest  = "schema.json"
  }

  file {
    src   = "scripts/validate.py"
    chmod = "755"
  }
}
```

| Attribute | Required | Description |
|-----------|----------|-------------|
| `src` | yes | Source path relative to plugin root |
| `dest` | no | Destination filename (defaults to basename) |
| `chmod` | no | File permissions (e.g., "755") |

### template_file Block

Renders a template and copies the result.

```hcl
claude_command "setup" {
  description = "Project setup"
  content     = file("commands/setup.md")

  template_file {
    src   = "scripts/config.py.tmpl"
    dest  = "config.py"
    vars = {
      api_endpoint = "https://api.example.com"
    }
  }
}
```

| Attribute | Required | Description |
|-----------|----------|-------------|
| `src` | yes | Source template path |
| `dest` | no | Destination filename |
| `chmod` | no | File permissions |
| `vars` | no | Additional template variables |

## Cross-Platform Plugins

Create plugins that work across multiple platforms by defining resources for each:

```hcl
package {
  name      = "code-review-tools"
  version   = "1.0.0"
  platforms = ["claude-code", "github-copilot", "cursor"]
}

# Shared content - write once, use everywhere
claude_skill "code-review" {
  description = "Thorough code review capability"
  content     = file("content/code-review.md")
}

copilot_skill "code-review" {
  description = "Thorough code review capability"
  content     = file("content/code-review.md")  # Same file!
}

# Platform-specific commands using templates
claude_command "review" {
  description = "Run code review"
  content     = templatefile("commands/review.md.tmpl", {
    tool_name = "Read"
  })
}

copilot_prompt "review" {
  description = "Run code review"
  content     = templatefile("commands/review.md.tmpl", {
    tool_name = "fetch"
  })
}

cursor_command "review" {
  description = "Run code review"
  content     = templatefile("commands/review.md.tmpl", {
    tool_name = "read_file"
  })
}
```

## Publishing Plugins

### Creating a Tarball

```bash
cd my-plugin
tar -czvf ../my-plugin-1.0.0.tar.gz .
```

### Publishing to Git

Simply push your plugin to a Git repository:

```bash
git tag v1.0.0
git push origin v1.0.0
```

Users can install directly:

```bash
dex install git+https://github.com/owner/my-plugin.git@v1.0.0
```

## Best Practices

1. **Use descriptive names** - Choose clear, meaningful names for skills and commands
2. **Document thoroughly** - Include usage examples in content files
3. **Share content** - Use `file()` to share content across platforms
4. **Use templates** - Use `templatefile()` for platform-specific variations
5. **Pin dependencies** - Use specific version ranges for stability
6. **Test on all platforms** - Verify your plugin works on each target platform
