resource "claudeplatform_service_account" "ci" {
  name              = "github-actions-ci"
  organization_role = "developer"
}
