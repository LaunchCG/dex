package resource

// PlatformOverride is the generic override for platforms with no extra fields.
// Used when a platform needs only disabled/content override capability.
type PlatformOverride struct {
	Disabled bool   `hcl:"disabled,optional"`
	Content  string `hcl:"content,optional"`
}
