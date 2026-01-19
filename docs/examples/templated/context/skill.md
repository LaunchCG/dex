# Platform-Aware Skill

This skill demonstrates Jinja2 template features in Dex plugins.

## Project Information

- **Project Name**: {{ env.project.name }}
- **Plugin**: {{ plugin.name }} v{{ plugin.version }}

## Platform Detection

{% if platform.os == "windows" %}
You are running on **Windows**.

### Windows-Specific Instructions

- Use backslashes for paths: `C:\Users\{{ env.USER | default("user") }}\Projects`
- Run commands in PowerShell or Command Prompt
- Use `dir` instead of `ls`

{% elif platform.os == "macos" %}
You are running on **macOS**.

### macOS-Specific Instructions

- Use forward slashes for paths: `/Users/{{ env.USER | default("user") }}/Projects`
- Run commands in Terminal or iTerm
- Homebrew is recommended for package management

{% else %}
You are running on **Linux** ({{ platform.os }}).

### Linux-Specific Instructions

- Use forward slashes for paths: `/home/{{ env.USER | default("user") }}/Projects`
- Run commands in your preferred terminal
- Use your distribution's package manager

{% endif %}

## Architecture

Your system architecture is: **{{ platform.arch }}**

{% if platform.arch == "arm64" %}
Note: You're on ARM64. Some tools may need Rosetta 2 (macOS) or ARM-specific builds.
{% endif %}

## Agent Information

You are using the **{{ agent.name }}** AI coding assistant.

## Path Utilities

Example path operations:
- Home directory: `{{ env.home }}`
- Project root: `{{ env.project.root }}`
