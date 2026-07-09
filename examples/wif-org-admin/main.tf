# Granting a WIF identity org admin, expressed as Terraform.
#
# Effective permissions are the INTERSECTION of two things, and the pieces
# split deliberately between Terraform and the Console:
#
#   1. service account organization_role = "admin"   → Terraform CAN create this
#   2. federation rule oauth_scope = "org:admin"     → Console ONLY (anti-self-escalation:
#      API callers can never create or modify org:admin rules, even when they
#      already hold org:admin themselves)
#
# So the bootstrap is: Terraform creates the admin service account and a
# dedicated issuer (below), a human wires them together once in the Console,
# and every other stack authenticates through that rule with zero static
# credentials (see the `federation` provider block at the bottom).
#
# This stack itself runs under a human org:admin OAuth token:
#   ant auth login --profile admin --scope "org:admin"
#   export ANTHROPIC_OAUTH_TOKEN=$(ant auth print-credentials --profile admin --access-token)

terraform {
  required_providers {
    claudeplatform = {
      source = "rileydakota/claude-platform"
    }
  }
}

provider "claudeplatform" {}

data "claudeplatform_organization" "me" {}

# ---------------------------------------------------------------------------
# Bootstrap half — created by Terraform, wired together in the Console
# ---------------------------------------------------------------------------

# The admin-role identity the CI workload will act as. A rule granting
# org:admin must target an admin-role service account; role changes force
# replacement, so this is the only admin SA you should ever need.
resource "claudeplatform_service_account" "terraform" {
  name              = "terraform-org-admin"
  organization_role = "admin"
}

# Dedicated issuer for the bootstrap rule ONLY. Once the Console-created
# org:admin rule backs this issuer, the API refuses to update it — which is
# exactly why it must not be shared with the workspace-scoped rules below
# (those need to stay Terraform-updatable).
resource "claudeplatform_federation_issuer" "github_actions_bootstrap" {
  name       = "github-actions-bootstrap"
  issuer_url = "https://token.actions.githubusercontent.com"

  jwks {
    type = "discovery"
  }

  lifecycle {
    # Updates 400 once the org:admin rule references this issuer, and archive
    # 400s while the rule is live — treat it as write-once.
    prevent_destroy = true
  }
}

# >>> MANUAL CONSOLE STEP (once per organization) <<<
#
# Console → Settings → Workload identity → Connect workload:
#   issuer:          github-actions-bootstrap (created above)
#   service account: terraform-org-admin      (created above)
#   oauth scope:     org:admin                (under "Advanced rule options")
#   subject:         repo:my-org/infra-repo:ref:refs/heads/main
#
# Pin the subject to the protected branch EXACTLY — a trailing wildcard like
# repo:my-org/infra-repo:* also matches fork-triggered pull_request runs,
# i.e. anyone who can open a PR could mint an org:admin token.
#
# Feed the resulting rule ID in via var.bootstrap_rule_id so downstream
# stacks can reference it. It cannot be a Terraform resource by design.

variable "bootstrap_rule_id" {
  type        = string
  description = "fdrl_... ID of the Console-created org:admin bootstrap rule."
}

# ---------------------------------------------------------------------------
# Downstream half — everything the CI identity manages from here on
# ---------------------------------------------------------------------------

resource "claudeplatform_workspace" "ci" {
  name = "ci-inference"
  tags = {
    managed_by = "terraform"
  }
}

# Developer-role identity for actual inference workloads. Keep runtime
# identities developer-role; only the bootstrap SA above is admin.
resource "claudeplatform_service_account" "inference" {
  name              = "inference-worker"
  organization_role = "developer"
}

# A federation rule can only target a service account that is a member of the
# rule's workspace (implicit membership covers the default workspace only).
resource "claudeplatform_service_account_workspace" "inference_ci" {
  service_account_id = claudeplatform_service_account.inference.id
  workspace_id       = claudeplatform_workspace.ci.id
}

# Separate issuer for workspace-scoped rules, so it stays API-updatable.
resource "claudeplatform_federation_issuer" "github_actions" {
  name       = "github-actions"
  issuer_url = "https://token.actions.githubusercontent.com"

  jwks {
    type = "discovery"
  }
}

# workspace:inference is the ceiling for tokens minted under this rule — even
# though nothing here is admin, the intersection rule means a broader-role SA
# would still be capped to inference in this workspace.
resource "claudeplatform_federation_rule" "app_inference" {
  name               = "app-inference"
  issuer_id          = claudeplatform_federation_issuer.github_actions.id
  service_account_id = claudeplatform_service_account_workspace.inference_ci.service_account_id
  workspace_id       = claudeplatform_service_account_workspace.inference_ci.workspace_id

  oauth_scope            = "workspace:inference"
  token_lifetime_seconds = 600

  match {
    subject_prefix = "repo:my-org/app-repo:ref:refs/heads/main"
    claims = {
      repository_owner = "my-org"
    }
  }
}

# ---------------------------------------------------------------------------
# How downstream stacks authenticate (for reference — lives in THEIR provider
# block, not here): the provider exchanges the runner's OIDC token against the
# bootstrap rule at plan time. No static credentials anywhere.
# ---------------------------------------------------------------------------
#
# provider "claudeplatform" {
#   federation {
#     federation_rule_id  = var.bootstrap_rule_id
#     organization_id     = data.claudeplatform_organization.me.id
#     service_account_id  = claudeplatform_service_account.terraform.id
#     identity_token_file = "/var/run/secrets/anthropic.com/token"
#   }
# }

output "organization_id" {
  value = data.claudeplatform_organization.me.id
}

output "terraform_service_account_id" {
  description = "Target this from the Console bootstrap rule."
  value       = claudeplatform_service_account.terraform.id
}

output "bootstrap_issuer_id" {
  description = "Select this issuer in the Console bootstrap rule."
  value       = claudeplatform_federation_issuer.github_actions_bootstrap.id
}
