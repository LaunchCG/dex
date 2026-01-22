# Dex Resources

Dex uses HCL (HashiCorp Configuration Language) to define resources that are installed into AI agent projects. This document covers all available resource types, their options, and examples.

## Table of Contents

- [HCL Functions](#hcl-functions)
- [Template Variables](#template-variables)
- [File Blocks](#file-blocks)
- [Claude Code Resources](#resources)
  - [claude_skill](#claude_skill)
  - [claude_command](#claude_command)
  - [claude_subagent](#claude_subagent)
  - [claude_rule](#claude_rule)
  - [claude_rules](#claude_rules)
  - [claude_settings](#claude_settings)
  - [claude_mcp_server](#claude_mcp_server)
- [GitHub Copilot Resources](#github-copilot-resources)
  - [copilot_instruction](#copilot_instruction)
  - [copilot_mcp_server](#copilot_mcp_server)
  - [copilot_instructions](#copilot_instructions)
  - [copilot_prompt](#copilot_prompt)
  - [copilot_agent](#copilot_agent)
  - [copilot_skill](#copilot_skill)
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
| `{{ .PluginName }}` | Name of the plugin being installed |
| `{{ .PluginVersion }}` | Version of the plugin |
| `{{ .ProjectRoot }}` | Absolute path to the project root |
| `{{ .Platform }}` | Target platform (e.g., `claude-code`) |

### Example Template

```markdown
# Setup

Install dependencies:

```bash
cd {{ .ComponentDir }}
pip install -r requirements.txt
```

This skill is part of {{ .PluginName }} v{{ .PluginVersion }}.
```

---

## File Blocks

Resources that support additional files use `file` and `template_file` blocks. Files are installed relative to the component's install directory.

### file

Copies a static file alongside the resource.

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `src` | string | yes | Source path relative to plugin root |
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
| `src` | string | yes | Source template path relative to plugin root |
| `dest` | string | no | Destination filename (defaults to basename without `.tmpl`) |
| `chmod` | string | no | File permissions |
| `vars` | map | no | Additional variables for this template |

```hcl
template_file {
  src   = "scripts/config.py.tmpl"
  dest  = "config.py"
  vars = {
    api_endpoint = "https://api.example.com"
  }
}
```

---

## Resources

### claude_skill

Skills provide specialized knowledge or capabilities to Claude. Each skill is installed to `.claude/skills/{plugin}-{name}/SKILL.md`.

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Block label identifying this skill |
| `description` | string | yes | Explains when and how to use this skill |
| `content` | string | yes | The skill's instructions/knowledge |
| `argument_hint` | string | no | Hint shown during autocomplete (e.g., `"[filename]"`) |
| `disable_model_invocation` | bool | no | Prevent Claude from auto-loading; user must invoke manually |
| `user_invocable` | bool | no | Set to `false` to hide from `/` menu (default: true) |
| `allowed_tools` | list(string) | no | Tools Claude can use without asking (e.g., `["Read", "Grep"]`) |
| `model` | string | no | Model to use: `sonnet`, `haiku`, or `opus` |
| `context` | string | no | Set to `"fork"` to run in isolated subagent |
| `agent` | string | no | Subagent type when `context = "fork"` (e.g., `"Explore"`, `"Plan"`) |
| `metadata` | map(string) | no | Additional frontmatter fields |

#### Nested Blocks

- `file` - Static files to copy alongside the skill
- `template_file` - Template files to render and copy

#### Examples

**Simple skill with inline content:**

```hcl
claude_skill "code-review" {
  name        = "code-review"
  description = "Performs thorough code reviews focusing on correctness and maintainability"

  content = <<-EOT
    When reviewing code:
    1. Check for bugs and edge cases
    2. Evaluate code style and readability
    3. Look for security vulnerabilities
    4. Suggest improvements with examples
  EOT
}
```

**Skill with content from file:**

```hcl
claude_skill "testing" {
  name        = "testing"
  description = "Helps write comprehensive tests using pytest"
  content     = file("skills/testing.md")
}
```

**Skill with helper files:**

```hcl
claude_skill "data-validation" {
  name        = "data-validation"
  description = "Validates JSON data against schemas"
  content     = file("skills/data-validation.md")

  file {
    src = "schemas/user.schema.json"
  }

  file {
    src   = "scripts/validate.py"
    chmod = "755"
  }

  file {
    src = "scripts/requirements.txt"
  }
}
```

**Skill with dynamic content:**

```hcl
claude_skill "deployment" {
  name        = "deployment"
  description = "Guides deployment to configured environment"
  content     = templatefile("skills/deployment.md.tmpl", {
    environment = "production"
    region      = "us-east-1"
  })
}
```

**Skill with argument hint and tool restrictions:**

```hcl
claude_skill "file-analyzer" {
  name          = "file-analyzer"
  description   = "Analyzes a specific file for issues and improvements"
  argument_hint = "[filename]"

  content = <<-EOT
    Analyze the specified file for:
    1. Code quality issues
    2. Performance bottlenecks
    3. Security vulnerabilities
    4. Suggestions for improvement
  EOT

  allowed_tools = ["Read", "Grep", "Glob"]
  model         = "sonnet"
}
```

**Skill hidden from menu (internal use only):**

```hcl
claude_skill "internal-helper" {
  name           = "internal-helper"
  description    = "Internal skill used by other skills"
  user_invocable = false

  content = <<-EOT
    This skill provides helper functionality for other skills.
    It is not intended to be invoked directly by users.
  EOT
}
```

**Skill that runs in isolated subagent:**

```hcl
claude_skill "codebase-explorer" {
  name        = "codebase-explorer"
  description = "Explores the codebase to answer architectural questions"
  context     = "fork"
  agent       = "Explore"

  content = <<-EOT
    Explore the codebase to understand:
    1. Directory structure and organization
    2. Key components and their relationships
    3. Data flow and dependencies
  EOT

  allowed_tools = ["Read", "Glob", "Grep"]
}
```

**Skill that prevents auto-loading:**

```hcl
claude_skill "dangerous-operations" {
  name                     = "dangerous-operations"
  description              = "Performs potentially destructive operations - use with caution"
  disable_model_invocation = true

  content = <<-EOT
    WARNING: This skill can perform destructive operations.
    Only invoke this skill when explicitly requested by the user.

    Available operations:
    - Mass file deletion
    - Database truncation
    - Cache clearing
  EOT
}
```

---

### claude_command

Commands are user-invokable actions accessible via `/{name}` syntax. Each command is installed to `.claude/commands/{plugin}-{name}.md`.

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Command name (invoked as `/{name}`) |
| `description` | string | yes | Brief description shown to user |
| `content` | string | yes | Command instructions |
| `argument_hint` | string | no | Hint for arguments (e.g., `"[environment]"`) |
| `allowed_tools` | list(string) | no | Tools this command can use |
| `model` | string | no | Model to use: `sonnet`, `haiku`, or `opus` |

#### Nested Blocks

- `file` - Static files to copy alongside the command
- `template_file` - Template files to render and copy

#### Examples

**Simple command:**

```hcl
claude_command "test" {
  name        = "test"
  description = "Run project tests"

  content = <<-EOT
    Run the project's test suite:
    1. Identify the test framework (pytest, jest, go test, etc.)
    2. Run all tests
    3. Report results and any failures
  EOT
}
```

**Command with arguments and tool restrictions:**

```hcl
claude_command "deploy" {
  name          = "deploy"
  description   = "Deploy the application to a specified environment"
  argument_hint = "[environment]"

  content = <<-EOT
    Deploy the application to the specified environment:
    1. Run the test suite first
    2. Build the application
    3. Push to container registry
    4. Update the Kubernetes deployment
  EOT

  allowed_tools = ["Bash(docker:*)", "Bash(kubectl:*)"]
  model         = "sonnet"
}
```

**Command with helper files:**

```hcl
claude_command "migrate" {
  name        = "migrate"
  description = "Run database migrations"
  content     = file("commands/migrate.md")

  file {
    src   = "scripts/migrate.sh"
    chmod = "755"
  }
}
```

---

### claude_subagent

Subagents are specialized agents that can be spawned by Claude for specific tasks. Each subagent is installed to `.claude/agents/{plugin}-{name}.md`.

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Agent identifier |
| `description` | string | yes | When to use this agent |
| `content` | string | yes | Agent instructions |
| `model` | string | no | Model: `inherit`, `sonnet`, `haiku`, or `opus` |
| `color` | string | no | Display color: `blue`, `green`, `yellow`, `red`, `purple` |
| `tools` | list(string) | no | Allowed tools for this agent |

#### Nested Blocks

- `file` - Static files to copy alongside the subagent
- `template_file` - Template files to render and copy

#### Examples

**Test runner agent:**

```hcl
claude_subagent "test-runner" {
  name        = "test-runner"
  description = "Runs tests and reports results. Use when you need to verify code changes."

  content = <<-EOT
    You are a test runner agent. Your job is to:
    1. Identify relevant test files for the changes
    2. Run tests using the appropriate framework
    3. Report results clearly, including any failures
    4. Suggest fixes for failing tests
  EOT

  model = "haiku"
  color = "green"
  tools = ["Bash", "Read", "Glob", "Grep"]
}
```

**Code reviewer agent:**

```hcl
claude_subagent "code-reviewer" {
  name        = "code-reviewer"
  description = "Reviews code for quality, security, and best practices"
  content     = file("agents/code-reviewer.md")

  model = "sonnet"
  color = "blue"
  tools = ["Read", "Glob", "Grep"]
}
```

---

### claude_rule

Rules are merged into the project's `CLAUDE.md` file. Multiple plugins can contribute rules which are combined together.

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Rule identifier |
| `description` | string | yes | Rule description |
| `content` | string | yes | Rule content |
| `paths` | list(string) | no | File patterns to scope when this rule applies |

#### Nested Blocks

- `file` - Static files to copy alongside the rule
- `template_file` - Template files to render and copy

#### Examples

**Global rule:**

```hcl
claude_rule "no-console-log" {
  name        = "no-console-log"
  description = "Avoid console.log in production code"

  content = <<-EOT
    Do not use console.log() in production code.
    Use the project's logging framework instead.
  EOT
}
```

**Path-scoped rule:**

```hcl
claude_rule "typescript-strict" {
  name        = "typescript-strict"
  description = "TypeScript strict mode requirements"

  content = <<-EOT
    When writing TypeScript:
    - Always use explicit types, avoid `any`
    - Use strict null checks
    - Prefer interfaces over type aliases for object shapes
  EOT

  paths = ["src/**/*.ts", "src/**/*.tsx"]
}
```

---

### claude_rules

A standalone rules file owned by a single plugin. Installed to `.claude/rules/{plugin}-{name}.md`.

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Rules file identifier |
| `description` | string | yes | Rules description |
| `content` | string | yes | Rules content |
| `paths` | list(string) | no | File patterns to scope when these rules apply |

#### Nested Blocks

- `file` - Static files to copy alongside the rules
- `template_file` - Template files to render and copy

#### Examples

```hcl
claude_rules "security" {
  name        = "security"
  description = "Security best practices for web applications"
  content     = file("rules/security.md")
  paths       = ["**/*.ts", "**/*.js"]
}
```

---

### claude_settings

Settings are merged into `.claude/settings.json`. Multiple plugins can contribute permissions and environment variables. Project-level settings override plugin settings.

#### Attributes

**Permissions (mergeable):**

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `allow` | list(string) | no | Tool patterns to automatically allow |
| `ask` | list(string) | no | Tool patterns requiring confirmation |
| `deny` | list(string) | no | Tool patterns to block |

**Environment:**

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `env` | map(string) | no | Environment variables to set |

**Global settings (project-level only):**

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `enable_all_project_mcp_servers` | bool | no | Auto-approve project MCP servers |
| `enabled_mcp_servers` | list(string) | no | Specific approved MCP servers |
| `disabled_mcp_servers` | list(string) | no | Rejected MCP servers |
| `respect_gitignore` | bool | no | Filter suggestions by gitignore |
| `include_co_authored_by` | bool | no | Include co-author in commits |
| `model` | string | no | Override default model |
| `output_style` | string | no | Response style preference |
| `always_thinking_enabled` | bool | no | Enable extended thinking |
| `plans_directory` | string | no | Custom plan files location |

#### Examples

**Plugin contributing permissions:**

```hcl
claude_settings "node-tools" {
  name = "node-tools"

  allow = [
    "Bash(npm:*)",
    "Bash(npx:*)",
    "Bash(node:*)",
  ]

  env = {
    NODE_ENV = "development"
  }
}
```

**Project-level settings with global options:**

```hcl
claude_settings "project" {
  name = "project"

  allow = ["Bash(docker:*)"]
  deny  = ["Bash(rm -rf /)"]

  env = {
    DEBUG = "true"
  }

  enable_all_project_mcp_servers = true
  respect_gitignore              = true
  include_co_authored_by         = true
}
```

---

### claude_mcp_server

MCP servers provide additional tools and capabilities to Claude. Configurations are merged into `.mcp.json`.

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Server identifier |
| `description` | string | no | Server description |
| `type` | string | yes | Server type: `command` or `http` |
| `command` | string | conditional | Command to run (required for `type = "command"` unless `source` is set) |
| `args` | list(string) | no | Command-line arguments |
| `env` | map(string) | no | Environment variables |
| `source` | string | conditional | Shortcut for package managers: `npm:`, `uvx:`, `pip:` |
| `url` | string | conditional | HTTP endpoint (required for `type = "http"`) |

#### Examples

**Command-based MCP server:**

```hcl
claude_mcp_server "filesystem" {
  name    = "filesystem"
  type    = "command"
  command = "npx"
  args    = ["-y", "@anthropic/mcp-filesystem"]

  env = {
    HOME = env("HOME")
  }
}
```

**Using source shortcut:**

```hcl
claude_mcp_server "postgres" {
  name   = "postgres"
  type   = "command"
  source = "uvx:mcp-postgres"

  env = {
    DATABASE_URL = env("DATABASE_URL")
  }
}
```

**HTTP-based MCP server:**

```hcl
claude_mcp_server "remote-api" {
  name        = "remote-api"
  description = "Remote API integration server"
  type        = "http"
  url         = "https://mcp.example.com/api"
}
```

**MCP server from git repository:**

```hcl
claude_mcp_server "serena" {
  name    = "serena"
  type    = "command"
  command = "uvx"
  args    = ["--from", "git+https://github.com/oraios/serena", "serena", "start-mcp-server"]
}
```

---

## GitHub Copilot Resources

The following resources are available for GitHub Copilot projects (`agentic_platform = "github-copilot"`).

### copilot_instruction

Instructions (singular) are merged into the project's `.github/copilot-instructions.md` file. Multiple plugins can contribute instructions which are combined together using marker comments.

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Block label identifying this instruction |
| `description` | string | yes | Explains what this instruction provides |
| `content` | string | yes | The instruction content |

#### Nested Blocks

- `file` - Static files to copy alongside the instruction
- `template_file` - Template files to render and copy

#### Examples

**Global instruction:**

```hcl
copilot_instruction "coding-standards" {
  name        = "coding-standards"
  description = "Project coding standards"

  content = <<-EOT
    Always follow these coding standards:
    - Use TypeScript strict mode
    - Prefer async/await over callbacks
    - Document all public APIs
  EOT
}
```

---

### copilot_mcp_server

MCP servers provide additional tools and capabilities to GitHub Copilot. Configurations are merged into `.vscode/mcp.json`.

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Server identifier |
| `description` | string | no | Server description |
| `type` | string | yes | Server type: `stdio`, `http`, or `sse` |
| `command` | string | conditional | Command to run (required for `type = "stdio"`) |
| `args` | list(string) | no | Command-line arguments |
| `env` | map(string) | no | Environment variables |
| `env_file` | string | no | Path to an env file to load |
| `url` | string | conditional | HTTP/SSE endpoint (required for `type = "http"` or `type = "sse"`) |
| `headers` | map(string) | no | HTTP headers for http/sse servers |

#### Examples

**Stdio-based MCP server:**

```hcl
copilot_mcp_server "filesystem" {
  name    = "filesystem"
  type    = "stdio"
  command = "npx"
  args    = ["-y", "@anthropic/mcp-filesystem"]

  env = {
    HOME = env("HOME")
  }
}
```

**HTTP-based MCP server:**

```hcl
copilot_mcp_server "context7" {
  name        = "context7"
  description = "Context7 documentation server"
  type        = "http"
  url         = "https://mcp.context7.com/mcp"
}
```

---

### copilot_instructions

Standalone instruction files owned by a single plugin. Installed to `.github/instructions/{plugin}-{name}.instructions.md`.

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Instructions file identifier |
| `description` | string | yes | Instructions description |
| `content` | string | yes | Instructions content |
| `apply_to` | string | no | Glob pattern for selective application |

#### Nested Blocks

- `file` - Static files to copy alongside the instructions
- `template_file` - Template files to render and copy

#### Examples

```hcl
copilot_instructions "typescript" {
  name        = "typescript"
  description = "TypeScript best practices"
  apply_to    = "**/*.ts"

  content = <<-EOT
    When writing TypeScript:
    - Always use explicit types, avoid `any`
    - Use strict null checks
    - Prefer interfaces over type aliases for object shapes
  EOT
}
```

---

### copilot_prompt

Prompts are user-invokable actions in GitHub Copilot. Each prompt is installed to `.github/prompts/{plugin}-{name}.prompt.md`.

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Prompt name |
| `description` | string | yes | Brief description |
| `content` | string | yes | Prompt instructions |
| `argument_hint` | string | no | Hint for arguments |
| `agent` | string | no | Agent mode: `ask`, `edit`, `agent`, or custom |
| `model` | string | no | Model selection |
| `tools` | list(string) | no | Tools to enable |

#### Nested Blocks

- `file` - Static files to copy alongside the prompt
- `template_file` - Template files to render and copy

#### Examples

**Simple prompt:**

```hcl
copilot_prompt "review" {
  name        = "review"
  description = "Review code for issues"
  agent       = "ask"

  content = <<-EOT
    Review this code for:
    1. Bugs and edge cases
    2. Security vulnerabilities
    3. Performance issues
    4. Code style improvements
  EOT
}
```

**Prompt with tools:**

```hcl
copilot_prompt "refactor" {
  name        = "refactor"
  description = "Refactor code to improve quality"
  agent       = "edit"
  model       = "gpt-4o"
  tools       = ["fetch", "search"]

  content = <<-EOT
    Refactor the selected code to:
    1. Improve readability
    2. Reduce complexity
    3. Follow best practices
  EOT
}
```

---

### copilot_agent

Agents are specialized agents that can be used by GitHub Copilot for specific tasks. Each agent is installed to `.github/agents/{plugin}-{name}.agent.md`.

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Agent identifier |
| `description` | string | yes | When to use this agent |
| `content` | string | yes | Agent instructions |
| `model` | string | no | Model selection |
| `tools` | list(string) | no | Available tools for this agent |
| `handoffs` | list(string) | no | Sequential workflow transitions to other agents |
| `infer` | bool | no | Enable subagent usage (default: true) |
| `target` | string | no | Target environment: `vscode` or `github-copilot` |

#### Nested Blocks

- `file` - Static files to copy alongside the agent
- `template_file` - Template files to render and copy

#### Examples

**Test runner agent:**

```hcl
copilot_agent "test-runner" {
  name        = "test-runner"
  description = "Runs tests and reports results"

  content = <<-EOT
    You are a test runner agent. Your job is to:
    1. Identify relevant test files for the changes
    2. Run tests using the appropriate framework
    3. Report results clearly, including any failures
    4. Suggest fixes for failing tests
  EOT

  tools = ["fetch", "search"]
  target = "vscode"
}
```

**Agent with handoffs:**

```hcl
copilot_agent "planner" {
  name        = "planner"
  description = "Creates implementation plans"

  content = <<-EOT
    Create a detailed implementation plan for the requested feature.
    Break down the work into small, manageable tasks.
  EOT

  tools    = ["fetch", "search"]
  handoffs = ["implementer", "reviewer"]
}
```

---

### copilot_skill

Skills provide specialized knowledge or capabilities to GitHub Copilot. Each skill is installed to `.github/skills/{plugin}-{name}/SKILL.md`.

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Skill name (max 64 characters) |
| `description` | string | yes | When and how to use this skill (max 1024 characters) |
| `content` | string | yes | The skill's instructions/knowledge |

#### Nested Blocks

- `file` - Static files to copy alongside the skill
- `template_file` - Template files to render and copy

#### Examples

**Simple skill:**

```hcl
copilot_skill "testing" {
  name        = "testing"
  description = "Best practices for writing comprehensive tests"

  content = <<-EOT
    When writing tests:
    1. Test both happy paths and edge cases
    2. Use descriptive test names
    3. Follow the Arrange-Act-Assert pattern
    4. Mock external dependencies
  EOT
}
```

**Skill with helper files:**

```hcl
copilot_skill "data-validation" {
  name        = "data-validation"
  description = "Validates JSON data against schemas"
  content     = file("skills/data-validation.md")

  file {
    src = "schemas/user.schema.json"
  }

  file {
    src   = "scripts/validate.py"
    chmod = "755"
  }
}
```

---

## Package Configuration

A `package.hcl` file defines a plugin's metadata and resources.

### Package Block

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

Variables allow users to customize plugin behavior at installation time.

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Variable identifier (block label) |
| `description` | string | no | Variable description |
| `default` | string | no | Default value |
| `required` | bool | no | Whether user must provide a value |
| `env` | string | no | Environment variable to read from |

### Complete Example

```hcl
package {
  name        = "python-tools"
  version     = "1.0.0"
  description = "Python development tools for Claude Code"
  author      = "example"
  license     = "MIT"
  platforms   = ["claude-code"]
}

variable "python_version" {
  description = "Python version to use"
  default     = "3.11"
}

variable "test_framework" {
  description = "Test framework (pytest or unittest)"
  default     = "pytest"
}

claude_skill "python-best-practices" {
  name        = "python-best-practices"
  description = "Python coding standards and best practices"
  content     = file("skills/python-best-practices.md")
}

claude_command "test" {
  name        = "test"
  description = "Run Python tests"
  content     = file("commands/test.md")
}

claude_settings "python" {
  name = "python"

  allow = [
    "Bash(python:*)",
    "Bash(pip:*)",
    "Bash(pytest:*)",
  ]

  env = {
    PYTHONDONTWRITEBYTECODE = "1"
  }
}

claude_mcp_server "python-lsp" {
  name    = "python-lsp"
  type    = "command"
  source  = "pip:python-lsp-server"
}
```

---

## Project Configuration

A `dex.hcl` file configures a dex-managed project.

### Project Block

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | no | Project name (defaults to directory name) |
| `agentic_platform` | string | yes | Target AI platform (e.g., `claude-code`) |

### Registry Block

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Registry identifier (block label) |
| `path` | string | conditional | Local filesystem path (for `file://` registries) |
| `url` | string | conditional | Remote URL (for `https://` registries) |

### Plugin Block

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Plugin identifier (block label) |
| `source` | string | conditional | Direct source URL (`git+https://`, `file://`) |
| `version` | string | no | Version constraint |
| `registry` | string | conditional | Registry name to fetch from |
| `config` | map(string) | no | Plugin-specific configuration values |

### Complete Example

```hcl
project {
  name             = "my-webapp"
  agentic_platform = "claude-code"
}

registry "internal" {
  path = "/path/to/internal-plugins"
}

registry "community" {
  url = "https://plugins.example.com"
}

plugin "python-tools" {
  registry = "community"
  version  = "^1.0.0"

  config = {
    python_version = "3.12"
    test_framework = "pytest"
  }
}

plugin "internal-tools" {
  registry = "internal"
  version  = "latest"
}

plugin "custom-plugin" {
  source  = "git+https://github.com/user/custom-plugin.git"
  version = "v2.0.0"
}
```
