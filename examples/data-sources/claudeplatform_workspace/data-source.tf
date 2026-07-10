# Look up by exact name...
data "claudeplatform_workspace" "ml_team" {
  name = "ml-team"
}

# ...or by ID.
data "claudeplatform_workspace" "by_id" {
  id = "wrkspc_01WgQMyXabc123"
}
