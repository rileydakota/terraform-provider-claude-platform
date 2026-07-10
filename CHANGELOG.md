# Changelog

## Unreleased

### Added

- Registry-layout examples (`examples/resources/<type>/resource.tf` +
  `import.sh`, `examples/data-sources/<type>/data-source.tf`) for every
  resource and data source; the generated registry docs now include Example
  Usage and Import sections.
- `golangci-lint` config (with a depguard rule banning SDKv2 imports) and a
  CI lint job.
- CI docs drift check: `make generate` must leave the tree clean.
- Dependabot for Go modules and GitHub Actions.

### Changed

- `tfplugindocs` is now pinned in a `tools/` module (was `@latest`);
  `make docs` runs `go generate` there, and a new `make generate` target
  chains fmt + docs.

### Fixed

- CI `terraform fmt` check referenced the removed `modules/` directory and
  silently passed despite erroring (the setup-terraform wrapper swallowed the
  exit code). The check now targets `examples/` only and the wrapper is
  disabled.

## v0.1.1

### Fixed

- `claudeplatform_service_account_workspace`: the membership endpoint requires
  a `workspace_role` field (discovered on first live apply — the API returned
  `400 workspace_role: Field required`). The resource now has a
  `workspace_role` attribute, defaulting to `workspace_developer`.

## v0.1.0

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
