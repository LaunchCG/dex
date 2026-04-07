# CLI Reference

Complete reference for Dex command-line commands.

## Global Options

```
--help, -h           Show help message
--verbose, -v        Increase verbosity (-v info, -vv debug, -vvv trace)
```

**Verbosity levels:**

| Level | Flag | Description |
|-------|------|-------------|
| Default | (none) | Only warnings and errors |
| Info | `-v` | Progress information |
| Debug | `-vv` | Detailed debug info with timestamps |
| Trace | `-vvv` | Full trace with file paths |

**Examples:**

```bash
# Show detailed installation progress
dex -v install my-package

# Full debug output
dex -vv install my-package
```

## Commands

### dex init

Initialize a new Dex project.

```bash
dex init [OPTIONS]
```

**Options:**

| Option | Short | Description | Default |
|--------|-------|-------------|---------|
| `--agent` | `-a` | Target AI agent platform | `claude-code` |
| `--name` | `-n` | Project name | Directory name |
| `--path` | `-p` | Project directory | Current directory |

**Examples:**

```bash
# Initialize in current directory
dex init

# Initialize for Claude Code with custom name
dex init --agent claude-code --name my-project

# Initialize in a specific directory
dex init --path /path/to/project
```

**Exit Codes:**

- `0` - Success
- `1` - Project already initialized or invalid options

---

### dex sync

Synchronize packages to match dex.hcl configuration.

Without arguments, syncs all packages: installs missing, updates outdated, and prunes orphaned packages. With arguments, installs or updates specific packages.

```bash
dex sync [PACKAGES...] [OPTIONS]
```

**Arguments:**

| Argument | Description |
|----------|-------------|
| `PACKAGES` | Optional package names/specs (e.g., `package@^1.0.0`) |

**Version specifiers:**

| Format | Example | Description |
|--------|---------|-------------|
| Exact | `package@1.2.0` | Exact version |
| Caret | `package@^1.0.0` | Compatible with 1.x.x |
| Tilde | `package@~1.2.0` | Compatible with 1.2.x |
| Range | `package@>=1.0.0` | 1.0.0 or higher |
| Latest | `package` | Latest version |

**Options:**

| Option | Short | Description | Default |
|--------|-------|-------------|---------|
| `--source` | `-s` | Direct source path (file://) | - |
| `--registry` | `-r` | Registry to use (overrides default) | - |
| `--no-save` | - | Don't save to config file (packages are saved by default) | `false` |
| `--no-lock` | - | Don't update lock file | `false` |
| `--force` | `-f` | Overwrite existing files even if not managed by dex | `false` |
| `--path` | `-p` | Project directory | Current directory |
| `--namespace` | - | Namespace resources with package name | `false` |
| `--dry-run` | `-n` | Show what would change without making changes | `false` |
| `--git-exclude` | - | Update `.git/info/exclude` to locally hide dex-managed files from git | `false` |

**File Conflict Protection:**

By default, Dex will refuse to overwrite files that exist but aren't tracked in the manifest. This prevents accidental loss of user modifications. Use `--force` to override this protection.

**Examples:**

```bash
# Sync all packages (install missing, update outdated, prune orphaned)
dex sync

# Preview what sync would do
dex sync --dry-run

# Install a specific package
dex sync my-package

# Install with version specifier
dex sync my-package@^1.0.0

# Install from a specific registry
dex sync my-package --registry file:./my-registry

# Install from local source
dex sync --source file:./local-package

# Install without saving to dex.hcl
dex sync my-package --no-save

# Install without updating lock file
dex sync --no-lock

# Force overwrite existing files
dex sync my-package --force
```

**Exit Codes:**

- `0` - All operations successful
- `1` - One or more operations failed (including file conflicts)

---

### dex list

List installed packages.

```bash
dex list [OPTIONS]
```

**Options:**

| Option | Short | Description | Default |
|--------|-------|-------------|---------|
| `--tree` | `-t` | Show dependency tree | `false` |
| `--path` | `-p` | Project directory | Current directory |

**Examples:**

```bash
# List all packages
dex list

# Show dependency tree
dex list --tree
```

**Exit Codes:**

- `0` - Success

---

### dex uninstall

Uninstall packages from the project.

```bash
dex uninstall PACKAGES... [OPTIONS]
```

**Arguments:**

| Argument | Description |
|----------|-------------|
| `PACKAGES` | Package names to uninstall (required) |

**Options:**

| Option | Short | Description | Default |
|--------|-------|-------------|---------|
| `--remove` | `-r` | Also remove from dex.hcl (drop the dependency) | `false` |
| `--path` | `-p` | Project directory | Current directory |

**Behavior:**

Without `--remove`:
- Deletes installed files tracked in the manifest
- Cleans up MCP servers
- Keeps the package in dex.hcl (can reinstall with `dex sync`)

With `--remove`:
- Does all of the above
- Also removes the package from dex.hcl

**Examples:**

```bash
# Uninstall a package (keep in dex.hcl for later reinstall)
dex uninstall my-package

# Uninstall and remove from dex.hcl
dex uninstall my-package --remove

# Uninstall multiple packages
dex uninstall package-a package-b --remove
```

**Exit Codes:**

- `0` - Success

---

### dex info

Show information about an installed package.

```bash
dex info PACKAGE [OPTIONS]
```

**Arguments:**

| Argument | Description |
|----------|-------------|
| `PACKAGE` | Package name (required) |

**Options:**

| Option | Short | Description | Default |
|--------|-------|-------------|---------|
| `--path` | `-p` | Project directory | Current directory |

**Examples:**

```bash
# Show package info
dex info my-package
```

**Output includes:**

- Version
- Source or registry
- Lock file information (if locked)
- Dependencies

**Exit Codes:**

- `0` - Success
- `1` - Package not found

---

### dex version

Show the Dex version.

```bash
dex version
```

**Example:**

```bash
$ dex version
dex 0.1.0
```

**Exit Codes:**

- `0` - Success

---

### dex manifest

Show files managed by Dex.

```bash
dex manifest [PACKAGE] [OPTIONS]
```

**Arguments:**

| Argument | Description |
|----------|-------------|
| `PACKAGE` | Optional package name to filter by |

**Options:**

| Option | Short | Description | Default |
|--------|-------|-------------|---------|
| `--path` | `-p` | Project directory | Current directory |

**Examples:**

```bash
# Show all managed files
dex manifest

# Show files for a specific package
dex manifest my-package
```

**Exit Codes:**

- `0` - Success

---

---

## Environment Variables

| Variable | Description |
|----------|-------------|
| `DEX_DEBUG` | Enable debug output |

## File Locations

| File | Description |
|------|-------------|
| `dex.hcl` | Project configuration |
| `dex.lock` | Lock file |
| `.dex/` | Dex internal directory |
| `.dex/manifest.json` | Manifest tracking installed files |
| `.dex/cache/` | Backup cache for rollback (gitignored) |
| `.claude/` | Claude Code configuration directory |
| `.mcp.json` | MCP server configuration (project root) |
| `.claude/skills/` | Installed skills directory |
