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
dex -v install my-plugin

# Full debug output
dex -vv install my-plugin
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

### dex install

Install plugins.

```bash
dex install [PLUGINS...] [OPTIONS]
```

**Arguments:**

| Argument | Description |
|----------|-------------|
| `PLUGINS` | Optional plugin names/specs (e.g., `plugin@^1.0.0`) |

**Version specifiers:**

| Format | Example | Description |
|--------|---------|-------------|
| Exact | `plugin@1.2.0` | Exact version |
| Caret | `plugin@^1.0.0` | Compatible with 1.x.x |
| Tilde | `plugin@~1.2.0` | Compatible with 1.2.x |
| Range | `plugin@>=1.0.0` | 1.0.0 or higher |
| Latest | `plugin` | Latest version |

**Options:**

| Option | Short | Description | Default |
|--------|-------|-------------|---------|
| `--source` | `-s` | Direct source path (file://) | - |
| `--registry` | `-r` | Registry to use (overrides default) | - |
| `--save` | `-S` | Save installed plugins to dex.hcl | `false` |
| `--no-lock` | - | Don't update lock file | `false` |
| `--force` | `-f` | Overwrite existing files even if not managed by dex | `false` |
| `--path` | `-p` | Project directory | Current directory |

**File Conflict Protection:**

By default, Dex will refuse to overwrite files that exist but aren't tracked in the manifest. This prevents accidental loss of user modifications. Use `--force` to override this protection.

**Examples:**

```bash
# Install all plugins from dex.hcl
dex install

# Install specific plugin
dex install my-plugin

# Install with version specifier
dex install my-plugin@^1.0.0

# Install and save to dex.hcl
dex install my-plugin@^1.0.0 --save

# Install from a specific registry
dex install my-plugin --registry file:./my-registry

# Install from local source
dex install --source file:./local-plugin

# Install without updating lock file
dex install --no-lock

# Force overwrite existing files
dex install my-plugin --force
```

**Exit Codes:**

- `0` - All installations successful
- `1` - One or more installations failed (including file conflicts)

---

### dex list

List installed plugins.

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
# List all plugins
dex list

# Show dependency tree
dex list --tree
```

**Exit Codes:**

- `0` - Success

---

### dex uninstall

Uninstall plugins from the project.

```bash
dex uninstall PLUGINS... [OPTIONS]
```

**Arguments:**

| Argument | Description |
|----------|-------------|
| `PLUGINS` | Plugin names to uninstall (required) |

**Options:**

| Option | Short | Description | Default |
|--------|-------|-------------|---------|
| `--remove` | `-r` | Also remove from dex.hcl (drop the dependency) | `false` |
| `--path` | `-p` | Project directory | Current directory |

**Behavior:**

Without `--remove`:
- Deletes installed files tracked in the manifest
- Cleans up MCP servers
- Keeps the plugin in dex.hcl (can reinstall with `dex install`)

With `--remove`:
- Does all of the above
- Also removes the plugin from dex.hcl

**Examples:**

```bash
# Uninstall a plugin (keep in dex.hcl for later reinstall)
dex uninstall my-plugin

# Uninstall and remove from dex.hcl
dex uninstall my-plugin --remove

# Uninstall multiple plugins
dex uninstall plugin-a plugin-b --remove
```

**Exit Codes:**

- `0` - Success

---

### dex info

Show information about an installed plugin.

```bash
dex info PLUGIN [OPTIONS]
```

**Arguments:**

| Argument | Description |
|----------|-------------|
| `PLUGIN` | Plugin name (required) |

**Options:**

| Option | Short | Description | Default |
|--------|-------|-------------|---------|
| `--path` | `-p` | Project directory | Current directory |

**Examples:**

```bash
# Show plugin info
dex info my-plugin
```

**Output includes:**

- Version
- Source or registry
- Lock file information (if locked)
- Dependencies

**Exit Codes:**

- `0` - Success
- `1` - Plugin not found

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
dex manifest [PLUGIN] [OPTIONS]
```

**Arguments:**

| Argument | Description |
|----------|-------------|
| `PLUGIN` | Optional plugin name to filter by |

**Options:**

| Option | Short | Description | Default |
|--------|-------|-------------|---------|
| `--path` | `-p` | Project directory | Current directory |

**Examples:**

```bash
# Show all managed files
dex manifest

# Show files for a specific plugin
dex manifest my-plugin
```

**Exit Codes:**

- `0` - Success

---

### dex update-ignore

Update .gitignore with Dex-managed files.

```bash
dex update-ignore [OPTIONS]
```

Adds or updates a managed section in .gitignore that excludes all directories and files tracked by Dex. The section is marked with header and footer comments so it can be automatically updated.

**Options:**

| Option | Short | Description | Default |
|--------|-------|-------------|---------|
| `--print` | - | Print what would be written without modifying files | `false` |
| `--path` | `-p` | Project directory | Current directory |

**Examples:**

```bash
# Update .gitignore
dex update-ignore

# Preview changes without modifying
dex update-ignore --print
```

**Managed Section Format:**

```gitignore
# === Dex managed files (do not edit this section) ===
.dex/
.claude/
# === End Dex managed files ===
```

**Exit Codes:**

- `0` - Success

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
