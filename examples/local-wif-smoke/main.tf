# Live smoke test: named workspaces, each with its own workspace-scoped
# workload identity (service account -> membership -> federation rule),
# sharing one OIDC issuer.
#
# Requires an org:admin OAuth token (WIF endpoints reject admin API keys):
#   ant auth login --profile admin --scope "org:admin"
#   export ANTHROPIC_OAUTH_TOKEN=$(ant auth print-credentials --profile admin --access-token)
#   export TF_CLI_CONFIG_FILE=../../dev.tfrc
#   terraform apply
#
# Cleanup: terraform destroy (rules archive first, then issuer/SAs — ordering
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
  description = "Workspaces to create, each with its own scoped service account. Names must be lowercase alphanumeric plus hyphens (they seed WIF resource names)."
  default     = ["test", "dev"]

  validation {
    condition     = alltrue([for n in var.workspace_names : can(regex("^[a-z0-9-]{1,200}$", n))])
    error_message = "Workspace names must match ^[a-z0-9-]+$ so derived service account and rule names are valid."
  }
}

variable "ci_subject" {
  type        = string
  description = "JWT subject the smoke-test rules match. Harmless placeholder by default — no real workload can satisfy it."
  default     = "repo:rileydakota/does-not-exist:ref:refs/heads/main"
}

# One issuer shared by every workspace's rule.
resource "claudeplatform_federation_issuer" "github_actions" {
  name       = "wif-smoke-github-actions"
  issuer_url = "https://token.actions.githubusercontent.com"

  jwks {
    type = "discovery"
  }
}

resource "claudeplatform_workspace" "named" {
  for_each = toset(var.workspace_names)

  name = each.value
  tags = {
    managed_by = "terraform"
    purpose    = "wif-smoke-test"
  }
}

# One service account per workspace. The SA object is org-level, but its
# effective reach is scoped by membership + the rule below to exactly its
# workspace (the implicit default-workspace membership stays inert because no
# rule enables the default workspace).
resource "claudeplatform_service_account" "per_workspace" {
  for_each = claudeplatform_workspace.named

  name              = "wif-smoke-${each.key}"
  organization_role = "developer"
}

resource "claudeplatform_service_account_workspace" "per_workspace" {
  for_each = claudeplatform_workspace.named

  service_account_id = claudeplatform_service_account.per_workspace[each.key].id
  workspace_id       = each.value.id
}

resource "claudeplatform_federation_rule" "per_workspace" {
  for_each = claudeplatform_workspace.named

  name               = "wif-smoke-${each.key}"
  issuer_id          = claudeplatform_federation_issuer.github_actions.id
  service_account_id = claudeplatform_service_account_workspace.per_workspace[each.key].service_account_id
  workspace_id       = claudeplatform_service_account_workspace.per_workspace[each.key].workspace_id

  oauth_scope            = "workspace:inference"
  token_lifetime_seconds = 600

  match {
    subject_prefix = var.ci_subject
  }
}

output "organization" {
  value = data.claudeplatform_organization.me.name
}

# Per-workspace identity bundle: everything a workload in that workspace
# would need for the token exchange (plus the org id above).
output "workspace_identities" {
  value = {
    for name, ws in claudeplatform_workspace.named : name => {
      workspace_id       = ws.id
      service_account_id = claudeplatform_service_account.per_workspace[name].id
      federation_rule_id = claudeplatform_federation_rule.per_workspace[name].id
    }
  }
}
