# Configuration

Dex uses two configuration files:

- `sdlc.json` - Project configuration
- `sdlc.lock` - Lock file for deterministic installations

## sdlc.json Format

```json
{
  "agent": "claude-code",
  "project_name": "my-project",
  "plugins": {
    "plugin-a": "^1.0.0",
    "plugin-b": {
      "version": "~2.0.0",
      "registry": "custom"
    },
    "local-plugin": {
      "source": "file:./plugins/local"
    }
  },
  "registries": {
    "default": "file:./registry",
    "custom": "file:/path/to/custom/registry"
  },
  "default_registry": "default"
}
```

### Fields

#### `agent` (required)

The target AI agent platform. Currently supported:
- `claude-code` - Anthropic's Claude Code CLI
- `cursor` - Cursor IDE (planned)
- `codex` - OpenAI Codex (planned)
- `antigravity` - Custom agents

#### `project_name` (optional)

Human-readable project name. Defaults to the directory name. Used in template rendering.

#### `plugins`

Dictionary of plugin specifications. Keys are plugin names, values can be:

**Version string:**
```json
"plugin-name": "^1.0.0"
```

**Plugin spec object:**
```json
"plugin-name": {
  "version": "^1.0.0",
  "registry": "custom"
}
```

**Direct source:**
```json
"plugin-name": {
  "source": "file:./local-plugin"
}
```

#### `registries`

Named registry URLs. Supported formats:
- `file:///absolute/path` - Absolute path
- `file:./relative/path` - Relative path
- `file:../sibling/path` - Parent-relative path

#### `default_registry`

Name of the default registry to use when a plugin doesn't specify one.

## Registry Configuration

A registry is a directory containing:

```
registry/
├── registry.json           # Package index
├── plugin-a-1.0.0.tar.gz  # Plugin tarballs
├── plugin-a-1.1.0.tar.gz
└── plugin-b-2.0.0.tar.gz
```

### registry.json Format

```json
{
  "packages": {
    "plugin-a": {
      "versions": ["1.0.0", "1.1.0"],
      "latest": "1.1.0"
    },
    "plugin-b": {
      "versions": ["2.0.0"],
      "latest": "2.0.0"
    }
  }
}
```

## Lock File (sdlc.lock)

The lock file ensures deterministic installations:

```json
{
  "version": "1.0",
  "agent": "claude-code",
  "plugins": {
    "plugin-a": {
      "version": "1.1.0",
      "resolved": "file:///path/to/registry/plugin-a-1.1.0.tar.gz",
      "integrity": "sha512-...",
      "dependencies": {}
    }
  }
}
```

### When to Commit the Lock File

**Do commit** when:
- Working on a team project
- Deploying to production
- Ensuring reproducible builds

**Don't commit** when:
- Developing a plugin locally
- Testing different versions

### Updating Locked Versions

To update all plugins to their latest compatible versions:

```bash
rm sdlc.lock
dex install
```

To update a specific plugin:

```bash
dex install plugin-name@latest
```

## Version Specifiers

Dex supports standard semver version specifiers:

| Specifier | Meaning |
|-----------|---------|
| `1.0.0` | Exact version |
| `^1.0.0` | Compatible with 1.x.x (>=1.0.0 <2.0.0) |
| `~1.0.0` | Patch-level changes (~1.0.x) |
| `>=1.0.0` | Greater than or equal |
| `<2.0.0` | Less than |
| `latest` | Latest available version |

### Caret Ranges

- `^1.2.3` → `>=1.2.3 <2.0.0`
- `^0.2.3` → `>=0.2.3 <0.3.0`
- `^0.0.3` → `>=0.0.3 <0.0.4`

### Tilde Ranges

- `~1.2.3` → `>=1.2.3 <1.3.0`
- `~0.2.3` → `>=0.2.3 <0.3.0`
