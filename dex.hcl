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

registry "nexus" {
  url = "https://nexustemplateproduction.z13.web.core.windows.net"
}

package "base-dev" {}

package "code-review" {}

package "github-workflows" {}
