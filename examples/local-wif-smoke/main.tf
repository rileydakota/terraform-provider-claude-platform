# Live smoke test: named workspaces + a complete workload identity
# (service account -> workspace membership -> issuer -> federation rule).
#
# Requires an org:admin OAuth token (WIF endpoints reject admin API keys):
#   ant auth login --profile admin --scope "org:admin"
#   export ANTHROPIC_OAUTH_TOKEN=$(ant auth print-credentials --profile admin --access-token)
#   export TF_CLI_CONFIG_FILE=../../dev.tfrc
#   terraform apply
#
# Cleanup: terraform destroy (rule archives first, then issuer/SA — ordering
# is enforced by the dependency graph; everything is a soft delete).

terraform {
  required_providers {
    claudeplatform = {
      source = "rileydakota/claude-platform"
    }
  }
}

provider "claudeplatform" {}

data "claudeplatform_organization" "me" {}

variable "workspace_names" {
  type        = list(string)
  description = "Workspaces to create."
  default     = ["test", "dev"]
}

variable "ci_subject" {
  type        = string
  description = "JWT subject the smoke-test rule matches. Harmless placeholder by default — no real workload can satisfy it."
  default     = "repo:rileydakota/does-not-exist:ref:refs/heads/main"
}

resource "claudeplatform_workspace" "named" {
  for_each = toset(var.workspace_names)

  name = each.value
  tags = {
    managed_by = "terraform"
    purpose    = "wif-smoke-test"
  }
}

# --- Workload identity -------------------------------------------------------

resource "claudeplatform_service_account" "smoke" {
  name              = "wif-smoke-test"
  organization_role = "developer"
}

# Membership in the first named workspace, so the rule below can target it there.
resource "claudeplatform_service_account_workspace" "smoke" {
  service_account_id = claudeplatform_service_account.smoke.id
  workspace_id       = claudeplatform_workspace.named[var.workspace_names[0]].id
}

resource "claudeplatform_federation_issuer" "github_actions" {
  name       = "wif-smoke-github-actions"
  issuer_url = "https://token.actions.githubusercontent.com"

  jwks {
    type = "discovery"
  }
}

resource "claudeplatform_federation_rule" "smoke" {
  name               = "wif-smoke-test"
  issuer_id          = claudeplatform_federation_issuer.github_actions.id
  service_account_id = claudeplatform_service_account_workspace.smoke.service_account_id
  workspace_id       = claudeplatform_service_account_workspace.smoke.workspace_id

  oauth_scope            = "workspace:inference"
  token_lifetime_seconds = 600

  match {
    subject_prefix = var.ci_subject
  }
}

output "organization" {
  value = data.claudeplatform_organization.me.name
}

output "workspace_ids" {
  value = { for name, ws in claudeplatform_workspace.named : name => ws.id }
}

output "workload_identity" {
  value = {
    service_account_id = claudeplatform_service_account.smoke.id
    issuer_id          = claudeplatform_federation_issuer.github_actions.id
    rule_id            = claudeplatform_federation_rule.smoke.id
  }
}
