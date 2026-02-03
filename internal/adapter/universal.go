package adapter

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/launchcg/dex/internal/config"
	"github.com/launchcg/dex/internal/resource"
	"github.com/launchcg/dex/internal/template"
)

// PlanUniversalFile creates an installation plan for a universal file resource.
// This is shared across all adapters since file resources work the same way on all platforms.
// Files can have inline content or reference a source file, with optional templating.
func PlanUniversalFile(fileRes *resource.File, pkg *config.PackageConfig, pluginDir, projectRoot, platform string, ctx *InstallContext) (*Plan, error) {
	plan := NewPlan(pkg.Package.Name)

	// Load content from inline or source file
	var content string
	if fileRes.Content != nil {
		content = *fileRes.Content
	} else if fileRes.Src != nil {
		// Read from source file
		srcPath := filepath.Join(pluginDir, *fileRes.Src)
		data, err := os.ReadFile(srcPath)
		if err != nil {
			return nil, fmt.Errorf("reading source file %s: %w", *fileRes.Src, err)
		}
		content = string(data)
	}

	// Apply templating if enabled
	if fileRes.Template {
		// Create template context with the appropriate platform
		templateCtx := template.NewContext(pkg.Package.Name, pkg.Package.Version, projectRoot, platform)
		engine := template.NewEngine(pluginDir, templateCtx)

		// Convert vars to map[string]any
		vars := make(map[string]any)
		for k, v := range fileRes.Vars {
			vars[k] = v
		}

		// Render template
		var err error
		if fileRes.Src != nil {
			// Render from file with vars
			content, err = engine.RenderFileWithVars(*fileRes.Src, vars)
		} else {
			// Render from string with vars
			content, err = engine.RenderWithVars(content, vars)
		}
		if err != nil {
			return nil, fmt.Errorf("rendering template: %w", err)
		}
	}

	// Determine destination path with optional namespacing
	destPath := fileRes.Dest
	if ctx != nil && ctx.Namespace {
		// Apply namespacing to the filename (last component)
		dir := filepath.Dir(destPath)
		base := filepath.Base(destPath)
		namespacedBase := fmt.Sprintf("%s-%s", pkg.Package.Name, base)
		if dir == "." {
			destPath = namespacedBase
		} else {
			destPath = filepath.Join(dir, namespacedBase)
		}
	}

	// Ensure parent directory exists
	parentDir := filepath.Dir(destPath)
	if parentDir != "." {
		plan.AddDirectory(parentDir, true)
	}

	// Add file to plan
	plan.AddFile(destPath, content, fileRes.Chmod)

	return plan, nil
}

// PlanUniversalDirectory creates an installation plan for a universal directory resource.
// This is shared across all adapters since directory resources work the same way on all platforms.
func PlanUniversalDirectory(dirRes *resource.Directory, pkg *config.PackageConfig, ctx *InstallContext) (*Plan, error) {
	plan := NewPlan(pkg.Package.Name)

	// Determine path with optional namespacing
	dirPath := dirRes.Path
	if ctx != nil && ctx.Namespace {
		// Apply namespacing to the directory name (last component)
		parent := filepath.Dir(dirPath)
		base := filepath.Base(dirPath)
		namespacedBase := fmt.Sprintf("%s-%s", pkg.Package.Name, base)
		if parent == "." {
			dirPath = namespacedBase
		} else {
			dirPath = filepath.Join(parent, namespacedBase)
		}
	}

	// Add directory to plan with Parents flag
	plan.AddDirectory(dirPath, dirRes.Parents)

	return plan, nil
}
