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
