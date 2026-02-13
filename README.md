# Dex

Dex is a universal package manager for AI coding agents. It provides a standardized way to define, distribute, and install capabilities (skills, commands, rules, MCP servers) across multiple AI agent platforms using a single package format.

## Platform Support

| Resource Type | Claude Code | Cursor | GitHub Copilot |
|---------------|:-----------:|:------:|:--------------:|
| Skills | `claude_skill` | - | `copilot_skill` |
| Commands | `claude_command` | `cursor_command` | - |
| Prompts | - | - | `copilot_prompt` |
| Agents | `claude_subagent` | - | `copilot_agent` |
| Rules (merged) | `claude_rule` | `cursor_rule` | `copilot_instruction` |
| Rules (standalone) | `claude_rules` | `cursor_rules` | `copilot_instructions` |
| Settings | `claude_settings` | - | - |
| MCP Servers | `claude_mcp_server` | `cursor_mcp_server` | `copilot_mcp_server` |

## Installation

### Quick Install (Latest: v0.1.0)

**macOS / Linux:**

```bash
curl -fsSL https://dexartifactsproduction.z13.web.core.windows.net/install.sh | bash
```

**Windows (PowerShell):**

```powershell
irm https://dexartifactsproduction.z13.web.core.windows.net/install.ps1 | iex
```

**Install a specific version:**

```bash
# macOS / Linux
curl -fsSL https://dexartifactsproduction.z13.web.core.windows.net/install.sh | bash -s -- --version 0.1.0

# Windows (PowerShell)
& ([scriptblock]::Create((irm https://dexartifactsproduction.z13.web.core.windows.net/install.ps1))) -Version 0.1.0
```

The binary installs to `~/.bin/dex`. Make sure `~/.bin` is in your `PATH`.

### Build from Source

```bash
git clone https://github.com/launchcg/dex.git
cd dex
make build
make install
```

## Quick Start

```bash
# Initialize a project for a specific AI agent
dex init --platform claude-code

# Install a plugin from GitHub
dex install git+https://github.com/owner/my-plugin.git

# Install from a local directory (for development)
dex install file:///path/to/my-plugin

# List installed plugins
dex list

# Uninstall a plugin
dex uninstall my-plugin
```

## Package Format (package.hcl)

Plugins are defined using HCL (HashiCorp Configuration Language). This format enables content sharing across platforms using `file()` and platform-specific variations using `templatefile()`.

### Multi-Platform Example

This example shows a code review plugin that targets both Claude Code and GitHub Copilot, demonstrating content de-duplication:

```hcl
package {
  name        = "code-review-tools"
  version     = "1.0.0"
  description = "Code review capabilities for AI coding agents"
  platforms   = ["claude-code", "github-copilot"]
}

# Shared content via file() - written once, used by both platforms
claude_skill "code-review" {
  name        = "code-review"
  description = "Thorough code review capability"
  content     = file("content/code-review.md")  # Shared!
}

copilot_skill "code-review" {
  name        = "code-review"
  description = "Thorough code review capability"
  content     = file("content/code-review.md")  # Same shared content!
}

# Platform-specific variations via templatefile()
claude_command "review" {
  name        = "review"
  description = "Run code review on specified files"
  content     = templatefile("commands/review.md.tmpl", {
    tool_name = "Read"
  })
}

copilot_prompt "review" {
  name        = "review"
  description = "Run code review on specified files"
  content     = templatefile("commands/review.md.tmpl", {
    tool_name = "fetch"
  })
}
```

### Directory Structure

```
my-plugin/
├── package.hcl           # Plugin manifest
├── content/
│   └── code-review.md    # Shared skill content
└── commands/
    └── review.md.tmpl    # Template with platform variables
```

### Template Example (commands/review.md.tmpl)

```markdown
# Code Review

Use the {{ .tool_name }} tool to examine the specified files.

Review for:
1. Bugs and edge cases
2. Security vulnerabilities
3. Performance issues
4. Code style improvements
```

## Project Configuration (dex.hcl)

Configure which plugins are installed in your project:

```hcl
project {
  name             = "my-webapp"
  agentic_platform = "claude-code"
}

plugin "code-review-tools" {
  source = "git+https://github.com/owner/code-review-tools.git"
}

plugin "python-tools" {
  source = "git+https://github.com/owner/python-tools.git"
  config = {
    python_version = "3.12"
  }
}
```

## Documentation

See [docs/resources.md](docs/resources.md) for the complete reference on all resource types, HCL functions, and configuration options.
