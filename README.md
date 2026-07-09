# terraform-provider-claude-platform

Terraform provider for the [Anthropic Admin API](https://platform.claude.com/docs/en/manage-claude/admin-api):
manage Claude platform workspaces, organization members, invites, API keys, and the full
[Workload Identity Federation](https://platform.claude.com/docs/en/manage-claude/workload-identity-federation)
surface (service accounts, federation issuers, federation rules) as code.

> Status: early scaffold — compiles, schema-validated, not yet exercised against a live org.
> See [DESIGN.md](./DESIGN.md) for the full design and roadmap. Licensed under
> [MPL-2.0](./LICENSE).

## Resources

| Type | Notes |
|---|---|
| `claudeplatform_workspace` | destroy = archive (the API has no hard delete) |
| `claudeplatform_workspace_member` | import as `workspace_id/user_id` |
| `claudeplatform_organization_invite` | invites expire server-side after 21 days |
| `claudeplatform_service_account` | requires org:admin OAuth token |
| `claudeplatform_service_account_workspace` | explicit workspace membership; required before a rule can target the SA in a non-default workspace |
| `claudeplatform_federation_issuer` | requires org:admin OAuth token; `jwks` = discovery \| explicit_url \| inline |
| `claudeplatform_federation_rule` | requires org:admin OAuth token; API-manageable scopes only (`workspace:developer`, `workspace:inference`) |

Data sources: `claudeplatform_organization`, `claudeplatform_workspace`,
`claudeplatform_workspaces`, `claudeplatform_user`.

## Authentication

Three options, in the order the provider resolves them:

1. **OAuth bearer token** with the `org:admin` scope — the full surface, including WIF
   resources (which *reject* admin API keys):

   ```sh
   ant auth login --profile admin --scope "org:admin"
   export ANTHROPIC_OAUTH_TOKEN=$(ant auth print-credentials --profile admin --access-token)
   ```

2. **Admin API key** (`sk-ant-admin...`, env `ANTHROPIC_ADMIN_KEY`) — classic Admin API
   surface only (workspaces, members, invites, API keys).

3. **Federation block** — the provider performs the WIF token exchange itself at plan
   time, so CI runs with zero static credentials. Requires a one-time Console-created
   bootstrap rule (`oauth_scope: org:admin` pinned to a protected branch):

   ```hcl
   provider "claudeplatform" {
     federation {
       federation_rule_id  = "fdrl_..."
       organization_id     = "..."
       service_account_id  = "svac_..."
       identity_token_file = "/var/run/secrets/anthropic.com/token"
     }
   }
   ```

See [examples/wif-bootstrap](./examples/wif-bootstrap/main.tf) for an end-to-end
GitHub Actions setup.

## Development

```sh
make build   # go build
make test    # unit tests (schema validation; no credentials needed)
make testacc # acceptance tests against a real org (TF_ACC=1 + ANTHROPIC_OAUTH_TOKEN)
```

To use a local build, add a dev override to `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "rileydakota/claude-platform" = "/Users/you/go/bin"
  }
  direct {}
}
```

then `go install .` and run `terraform plan` (skip `terraform init`).

## Releasing

Releases are cut by pushing a `v*` tag; the [release workflow](.github/workflows/release.yml)
runs GoReleaser to build, sign, and publish Terraform Registry-compatible artifacts.
One-time setup for a fork/new namespace:

1. Generate a GPG signing key; add `GPG_PRIVATE_KEY` and `PASSPHRASE` repo secrets.
2. Publish the repo publicly (registry requires the `terraform-provider-*` name).
3. Add the GPG public key in [registry.terraform.io](https://registry.terraform.io) →
   Settings → Signing Keys, then publish the provider from the repo.
4. `git tag v0.1.0 && git push --tags`.

Registry docs under `docs/` are generated — edit schemas/examples, then `make docs`.

## Known API constraints the provider encodes

- **Deletes are archives** for workspaces, service accounts, issuers, and rules;
  archived resources are dropped from state on refresh.
- **API keys cannot be created** via the API (Console only) — the planned
  `claudeplatform_api_key` resource is import-only lifecycle management.
- **OAuth callers cannot self-escalate:** rules with `org:admin` or
  `workspace:manage_tunnels` scopes are Console-only; issuers backing such rules are
  not updatable via the API (keep bootstrap rules on a dedicated issuer).
- **Archive ordering:** archiving an issuer or service account 400s while a live rule
  references it — Terraform's dependency graph destroys rules first automatically.
