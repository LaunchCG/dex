# Pipa Package Manager Specification v0.1

## Overview

Pipa is a universal package manager for AI-augmented development tools (e.g. Claude Code, Cursor, etc) that provides a standardized way to define, distribute, and install capabilities across multiple AI agent platforms.

## Core Principles

1. **Write Once, Deploy Everywhere**: Plugins are defined in a platform-agnostic format
2. **Non-Prescriptive**: Authors declare components; adapters handle platform specifics
3. **Composable**: Plugins can depend on and reference other plugins
4. **Extensible**: Support for scripts, configs, MCP servers, and arbitrary files
5. **Context-Aware**: Templating engine for platform and environment-specific variations

---

## Project Configuration: `sdlc.json`

Project-level manifest that declares installed plugins and configuration.

```json
{
  "agent": "claude-code",
  "project_name": "my-awesome-project",
  "registries": {
    "company": "s3://company-packages",
    "public": "https://registry.pipa.dev"
  },
  "default_registry": "company",
  "plugins": {
    "python-best-practices": "^2.1.0",
    "git-workflow": {
      "version": "1.5.2"
    },
    "database-tools": {
      "registry": "public",
      "version": "latest"
    },
    "custom-plugin": {
      "source": "git://github.com/company/custom-plugin.git"
    },
    "local-plugin": {
      "source": "file:../plugins/local-plugin"
    }
  },
  "config": {
    "cache_dir": ".pipa/cache"
  }
}
```

### Fields

**`agent`** (string, required)
- Target AI agent platform: `"claude-code"` | `"cursor"` | `"codex"` | `"antigravity"`

**`project_name`** (string, optional)
- Human-readable project name for use in templates
- If not specified, defaults to the project root directory name

