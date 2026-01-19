# Plugin Development

Plugins are the core of Dex. They contain skills, commands, sub-agents, and MCP server configurations that extend AI coding assistants.

## Platform Mapping

Each component type maps to different locations and formats depending on the target platform:

### Skills

| Platform | Location | Format |
|----------|----------|--------|
| Claude Code | `.claude/skills/{plugin}-{skill}/SKILL.md` | YAML frontmatter (name, description) |
| Cursor | ❌ Not supported | - |
| GitHub Copilot | `.github/skills/{skill}/SKILL.md` | YAML frontmatter (name, description) |
| Codex | `.codex/skills/{skill}/SKILL.md` | YAML frontmatter (name, description, allowed-tools, license) |
| Antigravity | `.agent/skills/{skill}/SKILL.md` | YAML frontmatter (name, description) |

### Commands

| Platform | Location | Format |
|----------|----------|--------|
| Claude Code | `.claude/commands/{plugin}-{command}.md` | YAML frontmatter (argument_hint, allowed_tools, model) |
| Cursor | ❌ Not supported | - |
| GitHub Copilot | `.github/instructions/{command}.instructions.md` | YAML frontmatter (applyTo, excludeAgent) |
| GitHub Copilot | `.github/prompts/{command}.prompt.md` | Plain markdown (set `copilot_mode: "prompt"`) |
| Codex | ❌ Not supported (use AGENTS.md) | - |
| Antigravity | ❌ Not supported | - |

### Sub-agents

| Platform | Location | Format |
|----------|----------|--------|
| Claude Code | `.claude/agents/{plugin}-{agent}.md` | YAML frontmatter (model, color, tools) |
| Cursor | ❌ Not supported | - |
| GitHub Copilot | `.github/agents/{agent}.agent.md` | YAML frontmatter (name, description) |
| Codex | ❌ Not supported | - |
| Antigravity | ❌ Not supported | - |

### Instructions

Instructions are file-scoped guidance that apply to specific file patterns (e.g., all Python files).

| Platform | Location | Format |
|----------|----------|--------|
| Claude Code | ❌ Not supported | - |
| Cursor | ❌ Not supported | - |
| GitHub Copilot | `.github/instructions/{name}.instructions.md` | YAML frontmatter (applyTo, description, excludeAgent) |
| Codex | ❌ Not supported | - |
| Antigravity | ❌ Not supported | - |

### Rules

Rules provide project-wide guidelines and constraints.

| Platform | Location | Format |
|----------|----------|--------|
| Claude Code | `.claude/rules/{plugin}-{rule}.md` | YAML frontmatter (paths) or plain markdown |
| Cursor | `.cursor/rules/{plugin}-{rule}.mdc` | MDC frontmatter (description, globs, alwaysApply) |
| GitHub Copilot | `.github/copilot-instructions.md` | Plain markdown (appended) |
| Codex | `.codex/rules/{rule}.md` | Plain markdown (no frontmatter) |
| Antigravity | `.agent/rules/{rule}.md` | Plain markdown (no frontmatter) |

**Note:** Claude Code supports the `paths` field in rule configuration, which generates YAML frontmatter with file path patterns. If no `paths` are specified, rules are plain markdown.

### Prompts

Prompts are reusable prompt templates.

| Platform | Location | Format |
|----------|----------|--------|
| Claude Code | ❌ Not supported | - |
| Cursor | ❌ Not supported | - |
| GitHub Copilot | `.github/prompts/{name}.prompt.md` | Plain markdown |
| Codex | ❌ Not supported | - |
| Antigravity | ❌ Not supported | - |

### Agent File

Content injected into the main agent instruction file using marker-based management.

| Platform | Location | Format |
|----------|----------|--------|
| Claude Code | `CLAUDE.md` | Markdown with dex markers |
| Cursor | ❌ Not supported | - |
| GitHub Copilot | ❌ Not supported | - |
| Codex | `AGENTS.md` | Markdown with dex markers |
| Antigravity | ❌ Not supported | - |

### MCP Servers

| Platform | Location | Format |
|----------|----------|--------|
| Claude Code | `.mcp.json` | JSON (mcpServers object) |
| Cursor | `.cursor/mcp.json` | JSON (mcpServers object) |
| GitHub Copilot | `.vscode/mcp.json` | JSON (mcpServers object) |
| Codex | `~/.codex/config.toml` | TOML (mcp_servers table, global) |
| Antigravity | ❌ UI-only | Configured via Antigravity UI |

