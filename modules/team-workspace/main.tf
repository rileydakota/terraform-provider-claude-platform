# team-workspace: everything a team needs to run their own Claude platform
# workspace, provisioned centrally under the org-admin WIF identity.
#
# Delegation boundary this module encodes:
#   - Team humans get workspace_admin and self-manage members/API keys in the
#     Console from then on.
#   - Team CI gets a workspace-scoped WIF rule (developer or inference) and
#     mints its own short-lived tokens — no static keys to hand out.
#   - Changes to the workspace itself (this module's inputs) stay central:
#     there is no workspace-scoped admin credential in the Admin API, so
#     programmatic workspace administration flows through the org stack.

terraform {
  required_providers {
    claudeplatform = {
      source = "rileydakota/claude-platform"
    }
  }
}

resource "claudeplatform_workspace" "this" {
  name = var.name
  tags = merge({
    team       = var.name
    managed_by = "terraform"
  }, var.tags)
}

# --- Human access -----------------------------------------------------------

data "claudeplatform_user" "admins" {
  for_each = toset(var.admin_emails)
  email    = each.value
}

resource "claudeplatform_workspace_member" "admins" {
  for_each = data.claudeplatform_user.admins

  workspace_id   = claudeplatform_workspace.this.id
  user_id        = each.value.id
  workspace_role = "workspace_admin"
}

data "claudeplatform_user" "developers" {
  for_each = toset(var.developer_emails)
  email    = each.value
}

resource "claudeplatform_workspace_member" "developers" {
  for_each = data.claudeplatform_user.developers

  workspace_id   = claudeplatform_workspace.this.id
  user_id        = each.value.id
  workspace_role = "workspace_developer"
}

# --- CI / workload access ----------------------------------------------------

resource "claudeplatform_service_account" "ci" {
  name              = "${var.name}-ci"
  organization_role = "developer"
}

# Rules can only target a service account that is a member of the rule's
# workspace; implicit membership covers the default workspace only.
resource "claudeplatform_service_account_workspace" "ci" {
  service_account_id = claudeplatform_service_account.ci.id
  workspace_id       = claudeplatform_workspace.this.id
}

resource "claudeplatform_federation_rule" "ci" {
  name               = "${var.name}-ci"
  issuer_id          = var.issuer_id
  service_account_id = claudeplatform_service_account_workspace.ci.service_account_id
  workspace_id       = claudeplatform_service_account_workspace.ci.workspace_id

  oauth_scope            = var.oauth_scope
  token_lifetime_seconds = var.token_lifetime_seconds

  match {
    subject_prefix = var.ci_subject_prefix
    claims         = length(var.ci_match_claims) > 0 ? var.ci_match_claims : null
  }
}
