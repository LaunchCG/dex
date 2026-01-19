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