## Plugin Structure

```
my-plugin/
├── package.json          # Plugin manifest
├── context/              # Context files (Markdown with Jinja2)
│   ├── skill.md
│   └── command.md
├── files/                # Associated files
│   └── config.json
└── servers/              # MCP server scripts
    └── server.js
```

## package.json Manifest

```json
{
  "name": "my-plugin",
  "version": "1.0.0",
  "description": "A useful plugin",
  "skills": [
    {
      "name": "my-skill",
      "context": "./context/skill.md",
      "files": ["./files/config.json"]
    }
  ],
  "commands": [
    {
      "name": "my-command",
      "context": "./context/command.md"
    }
  ],
  "sub_agents": [],
  "mcp_servers": [
    {
      "name": "my-server",
      "type": "bundled",
      "path": "./servers/server.js"
    }
  ],
  "dependencies": {
    "other-plugin": "^1.0.0"
  }
}
```

### Manifest Fields

#### `name` (required)

Plugin name. Must:
- Start with a letter or number
- Contain only lowercase letters, numbers, hyphens, and underscores

#### `version` (required)

Semantic version string (e.g., "1.0.0", "2.0.0-beta.1").

#### `description` (required)

Human-readable plugin description.

#### `skills`, `commands`, `sub_agents`

Arrays of component configurations:

```json
{
  "name": "component-name",
  "context": "./path/to/context.md",
  "files": ["./optional/files.json"],
  "metadata": {
    "author": "Your Name"
  }
}
```

#### `mcp_servers`

MCP server configurations:

**Bundled (local script):**
```json
{
  "name": "server-name",
  "type": "bundled",
  "path": "./servers/server.js",
  "config": {
    "args": ["--port", "8080"],
    "env": {
      "API_KEY": "${API_KEY}"
    }
  }
}
```

**Remote (npm package):**
```json
{
  "name": "server-name",
  "type": "remote",
  "source": "npm:@example/mcp-server",
  "version": "1.0.0"
}
```

## Template Variables

Context files support Jinja2 templating with these variables:

### Platform Variables

```jinja
{{ platform.os }}    {# "windows" | "linux" | "macos" #}
{{ platform.arch }}  {# "x64" | "arm64" | "arm" | "x86" #}
```

### Agent Variables

```jinja
{{ agent.name }}     {# "claude-code" | "cursor" | etc. #}
```

### Environment Variables

```jinja
{{ env.project.name }}   {# Project name from sdlc.json #}
{{ env.project.root }}   {# Project root path #}
{{ env.home }}           {# User home directory #}
{{ env.PATH }}           {# Any environment variable #}
```

### Plugin Variables

```jinja
{{ plugin.name }}        {# Plugin name #}
{{ plugin.version }}     {# Plugin version #}
{{ plugin.description }} {# Plugin description #}
```

### Component Variables

```jinja
{{ component.name }}     {# Component name #}
{{ component.type }}     {# "skill" | "command" | "sub_agent" #}
```

### Context Variables

```jinja
{{ context.root }}       {# Installation directory relative to project root #}
```

The `context.root` variable provides the path where the component is installed, relative to the project root. This is useful for referencing files that are part of the component.

**Examples by component type:**

| Component | `context.root` Example |
|-----------|------------------------|
| Skill | `.claude/skills/my-plugin-my-skill/` |
| Command | `.claude/commands/` |
| Sub-agent | `.claude/agents/` |

**Use case: Reference a bundled script**

If your skill includes a script file at `files/setup.sh`, you can reference it in the context:

```markdown
Run the setup script:
\`\`\`bash
{{ env.project.root }}/{{ context.root }}files/setup.sh
\`\`\`
```

This renders to something like:
```markdown
Run the setup script:
\`\`\`bash
/path/to/project/.claude/skills/my-plugin-my-skill/files/setup.sh
\`\`\`
```

## Conditional Content

Use platform conditionals for OS-specific instructions:

```jinja
{% if platform.os is windows %}
Use Windows-specific commands here.
{% elif platform.os is macos %}
Use macOS-specific commands here.
{% else %}
Use Linux/Unix commands here.
{% endif %}
```

The `unix` test matches both Linux and macOS:

```jinja
{% if platform.os is unix %}
This works on Linux and macOS.
{% endif %}
```

## Platform-Specific Files

Specify different files for different platforms:

```json
{
  "name": "my-skill",
  "context": "./context/skill.md",
  "files": {
    "common": ["./files/common.json"],
    "platform": {
      "windows": ["./files/windows-config.json"],
      "unix": ["./files/unix-config.json"]
    }
  }
}
```

