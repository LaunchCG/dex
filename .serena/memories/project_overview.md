# Dex Project Overview

## Purpose
Dex is a universal package manager for AI coding agents. It allows plugins to define skills, commands, rules, etc. that get installed into appropriate locations for each supported agent (Claude Code, Cursor, Codex, GitHub Copilot, Antigravity).

## Tech Stack
- Python 3.11+
- Pydantic 2.x for data validation
- Jinja2 for templating
- Typer for CLI
- pytest for testing

## Project Structure
```
dex/
├── adapters/       # Platform-specific adapters (claude_code, cursor, codex, etc.)
├── cli/            # Command-line interface (main.py)
├── config/         # Configuration schemas (schemas.py) and parsing (parser.py)
├── core/           # Core functionality (installer.py, manifest.py, project.py)
├── registry/       # Package registry clients (local.py, base.py)
├── template/       # Template engine (engine.py) and context building (context.py)
└── utils/          # Utility functions (version.py, filesystem.py, platform.py)
```

## Key Concepts
- **Plugins**: Define skills, commands, rules, etc. in package.json
- **Adapters**: Platform-specific implementations for each AI agent
- **Installation Plans**: Adapters return plans that describe what to write
- **Manifest**: .dex/manifest.json tracks all installed files per plugin