**`registries`** (object, optional)
- Named registry definitions
- Key: registry name, Value: registry URL (s3://, az://, https://, etc.)

**`default_registry`** (string, optional)
- Name of the default registry to use for plugins without explicit registry
- Must match a key in `registries`

**`plugins`** (object, required)
- Key: plugin name
- Value: version specifier (string) or plugin specification object

**`config`** (object, optional)
- Package manager configuration options

---

## Lock File: `sdlc.lock`

Auto-generated file that pins exact versions for deterministic installs.

```json
{
  "version": "1.0",
  "agent": "claude-code",
  "plugins": {
    "python-best-practices": {
      "version": "2.1.3",
      "resolved": "https://registry.pipa.dev/python-best-practices-2.1.3.tar.gz",
      "integrity": "sha512-abc123...",
      "dependencies": {
        "git-workflow": "1.5.2",
        "testing-framework": "2.0.1"
      }
    },
    "git-workflow": {
      "version": "1.5.2",
      "resolved": "https://registry.pipa.dev/git-workflow-1.5.2.tar.gz",
      "integrity": "sha512-def456..."
    },
    "testing-framework": {
      "version": "2.0.1",
      "resolved": "https://registry.pipa.dev/testing-framework-2.0.1.tar.gz",
      "integrity": "sha512-ghi789..."
    }
  }
}
```

### Lock File Behavior

- Auto-generated on `pipa install`
- Updated on `pipa update`
- Used for reproducible installs across environments
- Should be committed to version control
- `pipa install` without arguments uses lock file versions

---

## Plugin Definition: `package.json`

Plugin-level manifest that defines a plugin and its components (skills, commands, sub-agents, MCP servers).

```json
{
  "name": "python-best-practices",
  "version": "2.1.0",
  "description": "Python coding standards, linting, and testing workflows",
  
  "skills": [
    {
      "name": "linting",
      "context": "./skills/linting.md",
      "files": [
        "configs/.ruff.toml",
        "configs/mypy.ini"
      ]
    },
    {
      "name": "testing",
      "context": "./skills/testing.md",
      "files": ["configs/pytest.ini"]
    }
  ],
  
  "commands": [
    {
      "name": "format-code",
      "context": "./commands/format.md",
      "files": {
        "platform": {
          "windows": ["scripts/format.bat"],
          "unix": ["scripts/format.sh"]
        }
      }
    }
  ],
  
  "sub_agents": [
    {
      "name": "code-reviewer",
      "context": "./agents/reviewer.md"
    }
  ],
  
  "mcp_servers": [
    {
      "name": "python-tools",
      "type": "bundled",
      "path": "./mcp/python-server.js"
    }
  ],
  
  "dependencies": {
    "git-workflow": "^1.0.0",
    "testing-framework": "^2.0.0"
  },
  
  "rules": {
    "overridable": true,
    "defaults": {
      "line_length": 100,
      "python_version": "3.11"
    }
  }
}
```

### Required Fields

**`name`** (string)
- Unique plugin identifier (kebab-case recommended)

**`version`** (string)
- Semver version (e.g., "1.2.0")

**`description`** (string)
- Short description of plugin functionality

### Optional Fields

**`skills`** (array)
- Skill definitions contained in this plugin

**`commands`** (array)
- Command definitions contained in this plugin

**`sub_agents`** (array)
- Sub-agent definitions contained in this plugin

**`mcp_servers`** (array)
- MCP server definitions

**`dependencies`** (object)
- Other plugins this plugin requires

**`rules`** (object)
- Default rules/configurations

**`metadata`** (object)
- Arbitrary metadata for platform-specific use

---

## Plugin Components

### Skills

Skills are multi-step workflows or contextual knowledge packages.

```json
{
  "skills": [
    {
      "name": "database-migrations",
      "context": "./skills/migrations.md",
      "files": {
        "common": ["templates/migration.sql"],
        "platform": {
          "windows": ["scripts/migrate.bat"],
          "unix": ["scripts/migrate.sh"]
        }
      }
    }
  ]
}
```

**Fields:**
- `name` (string, required): Unique skill identifier
- `context` (string | array, required): Path(s) to `.md` context file(s)
- `files` (array | object, optional): Associated files
- `metadata` (object, optional): Additional metadata

### Commands

Commands are direct CLI invocations or one-shot operations.

```json
{
  "commands": [
    {
      "name": "lint",
      "context": "./commands/lint.md",
      "files": ["scripts/lint.sh"]
    }
  ]
}
```

**Fields:**
- `name` (string, required): Unique command identifier
- `context` (string | array, required): Path(s) to `.md` context file(s)
- `files` (array | object, optional): Associated files

### Sub-Agents

Sub-agents are specialized AI personas with scoped responsibilities.

```json
{
  "sub_agents": [
    {
      "name": "security-auditor",
      "context": "./agents/security.md",
      "files": ["configs/security-rules.json"]
    }
  ]
}
```

**Fields:**
- `name` (string, required): Unique sub-agent identifier
- `context` (string | array, required): Path(s) to `.md` context file(s)
- `files` (array | object, optional): Associated files

---

## Package Distribution Formats

Plugins can be distributed in multiple formats:

### 1. Compressed Archive (.tar.gz)

```
python-best-practices-2.1.0.tar.gz
└── (extracted)
    ├── package.json
    ├── skills/
    │   ├── linting.md
    │   └── testing.md
    ├── commands/
    │   └── format.md
    ├── agents/
    │   └── reviewer.md
    ├── scripts/
    └── configs/
```

### 2. Git Repository

```
git://github.com/company/python-best-practices.git
└── (repository root)
    ├── package.json
    ├── skills/
    ├── commands/
    ├── agents/
    ├── scripts/
    └── configs/
```

### 3. Local Directory

```
file:../plugins/python-best-practices
└── (directory)
    ├── package.json
    ├── skills/
    ├── commands/
    └── ...
```

**Requirements**: All formats must have `package.json` at the root.

---

## Context Files

Context files are **actual markdown files** (`.md`) containing instructions for the AI agent.

**Example**: `skills/linting.md`
```markdown
# Python Linting Skill

Run comprehensive Python linting using ruff and mypy.

## Usage
When asked to lint Python files:
1. Run ruff for style checking
2. Run mypy for type checking  
3. Report issues with file:line:column format

## Configuration
Use the included .ruff.toml and mypy.ini configs.
```

Context files support templating (see Templating section).

---

## Files

### Simple File List

```json
{
  "files": [
    "scripts/lint.sh",
    "scripts/fix.sh",
    "configs/.ruff.toml",
    "configs/mypy.ini"
  ]
}
```

### Platform-Conditional Files

```json
{
  "files": {
    "common": [
      "configs/.ruff.toml",
      "configs/mypy.ini"
    ],
    "platform": {
      "windows": [
        "scripts/lint.bat",
        "scripts/fix.bat"
      ],
      "unix": [
        "scripts/lint.sh",
        "scripts/fix.sh"
      ],
      "linux": [
        "scripts/linux-specific.sh"
      ],
      "macos": [
        "scripts/mac-specific.sh"
      ]
    }
  }
}
```

**Platform values**: `windows`, `linux`, `macos`, `unix` (matches both linux and macos)

### File Installation Targets

```json
{
  "files": [
    {
      "src": "scripts/lint.sh",
      "dest": "bin/",
      "chmod": "+x"
    },
    {
      "src": "configs/.ruff.toml",
      "dest": "."
    }
  ]
}
```

---

## MCP Servers

Plugins can bundle or reference MCP servers.

### Bundled MCP Server

```json
{
  "mcp_servers": [
    {
      "name": "database-tools",
      "type": "bundled",
      "path": "./mcp/database-server.js",
      "config": {
        "args": ["--db", "${DB_URL}"],
        "env": {
          "DB_TIMEOUT": "5000"
        }
      }
    }
  ]
}
```

### Remote MCP Server

```json
{
  "mcp_servers": [
    {
      "name": "github-tools",
      "type": "remote",
      "source": "npm:@modelcontextprotocol/server-github",
      "version": "^0.5.0",
      "config": {
        "env": {
          "GITHUB_TOKEN": "${GITHUB_TOKEN}"
        }
      }
    }
  ]
}
```

### Platform-Conditional MCP

```json
{
  "mcp_servers": [
    {
      "name": "file-tools",
      "type": "bundled",
      "path": {
        "windows": "./mcp/file-server-win.exe",
        "unix": "./mcp/file-server"
      },
      "config": {
        "args": ["--root", "${project.root}"]
      }
    }
  ]
}
```

---

## Templating Engine

Context files (`.md` files) and configurations support templating for conditional content.

### Syntax

Uses `{{` `}}` for expressions, `{%` `%}` for logic blocks.

### Available Variables

```
# Operating System
platform.os          # "windows" | "linux" | "macos"
platform.arch        # "x64" | "arm64" | etc.

# AI Agent
agent.name           # "claude-code" | "cursor" | "codex" | "antigravity"

# Environment
env.project.root     # Current project path
env.*                # Any environment variable

# Plugin metadata
plugin.name          # Current plugin name
plugin.version       # Current plugin version
plugin.dependencies  # List of dependency names

# Component metadata (when in a skill/command/sub-agent context)
component.name       # Current component name
component.type       # "skill" | "command" | "sub-agent"
```

### Template Examples

**Conditional Instructions** (`skills/migrations.md`):
```markdown
# Database Migration Skill

{% if agent.name == "claude-code" %}
Use the bash_tool to execute migrations.
{% elif agent.name == "cursor" %}
Use the terminal integration to execute migrations.
{% endif %}

Run migrations using:
{% if platform.os == "windows" %}
  .\scripts\migrate.bat
{% else %}
  ./scripts/migrate.sh
{% endif %}
```

**Variable Substitution**:
```markdown
Project root: {{ env.project.root }}
Using {{ plugin.name }} version {{ plugin.version }}
Running on {{ platform.os }} ({{ platform.arch }})
This is the {{ component.name }} {{ component.type }}
```

**Conditional Context Files**:
```json
{
  "skills": [
    {
      "name": "advanced-features",
      "context": [
        "./skills/base.md",
        {
          "path": "./skills/windows-notes.md",
          "if": "platform.os == 'windows'"
        },
        {
          "path": "./skills/mcp-integration.md",
          "if": "agent.name == 'claude-code'"
        }
      ]
    }
  ]
}
```

---

## Dependencies

Plugins can depend on other plugins.

### Dependency Declaration

```json
{
  "dependencies": {
    "git-workflow": "^1.0.0",
    "python-env": "~2.3.0",
    "testing-framework": "latest"
  }
}
```

### Version Specifiers

- **Exact**: `"1.2.3"`
- **Caret** (compatible): `"^1.2.3"` (>= 1.2.3, < 2.0.0)
- **Tilde** (patch): `"~1.2.3"` (>= 1.2.3, < 1.3.0)
- **Latest**: `"latest"`
- **Git**: `"git://github.com/user/plugin.git"`
- **Local**: `"file:../plugins/custom"`

### Dependency Resolution

- Dependencies are installed recursively
- Conflicts resolved using semantic versioning rules
- Circular dependencies are detected and rejected
- Resolution results are written to `sdlc.lock`

---

## Platform Adapters

Adapters translate the standard format into platform-specific implementations.

### Adapter Responsibilities

1. **Context Placement**: Determine where to write component instructions
2. **Frontmatter Generation**: Add platform-specific metadata headers
3. **File Organization**: Copy files to appropriate platform directories
4. **MCP Integration**: Configure MCP servers per platform conventions
5. **Template Rendering**: Resolve templates with platform-specific values

### Example Adapter Outputs

**Claude Code**:
```
.skills/python-best-practices/
├── linting/
│   ├── SKILL.md              # Rendered from skills/linting.md
│   └── configs/
│       ├── .ruff.toml
│       └── mypy.ini
├── testing/
│   ├── SKILL.md
│   └── configs/
│       └── pytest.ini
└── format-code/
    ├── SKILL.md              # Rendered from commands/format.md
    └── scripts/
        └── format.sh
```

**Cursor**:
```
.cursor/
├── .cursorrules              # Appended contexts
└── skills/
    └── python-best-practices/
        ├── linting/
        ├── testing/
        └── format-code/
```

**Antigravity** (hypothetical):
```
.antigravity/
├── plugins/
│   └── python-best-practices/
│       ├── plugin.json       # Metadata
│       ├── skills/
│       ├── commands/
│       └── agents/
└── resources/
    └── python-best-practices/
        ├── scripts/
        └── configs/
```

---

## Rules and Defaults

Plugins can declare suggested rules or default configurations.

```json
{
  "rules": {
    "overridable": true,
    "defaults": {
      "max_line_length": 100,
      "use_type_hints": true,
      "linters": ["ruff", "mypy"]
    },
    "suggestions": [
      "Run linting before commits",
      "Fix auto-fixable issues automatically"
    ]
  }
}
```

Platform adapters can:
- Apply defaults to project configuration
- Present suggestions to user during installation
- Allow user override of defaults

---

## Complete Example

### Project: `sdlc.json`

```json
{
  "agent": "claude-code",
  "project_name": "Python Best Practices Demo",
  "registries": {
    "company": "s3://company-packages"
  },
  "default_registry": "company",
  "plugins": {
    "python-best-practices": "^2.1.0",
    "git-workflow": {
      "version": "1.5.2"
    },
    "custom-db-tools": {
      "source": "git://github.com/company/db-tools.git"
    }
  }
}
```

### Lock File: `sdlc.lock`

```json
{
  "version": "1.0",
  "agent": "claude-code",
  "plugins": {
    "python-best-practices": {
      "version": "2.1.3",
      "resolved": "https://registry.pipa.dev/python-best-practices-2.1.3.tar.gz",
      "integrity": "sha512-abc123...",
      "dependencies": {
        "git-workflow": "1.5.2",
        "testing-framework": "2.0.1"
      }
    },
    "git-workflow": {
      "version": "1.5.2",
      "resolved": "https://registry.pipa.dev/git-workflow-1.5.2.tar.gz",
      "integrity": "sha512-def456..."
    },
    "testing-framework": {
      "version": "2.0.1",
      "resolved": "https://registry.pipa.dev/testing-framework-2.0.1.tar.gz",
      "integrity": "sha512-ghi789..."
    }
  }
}
```

### Plugin: `package.json`

```json
{
  "name": "python-best-practices",
  "version": "2.1.0",
  "description": "Python coding standards, linting, and testing workflows",
  
  "skills": [
    {
      "name": "linting",
      "context": "./skills/linting.md",
      "files": {
        "common": [
          "configs/.ruff.toml",
          "configs/mypy.ini"
        ]
      }
    },
    {
      "name": "testing",
      "context": "./skills/testing.md",
      "files": ["configs/pytest.ini"]
    }
  ],
  
  "commands": [
    {
      "name": "format-code",
      "context": "./commands/format.md",
      "files": {
        "platform": {
          "windows": ["scripts/format.bat"],
          "unix": ["scripts/format.sh"]
        }
      }
    }
  ],
  
  "sub_agents": [
    {
      "name": "code-reviewer",
      "context": "./agents/reviewer.md"
    }
  ],
  
  "mcp_servers": [
    {
      "name": "python-tools",
      "type": "bundled",
      "path": "./mcp/python-server.js",
      "config": {
        "args": ["--project", "${project.root}"]
      }
    }
  ],
  
  "dependencies": {
    "git-workflow": "^1.0.0",
    "testing-framework": "^2.0.0"
  },
  
  "rules": {
    "overridable": true,
    "defaults": {
      "line_length": 100,
      "python_version": "3.11"
    }
  }
}
```

### Context File: `skills/linting.md`

```markdown
# Python Linting Skill

Comprehensive Python code quality checking using ruff and mypy.

## Overview

This skill provides automated linting for Python codebases using:
- **ruff** for style checking (configured via .ruff.toml)
- **mypy** for type checking (configured via mypy.ini)

## Usage

{% if platform.os == "windows" %}
Run linting with: `.\scripts\lint.bat`
{% else %}
Run linting with: `./scripts/lint.sh`
{% endif %}

The linting script will:
1. Execute ruff for PEP 8 compliance and code quality
2. Execute mypy for type hint verification
3. Report all issues with file:line:column format

## Configuration

This skill uses the following config files:
- `.ruff.toml` - Ruff linter configuration (max line length: {{ plugin.rules.defaults.line_length }})
- `mypy.ini` - MyPy type checker configuration

All configs are set to {{ plugin.name }} defaults but can be overridden.

{% if agent.name == "claude-code" %}
## MCP Integration

The python-tools MCP server is available with additional commands for:
- Running ruff with auto-fix
- Executing mypy with specific modules
- Batch processing multiple files

Access via the MCP tool interface.
{% endif %}

## Best Practices

- Run linting before committing code
- Fix auto-fixable issues automatically with ruff
- Maintain type hints for all public functions
- Keep line length under {{ plugin.rules.defaults.line_length }} characters
```

---

## CLI Interface

```bash
# Initialize project
pipa init
pipa init --agent claude-code

# Install plugins
pipa install python-best-practices
pipa install python-best-practices@2.1.0
pipa install git://github.com/user/plugin.git
pipa install file:../plugins/custom
pipa install  # Install all from sdlc.json using lock file

# Update plugins
pipa update python-best-practices
pipa update --all

# Remove plugins
pipa remove python-best-practices

# List installed plugins
pipa list
pipa list --tree  # Show dependency tree

# Show plugin info
pipa info python-best-practices

# Configure agent
pipa config set agent cursor
pipa config get agent

# Lock file management
pipa install --no-lock  # Install without updating lock file
pipa lock  # Regenerate lock file from sdlc.json

# Plugin development
pipa pack  # Create .tar.gz from package.json
pipa publish  # Publish to registry (future)
pipa link  # Link local plugin for development
```

---

---

## Registries

A registry is a storage location (S3, Azure Blob, HTTP server, etc.) that hosts plugin packages and a registry index.

### Registry Structure

**Standardized naming convention:** `{name}-{version}.tar.gz`

**Example registry root:**
```
s3://company-packages/
├── registry.json                  # Registry index
├── python-linter-1.0.0.tar.gz
├── python-linter-2.0.0.tar.gz
├── db-tools-1.5.0.tar.gz
└── git-workflow-1.0.0.tar.gz
```

### Registry Index: `registry.json`

Located at the registry root, this file declares available packages and versions.

```json
{
  "packages": {
    "python-linter": {
      "versions": ["1.0.0", "1.5.0", "2.0.0"],
      "latest": "2.0.0"
    },
    "db-tools": {
      "versions": ["1.0.0", "1.5.0"],
      "latest": "1.5.0"
    },
    "git-workflow": {
      "versions": ["1.0.0"],
      "latest": "1.0.0"
    }
  }
}
```

### Configuring Registries

In `sdlc.json`:

```json
{
  "registries": {
    "company": "s3://company-packages",
    "azure-reg": "az://myaccount/pipa-container",
    "public": "https://cdn.example.com/pipa"
  },
  "default_registry": "company",
  
  "plugins": {
    "python-linter": "^2.0.0",
    
    "db-tools": {
      "registry": "azure-reg",
      "version": "^1.0.0"
    },
    
    "custom-plugin": {
      "source": "git://github.com/user/plugin.git"
    }
  }
}
```

### Plugin Specification Formats

**Shorthand (string):**
```json
{
  "plugins": {
    "python-linter": "^2.0.0"  // Uses default_registry
  }
}
```

**Object with registry:**
```json
{
  "plugins": {
    "db-tools": {
      "registry": "azure-reg",
      "version": "^1.5.0"
    }
  }
}
```

**Object with direct source:**
```json
{
  "plugins": {
    "custom": {
      "source": "git://github.com/company/plugin.git"
    },
    "local-dev": {
      "source": "file:../plugins/local-dev"
    },
    "remote-package": {
      "source": "https://example.com/packages/plugin-2.0.0.tar.gz"
    }
  }
}
```

### Installation Flow

When installing `python-linter@^2.0.0` from registry `s3://company-packages`:

1. Fetch `s3://company-packages/registry.json`
2. Find latest compatible version for `python-linter` matching `^2.0.0` → `2.0.0`
3. Download `s3://company-packages/python-linter-2.0.0.tar.gz`
4. Extract and install

### Supported Registry Protocols

- **S3**: `s3://bucket-name/path`
- **Azure Blob**: `az://account/container/path`
- **HTTPS**: `https://example.com/path`
- **Git**: `git://github.com/user/repo.git` (no registry.json, direct install)
- **Local**: `file:../local/path` (no registry.json, direct install)

---

## Dependency Resolution

### Version Conflict Handling

When multiple plugins require the same dependency at incompatible versions, Pipa **errors and halts installation**.

**Example:**
- `plugin-a` requires `common-lib@^1.0.0`
- `plugin-b` requires `common-lib@^2.0.0`

**Result:** Error message with conflict details, requiring manual resolution.

**Resolution options:**
- Update plugin versions to be compatible
- Contact plugin maintainers to update dependencies
- Fork and modify one of the plugins

### Circular Dependency Detection

Circular dependencies are **detected and result in an error**.

**Example:**
- `plugin-a` depends on `plugin-b`
- `plugin-b` depends on `plugin-a`

**Result:** Error with dependency chain visualization.

### Compatible Version Selection

When multiple plugins depend on the same package with compatible version ranges, Pipa selects the **highest version that satisfies all constraints**.

**Example:**
- `plugin-a` requires `common-lib@^1.5.0`
- `plugin-b` requires `common-lib@^1.2.0`

**Result:** Installs `common-lib@1.5.x` (latest that satisfies both `^1.5.0` and `^1.2.0`)

---

## Environment Variables

### Declaration

Plugins can declare required environment variables in MCP server configs or metadata:

```json
{
  "mcp_servers": [
    {
      "name": "github-tools",
      "type": "remote",
      "source": "npm:@modelcontextprotocol/server-github",
      "config": {
        "env": {
          "GITHUB_TOKEN": "${GITHUB_TOKEN}"
        }
      }
    }
  ],
  "env_variables": {
    "GITHUB_TOKEN": {
      "description": "GitHub personal access token for API access",
      "required": true
    },
    "GITHUB_ORG": {
      "description": "Default GitHub organization",
      "required": false,
      "default": "my-org"
    }
  }
}
```

### Validation

During installation, Pipa:
1. Collects all required environment variables from all plugins being installed
2. Checks if each variable is currently set in the environment
3. Displays a summary at the end of installation

**Example output:**
```
✓ Installed python-best-practices@2.1.0
✓ Installed git-workflow@1.5.2
✓ Installed github-tools@1.0.0

⚠ Environment Variables Required:

  GITHUB_TOKEN (required by github-tools)
    Description: GitHub personal access token for API access
    Status: NOT SET

  DATABASE_URL (required by db-migration)
    Description: PostgreSQL connection string
    Status: NOT SET

  SLACK_TOKEN (required by notifications)
    Description: Slack bot token
    Status: SET ✓

Set these variables in your environment before using the installed plugins.
```

### Variable Reference Syntax

In configuration files, use `${VAR_NAME}` to reference environment variables:

```json
{
  "config": {
    "args": ["--token", "${API_TOKEN}"],
    "env": {
      "DATABASE_URL": "${DATABASE_URL}",
      "LOG_LEVEL": "${LOG_LEVEL:-info}"
    }
  }
}
```

The `${VAR:-default}` syntax provides a default value if the variable is not set.

---

## Template Engine (Jinja2)

Context files (`.md` files) use Jinja2 templating for conditional content and variable substitution.

### Syntax

- **Expressions**: `{{ variable }}`
- **Statements**: `{% if condition %}...{% endif %}`
- **Comments**: `{# comment #}`

### Available Variables

#### Platform Information
```
platform.os          # "windows" | "linux" | "macos"
platform.arch        # "x64" | "arm64" | "arm" | "x86"
```

#### Agent Information
```
agent.name           # "claude-code" | "cursor" | "codex" | "antigravity"
agent.version        # Agent version string (if available)
```

#### Environment Variables
```
env.project.root     # Current project root path
env.project.name     # Project name (from sdlc.json project_name or root directory name)
env.home            # User home directory
env.*               # Any environment variable (e.g., env.GITHUB_TOKEN)
```

**Note on `env.project.name`:**
- If `project_name` is specified in `sdlc.json`, that value is used
- Otherwise, defaults to the name of the project root directory
- Example: If project is in `/home/user/my-awesome-app/` and no `project_name` is set, `env.project.name` will be `"my-awesome-app"`

#### Plugin Metadata
```
plugin.name          # Current plugin name
plugin.version       # Current plugin version
plugin.description   # Plugin description
```

#### Component Metadata
```
component.name       # Current component name (skill/command/sub-agent)
component.type       # "skill" | "command" | "sub-agent"
```

### Built-in Filters

Jinja2 standard filters are available, plus custom filters:

#### String Filters
```
{{ text | upper }}           # UPPERCASE
{{ text | lower }}           # lowercase
{{ text | title }}           # Title Case
{{ text | capitalize }}      # Capitalize first letter
{{ text | replace("old", "new") }}
```

#### Path Filters
```
{{ path | basename }}        # Get filename from path
{{ path | dirname }}         # Get directory from path
{{ path | abspath }}         # Convert to absolute path
{{ path | normpath }}        # Normalize path separators
```

#### Formatting Filters
```
{{ items | join(", ") }}     # Join list with separator
{{ text | indent(4) }}       # Indent text by N spaces
{{ text | wordwrap(80) }}    # Wrap text at N characters
```

### Built-in Tests

```
{% if platform.os is windows %}
{% if env.DEBUG is defined %}
{% if version is version_compatible("^2.0.0") %}
```

### Control Structures

#### Conditionals
```markdown
{% if platform.os == "windows" %}
Run: `.\script.bat`
{% elif platform.os == "macos" %}
Run: `./script-mac.sh`
{% else %}
Run: `./script.sh`
{% endif %}
```

#### Loops
```markdown
Available commands:
{% for cmd in commands %}
- {{ cmd.name }}: {{ cmd.description }}
{% endfor %}
```

#### Macros
```markdown
{% macro code_block(language, code) %}
```{{ language }}
{{ code }}
```
{% endmacro %}

{{ code_block("python", "print('hello')") }}
```

### Template Examples

**Platform-specific instructions:**
```markdown
# Database Setup

{% if platform.os == "windows" %}
1. Download PostgreSQL installer from https://postgresql.org
2. Run the installer: `postgresql-installer.exe`
3. Set environment variable: `setx DATABASE_URL "postgresql://localhost/mydb"`
{% else %}
1. Install PostgreSQL:
   {% if platform.os == "macos" %}
   ```bash
   brew install postgresql
   ```
   {% else %}
   ```bash
   sudo apt-get install postgresql
   ```
   {% endif %}
2. Set environment variable: `export DATABASE_URL="postgresql://localhost/mydb"`
{% endif %}
```

**Agent-specific features:**
```markdown
# Running Tests

{% if agent.name == "claude-code" %}
Use the bash_tool to execute: `pytest tests/`
{% elif agent.name == "cursor" %}
Use the integrated terminal to run: `pytest tests/`
{% endif %}

{% if agent.name == "claude-code" %}
## MCP Integration
The test-runner MCP server provides additional commands for test execution.
{% endif %}
```

**Environment variable usage:**
```markdown
# Configuration

Project: {{ env.project.name }}
Project Root: {{ env.project.root }}
Environment: {{ env.ENVIRONMENT | default("development") | upper }}

{% if env.DEBUG is defined %}
🐛 Debug mode is enabled
{% endif %}

Database: {{ env.DATABASE_URL | default("Not configured") }}
```

**Component metadata:**
```markdown
# {{ component.name | title }} {{ component.type | title }}

Version: {{ plugin.version }}
Plugin: {{ plugin.name }}

{% if component.type == "skill" %}
This is a skill that provides contextual knowledge and workflows.
{% elif component.type == "command" %}
This is a command for direct execution.
{% endif %}
```

### Error Handling

**Undefined variables:**
- By default, undefined variables throw an exception
- Use `{% if var is defined %}` to check existence
- Use `{{ var | default("fallback") }}` for defaults

**Template syntax errors:**
- Pipa halts installation and displays error with line number. It should also roll back any
actions it did take.
- Error message shows the problematic template section
