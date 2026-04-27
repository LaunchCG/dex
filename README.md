# Dex

Dex is a universal package manager for AI coding agents. It provides a standardized way to define, distribute, and install capabilities (skills, commands, rules, MCP servers) across multiple AI agent platforms using a single package format.

## Platform Support

Resources are defined once using universal types and automatically translate to platform-specific formats:

| Resource Type | Claude Code | GitHub Copilot | Cursor |
|---------------|:-----------:|:--------------:|:------:|
| `skill` | Skill | Skill | Skill |
| `command` | Command | Prompt | Command |
| `agent` | Subagent | Agent | — |
| `rule` (merged) | Rule → CLAUDE.md | Instruction → copilot-instructions.md | Rule → AGENTS.md |
| `rules` (standalone) | .claude/rules/ | .github/instructions/ | .cursor/rules/ |
| `settings` | .claude/settings.json | — | — |
| `mcp_server` | .mcp.json | .vscode/mcp.json | .cursor/mcp.json |

Unsupported resource types are automatically skipped with a log warning — no errors, no configuration needed.

## Installation

### Quick Install

**macOS / Linux:**

```bash
curl -fsSL https://dexartifactsproduction.z13.web.core.windows.net/install.sh | bash
```

**Windows (PowerShell):**

```powershell
irm https://dexartifactsproduction.z13.web.core.windows.net/install.ps1 | iex
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

# Install a package from GitHub
dex sync git+https://github.com/owner/my-package.git

# Install from a local directory (for development)
dex sync file:///path/to/my-package

# List installed packages
dex list

# Uninstall a package
dex uninstall my-package
```

## Package Format (package.hcl)

Packages are defined using HCL (HashiCorp Configuration Language). Define resources once — they work across all supported platforms automatically.

```hcl
meta {
  name        = "code-review-tools"
  version     = "1.0.0"
  description = "Code review capabilities for AI coding agents"
}

# One skill definition works on Claude Code and GitHub Copilot.
# Cursor doesn't support skills — automatically skipped.
skill "code-review" {
  description = "Thorough code review capability"
  content     = file("content/code-review.md")
}

# One command definition works everywhere.
# Translates to: Claude command, Copilot prompt, Cursor command.
command "review" {
  description = "Run code review on specified files"
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

agent "code-reviewer" {
  description = "Specialized code review agent"
  content     = file("agents/code-reviewer.md")
}

rule "review-standards" {
  description = "Code review standards"
  content     = "All PRs must be reviewed before merging."
}

settings "permissions" {
  claude {
    enable_all_project_mcp_servers = true
  }
}
```

### Directory Structure

```
my-package/
├── package.hcl           # Package manifest
├── content/
│   └── code-review.md    # Shared skill content
├── commands/
│   └── review.md         # Command content
└── agents/
    └── code-reviewer.md  # Agent content
```

## Project Configuration (dex.hcl)

Configure which packages are installed in your project:

```hcl
project {
  name             = "my-webapp"
  default_platform = "claude-code"
}

package "code-review-tools" {
  source = "git+https://github.com/owner/code-review-tools.git"
}

package "python-tools" {
  source = "git+https://github.com/owner/python-tools.git"
  config = {
    python_version = "3.12"
  }
}
```

### Profiles

Switch between different configurations:

```hcl
profile "qa" {
  agent_instructions = "QA environment — focus on testing"

  package "qa-tools" {
    source = "git+https://github.com/owner/qa-tools.git"
  }
}
```

```bash
dex sync              # Default configuration
dex sync --profile qa # QA configuration
```

## Documentation

- [Resources Reference](docs/RESOURCES.md) — All resource types, platform overrides, and examples
- [Configuration](docs/configuration.md) — Project and lock file configuration
- [Package Development](docs/packages.md) — Creating and publishing packages
- [CLI Reference](docs/cli-reference.md) — All commands and options
