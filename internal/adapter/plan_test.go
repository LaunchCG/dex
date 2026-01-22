package adapter

import (
	"reflect"
	"testing"
)

func TestMergePlans_MCPServers(t *testing.T) {
	plan1 := NewPlan("test-plugin")
	plan1.MCPEntries = map[string]any{
		"mcpServers": map[string]any{
			"server1": map[string]any{"command": "cmd1"},
		},
	}

	plan2 := NewPlan("test-plugin")
	plan2.MCPEntries = map[string]any{
		"mcpServers": map[string]any{
			"server2": map[string]any{"command": "cmd2"},
		},
	}

	plan3 := NewPlan("test-plugin")
	plan3.MCPEntries = map[string]any{
		"mcpServers": map[string]any{
			"server3": map[string]any{"command": "cmd3"},
		},
	}

	merged := MergePlans(plan1, plan2, plan3)

	mcpServers, ok := merged.MCPEntries["mcpServers"].(map[string]any)
	if !ok {
		t.Fatalf("mcpServers not found or wrong type")
	}

	if len(mcpServers) != 3 {
		t.Errorf("expected 3 servers, got %d: %v", len(mcpServers), mcpServers)
	}

	for _, name := range []string{"server1", "server2", "server3"} {
		if _, ok := mcpServers[name]; !ok {
			t.Errorf("missing server: %s", name)
		}
	}
}

func TestMergeSettingsEntries_Arrays(t *testing.T) {
	dst := map[string]any{
		"allow": []any{"cmd1", "cmd2"},
	}
	src := map[string]any{
		"allow": []any{"cmd3", "cmd4"},
	}

	mergeSettingsEntries(dst, src)

	allow, ok := dst["allow"].([]any)
	if !ok {
		t.Fatalf("allow not found or wrong type")
	}

	if len(allow) != 4 {
		t.Errorf("expected 4 items, got %d: %v", len(allow), allow)
	}

	expected := []any{"cmd1", "cmd2", "cmd3", "cmd4"}
	if !reflect.DeepEqual(allow, expected) {
		t.Errorf("got %v, want %v", allow, expected)
	}
}
