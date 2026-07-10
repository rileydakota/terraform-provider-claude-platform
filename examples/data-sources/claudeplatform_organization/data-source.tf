data "claudeplatform_organization" "current" {}

output "organization_id" {
  value = data.claudeplatform_organization.current.id
}
