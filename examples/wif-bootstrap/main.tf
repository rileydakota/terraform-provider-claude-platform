# End-to-end example: GitHub Actions → Claude platform via Workload Identity
# Federation, fully managed from Terraform.
#
# Prerequisite (one Console click, deliberately not automatable): a bootstrap
# federation rule with oauth_scope org:admin pinned to your infra repo's main
# branch, so this configuration can authenticate with no static credentials.

terraform {
  required_providers {
    claudeplatform = {
      source = "rileydakota/claude-platform"
    }
  }
}

provider "claudeplatform" {
  # In CI: exchange the GitHub Actions OIDC token for a short-lived org:admin
  # bearer token. Locally, omit this block and use:
  #   export ANTHROPIC_OAUTH_TOKEN=$(ant auth print-credentials --profile admin --access-token)
  federation {
    federation_rule_id  = var.bootstrap_rule_id
    organization_id     = var.organization_id
    service_account_id  = var.bootstrap_service_account_id
    identity_token_file = "/var/run/secrets/anthropic.com/token"
  }
}

# A workspace per environment; its quota/billing/rate limits apply to tokens
# minted under the rules below.
resource "claudeplatform_workspace" "ci" {
  name = "ci-inference"
  tags = {
    env  = "prod"
    team = "platform"
  }
}

# The non-human identity CI inference tokens act as. Keep it developer-role;
# only the Console-created bootstrap rule needs an admin service account.
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

# Register GitHub Actions as an OIDC issuer. Keep this issuer dedicated to
# workspace-scoped rules — an issuer backing an org:admin rule can no longer
# be updated through the API.
resource "claudeplatform_federation_issuer" "github_actions" {
  name       = "github-actions"
  issuer_url = "https://token.actions.githubusercontent.com"

  jwks {
    type = "discovery"
  }
}

# JWTs from main-branch runs of my-org/my-repo may mint 10-minute
# inference-only tokens acting as the service account, in the CI workspace.
resource "claudeplatform_federation_rule" "gha_inference" {
  name               = "gha-inference"
  issuer_id          = claudeplatform_federation_issuer.github_actions.id
  service_account_id = claudeplatform_service_account_workspace.inference_ci.service_account_id
  workspace_id       = claudeplatform_service_account_workspace.inference_ci.workspace_id

  oauth_scope            = "workspace:inference"
  token_lifetime_seconds = 600

  match {
    # Exact match unless it ends in "*". Never use a broad wildcard like
    # "repo:my-org/my-repo:*" — that also matches fork-triggered PR runs.
    subject_prefix = "repo:my-org/my-repo:ref:refs/heads/main"
    claims = {
      repository_owner = "my-org"
    }
  }
}

variable "organization_id" {
  type = string
}

variable "bootstrap_rule_id" {
  type = string
}

variable "bootstrap_service_account_id" {
  type = string
}
