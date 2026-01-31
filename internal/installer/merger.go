package installer

import (
	"fmt"
	"regexp"
	"strings"
)

// MergeJSON merges two JSON objects.
// The 'overlay' values take precedence for simple values.
// Arrays and maps are recursively merged.
func MergeJSON(base, overlay map[string]any) map[string]any {
	if base == nil {
		base = make(map[string]any)
	}
	if overlay == nil {
		return base
	}

	result := make(map[string]any)

	// Copy base values
	for k, v := range base {
		result[k] = v
	}

	// Merge overlay values
	for k, v := range overlay {
		if existingVal, exists := result[k]; exists {
			// Handle different merge strategies based on type
			switch ev := existingVal.(type) {
			case map[string]any:
				if ov, ok := v.(map[string]any); ok {
					// Recursively merge maps
					result[k] = MergeJSON(ev, ov)
					continue
				}
			case []any:
				if ov, ok := v.([]any); ok {
					// Merge arrays (append unique values)
					result[k] = MergeJSONArrays(ev, ov)
					continue
				}
			}
		}
		// For other cases or non-matching types, overlay takes precedence
		result[k] = v
	}

	return result
}

// MergeJSONArrays merges JSON arrays by appending unique values.
// Uses string comparison for primitives.
func MergeJSONArrays(base, overlay []any) []any {
	if base == nil {
		return overlay
	}
	if overlay == nil {
		return base
	}

	// Build a set of existing values for deduplication
	seen := make(map[string]bool)
	result := make([]any, 0, len(base)+len(overlay))

	for _, v := range base {
		key := fmt.Sprintf("%v", v)
		if !seen[key] {
			seen[key] = true
			result = append(result, v)
		}
	}

	for _, v := range overlay {
		key := fmt.Sprintf("%v", v)
		if !seen[key] {
			seen[key] = true
			result = append(result, v)
		}
	}

	return result
}

// MergeMCPServers merges MCP server configurations.
// The format is: {"mcpServers": {"name": {...}}}
// Servers from overlay replace servers with the same name in base.
func MergeMCPServers(base, overlay map[string]any) map[string]any {
	return MergeMCPServersWithKey(base, overlay, "mcpServers")
}

// MergeMCPServersWithKey merges MCP server configurations using a custom key.
// The format is: {key: {"name": {...}}} where key is "mcpServers" (Claude) or "servers" (Copilot).
// Servers from overlay replace servers with the same name in base.
func MergeMCPServersWithKey(base, overlay map[string]any, key string) map[string]any {
	if base == nil {
		base = make(map[string]any)
	}
	if overlay == nil {
		return base
	}

	// Get or create the servers map in base
	baseServers, ok := base[key].(map[string]any)
	if !ok {
		baseServers = make(map[string]any)
	}

	// Get the servers map from overlay
	overlayServers, ok := overlay[key].(map[string]any)
	if !ok {
		// Check if overlay itself contains server configs directly
		overlayServers = overlay
	}

	// Merge servers
	for name, config := range overlayServers {
		if name == key {
			// Handle nested key
			if servers, ok := config.(map[string]any); ok {
				for n, c := range servers {
					baseServers[n] = c
				}
			}
		} else {
			baseServers[name] = config
		}
	}

	base[key] = baseServers
	return base
}

// RemoveMCPServers removes specified servers from the MCP config.
func RemoveMCPServers(config map[string]any, names []string) map[string]any {
	if config == nil {
		return config
	}

	servers, ok := config["mcpServers"].(map[string]any)
	if !ok {
		return config
	}

	for _, name := range names {
		delete(servers, name)
	}

	config["mcpServers"] = servers
	return config
}

// MergeAgentContent merges plugin content into CLAUDE.md using markers.
// Uses markers: <!-- dex:{plugin} --> ... <!-- /dex:{plugin} -->
func MergeAgentContent(existing, pluginName, content string) string {
	startMarker := fmt.Sprintf("<!-- dex:%s -->", pluginName)
	endMarker := fmt.Sprintf("<!-- /dex:%s -->", pluginName)
	markedContent := fmt.Sprintf("%s\n%s\n%s", startMarker, content, endMarker)

	// Check if markers already exist using regex
	pattern := regexp.MustCompile(
		fmt.Sprintf(`(?s)<!-- dex:%s -->.*?<!-- /dex:%s -->`,
			regexp.QuoteMeta(pluginName),
			regexp.QuoteMeta(pluginName)),
	)

	if pattern.MatchString(existing) {
		// Replace existing marked section
		return pattern.ReplaceAllString(existing, markedContent)
	}

	// Append new section
	if existing == "" {
		return markedContent
	}

	// Ensure proper spacing
	if !strings.HasSuffix(existing, "\n") {
		existing += "\n"
	}

	return existing + "\n" + markedContent
}

// RemoveAgentContent removes a plugin's content from CLAUDE.md.
func RemoveAgentContent(existing, pluginName string) string {
	// Match the entire section including markers
	pattern := regexp.MustCompile(
		fmt.Sprintf(`(?s)\n*<!-- dex:%s -->.*?<!-- /dex:%s -->\n*`,
			regexp.QuoteMeta(pluginName),
			regexp.QuoteMeta(pluginName)),
	)

	result := pattern.ReplaceAllString(existing, "\n")

	// Clean up multiple consecutive newlines
	result = strings.TrimSpace(result)
	if result != "" {
		result += "\n"
	}

	return result
}

// MergeSettingsArrays merges settings arrays (allow, ask, deny) by appending unique values.
func MergeSettingsArrays(base, overlay []string) []string {
	if base == nil {
		return overlay
	}
	if overlay == nil {
		return base
	}

	seen := make(map[string]bool)
	result := make([]string, 0, len(base)+len(overlay))

	for _, v := range base {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}

	for _, v := range overlay {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}

	return result
}

// MergeEnvMaps merges environment variable maps.
// Overlay values take precedence.
func MergeEnvMaps(base, overlay map[string]string) map[string]string {
	if base == nil {
		return overlay
	}
	if overlay == nil {
		return base
	}

	result := make(map[string]string)
	for k, v := range base {
		result[k] = v
	}
	for k, v := range overlay {
		result[k] = v
	}

	return result
}

// MergeProjectAgentContent merges project-level instructions with plugin-contributed sections.
// Project instructions appear at the top without markers.
// Plugin sections are marked with <!-- dex:plugin-name --> ... <!-- /dex:plugin-name -->
// Returns the merged content with project instructions first, then all plugin sections.
func MergeProjectAgentContent(existing, projectInstructions string) string {
	// Extract all plugin sections (anything between <!-- dex:* --> markers)
	// Use a simpler pattern that matches any plugin section
	markerPattern := regexp.MustCompile(`(?s)<!-- dex:[^>]+ -->.*?<!-- /dex:[^>]+ -->`)
	pluginSections := markerPattern.FindAllString(existing, -1)

	// Build the new content: project instructions + plugin sections
	var result strings.Builder

	// Add project instructions (if any)
	if projectInstructions != "" {
		result.WriteString(strings.TrimSpace(projectInstructions))
	}

	// Add plugin sections
	for _, section := range pluginSections {
		if result.Len() > 0 {
			result.WriteString("\n\n")
		}
		result.WriteString(section)
	}

	return result.String()
}
