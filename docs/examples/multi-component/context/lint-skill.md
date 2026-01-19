# Lint Skill

This skill provides code linting capabilities.

## Usage

Use this skill when you need to check code for style issues, potential bugs, or best practice violations.

## Instructions

1. Identify the programming language of the code
2. Apply appropriate linting rules
3. Report issues with line numbers and descriptions
4. Suggest fixes when possible

## Configuration

The linting configuration is stored in `settings.json`:

- `rules`: Which rules to enable/disable
- `severity`: Warning vs error thresholds
- `ignore`: Patterns to ignore

## Examples

```
Lint issue at line 42: Unused variable 'temp'
Lint issue at line 58: Missing type annotation for parameter 'data'
```
