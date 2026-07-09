# Changelog

## Unreleased (initial development)

### Added

- Provider with three authentication paths: `org:admin` OAuth token, admin API
  key, and a `federation` block that performs the Workload Identity Federation
  token exchange at plan time (zero static credentials in CI).
- Resources: `claudeplatform_workspace`, `claudeplatform_workspace_member`,
  `claudeplatform_organization_invite`, `claudeplatform_service_account`,
  `claudeplatform_service_account_workspace`,
  `claudeplatform_federation_issuer`, `claudeplatform_federation_rule`.
- Data sources: `claudeplatform_organization`, `claudeplatform_workspace`,
  `claudeplatform_workspaces`, `claudeplatform_user`.
- `team-workspace` module: workspace + human roles + workspace-scoped workload
  identity per team.

### Known limitations

- Not yet exercised against a live organization; the WIF endpoint
  request/response shapes are built from the published documentation.
- `claudeplatform_api_key` (import-only), `claudeplatform_organization_member`,
  and `claudeplatform_federation_rule_workspace` are designed but not yet
  implemented (see DESIGN.md).
