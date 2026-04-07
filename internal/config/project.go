package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/launchcg/dex/internal/resource"
)

// ResourceSet holds all universal resource slices. When adding a new resource type,
// update this struct and its methods (copyFrom, appendFrom, mergeFrom, buildResources).
type ResourceSet struct {
	Skills     []resource.Skill     `hcl:"skill,block"`
	Commands   []resource.Command   `hcl:"command,block"`
	Agents     []resource.Agent     `hcl:"agent,block"`
	Rules      []resource.Rule      `hcl:"rule,block"`
	RulesFiles []resource.Rules     `hcl:"rules,block"`
	Settings   []resource.Settings  `hcl:"settings,block"`
	MCPServers []resource.MCPServer `hcl:"mcp_server,block"`
}

// copyFrom replaces all resource fields with those from src.
func (r *ResourceSet) copyFrom(src *ResourceSet) {
	*r = *src
}

// appendFrom appends all resource slices from src into r.
func (r *ResourceSet) appendFrom(src *ResourceSet) {
	r.Skills = append(r.Skills, src.Skills...)
	r.Commands = append(r.Commands, src.Commands...)
	r.Agents = append(r.Agents, src.Agents...)
	r.Rules = append(r.Rules, src.Rules...)
	r.RulesFiles = append(r.RulesFiles, src.RulesFiles...)
	r.Settings = append(r.Settings, src.Settings...)
	r.MCPServers = append(r.MCPServers, src.MCPServers...)
}

// mergeFrom performs additive merge: same-name resources are replaced, new ones appended.
func (r *ResourceSet) mergeFrom(src *ResourceSet) {
	r.Skills = mergeByName(r.Skills, src.Skills, func(s resource.Skill) string { return s.Name })
	r.Commands = mergeByName(r.Commands, src.Commands, func(c resource.Command) string { return c.Name })
	r.Agents = mergeByName(r.Agents, src.Agents, func(a resource.Agent) string { return a.Name })
	r.Rules = mergeByName(r.Rules, src.Rules, func(v resource.Rule) string { return v.Name })
	r.RulesFiles = mergeByName(r.RulesFiles, src.RulesFiles, func(v resource.Rules) string { return v.Name })
	r.Settings = mergeByName(r.Settings, src.Settings, func(s resource.Settings) string { return s.Name })
	r.MCPServers = mergeByName(r.MCPServers, src.MCPServers, func(m resource.MCPServer) string { return m.Name })
}

// buildResources returns a unified Resource slice from the typed fields.
func (r *ResourceSet) buildResources() []resource.Resource {
	var res []resource.Resource
	for i := range r.Skills {
		res = append(res, &r.Skills[i])
	}
	for i := range r.Commands {
		res = append(res, &r.Commands[i])
	}
	for i := range r.Agents {
		res = append(res, &r.Agents[i])
	}
	for i := range r.Rules {
		res = append(res, &r.Rules[i])
	}
	for i := range r.RulesFiles {
		res = append(res, &r.RulesFiles[i])
	}
	for i := range r.Settings {
		res = append(res, &r.Settings[i])
	}
	for i := range r.MCPServers {
		res = append(res, &r.MCPServers[i])
	}
	return res
}

// ProjectConfig represents the dex.hcl file structure.
// This is the main configuration file for a dex-managed project.
//
// Note: gohcl does not decode into embedded struct fields, so resource fields must be
// declared directly here (and in LocalConfig and ProfileBlock). The ResourceSet type
// and its methods (copyFrom, mergeFrom, appendFrom, buildResources) centralize the
// field-iteration logic so that adding a new resource type only requires updating
// ResourceSet and the toResourceSet/applyResourceSet methods.
type ProjectConfig struct {
	// Project contains project metadata
	Project ProjectBlock `hcl:"project,block"`

	// Profiles defines named configuration variants
	Profiles []ProfileBlock `hcl:"profile,block"`

	// Registries defines package registry sources
	Registries []RegistryBlock `hcl:"registry,block"`

	// Packages defines package dependencies
	Packages []PackageBlock `hcl:"package,block"`

	// Universal resource types
	Skills     []resource.Skill     `hcl:"skill,block"`
	Commands   []resource.Command   `hcl:"command,block"`
	Agents     []resource.Agent     `hcl:"agent,block"`
	Rules      []resource.Rule      `hcl:"rule,block"`
	RulesFiles []resource.Rules     `hcl:"rules,block"`
	Settings   []resource.Settings  `hcl:"settings,block"`
	MCPServers []resource.MCPServer `hcl:"mcp_server,block"`

	// Resources is a unified view of all resources (populated after parsing)
	Resources []resource.Resource

	// Variables defines user-configurable variables for the project
	// Populated from first-pass extraction, not from HCL decode
	Variables []ProjectVariableBlock

	// ResolvedVars contains the resolved variable values (populated after parsing, not from HCL)
	ResolvedVars map[string]string
}

