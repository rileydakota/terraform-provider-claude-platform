resource "claudeplatform_federation_issuer" "github_actions" {
  name       = "github-actions"
  issuer_url = "https://token.actions.githubusercontent.com"

  jwks {
    type = "discovery"
  }
}

resource "claudeplatform_workspace" "ml_team" {
  name = "ml-team"
}

resource "claudeplatform_service_account" "ci" {
  name = "github-actions-ci"
}

# Mint workspace:inference tokens for CI runs on the protected main branch.
resource "claudeplatform_federation_rule" "ci_inference" {
  name               = "github-actions-main"
  issuer_id          = claudeplatform_federation_issuer.github_actions.id
  service_account_id = claudeplatform_service_account.ci.id
  workspace_id       = claudeplatform_workspace.ml_team.id
  oauth_scope        = "workspace:inference"

  match {
    subject_prefix = "repo:acme/inference-service:ref:refs/heads/main"
  }
}
