# Dex Development Guidelines

## Testing Rules

### File Content Assertions
When testing generated files (frontmatter, configs, etc.), **ALWAYS assert the entire expected content**, not just substrings. This ensures:
- No unexpected content is included
- The exact format is correct
- Changes to output are immediately caught

**BAD:**
```python
def test_generates_frontmatter(self):
    frontmatter = generate_frontmatter(skill)
    assert "name: test-skill" in frontmatter  # Only checks substring!
    assert "description:" in frontmatter
```

**GOOD:**
```python
def test_generates_frontmatter(self):
    frontmatter = generate_frontmatter(skill)
    expected = """\
---
name: test-skill
description: A test skill
version: 1.0.0
---
"""
    assert frontmatter == expected
```

### Test Coverage
- Every feature must have tests that validate the complete output
- Use fixtures for common test data
- Test both success and error cases

## Development Tools

**CRITICAL: Use MCP dev-toolkit tools for ALL development tasks. DO NOT use shell commands or Makefiles.**

This project provides MCP (Model Context Protocol) tools for all development operations. You MUST use these tools instead of running shell commands directly.

### Available Dev Tools

- `mcp__dev-toolkit-mcp__run_build` - Build the CLI binary to bin/dex
- `mcp__dev-toolkit-mcp__run_clean` - Remove built binary and coverage files
- `mcp__dev-toolkit-mcp__run_fmt` - Format code with go fmt
- `mcp__dev-toolkit-mcp__run_install` - Install binary to GOPATH/bin
- `mcp__dev-toolkit-mcp__run_install-user` - Install binary to ~/.bin
- `mcp__dev-toolkit-mcp__run_lint` - Run linter (fmt + vet)
- `mcp__dev-toolkit-mcp__run_test` - Run all tests
- `mcp__dev-toolkit-mcp__run_test-cover` - Run tests with coverage report
- `mcp__dev-toolkit-mcp__run_vet` - Run go vet

### Examples

**BAD:**
```bash
# DO NOT run shell commands directly
make build
make test
go fmt ./...
```

**GOOD:**
```python
# Use MCP dev-toolkit tools
mcp__dev-toolkit-mcp__run_build()
mcp__dev-toolkit-mcp__run_test()
mcp__dev-toolkit-mcp__run_fmt()
```

### Why MCP Tools?

- **Consistency**: Ensures consistent build and test environments
- **Tracking**: MCP tools are tracked and managed
- **Integration**: Better integration with the development workflow
- **No Makefiles**: Project uses MCP tools instead of traditional Makefiles

## Code Style

### Imports
- Use absolute imports
- Group imports: stdlib, third-party, local
- Sort alphabetically within groups

### Type Hints
- All public functions must have type hints
- Use `from __future__ import annotations` for forward references

## Project Structure

```
dex/
├── adapters/       # Platform-specific adapters (claude-code, cursor, etc.)
├── cli/            # Command-line interface
├── config/         # Configuration schemas and parsing
├── core/           # Core functionality (installer, manifest, etc.)
├── registry/       # Package registry clients
├── template/       # Template engine and context building
└── utils/          # Utility functions
```

## Adapters

Each platform adapter must implement:
- Directory structure methods (get_skills_directory, get_commands_directory, etc.)
- Installation planning (plan_skill_installation, plan_command_installation, etc.)
- Frontmatter generation
- MCP configuration

## Manifest System

The `.dex/manifest.json` tracks all files managed by dex:
- Files are stored as relative paths
- MCP servers are tracked per-plugin for cleanup
- Old files are removed during reinstall/upgrade