// ProjectBlock contains project metadata defined in the project {} block.
type ProjectBlock struct {
	// Name is the project name
	Name string `hcl:"name,attr"`

	// AgenticPlatform specifies the target AI agent platform (e.g., "claude-code", "cursor")
	AgenticPlatform string `hcl:"default_platform,attr"`

	// NamespaceAll enables namespacing for all installed packages
	NamespaceAll bool `hcl:"namespace_all,optional"`

	// NamespacePackages lists specific packages to namespace
	NamespacePackages []string `hcl:"namespace_packages,optional"`

	// AgentInstructions contains project-level instructions that appear at the top
	// of agent files (CLAUDE.md, AGENTS.md, copilot-instructions.md) before any
	// package-contributed content. This content is owned by the project, not packages.
	AgentInstructions string `hcl:"agent_instructions,optional"`

	// GitExclude controls whether dex sync automatically updates .git/info/exclude
	// to locally hide dex-managed files from git without modifying .gitignore
	GitExclude bool `hcl:"git_exclude,optional"`
}

// RegistryBlock defines a package registry source.
// Registries can be local (file://) or remote (https://).
type RegistryBlock struct {
	// Name is the unique identifier for this registry
	Name string `hcl:"name,label"`

	// Path is the local filesystem path for file:// registries
	Path string `hcl:"path,optional"`

	// URL is the remote URL for https:// registries
	URL string `hcl:"url,optional"`
}

// PackageBlock defines a package dependency.
// Packages can be sourced directly (git+https://, file://) or from a registry.
type PackageBlock struct {
	// Name is the unique identifier for this package dependency
	Name string `hcl:"name,label"`

	// Source is a direct source URL (git+https://, file://)
	Source string `hcl:"source,optional"`

	// Version is the version constraint for the package
	Version string `hcl:"version,optional"`

	// Registry is the name of the registry to fetch the package from
	Registry string `hcl:"registry,optional"`

	// Config provides package-specific configuration values
	Config map[string]string `hcl:"config,optional"`
}

// ProjectVariableBlock defines a user-configurable variable for the project.
// Variables can source values from environment variables with fallback defaults.
type ProjectVariableBlock struct {
	// Name is the variable identifier used in var.NAME references
	Name string `hcl:"name,label"`

	// Description explains what this variable controls
	Description string `hcl:"description,optional"`

	// Default is the default value if not provided by environment
	Default string `hcl:"default,optional"`

	// Env specifies an environment variable to read the value from
	Env string `hcl:"env,optional"`

	// Required indicates whether a value must be available
	Required bool `hcl:"required,optional"`
}

// ProfileBlock defines a named configuration variant.
// Profiles allow switching between different sets of packages, registries, and resources
// using `dex sync --profile <name>`. By default, profile contents are merged additively
// with the default config. Set exclude_defaults = true to start clean.
type ProfileBlock struct {
	// Name is the profile identifier used in --profile flag
	Name string `hcl:"name,label"`

	// ExcludeDefaults when true means only profile-defined items are used
	ExcludeDefaults bool `hcl:"exclude_defaults,optional"`

	// AgentInstructions overrides the project-level agent instructions
	AgentInstructions string `hcl:"agent_instructions,optional"`

	// Registries defines profile-specific package registry sources
	Registries []RegistryBlock `hcl:"registry,block"`

	// Packages defines profile-specific package dependencies
	Packages []PackageBlock `hcl:"package,block"`

	// Universal resource types
	Skills     []resource.Skill     `hcl:"skill,block"`
	Commands   []resource.Command   `hcl:"command,block"`
	Agents     []resource.Agent     `hcl:"agent,block"`
	Rules      []resource.Rule      `hcl:"rule,block"`
	RulesFiles []resource.Rules     `hcl:"rules,block"`
	Settings   []resource.Settings  `hcl:"settings,block"`
	MCPServers []resource.MCPServer `hcl:"mcp_server,block"`
}

// toResourceSet extracts the resource fields into a ResourceSet.
func (pb *ProfileBlock) toResourceSet() ResourceSet {
	return ResourceSet{
		Skills: pb.Skills, Commands: pb.Commands, Agents: pb.Agents,
		Rules: pb.Rules, RulesFiles: pb.RulesFiles, Settings: pb.Settings,
		MCPServers: pb.MCPServers,
	}
}

