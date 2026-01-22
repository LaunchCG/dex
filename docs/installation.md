# Installation

## Prerequisites

- Go 1.21 or higher
- Git

## Install from Source

```bash
git clone https://github.com/launchcg/dex.git
cd dex
make build
```

The binary is created at `bin/dex`.

## Install to PATH (Optional)

```bash
# Install to $GOPATH/bin
make install

# Or install to ~/.bin
make install-user
```

## Verify Installation

```bash
dex version
```

This should display the installed Dex version.

## Shell Completion

Dex uses Cobra, which supports shell completion:

**Bash:**
```bash
dex completion bash > /etc/bash_completion.d/dex
# Or for user-level:
dex completion bash > ~/.bash_completion.d/dex
```

**Zsh:**
```bash
dex completion zsh > "${fpath[1]}/_dex"
```

**Fish:**
```bash
dex completion fish > ~/.config/fish/completions/dex.fish
```

**PowerShell:**
```powershell
dex completion powershell > dex.ps1
```

## Upgrading

Pull the latest changes and rebuild:

```bash
cd dex
git pull
make build
```

## Uninstalling

Remove the binary:

```bash
rm $(which dex)
# Or if installed via make install:
rm $GOPATH/bin/dex
```
