resource "claudeplatform_workspace" "ml_team" {
  name = "ml-team"
}

resource "claudeplatform_service_account" "ci" {
  name = "github-actions-ci"
}

resource "claudeplatform_service_account_workspace" "ci_ml_team" {
  service_account_id = claudeplatform_service_account.ci.id
  workspace_id       = claudeplatform_workspace.ml_team.id
  workspace_role     = "workspace_developer"
}
