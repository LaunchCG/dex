package resource

import (
	"fmt"
	"sort"
)

// ResourceFactory creates a new empty resource of a given type.
// Used by the registry to instantiate resources during HCL parsing.
type ResourceFactory func() Resource

// resourceTypes maps resource type names to their factory functions.
var resourceTypes = map[string]ResourceFactory{
	// Universal resource types
	"skill":      func() Resource { return &Skill{} },
	"command":    func() Resource { return &Command{} },
	"agent":      func() Resource { return &Agent{} },
	"rule":       func() Resource { return &Rule{} },
	"rules":      func() Resource { return &Rules{} },
	"settings":   func() Resource { return &Settings{} },
	"mcp_server": func() Resource { return &MCPServer{} },
	"file":       func() Resource { return &File{} },
	"directory":  func() Resource { return &Directory{} },
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
