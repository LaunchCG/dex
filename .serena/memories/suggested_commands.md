# Suggested Commands

## MCP Runbook Tools (use these, NOT shell commands)
- `mcp__runbook__run_test` - Run all tests
- `mcp__runbook__run_test-cover` - Run tests with coverage
- `mcp__runbook__run_build` - Build the CLI binary
- `mcp__runbook__run_lint` - Run linter (fmt + vet)
- `mcp__runbook__run_fmt` - Format code
- `mcp__runbook__run_vet` - Run go vet
- `mcp__runbook__run_clean` - Clean build artifacts
- `mcp__runbook__run_install` - Install to GOPATH/bin
- `mcp__runbook__run_install-user` - Install to ~/.bin

Tasks config: `.runbook/tasks.yaml`
Prompts: `ci`, `fix-test-failures` (in `.runbook/prompts.yaml`)

## Running the CLI
```bash
./dex --help               # Show help
./dex sync                 # Sync plugins (install/update/prune)
./dex uninstall <plugin>   # Uninstall a plugin
./dex list                 # List installed plugins
```