// mergeByName merges overrides into defaults. If an override has the same name
// as a default (determined by getName), the override replaces that default.
// New overrides are appended.
func mergeByName[T any](defaults, overrides []T, getName func(T) string) []T {
	if len(overrides) == 0 {
		return defaults
	}

	// Build index of override names for quick lookup
	overrideNames := make(map[string]int, len(overrides))
	for i, o := range overrides {
		overrideNames[getName(o)] = i
	}

	// Replace matching defaults, track which overrides were used
	used := make(map[int]bool)
	result := make([]T, 0, len(defaults)+len(overrides))
	for _, d := range defaults {
		if idx, ok := overrideNames[getName(d)]; ok {
			result = append(result, overrides[idx])
			used[idx] = true
		} else {
			result = append(result, d)
		}
	}

	// Append overrides that didn't replace a default
	for i, o := range overrides {
		if !used[i] {
			result = append(result, o)
		}
	}

	return result
}

// ApplyProfile applies the named profile to this ProjectConfig.
// By default, profile contents are merged additively with defaults (same-name items
// are replaced). With exclude_defaults = true, defaults are cleared first.
// Returns an error if the profile is not found or if duplicate profile names exist.
func (p *ProjectConfig) ApplyProfile(name string) error {
	var profile *ProfileBlock
	seen := make(map[string]bool, len(p.Profiles))
	for i := range p.Profiles {
		if seen[p.Profiles[i].Name] {
			return fmt.Errorf("duplicate profile name: %s", p.Profiles[i].Name)
		}
		seen[p.Profiles[i].Name] = true
		if p.Profiles[i].Name == name {
			profile = &p.Profiles[i]
		}
	}

	if profile == nil {
		available := make([]string, 0, len(p.Profiles))
		for i := range p.Profiles {
			available = append(available, p.Profiles[i].Name)
		}
		return fmt.Errorf("profile %q not found; available profiles: %s", name, strings.Join(available, ", "))
	}

	profResources := profile.toResourceSet()

	// Registries are always merged (never wiped by exclude_defaults) since
	// profile packages may reference global registries
	p.Registries = mergeByName(p.Registries, profile.Registries, func(r RegistryBlock) string { return r.Name })

	if profile.ExcludeDefaults {
		p.Packages = profile.Packages
		p.Project.AgentInstructions = profile.AgentInstructions
		p.applyResourceSet(&profResources)
	} else {
		p.Packages = mergeByName(p.Packages, profile.Packages, func(pl PackageBlock) string { return pl.Name })
		if profile.AgentInstructions != "" {
			p.Project.AgentInstructions = profile.AgentInstructions
		}
		rs := p.toResourceSet()
		rs.mergeFrom(&profResources)
		p.applyResourceSet(&rs)
	}

	// Clear profiles to prevent double-apply
	p.Profiles = nil

	return nil
}

// LoadProjectWithProfile loads a dex.hcl file and optionally applies a named profile.
// If profile is empty, it behaves identically to LoadProject.
func LoadProjectWithProfile(dir string, profile string) (*ProjectConfig, error) {
	cfg, err := LoadProject(dir)
	if err != nil {
		return nil, err
	}

	if profile != "" {
		if err := cfg.ApplyProfile(profile); err != nil {
			return nil, err
		}
		cfg.buildResources()
	}

	return cfg, nil
}

// LoadProject loads a dex.hcl file from the given directory.
// It parses the HCL file, evaluates expressions, and returns the configuration.
// Uses two-pass parsing to first extract and resolve variables, then decode the full config.
func LoadProject(dir string) (*ProjectConfig, error) {
	filename := filepath.Join(dir, "dex.hcl")

	parser := NewParser()
	file, diags := parser.ParseFile(filename)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse %s: %s", filename, diags.Error())
	}

	// Pass 1: Extract and resolve variable blocks
	// remain contains the body with variable blocks removed
	variables, resolvedVars, remain, err := extractAndResolveProjectVariables(file.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve variables in %s: %w", filename, err)
	}

	// Pass 2: Decode the remaining body with resolved vars in the eval context
	ctx := NewProjectEvalContext(dir, resolvedVars)
	var config ProjectConfig
	diags = DecodeBody(remain, ctx, &config)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to decode %s: %s", filename, diags.Error())
	}

	// Store resolved variables for later use (populated from pass 1)
	config.Variables = variables
	config.ResolvedVars = resolvedVars

	// Build unified Resources slice from typed fields
	config.buildResources()

	return &config, nil
}