## Platform-Specific File Overrides

Dex supports platform-specific file overrides using file naming conventions. This works for **any file** in the plugin - context files, scripts, configs, etc.

### Convention

- `{filename}.{ext}` - default for all platforms
- `{filename}.{platform}.{ext}` - platform-specific override
- `{filename}.{platform1,platform2}.{ext}` - shared override for multiple platforms

Examples:
- `context.md` - default context
- `context.claude_code.md` - Claude Code override
- `config.json` - default config
- `config.cursor.json` - Cursor-specific config
- `setup.{codex,antigravity}.sh` - shared script for Codex + Antigravity

### Platform Identifiers

Use underscores (not hyphens) in file names:

| Identifier | Platform |
|------------|----------|
| `claude_code` | Claude Code |
| `cursor` | Cursor |
| `codex` | OpenAI Codex |
| `github_copilot` | GitHub Copilot |
| `antigravity` | Antigravity |

### Example Directory Structure

```
my-plugin/
├── package.json
├── context/
│   ├── skill.md                          # Default for all platforms
│   ├── skill.claude_code.md              # Claude Code only
│   └── skill.cursor.md                   # Cursor only
├── scripts/
│   ├── setup.sh                          # Default script
│   ├── setup.claude_code.sh              # Claude Code version
│   └── setup.{codex,antigravity}.sh      # Shared for Codex + Antigravity
└── configs/
    ├── settings.json                     # Default config
    └── settings.cursor.json              # Cursor-specific config
```

### Resolution Order

When installing, Dex resolves files in this order:

1. **Platform-specific override** (e.g., `script.claude_code.sh`)
2. **Multi-platform override** (e.g., `script.{claude_code,cursor}.sh`)
3. **Default file** (e.g., `script.sh`)

### Usage

Reference the default file path in your `package.json`:

```json
{
  "skills": [
    {
      "name": "my-skill",
      "context": "./context/skill.md",
      "files": ["./scripts/setup.sh", "./configs/settings.json"]
    }
  ]
}
```

Dex automatically resolves to the platform-specific version during installation. You don't need to modify the manifest.

## Custom Filters

Available Jinja2 filters:

| Filter | Example | Result |
|--------|---------|--------|
| `basename` | `{{ "/path/to/file.txt" \| basename }}` | `file.txt` |
| `dirname` | `{{ "/path/to/file.txt" \| dirname }}` | `/path/to` |
| `extension` | `{{ "file.txt" \| extension }}` | `.txt` |
| `to_posix` | `{{ "path\\to\\file" \| to_posix }}` | `path/to/file` |

## Publishing Plugins

### Creating a Tarball

```bash
cd my-plugin
tar -czvf ../my-plugin-1.0.0.tar.gz .
```

### Adding to a Registry

1. Copy the tarball to the registry directory
2. Update `registry.json`:

```json
{
  "packages": {
    "my-plugin": {
      "versions": ["1.0.0"],
      "latest": "1.0.0"
    }
  }
}
```

## Platform-Specific Metadata

Each component (skill, command, sub_agent) supports a `metadata` field for passing platform-specific configuration. Each adapter only reads the keys it recognizes; unknown keys are ignored.

### Cursor Metadata

Cursor rules use MDC format with frontmatter:

```json
{
  "name": "code-style",
  "description": "Enforce code style guidelines",
  "context": "./skills/code-style.md",
  "glob": "**/*.ts",
  "always": true
}
```

| Field | Type | Description |
|-------|------|-------------|
| `glob` | string | File patterns to auto-attach rule (e.g., `"**/*.py"`, `"src/**/*.ts"`) |
| `always` | boolean | If true, applies to every chat session |

**Generated frontmatter:**
```yaml
---
description: Enforce code style guidelines
globs: **/*.ts
alwaysApply: true
---
```

### Claude Code Rule Metadata

Claude Code rules support file path scoping via the `paths` field:

```json
{
  "name": "testing-rules",
  "description": "Testing guidelines for test files",
  "context": "./rules/testing.md",
  "paths": ["tests/**/*.py", "**/*_test.py"]
}
```

| Field | Type | Description |
|-------|------|-------------|
| `paths` | string or string[] | File patterns to scope the rule (single path or array) |

**Generated frontmatter (when paths specified):**
```yaml
---
paths:
  - tests/**/*.py
  - **/*_test.py
---
```

If no `paths` are specified, Claude Code rules have no frontmatter (plain markdown).

