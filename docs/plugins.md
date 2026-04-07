# Package Development

Packages are the core of Dex. They contain skills, commands, rules, agents, and MCP server configurations that extend AI coding assistants.

## Package Structure

```
my-package/
├── package.hcl           # Package manifest (required)
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

The `package.hcl` file defines your package's metadata and resources using HCL syntax.

```hcl
meta {
  name        = "my-package"
  version     = "1.0.0"
  description = "A useful package for AI coding assistants"
  author      = "Your Name"
  license     = "MIT"
  repository  = "https://github.com/owner/my-package"
  platforms   = ["claude-code", "cursor", "github-copilot"]
}

variable "api_endpoint" {
  description = "API endpoint URL"
  default     = "https://api.example.com"
}

skill "code-review" {
  description = "Performs thorough code reviews"
  content     = file("skills/code-review.md")
}

command "deploy" {
  description = "Deploy the application"
  content     = templatefile("commands/deploy.md.tmpl", {
    environment = "production"
  })
}
```

### Meta Block

| Attribute | Required | Description |
|-----------|----------|-------------|
| `name` | yes | Package name (lowercase, hyphens allowed) |
| `version` | yes | Semantic version (e.g., "1.0.0") |
| `description` | no | Human-readable description |
| `author` | no | Package author |
| `license` | no | License identifier (e.g., "MIT") |
| `repository` | no | Source repository URL |
| `platforms` | no | Supported platforms (empty = all) |

### Variable Block

Variables allow users to customize package behavior at installation time.

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

Reads the contents of a file relative to the package directory.

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
| `{{ .PackageName }}` | Name of the package being installed |
| `{{ .PackageVersion }}` | Version of the package |
| `{{ .ProjectRoot }}` | Absolute path to the project root |
| `{{ .Platform }}` | Target platform (e.g., `claude-code`) |

### Template Example

```markdown
# Setup for {{ .PackageName }}

Install dependencies:

```bash
cd {{ .ComponentDir }}
pip install -r requirements.txt
```

This skill is part of {{ .PackageName }} v{{ .PackageVersion }}.
```

## Platform Resource Mapping

Each resource type maps to different locations depending on the target platform:

### Skills

| Platform | Location |
|----------|----------|
| Claude Code | `.claude/skills/{pkg}-{name}/SKILL.md` |
| GitHub Copilot | `.github/skills/{pkg}-{name}/SKILL.md` |
| Cursor | Not supported |

### Commands

| Platform | Location |
|----------|----------|
| Claude Code | `.claude/commands/{pkg}-{name}.md` |
| Cursor | `.cursor/commands/{pkg}-{name}.md` |
| GitHub Copilot | `.github/prompts/{pkg}-{name}.prompt.md` |

### Agents

| Platform | Location |
|----------|----------|
| Claude Code | `.claude/agents/{pkg}-{name}.md` |
| GitHub Copilot | `.github/agents/{pkg}-{name}.agent.md` |
| Cursor | Not supported |

### Rules (Merged)

Content merged into a single file with markers.

| Platform | Location |
|----------|----------|
| Claude Code | `CLAUDE.md` |
| Cursor | `AGENTS.md` |
| GitHub Copilot | `.github/copilot-instructions.md` |

### Rules (Standalone)

Individual rule files per package.

| Platform | Location |
|----------|----------|
| Claude Code | `.claude/rules/{pkg}-{name}.md` |
| Cursor | `.cursor/rules/{pkg}-{name}.mdc` |
| GitHub Copilot | `.github/instructions/{pkg}-{name}.instructions.md` |

### MCP Servers

| Platform | Location |
|----------|----------|
| Claude Code | `.mcp.json` |
| Cursor | `.cursor/mcp.json` |
| GitHub Copilot | `.vscode/mcp.json` |

## Universal Resource Types

All resources use universal block types that translate to the correct platform format at install time. See [RESOURCES.md](RESOURCES.md) for complete documentation.

- `skill` - Specialized knowledge/capabilities (Claude Code, GitHub Copilot)
- `command` - User-invokable commands (all platforms)
- `agent` - Specialized agent behaviors (Claude Code, GitHub Copilot)
- `rule` - Rules merged into agent file (all platforms)
- `rules` - Standalone rule files (all platforms)
- `settings` - Platform settings/permissions (Claude Code only)
- `mcp_server` - MCP server configurations (all platforms)

## File and Template File Blocks

Resources can include additional files:

### file Block

Copies a static file alongside the resource.

```hcl
skill "data-validation" {
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

### template_file Block

Renders a template and copies the result.

```hcl
command "setup" {
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

## Cross-Platform Packages

With universal resource types, cross-platform packages are simple — define once, use everywhere:

```hcl
meta {
  name      = "code-review-tools"
  version   = "1.0.0"
}

# One skill definition works on Claude Code and GitHub Copilot.
# Cursor doesn't support skills — automatically skipped with a log warning.
skill "code-review" {
  description = "Thorough code review capability"
  content     = file("content/code-review.md")
}

# One command definition works on all platforms.
# Translates to: Claude command, Copilot prompt, Cursor command.
command "review" {
  description = "Run code review"
  content     = file("commands/review.md")

  # Platform-specific overrides where needed
  claude {
    allowed_tools = ["Read", "Grep"]
  }

  copilot {
    agent = "edit"
    tools = ["terminal"]
  }
}
```

## Publishing Packages

### Creating a Tarball

```bash
cd my-package
tar -czvf ../my-package-1.0.0.tar.gz .
```

### Publishing to Git

Simply push your package to a Git repository:

```bash
git tag v1.0.0
git push origin v1.0.0
```

Users can install directly:

```bash
dex sync git+https://github.com/owner/my-package.git@v1.0.0
```

## Best Practices

1. **Use descriptive names** - Choose clear, meaningful names for skills and commands
2. **Document thoroughly** - Include usage examples in content files
3. **Share content** - Use `file()` to share content across platforms
4. **Use templates** - Use `templatefile()` for platform-specific variations
5. **Pin dependencies** - Use specific version ranges for stability
6. **Test on all platforms** - Verify your package works on each target platform