// toResourceSet extracts the resource fields into a ResourceSet.
func (p *ProjectConfig) toResourceSet() ResourceSet {
	return ResourceSet{
		Skills: p.Skills, Commands: p.Commands, Agents: p.Agents,
		Rules: p.Rules, RulesFiles: p.RulesFiles, Settings: p.Settings,
		MCPServers: p.MCPServers,
	}
}

// applyResourceSet writes the ResourceSet fields back into ProjectConfig.
func (p *ProjectConfig) applyResourceSet(r *ResourceSet) {
	p.Skills = r.Skills
	p.Commands = r.Commands
	p.Agents = r.Agents
	p.Rules = r.Rules
	p.RulesFiles = r.RulesFiles
	p.Settings = r.Settings
	p.MCPServers = r.MCPServers
}

// buildResources populates the Resources slice from the typed resource fields.
func (p *ProjectConfig) buildResources() {
	rs := p.toResourceSet()
	p.Resources = rs.buildResources()
}

// toLocalConfig extracts the resource slices from this ProjectConfig into a LocalConfig
// so that MergeLocal can delegate to LocalConfig.merge (the single merge source of truth).
// ResolvedVars is intentionally omitted: MergeLocal handles var precedence separately.
func (p *ProjectConfig) toLocalConfig() *LocalConfig {
	rs := p.toResourceSet()
	lc := &LocalConfig{
		Registries: p.Registries,
		Packages:   p.Packages,
		Variables:  p.Variables,
	}
	lc.applyResourceSet(&rs)
	return lc
}

// applyLocalConfig writes the merged resource slices from l back into this ProjectConfig.
// Used by MergeLocal after delegating to LocalConfig.merge.
// ResolvedVars is intentionally omitted: MergeLocal handles var precedence separately.
// IMPORTANT: this leaves p.Resources stale. Callers must call p.buildResources() afterward.
// applyLocalConfig is unexported to enforce this — MergeLocal is the only call site.
func (p *ProjectConfig) applyLocalConfig(l *LocalConfig) {
	p.Registries = l.Registries
	p.Packages = l.Packages
	rs := l.toResourceSet()
	p.applyResourceSet(&rs)
	p.Variables = l.Variables
}

// MergeLocal appends all resources from a LocalConfig into this ProjectConfig and
// rebuilds the unified Resources slice. Resource slice merging is delegated to
// LocalConfig.merge so the merge logic lives in one place. Project-defined resolved
// vars take precedence over local config vars.
func (p *ProjectConfig) MergeLocal(local *LocalConfig) {
	base := p.toLocalConfig()
	base.merge(local)
	p.applyLocalConfig(base)

	// Var precedence at the project level: skip-if-exists (project wins).
	// This is asymmetric with LocalConfig.merge, which uses last-writer-wins within
	// local configs. The full chain is intentional:
	//   project vars > per-project local vars > global local vars
	// merge() above has already collapsed global+per-project into local using
	// last-writer-wins, so here we only copy vars not already set by the project.
	if p.ResolvedVars == nil {
		p.ResolvedVars = make(map[string]string)
	}
	for k, v := range local.ResolvedVars {
		if _, exists := p.ResolvedVars[k]; !exists {
			p.ResolvedVars[k] = v
		}
	}

	p.buildResources()
}