### Cross-Platform Rule Example

This rule works on both Claude Code and Cursor:

```json
{
  "rules": [
    {
      "name": "typescript-style",
      "description": "TypeScript coding style guidelines",
      "context": "./rules/typescript.md",
      "glob": "**/*.ts",
      "paths": ["src/**/*.ts", "lib/**/*.ts"],
      "always": false
    }
  ]
}
```

When installed:
- **Cursor**: Uses `glob` and `always` → MDC frontmatter with `globs: **/*.ts`
- **Claude Code**: Uses `paths` → YAML frontmatter with `paths` array
- **Codex/Antigravity**: Plain markdown (no frontmatter)

### GitHub Copilot Metadata

GitHub Copilot uses instruction files with optional frontmatter:

```json
{
  "name": "python-lint",
  "description": "Python linting instructions",
  "context": "./commands/lint.md",
  "metadata": {
    "applyTo": "**/*.py",
    "excludeAgent": "code-review"
  }
}
```

| Key | Type | Description |
|-----|------|-------------|
| `applyTo` | string | Glob pattern for auto-attachment |
| `excludeAgent` | string | Agent to exclude (`"code-review"` or `"coding-agent"`) |

**Generated frontmatter (only if `applyTo` is set):**
```yaml
---
applyTo: "**/*.py"
excludeAgent: "code-review"
---
```

### Claude Code Metadata

Claude Code supports rich metadata for commands and sub-agents:

**Command metadata:**
```json
{
  "name": "deploy",
  "description": "Deploy to production",
  "context": "./commands/deploy.md",
  "metadata": {
    "argument_hint": "[environment] [--dry-run]",
    "allowed_tools": "Bash(deploy:*), Read",
    "model": "sonnet"
  }
}
```

| Key | Type | Description |
|-----|------|-------------|
| `argument_hint` | string | Hint for command arguments (e.g., `"[file] [options]"`) |
| `allowed_tools` | string/array | Tools this command can use |
| `model` | string | Model to use (`"sonnet"`, `"haiku"`, `"opus"`) |

**Sub-agent metadata:**
```json
{
  "name": "reviewer",
  "description": "Code review specialist",
  "context": "./agents/reviewer.md",
  "metadata": {
    "model": "inherit",
    "color": "green",
    "tools": ["Read", "Grep", "Glob"]
  }
}
```

| Key | Type | Description |
|-----|------|-------------|
| `model` | string | Model to use (`"inherit"` or specific model) |
| `color` | string | Agent color in UI (`"blue"`, `"green"`, etc.) |
| `tools` | array/string | Tools available to this agent |

### Codex Metadata

Codex skills support a short description for UI:

```json
{
  "name": "testing",
  "description": "Comprehensive testing assistance skill",
  "context": "./skills/testing.md",
  "metadata": {
    "short-description": "Help with tests"
  }
}
```

| Key | Type | Description |
|-----|------|-------------|
| `short-description` | string | Shorter description for UI display |

### Cross-Platform Example

This plugin works on all platforms, using platform-specific metadata:

```json
{
  "name": "universal-plugin",
  "version": "1.0.0",
  "description": "Works on all platforms",
  "skills": [
    {
      "name": "code-review",
      "description": "Automated code review",
      "context": "./skills/review.md",
      "metadata": {
        "globs": "**/*.{ts,js,py}",
        "alwaysApply": false,
        "short-description": "Review code"
      }
    }
  ],
  "commands": [
    {
      "name": "lint",
      "description": "Run linting",
      "context": "./commands/lint.md",
      "metadata": {
        "applyTo": "**/*.py",
        "argument_hint": "[files...]",
        "allowed_tools": "Bash(lint:*)"
      }
    }
  ]
}
```

When installed:
- **Cursor**: Uses `globs` and `alwaysApply` for the rule
- **GitHub Copilot**: Uses `applyTo` for the instruction
- **Claude Code**: Uses `argument_hint` and `allowed_tools` for the command
- **Codex**: Uses `short-description` for the skill
- **Antigravity**: Uses basic skill metadata (name, description)

## Best Practices

1. **Use descriptive names** - Choose clear, meaningful names for skills and commands
2. **Document thoroughly** - Include usage examples in context files
3. **Handle platform differences** - Test on all target platforms
4. **Pin dependencies** - Use specific version ranges for stability
5. **Include metadata** - Add author and category information
6. **Cross-platform metadata** - Include metadata for all target platforms; unused keys are ignored
