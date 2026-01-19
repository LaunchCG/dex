# Installation

## Prerequisites

- Python 3.11 or higher
- pip or uv package manager

## Install via pip

```bash
pip install dex
```

## Install via uv (Recommended)

[uv](https://github.com/astral-sh/uv) is a fast Python package installer:

```bash
uv pip install dex
```

## Install from Source

```bash
git clone https://github.com/dex-dev/dex.git
cd dex
pip install -e .
```

## Development Installation

For development with all test dependencies:

```bash
git clone https://github.com/dex-dev/dex.git
cd dex
pip install -e ".[dev]"
```

## Verify Installation

```bash
dex version
```

This should display the installed Dex version.

## Shell Completion

Dex is built with Typer, which supports shell completion. To enable:

**Bash:**
```bash
dex --install-completion bash
```

**Zsh:**
```bash
dex --install-completion zsh
```

**Fish:**
```bash
dex --install-completion fish
```

## Upgrading

```bash
pip install --upgrade dex
```

Or with uv:

```bash
uv pip install --upgrade dex
```

## Uninstalling

```bash
pip uninstall dex
```