// Validate checks the project config for errors.
// It ensures required fields are present and values are valid.
func (p *ProjectConfig) Validate() error {
	// Validate project block
	// Name is optional - will default to directory name if not specified
	if p.Project.AgenticPlatform == "" {
		return fmt.Errorf("project.default_platform is required")
	}

	// Validate profiles
	profileNames := make(map[string]bool)
	for _, prof := range p.Profiles {
		if prof.Name == "" {
			return fmt.Errorf("profile name is required")
		}
		if profileNames[prof.Name] {
			return fmt.Errorf("duplicate profile name: %s", prof.Name)
		}
		profileNames[prof.Name] = true
	}

	// Validate variables
	varNames := make(map[string]bool)
	for _, v := range p.Variables {
		if v.Name == "" {
			return fmt.Errorf("variable name is required")
		}
		if varNames[v.Name] {
			return fmt.Errorf("duplicate variable name: %s", v.Name)
		}
		varNames[v.Name] = true

		// Required variables should not have defaults
		if v.Required && v.Default != "" {
			return fmt.Errorf("variable %q is marked required but has a default value", v.Name)
		}
	}

	// Validate registries
	registryNames := make(map[string]bool)
	for _, reg := range p.Registries {
		if reg.Name == "" {
			return fmt.Errorf("registry name is required")
		}
		if registryNames[reg.Name] {
			return fmt.Errorf("duplicate registry name: %s", reg.Name)
		}
		registryNames[reg.Name] = true

		// Must have either path or URL, but not both
		if reg.Path == "" && reg.URL == "" {
			return fmt.Errorf("registry %q must have either path or url", reg.Name)
		}
		if reg.Path != "" && reg.URL != "" {
			return fmt.Errorf("registry %q cannot have both path and url", reg.Name)
		}
	}

	// Validate packages
	packageNames := make(map[string]bool)
	for _, pkg := range p.Packages {
		if pkg.Name == "" {
			return fmt.Errorf("package name is required")
		}
		if packageNames[pkg.Name] {
			return fmt.Errorf("duplicate package name: %s", pkg.Name)
		}
		packageNames[pkg.Name] = true

		// Cannot have both source and registry
		if pkg.Source != "" && pkg.Registry != "" {
			return fmt.Errorf("package %q cannot have both source and registry", pkg.Name)
		}

		// If using registry, it must exist
		if pkg.Registry != "" && !registryNames[pkg.Registry] {
			return fmt.Errorf("package %q references unknown registry: %s", pkg.Name, pkg.Registry)
		}
	}

	return nil
}

// AddPackage adds a package block with a source to the dex.hcl file.
// It appends the package block to the end of the file.
func AddPackage(dir string, name string, source string, version string) error {
	return AddPackageToConfig(dir, name, source, "", version)
}

// AddPackageToConfig adds a package block to the dex.hcl file.
// Supports both source-based and registry-based packages.
// Exactly one of source or registryName must be non-empty.
func AddPackageToConfig(dir, name, source, registryName, version string) error {
	if source == "" && registryName == "" {
		return fmt.Errorf("either source or registry must be specified for package %q", name)
	}

	filename := filepath.Join(dir, "dex.hcl")

	// Read existing content
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", filename, err)
	}

	// Check if package already exists
	existingConfig, err := LoadProject(dir)
	if err == nil {
		for _, p := range existingConfig.Packages {
			if p.Name == name {
				// Package already exists, skip
				return nil
			}
		}
	}

	// Build the package block
	var packageBlock string
	if source != "" {
		if version != "" {
			packageBlock = fmt.Sprintf("\npackage %q {\n  source  = %q\n  version = %q\n}\n", name, source, version)
		} else {
			packageBlock = fmt.Sprintf("\npackage %q {\n  source = %q\n}\n", name, source)
		}
	} else {
		if version != "" {
			packageBlock = fmt.Sprintf("\npackage %q {\n  registry = %q\n  version  = %q\n}\n", name, registryName, version)
		} else {
			packageBlock = fmt.Sprintf("\npackage %q {\n  registry = %q\n}\n", name, registryName)
		}
	}

	// Append to file
	newContent := string(content) + packageBlock
	if err := os.WriteFile(filename, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", filename, err)
	}

	return nil
}

// AddRegistry adds a registry block to the dex.hcl file.
// It appends the registry block to the end of the file.
// Exactly one of url or path must be provided.
func AddRegistry(dir string, name string, url string, path string, force bool) error {
	// Validate exactly one of url or path is provided
	if url == "" && path == "" {
		return fmt.Errorf("exactly one of --url or --local must be provided")
	}
	if url != "" && path != "" {
		return fmt.Errorf("cannot specify both --url and --local")
	}

	filename := filepath.Join(dir, "dex.hcl")

	// Read existing content
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", filename, err)
	}

	// Check if registry already exists
	existingConfig, err := LoadProject(dir)
	if err == nil {
		for _, r := range existingConfig.Registries {
			if r.Name == name {
				if !force {
					return fmt.Errorf("registry %q already exists; use --force to overwrite", name)
				}
				// Remove the existing registry block using regex
				re := regexp.MustCompile(`(?m)\n?registry\s+"` + regexp.QuoteMeta(name) + `"\s*\{[^}]*\}\n?`)
				content = re.ReplaceAll(content, []byte(""))
				break
			}
		}
	}

	// Build the registry block
	var registryBlock string
	if url != "" {
		registryBlock = fmt.Sprintf("\nregistry %q {\n  url = %q\n}\n", name, url)
	} else {
		registryBlock = fmt.Sprintf("\nregistry %q {\n  path = %q\n}\n", name, path)
	}

	// Append to file
	newContent := string(content) + registryBlock
	if err := os.WriteFile(filename, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", filename, err)
	}

	return nil
}
