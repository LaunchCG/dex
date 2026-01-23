package {
  name = "test-plugin"
  version = "2.0.0"
  description = "Test plugin version 2.0.0"
}

claude_rule "test-rule-v2" {
  description = "Test rule from v2.0.0"
  content = "This is the v2.0.0 rule content"
}

claude_skill "test-skill" {
  description = "A test skill added in v2"
  content = "Skill content for version 2"
}
