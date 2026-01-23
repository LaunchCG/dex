// git.go provides a Git repository registry for fetching packages from git sources.

package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"

	"github.com/dex-tools/dex/internal/errors"
	"github.com/dex-tools/dex/pkg/version"
)

// GitRef represents a parsed Git reference.
type GitRef struct {
	Type  string // "tag", "branch", "commit", or "default"
	Value string // The ref value (empty for "default")
}

// GitRegistry handles git+https:// and git+ssh:// sources.
// Authentication is handled externally via:
//   - HTTPS: Git credential helpers (configured via git config)
//   - SSH: SSH agent or ~/.ssh keys
type GitRegistry struct {
	repoURL string     // The actual git URL (without git+ prefix)
	ref     GitRef     // Branch, tag, or commit (from URL fragment)
	mode    SourceMode // Registry or package mode
	cache   *Cache     // Cache for cloned repositories
}

// NewGitRegistry creates a registry from a git URL.
//
// URL formats:
//   - git+https://github.com/user/repo.git
//   - git+https://github.com/user/repo.git#v1.0.0
//   - git+https://github.com/user/repo.git#tag=v1.0.0
//   - git+https://github.com/user/repo.git#branch=main
//   - git+ssh://git@github.com/user/repo.git#v1.0.0
//
// Authentication is handled externally:
//   - HTTPS: Configure git credential helpers
//   - SSH: Use ssh-agent or ~/.ssh/config
func NewGitRegistry(url string, mode SourceMode) (*GitRegistry, error) {
	repoURL, ref, err := parseGitURL(url)
	if err != nil {
		return nil, err
	}

	// Set up cache directory
	defaultCache, err := DefaultCache()
	if err != nil {
		return nil, errors.NewRegistryError(url, "connect", err)
	}
	cache := NewCache(defaultCache.GetCacheDir("git"))

	return &GitRegistry{
		repoURL: repoURL,
		ref:     ref,
		mode:    mode,
		cache:   cache,
	}, nil
}

// Protocol returns "git".
func (r *GitRegistry) Protocol() string {
	return "git"
}

// Mode returns the source mode (registry or package).
func (r *GitRegistry) Mode() SourceMode {
	return r.mode
}

// RepoURL returns the Git repository URL.
func (r *GitRegistry) RepoURL() string {
	return r.repoURL
}

// Ref returns the Git reference.
func (r *GitRegistry) Ref() GitRef {
	return r.ref
}

// GetPackageInfo returns package info by cloning/fetching the repo.
// For git repos, we list tags as versions.
func (r *GitRegistry) GetPackageInfo(name string) (*PackageInfo, error) {
	// Clone the repo to get package.json or registry.json
	clonePath, err := r.cloneToCache()
	if err != nil {
		return nil, errors.NewRegistryError(r.repoURL, "clone", err)
	}

	if r.mode == ModeRegistry {
		return r.getPackageFromRegistry(clonePath, name)
	}
	return r.getPackageFromManifest(clonePath, name)
}

// getPackageFromRegistry extracts package info from registry.json.
func (r *GitRegistry) getPackageFromRegistry(clonePath, name string) (*PackageInfo, error) {
	registryFile := filepath.Join(clonePath, "registry.json")
	data, err := os.ReadFile(registryFile)
	if err != nil {
		return nil, errors.NewNotFoundError("registry.json", r.repoURL)
	}

	var index RegistryIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, errors.NewRegistryError(r.repoURL, "fetch",
			fmt.Errorf("failed to parse registry.json: %w", err))
	}

	entry, ok := index.Packages[name]
	if !ok {
		return nil, errors.NewNotFoundError("package", name)
	}

	return &PackageInfo{
		Name:     name,
		Versions: entry.Versions,
		Latest:   entry.Latest,
	}, nil
}

