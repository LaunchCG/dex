package installer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
