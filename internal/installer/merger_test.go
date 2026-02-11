package installer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeProjectAgentContent_EmptyFile(t *testing.T) {
	result := MergeProjectAgentContent("", "# Project Instructions\nUse TypeScript")

	expected := "# Project Instructions\nUse TypeScript"
	assert.Equal(t, expected, result)
}

func TestMergeProjectAgentContent_WithPluginSections(t *testing.T) {
	existing := `# Old project content

<!-- dex:plugin-a -->
Plugin A rules
<!-- /dex:plugin-a -->

<!-- dex:plugin-b -->
Plugin B rules
<!-- /dex:plugin-b -->`

	projectInstructions := "# New Project Instructions\nAlways use TypeScript"

	result := MergeProjectAgentContent(existing, projectInstructions)

	expected := `# New Project Instructions
Always use TypeScript

<!-- dex:plugin-a -->
Plugin A rules
<!-- /dex:plugin-a -->

<!-- dex:plugin-b -->
Plugin B rules
<!-- /dex:plugin-b -->`

	assert.Equal(t, expected, result)
}

func TestMergeProjectAgentContent_OnlyPlugins(t *testing.T) {
	existing := `<!-- dex:plugin-a -->
Plugin A rules
<!-- /dex:plugin-a -->

<!-- dex:plugin-b -->
Plugin B rules
<!-- /dex:plugin-b -->`

	projectInstructions := "# Project Instructions"

	result := MergeProjectAgentContent(existing, projectInstructions)

	expected := `# Project Instructions

<!-- dex:plugin-a -->
Plugin A rules
<!-- /dex:plugin-a -->

<!-- dex:plugin-b -->
Plugin B rules
<!-- /dex:plugin-b -->`

	assert.Equal(t, expected, result)
}

func TestMergeProjectAgentContent_RemoveProjectInstructions(t *testing.T) {
	existing := `# Old Project Content
This should be removed

<!-- dex:plugin-a -->
Plugin A rules
<!-- /dex:plugin-a -->`

	// Empty project instructions should remove the project section
	result := MergeProjectAgentContent(existing, "")

	expected := `<!-- dex:plugin-a -->
Plugin A rules
<!-- /dex:plugin-a -->`

	assert.Equal(t, expected, result)
}

func TestMergeProjectAgentContent_NoPlugins(t *testing.T) {
	existing := "# Old project content\nSome old stuff"

	projectInstructions := "# New Project Content\nNew stuff"

	result := MergeProjectAgentContent(existing, projectInstructions)

	expected := "# New Project Content\nNew stuff"
	assert.Equal(t, expected, result)
}

func TestMergeProjectAgentContent_MultilinePlugin(t *testing.T) {
	existing := `<!-- dex:my-plugin -->
Rule 1: Do this
Rule 2: Do that

Multiple lines here
<!-- /dex:my-plugin -->`

	projectInstructions := "# Project\nUse strict mode"

	result := MergeProjectAgentContent(existing, projectInstructions)

	expected := `# Project
Use strict mode

<!-- dex:my-plugin -->
Rule 1: Do this
Rule 2: Do that

Multiple lines here
<!-- /dex:my-plugin -->`

	assert.Equal(t, expected, result)
}

func TestMergeProjectAgentContent_MultipleUpdates(t *testing.T) {
	// Start with empty
	result := MergeProjectAgentContent("", "# V1")
	assert.Equal(t, "# V1", result)

	// Add a plugin
	result += "\n\n<!-- dex:plugin-a -->\nPlugin A\n<!-- /dex:plugin-a -->"

	// Update project instructions
	result = MergeProjectAgentContent(result, "# V2 Updated")

	expected := `# V2 Updated

<!-- dex:plugin-a -->
Plugin A
<!-- /dex:plugin-a -->`

	assert.Equal(t, expected, result)

	// Add another plugin
	result += "\n\n<!-- dex:plugin-b -->\nPlugin B\n<!-- /dex:plugin-b -->"

	// Update project instructions again
	result = MergeProjectAgentContent(result, "# V3 Final")

	expected = `# V3 Final

<!-- dex:plugin-a -->
Plugin A
<!-- /dex:plugin-a -->

<!-- dex:plugin-b -->
Plugin B
<!-- /dex:plugin-b -->`

	assert.Equal(t, expected, result)
}

func TestMergeProjectAgentContent_PreservesPluginOrder(t *testing.T) {
	existing := `Old project stuff

<!-- dex:first -->
First plugin
<!-- /dex:first -->

<!-- dex:second -->
Second plugin
<!-- /dex:second -->

<!-- dex:third -->
Third plugin
<!-- /dex:third -->`

	projectInstructions := "# New Project"

	result := MergeProjectAgentContent(existing, projectInstructions)

	expected := `# New Project

<!-- dex:first -->
First plugin
<!-- /dex:first -->

<!-- dex:second -->
Second plugin
<!-- /dex:second -->

<!-- dex:third -->
Third plugin
<!-- /dex:third -->`

	assert.Equal(t, expected, result)
}

