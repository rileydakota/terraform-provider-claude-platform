data "claudeplatform_user" "alice" {
  email = "alice@example.com"
}

resource "claudeplatform_workspace" "ml_team" {
  name = "ml-team"
}

resource "claudeplatform_workspace_member" "alice" {
  workspace_id   = claudeplatform_workspace.ml_team.id
  user_id        = data.claudeplatform_user.alice.id
  workspace_role = "workspace_developer"
}
