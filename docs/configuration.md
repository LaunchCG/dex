# Configuration

Dex uses HCL (HashiCorp Configuration Language) for configuration files:

- `dex.hcl` - Project configuration
- `dex.lock` - Lock file for deterministic installations

## Project Configuration (dex.hcl)

The `dex.hcl` file defines your project settings, registries, and packages.

```hcl
project {
  name             = "my-project"
  default_platform = "claude-code"
}

registry "internal" {
  path = "/path/to/internal-packages"
}

registry "community" {
  url = "https://packages.example.com"
}

package "python-tools" {
  registry = "community"
  version  = "^1.0.0"

  config = {
    python_version = "3.12"
    test_framework = "pytest"
  }
}

package "custom-package" {
  source  = "git+https://github.com/user/custom-package.git"
  version = "v2.0.0"
}
```

### Project Block

| Attribute | Required | Description |
|-----------|----------|-------------|
| `name` | no | Project name (defaults to directory name) |
| `default_platform` | yes | Target AI platform |

### Supported Platforms

| Platform | Value |
|----------|-------|
| Claude Code | `claude-code` |
| Cursor | `cursor` |
| GitHub Copilot | `github-copilot` |

### Registry Block

Registries define where to fetch packages from.

| Attribute | Required | Description |
|-----------|----------|-------------|
| `name` | yes | Registry identifier (block label) |
| `path` | conditional | Local filesystem path |
| `url` | conditional | Remote URL |

**Supported registry formats:**

| Format | Example | Description |
|--------|---------|-------------|
| Local | `path = "/path/to/packages"` | Local filesystem directory |
| HTTPS | `url = "https://registry.example.com"` | Remote HTTP registry |
| Git | `source = "git+https://github.com/org/repo.git"` | Git repository (in package block) |
| S3 | `url = "s3://bucket/prefix"` | AWS S3 bucket |
| Azure | `url = "az://container/prefix"` | Azure Blob Storage |

### Package Block

| Attribute | Required | Description |
|-----------|----------|-------------|
| `name` | yes | Package identifier (block label) |
| `source` | conditional | Direct source URL |
| `registry` | conditional | Registry name to fetch from |
| `version` | no | Version constraint |
| `config` | no | Package configuration values |

**Package source options:**

```hcl
# From a named registry
package "my-package" {
  registry = "community"
  version  = "^1.0.0"
}

# From a git repository
package "my-package" {
  source  = "git+https://github.com/owner/repo.git"
  version = "v1.0.0"
}

# From local filesystem
package "my-package" {
  source = "file:///path/to/package"
}
```

## Lock File (dex.lock)

The lock file ensures deterministic installations by recording exact versions and checksums:

```json
{
  "version": "1.0",
  "platform": "claude-code",
  "packages": {
    "python-tools": {
      "version": "1.2.0",
      "resolved": "https://registry.example.com/python-tools-1.2.0.tar.gz",
      "integrity": "sha512-abc123...",
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
- Developing a package locally
- Testing different versions

### Updating Locked Versions

To update all packages to their latest compatible versions:

```bash
dex sync
```

To update a specific package:

```bash
dex sync package-name@latest
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

### Caret Ranges (^)

Allows changes that do not modify the left-most non-zero digit:

- `^1.2.3` → `>=1.2.3 <2.0.0`
- `^0.2.3` → `>=0.2.3 <0.3.0`
- `^0.0.3` → `>=0.0.3 <0.0.4`

### Tilde Ranges (~)

Allows patch-level changes:

- `~1.2.3` → `>=1.2.3 <1.3.0`
- `~0.2.3` → `>=0.2.3 <0.3.0`

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
| `.dex/manifest.json` | Tracks installed files |
| `.dex/cache/` | Package cache (gitignored) |
