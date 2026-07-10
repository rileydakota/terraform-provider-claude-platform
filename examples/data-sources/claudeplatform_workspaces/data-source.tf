data "claudeplatform_workspaces" "all" {
  include_archived = false
}

output "workspace_names" {
  value = [for w in data.claudeplatform_workspaces.all.workspaces : w.name]
}
