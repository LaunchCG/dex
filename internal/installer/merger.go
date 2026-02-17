package installer

import (
	"fmt"
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
// Non-server keys (e.g., "inputs") are also merged: arrays are deduplicated by "id" field.
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
	overlayServers, hasKey := overlay[key].(map[string]any)
	if !hasKey {
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

	// Handle non-server keys (e.g., "inputs") when overlay has the proper structure
	if hasKey {
		for k, v := range overlay {
			if k == key {
				continue
			}
			if overlayArr, ok := v.([]any); ok {
				if baseArr, ok := base[k].([]any); ok {
					base[k] = mergeInputArrays(baseArr, overlayArr)
				} else {
					base[k] = overlayArr
				}
			} else {
				base[k] = v
			}
		}
	}

	return base
}

// mergeInputArrays merges two arrays of objects, deduplicating by "id" field.
// When both arrays contain an object with the same "id", the overlay version wins.
func mergeInputArrays(base, overlay []any) []any {
	seen := make(map[string]int) // id -> index in result
	result := make([]any, 0, len(base)+len(overlay))

	for _, item := range base {
		if m, ok := item.(map[string]any); ok {
			if id, ok := m["id"].(string); ok {
				seen[id] = len(result)
			}
		}
		result = append(result, item)
	}

	for _, item := range overlay {
		if m, ok := item.(map[string]any); ok {
			if id, ok := m["id"].(string); ok {
				if idx, exists := seen[id]; exists {
					result[idx] = item // replace existing
					continue
				}
				seen[id] = len(result)
			}
		}
		result = append(result, item)
	}

	return result
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
