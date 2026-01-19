# Code Style Guidelines

## Imports
- Use absolute imports
- Group imports: stdlib, third-party, local
- Sort alphabetically within groups (isort handles this)

## Type Hints
- All public functions must have type hints
- Use `from __future__ import annotations` for forward references

## Testing
- **CRITICAL**: When testing generated files, ALWAYS assert the ENTIRE expected content, not substrings
- Use fixtures for common test data
- Test both success and error cases

## Code Quality
- Line length: 100 (configured in pyproject.toml)
- Use ruff, flake8, black, isort, mypy
- Strict mypy mode enabled

## Pydantic
- Use Pydantic v2 patterns (model_validate, model_dump)
- Use Field for default factories
