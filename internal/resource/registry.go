package resource

import (
	"fmt"
	"sort"
)

// ResourceFactory creates a new empty resource of a given type.
// Used by the registry to instantiate resources during HCL parsing.
type ResourceFactory func() Resource

// resourceTypes maps HCL block type names to their factory functions.
var resourceTypes = map[string]ResourceFactory{
	// Claude Code resources
	"claude_skill":      func() Resource { return &ClaudeSkill{} },
	"claude_command":    func() Resource { return &ClaudeCommand{} },
	"claude_subagent":   func() Resource { return &ClaudeSubagent{} },
	"claude_rule":       func() Resource { return &ClaudeRule{} },
	"claude_rules":      func() Resource { return &ClaudeRules{} },
	"claude_settings":   func() Resource { return &ClaudeSettings{} },
	"claude_mcp_server": func() Resource { return &ClaudeMCPServer{} },

	// GitHub Copilot resources - merged (multiple plugins contribute to same file)
	"copilot_instruction": func() Resource { return &CopilotInstruction{} },
	"copilot_mcp_server":  func() Resource { return &CopilotMCPServer{} },

	// GitHub Copilot resources - standalone (one file per resource)
	"copilot_instructions": func() Resource { return &CopilotInstructions{} },
	"copilot_prompt":       func() Resource { return &CopilotPrompt{} },
	"copilot_agent":        func() Resource { return &CopilotAgent{} },
	"copilot_skill":        func() Resource { return &CopilotSkill{} },

	// Cursor resources - merged (multiple plugins contribute to same file)
	"cursor_rule":       func() Resource { return &CursorRule{} },
	"cursor_mcp_server": func() Resource { return &CursorMCPServer{} },

	// Cursor resources - standalone (one file per resource)
	"cursor_rules":   func() Resource { return &CursorRules{} },
	"cursor_command": func() Resource { return &CursorCommand{} },
}

// NewResource creates a new resource instance by type name.
// Returns an error if the type name is not registered.
func NewResource(typeName string) (Resource, error) {
	factory, ok := resourceTypes[typeName]
	if !ok {
		return nil, fmt.Errorf("unknown resource type: %q", typeName)
	}
	return factory(), nil
}

// RegisteredTypes returns all registered resource type names in sorted order.
func RegisteredTypes() []string {
	types := make([]string, 0, len(resourceTypes))
	for typeName := range resourceTypes {
		types = append(types, typeName)
	}
	sort.Strings(types)
	return types
}

// IsRegisteredType checks if a type name is registered.
func IsRegisteredType(typeName string) bool {
	_, ok := resourceTypes[typeName]
	return ok
}

// RegisterResourceType registers a new resource type with its factory function.
// This allows extending the resource system with additional types.
// Panics if the type name is already registered.
func RegisterResourceType(typeName string, factory ResourceFactory) {
	if _, exists := resourceTypes[typeName]; exists {
		panic(fmt.Sprintf("resource type %q is already registered", typeName))
	}
	resourceTypes[typeName] = factory
}
