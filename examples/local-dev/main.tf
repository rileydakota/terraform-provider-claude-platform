# Local smoke test against the dev-override build.
#
#   cd examples/local-dev
#   export TF_CLI_CONFIG_FILE=../../dev.tfrc
#   export ANTHROPIC_ADMIN_KEY=sk-ant-admin-...   # or ANTHROPIC_OAUTH_TOKEN
#   terraform apply                                # no `terraform init` with dev overrides
#
# Cleanup: terraform destroy (archives the workspace).

terraform {
  required_providers {
    claudeplatform = {
      source = "rileydakota/claude-platform"
    }
  }
}

# Credentials come from the environment:
# ANTHROPIC_OAUTH_TOKEN (preferred) or ANTHROPIC_ADMIN_KEY.
provider "claudeplatform" {}

data "claudeplatform_organization" "me" {}

resource "claudeplatform_workspace" "test" {
  name = "test"
  tags = {
    managed_by = "terraform"
    purpose    = "provider-smoke-test"
  }
}

output "organization" {
  value = data.claudeplatform_organization.me.name
}

output "workspace_id" {
  value = claudeplatform_workspace.test.id
}

output "workspace_display_color" {
  value = claudeplatform_workspace.test.display_color
}
