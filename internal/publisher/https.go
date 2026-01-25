package publisher

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/launchcg/dex/internal/errors"
)

// HTTPSPublisher provides manual upload instructions for HTTPS registries.
// HTTPS registries are read-only from dex's perspective.
type HTTPSPublisher struct {
	url string
}

// NewHTTPSPublisher creates a publisher for an HTTPS registry.
// This publisher only provides manual instructions since HTTPS registries
// are typically read-only.
func NewHTTPSPublisher(url string) (*HTTPSPublisher, error) {
	return &HTTPSPublisher{
		url: strings.TrimSuffix(url, "/"),
	}, nil
}

// Protocol returns "https".
func (p *HTTPSPublisher) Protocol() string {
	return "https"
}

// Publish returns manual instructions for uploading to an HTTPS registry.
func (p *HTTPSPublisher) Publish(tarballPath string) (*PublishResult, error) {
	// Parse tarball to get name and version
	info, err := ParseTarball(tarballPath)
	if err != nil {
		return nil, errors.NewPublishError("", p.url, "validate", err)
	}

	// Compute integrity hash
	integrity, err := ComputeTarballHash(tarballPath)
	if err != nil {
		return nil, errors.NewPublishError(info.Name, p.url, "validate", err)
	}

	filename := filepath.Base(tarballPath)
	tarballURL := p.url + "/" + filename

	instructions := fmt.Sprintf(`HTTPS registries require manual upload. Please follow these steps:

1. Upload the tarball to your registry:
   Target URL: %s

2. Update your registry.json to include the new version:
   {
     "packages": {
       "%s": {
         "versions": [..., "%s"],
         "latest": "%s"
       }
     }
   }

3. Tarball details:
   - File: %s
   - Integrity: %s

Note: The exact upload method depends on your registry hosting.
Common methods include:
  - scp/rsync for static file servers
  - Git commit for GitHub Pages
  - Web UI upload for CDN services
`, tarballURL, info.Name, info.Version, info.Version, filename, integrity)

	return &PublishResult{
		Name:               info.Name,
		Version:            info.Version,
		URL:                tarballURL,
		Integrity:          integrity,
		ManualInstructions: instructions,
	}, nil
}
