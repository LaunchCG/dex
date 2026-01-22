# Suggested Commands

## Testing
```bash
go test ./...           # Run all tests
go test ./internal/...  # Run internal package tests
go test -v ./...        # Run with verbose output
go test -cover ./...    # Run with coverage
```

## Building
```bash
go build -o dex ./cmd/dex  # Build the CLI binary
```

## Running the CLI
```bash
./dex --help               # Show help
./dex init                 # Initialize a new project
./dex install              # Install plugins
./dex uninstall <plugin>   # Uninstall a plugin
./dex list                 # List installed plugins
```

## Linting
```bash
go fmt ./...              # Format code
go vet ./...              # Run vet
```
