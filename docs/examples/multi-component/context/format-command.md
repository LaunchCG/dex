# Format Command

This command formats code according to project standards.

## Usage

Invoke with `/format` to format the current file or selection.

## Instructions

1. Detect the file type
2. Apply the appropriate formatter
3. Preserve semantic meaning while improving style
4. Report any formatting changes made

## Supported Languages

- Python (Black style)
- JavaScript/TypeScript (Prettier style)
- JSON (2-space indent)
- YAML (2-space indent)
- Markdown (wrap at 80 columns)

## Options

- `--check`: Only check if formatting is needed, don't modify
- `--diff`: Show what would change