// getPackageFromManifest extracts package info from package.json.
func (r *GitRegistry) getPackageFromManifest(clonePath, name string) (*PackageInfo, error) {
	packageFile := filepath.Join(clonePath, "package.json")
	data, err := os.ReadFile(packageFile)
	if err != nil {
		return nil, errors.NewNotFoundError("package.json", r.repoURL)
	}

	// Parse package.json for metadata
	var pkg struct {
		Name        string `json:"name"`
		Version     string `json:"version"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, errors.NewRegistryError(r.repoURL, "fetch",
			fmt.Errorf("failed to parse package.json: %w", err))
	}

	// Verify name matches if provided
	if name != "" && pkg.Name != "" && !NamesMatch(name, pkg.Name) {
		return nil, errors.NewNotFoundError("package", name)
	}

	// Use name from package.json, fall back to provided name
	pkgName := pkg.Name
	if pkgName == "" {
		pkgName = name
	}

	// Get tags as versions
	tags, err := r.listTags()
	if err != nil {
		// Non-fatal: just use manifest version
		tags = nil
	}

	// Filter to semver-like tags
	var versions []string
	for _, tag := range tags {
		// Strip 'v' prefix for normalization
		v := strings.TrimPrefix(tag, "v")
		if _, err := version.Parse(v); err == nil {
			versions = append(versions, v)
		}
	}

	// Sort versions
	parsedVersions := version.SortStrings(versions)
	versions = make([]string, len(parsedVersions))
	for i, v := range parsedVersions {
		versions[i] = v.String()
	}

	// If no tags found, use version from package.json
	if len(versions) == 0 && pkg.Version != "" {
		versions = []string{pkg.Version}
	}

	latest := ""
	if len(versions) > 0 {
		latest = versions[len(versions)-1]
	}

	return &PackageInfo{
		Name:        pkgName,
		Versions:    versions,
		Latest:      latest,
		Description: pkg.Description,
	}, nil
}

// ResolvePackage resolves a version to a specific commit.
func (r *GitRegistry) ResolvePackage(name, versionSpec string) (*ResolvedPackage, error) {
	info, err := r.GetPackageInfo(name)
	if err != nil {
		return nil, err
	}

	// Determine resolved version
	var resolvedVersion string
	if versionSpec == "latest" || versionSpec == "" {
		resolvedVersion = info.Latest
	} else {
		// Parse constraint and find best match
		constraint, err := version.ParseConstraint(versionSpec)
		if err != nil {
			return nil, errors.NewVersionError(name, versionSpec, info.Versions, "invalid version constraint")
		}

		var parsedVersions []*version.Version
		for _, v := range info.Versions {
			if parsed, err := version.Parse(v); err == nil {
				parsedVersions = append(parsedVersions, parsed)
			}
		}

		best := constraint.FindBest(parsedVersions)
		if best == nil {
			return nil, errors.NewVersionError(name, versionSpec, info.Versions, "")
		}
		resolvedVersion = best.String()
	}

	// Build resolved URL with appropriate ref
	tags, _ := r.listTags()
	var ref string
	if contains(tags, "v"+resolvedVersion) {
		ref = "v" + resolvedVersion
	} else if contains(tags, resolvedVersion) {
		ref = resolvedVersion
	} else if r.ref.Value != "" {
		ref = r.ref.Value
	}

	resolvedURL := "git+" + r.repoURL
	if ref != "" {
		resolvedURL += "#tag=" + ref
	}

	return &ResolvedPackage{
		Name:    name,
		Version: resolvedVersion,
		URL:     resolvedURL,
	}, nil
}

// FetchPackage clones/copies the repo to destDir.
// Uses shallow clone (depth 1) for efficiency.
// Authentication is handled externally via git credential helpers or SSH agent.
func (r *GitRegistry) FetchPackage(resolved *ResolvedPackage, destDir string) (string, error) {
	// Parse the resolved URL to get the ref
	_, ref, err := parseGitURL(resolved.URL)
	if err != nil {
		return "", errors.NewRegistryError(resolved.URL, "parse", err)
	}

	// Check cache first
	cacheKey := r.getCacheKeyForRef(ref)
	if r.cache.Has(cacheKey) {
		cachedPath := r.cache.GetPath(cacheKey)
		// Copy from cache to destination
		pluginDir := filepath.Join(destDir, resolved.Name)
		if err := os.RemoveAll(pluginDir); err != nil && !os.IsNotExist(err) {
			return "", fmt.Errorf("failed to remove existing directory: %w", err)
		}
		if err := copyDir(cachedPath, pluginDir); err != nil {
			return "", fmt.Errorf("failed to copy from cache: %w", err)
		}
		return pluginDir, nil
	}

	// Clone to destination
	pluginDir := filepath.Join(destDir, resolved.Name)
	if err := os.RemoveAll(pluginDir); err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to remove existing directory: %w", err)
	}

	cloneOpts := &git.CloneOptions{
		URL:   r.repoURL,
		Depth: 1,
	}

	// Set reference if specified
	if ref.Value != "" {
		switch ref.Type {
		case "tag":
			cloneOpts.ReferenceName = plumbing.NewTagReferenceName(ref.Value)
		case "branch":
			cloneOpts.ReferenceName = plumbing.NewBranchReferenceName(ref.Value)
		}
	}

	// Clone the repository
	// Authentication is handled externally via:
	// - HTTPS: Git credential helpers
	// - SSH: SSH agent or ~/.ssh keys
	_, err = git.PlainClone(pluginDir, false, cloneOpts)
	if err != nil {
		return "", errors.NewRegistryError(r.repoURL, "clone", err)
	}

	// Remove .git directory to reduce size
	gitDir := filepath.Join(pluginDir, ".git")
	if err := os.RemoveAll(gitDir); err != nil {
		return "", fmt.Errorf("failed to remove .git directory: %w", err)
	}

	// Cache for future use by copying to cache directory
	cachePath := r.cache.GetPath(cacheKey)
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}
	if err := copyDir(pluginDir, cachePath); err != nil {
		return "", fmt.Errorf("failed to copy to cache: %w", err)
	}

	return pluginDir, nil
}

// ListPackages returns nil (git repos don't support listing without cloning).
func (r *GitRegistry) ListPackages() ([]string, error) {
	if r.mode == ModeRegistry {
		clonePath, err := r.cloneToCache()
		if err != nil {
			return nil, err
		}

		// Read registry.json and list packages
		indexPath := filepath.Join(clonePath, "registry.json")
		data, err := os.ReadFile(indexPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read registry.json: %w", err)
		}

		var index RegistryIndex
		if err := json.Unmarshal(data, &index); err != nil {
			return nil, fmt.Errorf("failed to parse registry.json: %w", err)
		}

		packages := make([]string, 0, len(index.Packages))
		for name := range index.Packages {
			packages = append(packages, name)
		}
		sort.Strings(packages)
		return packages, nil
	}

	// For package mode, return nil (can't list without knowing the name)
	return nil, nil
}

// parseGitURL parses the URL and extracts the ref from fragment.
func parseGitURL(url string) (repoURL string, ref GitRef, err error) {
	if !strings.HasPrefix(url, "git+") {
		return "", GitRef{}, fmt.Errorf("invalid git URL: must start with 'git+': %s", url)
	}

	// Remove git+ prefix
	gitURL := url[4:]

	ref = GitRef{Type: "default", Value: ""}

	// Split URL and fragment
	if idx := strings.Index(gitURL, "#"); idx != -1 {
		repoURL = gitURL[:idx]
		fragment := gitURL[idx+1:]

		if strings.Contains(fragment, "=") {
			// Explicit ref type: tag=v1.0.0 or branch=main
			parts := strings.SplitN(fragment, "=", 2)
			refType := parts[0]
			refValue := parts[1]

			switch refType {
			case "tag", "branch", "commit":
				ref = GitRef{Type: refType, Value: refValue}
			default:
				return "", GitRef{}, fmt.Errorf("invalid ref type: %s (must be tag, branch, or commit)", refType)
			}
		} else {
			// Implicit ref (assume tag)
			ref = GitRef{Type: "tag", Value: fragment}
		}
	} else {
		repoURL = gitURL
	}

	// Validate URL scheme
	if !strings.HasPrefix(repoURL, "https://") &&
		!strings.HasPrefix(repoURL, "ssh://") &&
		!strings.HasPrefix(repoURL, "git@") {
		return "", GitRef{}, fmt.Errorf("invalid git URL scheme: must be https://, ssh://, or git@: %s", repoURL)
	}

	return repoURL, ref, nil
}

// getCacheKey returns a unique cache key for this repo+ref.
func (r *GitRegistry) getCacheKey() string {
	return r.getCacheKeyForRef(r.ref)
}

// getCacheKeyForRef returns a unique cache key for this repo with a specific ref.
func (r *GitRegistry) getCacheKeyForRef(ref GitRef) string {
	refStr := "HEAD"
	if ref.Value != "" {
		refStr = ref.Type + "=" + ref.Value
	}
	return r.repoURL + "#" + refStr
}

// cloneToCache clones the repo to cache directory.
// Authentication is handled externally via git credential helpers or SSH agent.
func (r *GitRegistry) cloneToCache() (string, error) {
	cacheKey := r.getCacheKey()

	// Check cache
	if r.cache.Has(cacheKey) {
		return r.cache.GetPath(cacheKey), nil
	}

	// Clone to temp directory
	tempDir, err := os.MkdirTemp("", "dex-git-clone-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	cloneDest := filepath.Join(tempDir, "repo")

	cloneOpts := &git.CloneOptions{
		URL:   r.repoURL,
		Depth: 1,
	}

	// Set reference if specified
	if r.ref.Value != "" {
		switch r.ref.Type {
		case "tag":
			cloneOpts.ReferenceName = plumbing.NewTagReferenceName(r.ref.Value)
		case "branch":
			cloneOpts.ReferenceName = plumbing.NewBranchReferenceName(r.ref.Value)
		}
	}

	// Clone the repository
	// Authentication is handled externally via:
	// - HTTPS: Git credential helpers
	// - SSH: SSH agent or ~/.ssh keys
	_, err = git.PlainClone(cloneDest, false, cloneOpts)
	if err != nil {
		return "", fmt.Errorf("failed to clone repository: %w", err)
	}

	// Remove .git directory to reduce size
	gitDir := filepath.Join(cloneDest, ".git")
	if err := os.RemoveAll(gitDir); err != nil {
		return "", fmt.Errorf("failed to remove .git directory: %w", err)
	}

	// Cache the clone
	cachePath := r.cache.GetPath(cacheKey)
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	if err := copyDir(cloneDest, cachePath); err != nil {
		return "", fmt.Errorf("failed to copy to cache: %w", err)
	}

	return cachePath, nil
}

// listTags returns all tags in the repository.
// Uses ls-remote which doesn't require authentication for public repos.
// For private repos, authentication is handled externally.
func (r *GitRegistry) listTags() ([]string, error) {
	// Use git ls-remote to list tags without cloning
	rem := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
		Name: "origin",
		URLs: []string{r.repoURL},
	})

	refs, err := rem.List(&git.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list remote refs: %w", err)
	}

	var tags []string
	for _, ref := range refs {
		if ref.Name().IsTag() {
			tagName := ref.Name().Short()
			tags = append(tags, tagName)
		}
	}

	return tags, nil
}

// copyDir copies a directory recursively.
func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile copies a single file.
func copyFile(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, data, srcInfo.Mode())
}

// contains checks if a string slice contains a value.
func contains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}
