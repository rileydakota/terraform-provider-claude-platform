terraform {
  required_providers {
    claudeplatform = {
      source = "rileydakota/claude-platform"
    }
  }
}

# Option 1: org:admin OAuth token (full surface, including WIF resources).
#   ant auth login --profile admin --scope "org:admin"
#   export ANTHROPIC_OAUTH_TOKEN=$(ant auth print-credentials --profile admin --access-token)
provider "claudeplatform" {}

# Option 2: admin API key (env ANTHROPIC_ADMIN_KEY). Classic Admin API only —
# service accounts and federation issuers/rules will error, since those
# endpoints reject admin API keys.
# provider "claudeplatform" {
#   admin_api_key = var.admin_api_key
# }
