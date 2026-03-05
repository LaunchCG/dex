project {
  name            = "dex"
  agentic_platform = "claude-code"

  git_exclude = true
}

claude_settings "project_permissions" {
  enable_all_project_mcp_servers = true
}

registry "nexus" {
  url = "https://nexustemplateproduction.z13.web.core.windows.net"
}

plugin "base-dev" {}

plugin "code-review" {}

plugin "github-workflows" {}
