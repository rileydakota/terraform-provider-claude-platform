output "workspace_id" {
  value = claudeplatform_workspace.this.id
}

output "service_account_id" {
  value = claudeplatform_service_account.ci.id
}

output "federation_rule_id" {
  value = claudeplatform_federation_rule.ci.id
}

# Everything the team's workload needs to authenticate — hand this to them
# (e.g. as GitHub Actions variables). The SDKs pick these up automatically
# and perform the token exchange with no static credentials.
output "workload_env" {
  description = "Environment variables for the team's CI/workload."
  value = {
    ANTHROPIC_FEDERATION_RULE_ID = claudeplatform_federation_rule.ci.id
    ANTHROPIC_SERVICE_ACCOUNT_ID = claudeplatform_service_account.ci.id
    ANTHROPIC_WORKSPACE_ID       = claudeplatform_workspace.this.id
  }
}
