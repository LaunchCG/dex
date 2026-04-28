project {
  name            = "dex"
  default_platform = "claude-code"

  git_exclude = true
}

settings "project_permissions" {
  claude {
    enable_all_project_mcp_servers = true
  }
}

package "base-dev" {}

package "code-review" {}

registry "nexus" {
  url = "https://regproduction.z13.web.core.windows.net"
}
