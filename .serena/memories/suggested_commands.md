# Suggested Commands

## Testing
```bash
./test.sh           # Run all tests
uv run pytest       # Run pytest directly
uv run pytest -k "test_name"  # Run specific test
```

## Linting
```bash
./lint.sh           # Run all linters
uv run ruff check . # Ruff lint check
uv run black . --check  # Black format check
uv run mypy .       # Type checking
```

## Formatting
```bash
uv run black .      # Format with black
uv run isort .      # Sort imports
```

## Running the CLI
```bash
uv run dex --help  # Run the CLI
```
