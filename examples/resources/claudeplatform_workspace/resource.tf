resource "claudeplatform_workspace" "ml_team" {
  name = "ml-team"

  tags = {
    team        = "ml"
    cost-center = "research"
  }
}
