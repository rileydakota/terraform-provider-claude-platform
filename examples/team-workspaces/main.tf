# Central org stack: one shared issuer, one team-workspace module call per
# team. Runs locally under your human org:admin token first, then in CI under
# the bootstrap WIF identity (see examples/wif-org-admin).
#
# After apply, each team self-manages:
#   - humans: workspace_admin in the Console (members, API keys)
#   - CI: mints its own short-lived tokens via the rule (workload_env output)
# New teams / workspace changes come back here as PRs — the Admin API has no
# workspace-scoped admin credential, so central Terraform is the write path.

terraform {
  required_providers {
    claudeplatform = {
      source = "rileydakota/claude-platform"
    }
  }
}

provider "claudeplatform" {}

data "claudeplatform_organization" "me" {}

# One issuer shared by every team's workspace-scoped rule. Keep the org:admin
# bootstrap rule on its own dedicated issuer, never this one.
resource "claudeplatform_federation_issuer" "github_actions" {
  name       = "github-actions"
  issuer_url = "https://token.actions.githubusercontent.com"

  jwks {
    type = "discovery"
  }
}

module "team_atlas" {
  source = "../../modules/team-workspace"

  name              = "atlas"
  issuer_id         = claudeplatform_federation_issuer.github_actions.id
  ci_subject_prefix = "repo:my-org/atlas:ref:refs/heads/main"
  ci_match_claims   = { repository_owner = "my-org" }
  admin_emails      = ["atlas-lead@example.com"]

  oauth_scope            = "workspace:developer"
  token_lifetime_seconds = 900
}

module "team_arbiter" {
  source = "../../modules/team-workspace"

  name              = "arbiter"
  issuer_id         = claudeplatform_federation_issuer.github_actions.id
  ci_subject_prefix = "repo:my-org/arbiter:ref:refs/heads/main"
  ci_match_claims   = { repository_owner = "my-org" }

  # Inference-only: this team's CI can call Messages/Models but not manage
  # Files, Skills, or agents in the workspace.
  oauth_scope = "workspace:inference"
}

# Hand each team the full set of variables their workload needs (the SDKs
# read these and perform the token exchange automatically; the identity token
# itself comes from the runner's OIDC provider).
output "team_workload_env" {
  value = {
    atlas   = merge(module.team_atlas.workload_env, { ANTHROPIC_ORGANIZATION_ID = data.claudeplatform_organization.me.id })
    arbiter = merge(module.team_arbiter.workload_env, { ANTHROPIC_ORGANIZATION_ID = data.claudeplatform_organization.me.id })
  }
}
