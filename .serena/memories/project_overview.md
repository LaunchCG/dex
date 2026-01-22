# Dex Project Overview

## Purpose
Dex is a universal package manager for AI coding agents. It allows plugins to define skills, commands, subagents, rules, settings, and MCP servers that get installed into appropriate locations for each supported agent.

## Tech Stack
- Go 1.21+
- hashicorp/hcl/v2 for HCL parsing
- spf13/cobra for CLI
- Standard library testing

## Project Structure
```
dexv2/
├── cmd/dex/           # CLI entry point (main.go)
├── internal/
│   ├── adapter/       # Platform-specific adapters (claude.go)
│   ├── cli/           # Cobra commands (root.go, init.go, install.go, etc.)
│   ├── config/        # HCL parsing (project.go, package.go, parse.go)
│   ├── errors/        # Custom error types
│   ├── installer/     # Installation logic (installer.go)
│   ├── lockfile/      # dex.lock handling
│   ├── manifest/      # .dex/manifest.json handling
│   ├── registry/      # Package registry clients (local.go, git.go)
│   └── resource/      # Resource types (claude.go, resource.go)
├── pkg/
│   └── version/       # Semver utilities
├── testdata/          # Test fixtures
└── docs/              # Documentation
```

## Template Engine
The template engine (`internal/template/`) provides Go text/template rendering for plugin content:

- **Context**: Holds built-in variables (PluginName, PluginVersion, ProjectRoot, Platform, ComponentDir) plus user-defined variables
- **Engine**: Renders templates with context, supports file inclusion and nested templates
- **Built-in functions**: `file()` (include raw file), `env()` (read env var), `templatefile()` (render nested template)
- **Integration**: Adapter uses template engine to render resource content and template_file blocks

## Key Concepts
- **Plugins**: Define resources in package.hcl (HCL format)
- **Resources**: claude_skill, claude_command, claude_subagent, claude_rule, claude_rules, claude_settings, claude_mcp_server
- **Adapters**: Platform-specific implementations (currently only claude-code)
- **Installation Plans**: Adapters return plans describing files to write
- **Manifest**: .dex/manifest.json tracks all installed files per plugin
- **Lock File**: dex.lock tracks resolved versions and integrity hashes

## Configuration Files
- **dex.hcl**: Project configuration (project name, platform, plugins)
- **package.hcl**: Plugin definition (resources to install)
- **.dex/manifest.json**: Tracks installed files
- **dex.lock**: Locks resolved versions
