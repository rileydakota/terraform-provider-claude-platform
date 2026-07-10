resource "claudeplatform_federation_issuer" "github_actions" {
  name       = "github-actions"
  issuer_url = "https://token.actions.githubusercontent.com"

  jwks {
    type = "discovery"
  }
}