func TestMergeProjectAgentContent_WhitespaceHandling(t *testing.T) {
	existing := `

<!-- dex:plugin -->
Content
<!-- /dex:plugin -->

   `

	projectInstructions := "  # Project  \n  Content  "

	result := MergeProjectAgentContent(existing, projectInstructions)

	// TrimSpace trims leading/trailing whitespace from the whole string, preserves internal whitespace
	expected := "# Project  \n  Content\n\n<!-- dex:plugin -->\nContent\n<!-- /dex:plugin -->"

	assert.Equal(t, expected, result)
}

func TestMergeProjectAgentContent_EmptyProjectAndNoPlugins(t *testing.T) {
	result := MergeProjectAgentContent("", "")
	assert.Equal(t, "", result)
}

func TestMergeMCPServersWithKey_PreservesInputs(t *testing.T) {
	base := map[string]any{
		"servers": map[string]any{
			"existing": map[string]any{"type": "stdio", "command": "cmd1"},
		},
		"inputs": []any{
			map[string]any{"id": "token", "type": "promptString", "description": "API Token"},
		},
	}

	overlay := map[string]any{
		"servers": map[string]any{
			"new-server": map[string]any{"type": "stdio", "command": "cmd2"},
		},
		"inputs": []any{
			map[string]any{"id": "org", "type": "promptString", "description": "Org Name"},
		},
	}

	result := MergeMCPServersWithKey(base, overlay, "servers")

	// Servers should be merged
	servers := result["servers"].(map[string]any)
	assert.Contains(t, servers, "existing")
	assert.Contains(t, servers, "new-server")

	// Inputs should be merged
	inputs := result["inputs"].([]any)
	assert.Len(t, inputs, 2)
	assert.Equal(t, "token", inputs[0].(map[string]any)["id"])
	assert.Equal(t, "org", inputs[1].(map[string]any)["id"])
}

func TestMergeMCPServersWithKey_DeduplicatesInputs(t *testing.T) {
	base := map[string]any{
		"servers": map[string]any{},
		"inputs": []any{
			map[string]any{"id": "shared", "type": "promptString", "description": "Old"},
			map[string]any{"id": "base_only", "type": "promptString", "description": "Base"},
		},
	}

	overlay := map[string]any{
		"servers": map[string]any{
			"server1": map[string]any{"type": "stdio", "command": "cmd1"},
		},
		"inputs": []any{
			map[string]any{"id": "shared", "type": "promptString", "description": "New"},
			map[string]any{"id": "overlay_only", "type": "promptString", "description": "Overlay"},
		},
	}

	result := MergeMCPServersWithKey(base, overlay, "servers")

	inputs := result["inputs"].([]any)
	assert.Len(t, inputs, 3)

	// "shared" should have the overlay's description
	assert.Equal(t, "shared", inputs[0].(map[string]any)["id"])
	assert.Equal(t, "New", inputs[0].(map[string]any)["description"])

	// "base_only" preserved
	assert.Equal(t, "base_only", inputs[1].(map[string]any)["id"])

	// "overlay_only" added
	assert.Equal(t, "overlay_only", inputs[2].(map[string]any)["id"])
}

func TestMergeMCPServersWithKey_NoInputsInOverlay(t *testing.T) {
	base := map[string]any{
		"servers": map[string]any{
			"existing": map[string]any{"type": "stdio", "command": "cmd1"},
		},
		"inputs": []any{
			map[string]any{"id": "token", "type": "promptString", "description": "Token"},
		},
	}

	overlay := map[string]any{
		"servers": map[string]any{
			"new": map[string]any{"type": "stdio", "command": "cmd2"},
		},
	}

	result := MergeMCPServersWithKey(base, overlay, "servers")

	// Base inputs should be preserved
	inputs := result["inputs"].([]any)
	assert.Len(t, inputs, 1)
	assert.Equal(t, "token", inputs[0].(map[string]any)["id"])
}

func TestMergeInputArrays(t *testing.T) {
	t.Run("empty base", func(t *testing.T) {
		result := mergeInputArrays(nil, []any{
			map[string]any{"id": "a", "description": "A"},
		})
		assert.Len(t, result, 1)
		assert.Equal(t, "a", result[0].(map[string]any)["id"])
	})

	t.Run("empty overlay", func(t *testing.T) {
		result := mergeInputArrays([]any{
			map[string]any{"id": "a", "description": "A"},
		}, nil)
		assert.Len(t, result, 1)
	})

	t.Run("deduplication by id", func(t *testing.T) {
		base := []any{
			map[string]any{"id": "a", "description": "Old A"},
			map[string]any{"id": "b", "description": "B"},
		}
		overlay := []any{
			map[string]any{"id": "a", "description": "New A"},
			map[string]any{"id": "c", "description": "C"},
		}
		result := mergeInputArrays(base, overlay)
		assert.Len(t, result, 3)
		assert.Equal(t, "New A", result[0].(map[string]any)["description"])
		assert.Equal(t, "B", result[1].(map[string]any)["description"])
		assert.Equal(t, "C", result[2].(map[string]any)["description"])
	})
}

func TestMergeProjectAgentContent_PluginWithHyphens(t *testing.T) {
	existing := `<!-- dex:my-cool-plugin -->
Plugin content
<!-- /dex:my-cool-plugin -->`

	projectInstructions := "# Project"

	result := MergeProjectAgentContent(existing, projectInstructions)

	expected := `# Project

<!-- dex:my-cool-plugin -->
Plugin content
<!-- /dex:my-cool-plugin -->`

	assert.Equal(t, expected, result)
}
