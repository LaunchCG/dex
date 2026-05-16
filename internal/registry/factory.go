package registry

import (
	"fmt"
)

// NewRegistry creates a registry from a URL and mode.
// It parses the URL to determine the protocol and creates the appropriate
// registry implementation.
//
// Supported protocols:
//   - file:// - Local filesystem (LocalRegistry)
//   - git+https://, git+ssh:// - Git repositories (GitRegistry)
//   - https://, http:// - HTTP/HTTPS servers (HTTPSRegistry)
//   - s3:// - Amazon S3 (S3Registry)
//   - az:// - Azure Blob Storage (AzureRegistry)
//
// The mode parameter controls how the source is interpreted:
//   - ModeAuto: Auto-detect based on presence of registry.json
//   - ModeRegistry: Expect registry.json index
//   - ModePackage: Expect single package with package.hcl
//
// For backend-specific overrides (e.g. S3 region), use NewRegistryWithConfig.
func NewRegistry(url string, mode SourceMode) (Registry, error) {
	return NewRegistryWithConfig(url, mode, nil)
}

// NewRegistryWithConfig is like NewRegistry but accepts backend-specific
// settings declared in the project's registry block (e.g. {"region":
// "us-west-2"} for S3). Only the S3 and Azure backends consult cfg.
func NewRegistryWithConfig(url string, mode SourceMode, cfg map[string]string) (Registry, error) {
	protocol, path, err := ParseSource(url)
	if err != nil {
		return nil, err
	}

	switch protocol {
	case "file":
		return NewLocalRegistry(path, mode)

	case "git":
		// Reconstruct the full git URL (git+ prefix was stripped by ParseSource)
		return NewGitRegistry("git+"+path, mode)

	case "https", "http":
		// path is the full URL for HTTP(S)
		return NewHTTPSRegistry(path, mode)

	case "s3":
		// Reconstruct the full s3:// URL
		return NewS3Registry("s3://"+path, mode, cfg)

	case "az":
		// Reconstruct the full az:// URL
		return NewAzureRegistry("az://"+path, mode, cfg)

	default:
		return nil, fmt.Errorf("unsupported protocol: %s", protocol)
	}
}

// MustNewRegistry creates a registry from a URL and mode, panicking on error.
// This is useful for initialization code where errors are not expected.
func MustNewRegistry(url string, mode SourceMode) Registry {
	reg, err := NewRegistry(url, mode)
	if err != nil {
		panic(fmt.Sprintf("failed to create registry from %s: %v", url, err))
	}
	return reg
}
