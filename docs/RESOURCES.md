# Dex Resources

Dex uses HCL (HashiCorp Configuration Language) to define resources that are installed into AI agent projects. Resources are **universal** — define them once and they translate automatically to the correct format for each platform (Claude Code, GitHub Copilot, Cursor).

## Table of Contents

- [HCL Functions](#hcl-functions)
- [Template Variables](#template-variables)
- [File Blocks](#file-blocks)
- [Universal Resource Types](#universal-resource-types)
  - [skill](#skill)
  - [command](#command)
  - [agent](#agent)
  - [rule](#rule)
  - [rules](#rules)
  - [settings](#settings)
  - [mcp_server](#mcp_server)
  - [file](#file-resource)
  - [directory](#directory)
- [Platform Override Blocks](#platform-override-blocks)
- [Platform Support Matrix](#platform-support-matrix)
- [Package Configuration](#package-configuration)
- [Project Configuration](#project-configuration)

---

## HCL Functions

These functions are available in `package.hcl` files for reading files and environment variables.

### file()

Reads the contents of a file relative to the package directory.

```hcl
content = file("skills/my-skill.md")
```

### templatefile()

Reads a template file and renders it with the provided variables using Go's `text/template` syntax.

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
  DEBUG   = env("DEBUG", "false")  # "false" is used if DEBUG is not set
}
```

---

## Template Variables

These variables are available when rendering templates (via `templatefile()` or `template_file` blocks):

| Variable | Description |
|----------|-------------|
| `{{ .ComponentDir }}` | Absolute path to the installed component directory |
| `{{ .PackageName }}` | Name of the package being installed |
| `{{ .PackageVersion }}` | Version of the package |
| `{{ .ProjectRoot }}` | Absolute path to the project root |
| `{{ .Platform }}` | Target platform (e.g., `claude-code`) |

---

## File Blocks

Resources that support additional files use `file` and `template_file` blocks. Files are installed relative to the component's install directory.

### file

Copies a static file alongside the resource.

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `src` | string | yes | Source path relative to package root |
| `dest` | string | no | Destination filename (defaults to basename of src) |
| `chmod` | string | no | File permissions (e.g., `"755"`, `"600"`) |

```hcl
file {
  src   = "scripts/run-tests.sh"
  dest  = "run-tests.sh"
  chmod = "755"
}
```

### template_file

Renders a template file and copies it alongside the resource.

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `src` | string | yes | Source template path relative to package root |
| `dest` | string | no | Destination filename (defaults to basename without `.tmpl`) |
| `chmod` | string | no | File permissions |
| `vars` | map | no | Additional variables for this template |

```hcl
template_file {
  src  = "config/setup.sh.tmpl"
  dest = "setup.sh"
  chmod = "755"
  vars = {
    project_name = "MyProject"
  }
}
```

---

## Universal Resource Types

All resources are declared using universal block types. Dex translates them to the correct platform-specific format at install time. If a resource type is not supported by the target platform, it is logged and skipped.

### skill

Skills provide specialized knowledge or capabilities to the AI assistant. Installed as standalone files in the platform's skills directory.

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Skill identifier (block label) |
| `description` | string | yes | When and how to use this skill |
| `content` | string | yes | Main body/instructions |
| `file` | block | no | Static files to copy alongside |
| `template_file` | block | no | Template files to render and copy |
| `platforms` | list(string) | no | Limit to specific platforms (empty = all) |
| `claude` | block | no | Claude-specific overrides |
| `copilot` | block | no | Copilot-specific overrides |
| `cursor` | block | no | Cursor-specific overrides |

**Supported platforms:** Claude Code, GitHub Copilot, Cursor.

```hcl
skill "python-best-practices" {
  description = "Python coding standards and best practices"
  content     = file("skills/python-best-practices.md")

  claude {
    allowed_tools = ["Bash", "Read"]
    model         = "sonnet"
  }

  cursor {
    license       = "MIT"
    compatibility = "requires python 3.11"
  }
}
```

### command

Commands can be invoked by users (e.g., `/command-name`). Translates to Claude commands, Copilot prompts, or Cursor commands.

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Command identifier (block label) |
| `description` | string | yes | What this command does |
| `content` | string | yes | Command body/instructions |
| `file` | block | no | Static files to copy alongside |
| `template_file` | block | no | Template files to render and copy |
| `platforms` | list(string) | no | Limit to specific platforms |
| `claude` | block | no | Claude-specific overrides |
| `copilot` | block | no | Copilot-specific overrides |
| `cursor` | block | no | Cursor-specific overrides |

**Supported platforms:** Claude Code, GitHub Copilot, Cursor.

```hcl
command "test" {
  description = "Run project tests"
  content     = file("commands/test.md")

  claude {
    allowed_tools = ["Bash"]
  }

  copilot {
    agent = "edit"
    tools = ["terminal"]
  }
}
```

### agent

Agents provide specialized agent behaviors. Translates to Claude subagents or Copilot agents.

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Agent identifier (block label) |
| `description` | string | yes | What this agent does |
| `content` | string | yes | Agent instructions |
| `file` | block | no | Static files to copy alongside |
| `template_file` | block | no | Template files to render and copy |
| `platforms` | list(string) | no | Limit to specific platforms |
| `claude` | block | no | Claude-specific overrides |
| `copilot` | block | no | Copilot-specific overrides |
| `cursor` | block | no | Cursor-specific overrides (disabled = true only) |

**Supported platforms:** Claude Code, GitHub Copilot. Cursor does not support agents (skipped with warning).

```hcl
agent "code-reviewer" {
  description = "Reviews code for quality issues"
  content     = file("agents/code-reviewer.md")

  claude {
    model = "opus"
    color = "blue"
    tools = ["Read", "Grep", "Glob"]
  }

  copilot {
    model = "gpt-4"
    tools = ["terminal"]
  }
}
```

### rule

Rules are merged into the platform's main agent file (CLAUDE.md, copilot-instructions.md, AGENTS.md). Multiple packages can contribute rules that are combined together.

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Rule identifier (block label) |
| `description` | string | yes | What this rule provides |
| `content` | string | yes | Rule text |
| `file` | block | no | Static files to copy alongside |
| `template_file` | block | no | Template files to render and copy |
| `platforms` | list(string) | no | Limit to specific platforms |
| `claude` | block | no | Claude-specific overrides |
| `copilot` | block | no | Copilot-specific overrides |
| `cursor` | block | no | Cursor-specific overrides |

**Supported platforms:** Claude Code, GitHub Copilot, Cursor.

```hcl
rule "linting" {
  description = "Linting standards"
  content     = "Always run the linter before committing code."
}
```

### rules

Standalone rule files owned by a single package. Installed as individual files in the platform's rules directory.

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Rules identifier (block label) |
| `description` | string | yes | What these rules provide |
| `content` | string | conditional | Rules text (required if no file blocks) |
| `file` | block | no | Static files to copy alongside |
| `template_file` | block | no | Template files to render and copy |
| `platforms` | list(string) | no | Limit to specific platforms |
| `claude` | block | no | Claude-specific overrides |
| `copilot` | block | no | Copilot-specific overrides |
| `cursor` | block | no | Cursor-specific overrides |

**Supported platforms:** Claude Code, GitHub Copilot, Cursor.

```hcl
rules "code-review-standards" {
  description = "Code review standards and practices"
  content     = file("rules/code-review.md")

  cursor {
    globs       = ["*.go", "*.ts"]
    always_apply = true
  }
}
```

### settings

Platform settings (permissions, environment variables, model preferences). All platform-specific fields go inside the corresponding platform override block.

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Settings identifier (block label) |
| `platforms` | list(string) | no | Limit to specific platforms |
| `claude` | block | no | Claude Code settings (see Claude Code Overrides) |
| `copilot` | block | no | Copilot settings (disabled only, for now) |
| `cursor` | block | no | Cursor settings (disabled only, for now) |

**Claude settings fields** (inside `claude {}` block):
`allow`, `ask`, `deny`, `env`, `enable_all_project_mcp_servers`, `enabled_mcp_servers`, `disabled_mcp_servers`, `respect_gitignore`, `include_co_authored_by`, `model`, `output_style`, `always_thinking_enabled`, `plans_directory`

**Supported platforms:** Claude Code. Other platforms skip with warning.

```hcl
settings "python" {
  claude {
    allow = [
      "Bash(python:*)",
      "Bash(pip:*)",
      "Bash(pytest:*)",
    ]

    env = {
      PYTHONDONTWRITEBYTECODE = "1"
    }

    enable_all_project_mcp_servers = true
  }
}
```

### mcp_server

MCP (Model Context Protocol) servers provide additional tools and capabilities. Supported across all platforms with platform-specific translation.

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Server identifier (block label) |
| `description` | string | no | What this server provides |
| `command` | string | conditional | Executable to run (mutually exclusive with `url`) |
| `args` | list(string) | no | Command-line arguments |
| `env` | map(string) | no | Environment variables |
| `url` | string | conditional | HTTP/SSE endpoint (mutually exclusive with `command`) |
| `env_file` | string | no | Path to env file |
| `headers` | map(string) | no | HTTP headers (URL-based servers only) |
| `input` | block | no | VS Code input prompts for dynamic config |
| `platforms` | list(string) | no | Limit to specific platforms |
| `claude` | block | no | Claude-specific overrides |
| `copilot` | block | no | Copilot-specific overrides |
| `cursor` | block | no | Cursor-specific overrides |

**Supported platforms:** Claude Code, GitHub Copilot, Cursor.

```hcl
mcp_server "database" {
  command = "npx"
  args    = ["-y", "@database/mcp-server"]

  env = {
    DB_HOST = env("DB_HOST", "localhost")
  }
}

mcp_server "api-gateway" {
  url = "https://api.example.com/mcp"

  headers = {
    Authorization = "Bearer ${env("API_TOKEN")}"
  }

  claude {
    disabled = true  # Not needed for Claude
  }
}
```

### file (resource)

Universal file resource — copies a file to the project. Works across all platforms.

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | File identifier (block label) |
| `path` | string | yes | Destination path relative to project root |
| `content` | string | conditional | Inline file content |
| `src` | string | conditional | Source file path relative to package root |
| `chmod` | string | no | File permissions |

### directory

Universal directory resource — creates a directory in the project. Works across all platforms.

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Directory identifier (block label) |
| `path` | string | yes | Directory path relative to project root |

---

## Platform Override Blocks

Every universal resource supports optional platform override blocks (`claude {}`, `copilot {}`, `cursor {}`) for platform-specific customization.

### Common Override Fields

All platform overrides support:

| Field | Type | Description |
|-------|------|-------------|
| `disabled` | bool | Set to `true` to skip this resource on this platform |
| `content` | string | Override the base content for this platform |

### Claude Code Overrides

Claude overrides vary by resource type. Field names are HCL attributes; the emitted YAML/JSON key is in parentheses when it differs.

**skill:**
`argument_hint` (`argument-hint`), `arguments`, `when_to_use`, `disable_model_invocation` (`disable-model-invocation`), `user_invocable` (`user-invocable`), `allowed_tools` (`allowed-tools`), `model`, `effort`, `context` (`fork` runs the skill in an isolated subagent), `agent` (subagent type when `context = "fork"`), `paths` (glob patterns for auto-activation), `shell` (`bash` or `powershell`), `metadata`, `hooks`

**command:**
Claude Code merged custom commands into the skills system, so commands accept **the same fields as `skill`** above. `CommandClaudeOverride` is a Go type alias for `SkillClaudeOverride` — the two always stay in sync.

**agent:**
`model`, `color`, `tools`, `disallowed_tools` (`disallowedTools`), `permission_mode` (`permissionMode`), `max_turns` (`maxTurns`), `skills` (names of skills to preload), `mcp_servers` (`mcpServers`), `memory` (`user`/`project`/`local`), `background`, `effort`, `isolation` (`worktree` to run in an isolated git worktree), `initial_prompt` (`initialPrompt`), `hooks`

**rule / rules:**
`paths`

**settings:**
`allow`, `ask`, `deny`, `env`, `enable_all_project_mcp_servers`, `enabled_mcp_servers`, `disabled_mcp_servers`, `respect_gitignore`, `include_co_authored_by`, `model`, `output_style`, `always_thinking_enabled`, `plans_directory`, `additional_directories` (`additionalDirectories` — extra file-access directories), `auto_memory_directory` (`autoMemoryDirectory`), `include_git_instructions` (`includeGitInstructions` — default `true`), `agent` (default subagent for sessions)

### GitHub Copilot Overrides

**command:**
`argument_hint`, `agent`, `model`, `tools`

**agent:**
`model`, `tools`, `handoffs`, `infer`, `target`

**rules:**
`apply_to`

### Cursor Overrides

**skill:**
`license`, `compatibility`, `disable_model_invocation`, `metadata`

Cursor skills deliberately have a smaller frontmatter than Claude's — no `allowed_tools`, `model`, `argument_hint`, or `context` fields are supported by Cursor.

**command:**
Cursor commands are plain markdown with no documented frontmatter. Only the generic `disabled` / `content` override fields apply.

**rules:**
`globs`, `always_apply` (emitted as `alwaysApply` in the `.mdc` frontmatter)

**mcp_server:**
`auth` block (for HTTP/SSE servers): `client_id` (required), `client_secret`, `scopes`. Emitted as `CLIENT_ID` / `CLIENT_SECRET` / `scopes` per Cursor's documented casing.

```hcl
mcp_server "oauth-api" {
  url = "https://api.example.com/mcp"

  cursor {
    auth {
      client_id     = "my-client-id"
      client_secret = env("CLIENT_SECRET")
      scopes        = ["read", "write"]
    }
  }
}
```

---

## Platform Support Matrix

| Resource Type | Claude Code | GitHub Copilot | Cursor |
|---------------|:-----------:|:--------------:|:------:|
| `skill` | Skill | Skill | Skill |
| `command` | Command | Prompt | Command |
| `agent` | Subagent | Agent | -- |
| `rule` | Rule (CLAUDE.md) | Instruction (copilot-instructions.md) | Rule (AGENTS.md) |
| `rules` | Rules (.claude/rules/) | Instructions (.github/instructions/) | Rules (.cursor/rules/) |
| `settings` | Settings (.claude/settings.json) | -- | -- |
| `mcp_server` | MCP Server (.mcp.json) | MCP Server (.vscode/mcp.json) | MCP Server (.cursor/mcp.json) |
| `file` | File | File | File |
| `directory` | Directory | Directory | Directory |

`--` = Not supported. Resource is logged as skipped and produces no output.

---

## Package Configuration

A `package.hcl` file defines a package's metadata and resources.

### Meta Block

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Package name |
| `version` | string | yes | Package version (semver recommended) |
| `description` | string | no | Package description |
| `author` | string | no | Package author |
| `license` | string | no | License identifier (e.g., `MIT`) |
| `repository` | string | no | Source repository URL |
| `platforms` | list(string) | no | Supported platforms (empty = all) |

### Variable Block

Variables allow users to customize package behavior at installation time.

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Variable identifier (block label) |
| `description` | string | no | Variable description |
| `default` | string | no | Default value |
| `required` | bool | no | Whether user must provide a value |
| `env` | string | no | Environment variable to read from |

### Complete Example

```hcl
meta {
  name        = "python-tools"
  version     = "1.0.0"
  description = "Python development tools"
  author      = "example"
  license     = "MIT"
}

variable "python_version" {
  description = "Python version to use"
  default     = "3.11"
}

skill "python-best-practices" {
  description = "Python coding standards and best practices"
  content     = file("skills/python-best-practices.md")
}

command "test" {
  description = "Run Python tests"
  content     = file("commands/test.md")
}

settings "python" {
  claude {
    allow = [
      "Bash(python:*)",
      "Bash(pip:*)",
      "Bash(pytest:*)",
    ]

    env = {
      PYTHONDONTWRITEBYTECODE = "1"
    }
  }
}

mcp_server "python-lsp" {
  command = "pip"
  args    = ["run", "python-lsp-server"]
}
```

---

## Project Configuration

A `dex.hcl` file configures a dex-managed project.

### Project Block

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | no | Project name (defaults to directory name) |
| `default_platform` | string | yes | Target AI platform (e.g., `claude-code`) |
| `agent_instructions` | string | no | Project-level instructions added to agent file |
| `git_exclude` | bool | no | Auto-update .git/info/exclude |

### Registry Block

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Registry identifier (block label) |
| `path` | string | conditional | Local filesystem path (for `file://` registries) |
| `url` | string | conditional | Remote URL (for `https://` registries) |

### Package Block

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Package identifier (block label) |
| `source` | string | conditional | Direct source URL (`git+https://`, `file://`) |
| `version` | string | no | Version constraint |
| `registry` | string | conditional | Registry name to fetch from |
| `config` | map(string) | no | Package-specific configuration values |

### Profile Block

Profiles define named configuration variants activated with `dex sync --profile <name>`.

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Profile identifier (block label) |
| `exclude_defaults` | bool | no | Start clean (no defaults inherited) |
| `agent_instructions` | string | no | Override project agent instructions |

Profiles support all resource types (`skill`, `command`, `rule`, `rules`, `agent`, `settings`, `mcp_server`), `registry`, and `package` blocks inside them. By default, profile contents are merged additively with defaults (same-name items replaced). With `exclude_defaults = true`, only profile-defined items are used.

### Complete Example

```hcl
project {
  name             = "my-webapp"
  default_platform = "claude-code"
  agent_instructions = "This is a web application built with React and Go."
}

registry "internal" {
  url = "https://packages.example.com"
}

package "web-tools" {
  registry = "internal"
  version  = "^1.0.0"
}

rule "project-standards" {
  description = "Project coding standards"
  content     = "Follow the project style guide."
}

settings "permissions" {
  claude {
    enable_all_project_mcp_servers = true
  }
}

profile "qa" {
  agent_instructions = "QA environment - focus on testing"

  package "qa-tools" {
    registry = "internal"
  }
}
```
