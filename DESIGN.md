# terraform-provider-claude-platform — Design

Terraform provider for the Anthropic **Admin API** (`/v1/organizations/*`): org members,
invites, workspaces, API keys, and the Workload Identity Federation (WIF) surface —
service accounts, federation issuers, and federation rules.

Registry address (planned): `rileydakota/claude-platform`. Resource type prefix: `claudeplatform_`.

```hcl
terraform {
  required_providers {
    claudeplatform = {
      source = "rileydakota/claude-platform"
    }
  }
}
```

## 1. Authentication model

The Admin API accepts **two credential types**, and they are *not* interchangeable:

| Credential | Header | Works on |
|---|---|---|
| Admin API key (`sk-ant-admin...`) | `x-api-key` | users, invites, workspaces, workspace members, API keys, reports |
| OAuth bearer with `org:admin` scope | `authorization: Bearer` | everything above **plus** service accounts, federation issuers, federation rules (which *reject* admin keys) |

Provider config therefore supports three ways in, resolved in this order:

```hcl
provider "claudeplatform" {
  # 1. OAuth bearer token (env: ANTHROPIC_OAUTH_TOKEN). Full surface.
  oauth_token = "..."

  # 2. Admin API key (env: ANTHROPIC_ADMIN_KEY). Classic surface only —
  #    WIF resources fail fast with a clear error.
  admin_api_key = "..."

  # 3. WIF token exchange performed by the provider itself (CI-native path):
  #    exchanges an OIDC JWT for a short-lived org:admin bearer at plan time.
  federation {
    federation_rule_id  = "fdrl_..."
    organization_id     = "00000000-0000-0000-0000-000000000000"
    service_account_id  = "svac_..."
    # workspace_id      = "wrkspc_..."      # only if the rule spans multiple workspaces
    identity_token_file = "/var/run/secrets/anthropic.com/token"
    # identity_token    = "eyJ..."          # alternative to _file
  }
}
```

The `federation` block is the intended production path: one Console-created bootstrap
rule (`oauth_scope: org:admin`, pinned to a protected branch) lets CI manage the whole
org with no static credentials — exactly the flow Anthropic documents in
"Manage WIF with the Admin API".

**org:admin OAuth caller constraints** (enforced server-side; surfaced in our docs):
- Can only create/modify rules scoped `workspace:developer` or `workspace:inference`.
  `org:admin` / `workspace:manage_tunnels` rules are Console-only (deliberate
  anti-self-escalation design).
- Cannot update an issuer that backs a rule with any other scope — keep the bootstrap
  rule on a *dedicated issuer* so the workspace-scoped issuers stay API-managable.

## 2. Resources

| Resource | Endpoints | Delete semantics | Notes |
|---|---|---|---|
| `claudeplatform_workspace` | `POST/GET/POST /workspaces{,/{id}}`, `POST /{id}/archive` | archive (soft) | `name`, `tags`; `data_residency` + CMEK on roadmap |
| `claudeplatform_workspace_member` | `/workspaces/{wid}/members` | DELETE | composite import id `wid/uid`; roles `workspace_user\|developer\|admin\|billing` |
| `claudeplatform_organization_invite` | `/invites` | DELETE | email+role force replacement; expires after 21 days (status tracked, expiry does not auto-recreate in v0) |
| `claudeplatform_organization_member` (roadmap) | `/users/{id}` | remove from org | adopt-only (users join via invite); manages `role` |
| `claudeplatform_api_key` (roadmap) | `/api_keys/{id}` | set `inactive` | **import-only** — the API cannot create keys (Console only). Manages `name`/`status`. |
| `claudeplatform_service_account` | `/service_accounts` | archive | OAuth-only; `organization_role` = `developer\|admin` (replace on change) |
| `claudeplatform_service_account_workspace` (roadmap) | `/service_accounts/{id}/workspaces` | DELETE | explicit workspace membership (default workspace is implicit) |
| `claudeplatform_federation_issuer` | `/federation_issuers` | archive | `jwks` union: `discovery` (+optional `discovery_base`) \| `explicit_url` \| `inline` (`keys_json`); optional `ca_cert_pem` |
| `claudeplatform_federation_rule` | `/federation_rules` | archive | `match` (subject_prefix/audience/claims/condition-CEL), target service account, `workspace_id` xor `applies_to_all_workspaces`, `oauth_scope`, `token_lifetime_seconds` (60–86400) |
| `claudeplatform_federation_rule_workspace` (roadmap) | `/federation_rules/{id}/workspaces` | DELETE | enable a rule in additional workspaces |

Cross-cutting semantics:

- **Archive-as-delete.** Workspaces, service accounts, issuers, and rules have no hard
  delete. `Delete` archives; `Read` treats `archived_at != null` (or 404) as gone and
  removes the resource from state. Archiving is idempotent server-side.
- **Reference ordering.** Archiving an issuer/service account returns 400 while a live
  rule references it. Terraform's dependency graph (rule references issuer + SA ids)
  destroys rules first, so this works naturally; no special handling needed.
- **Import everywhere.** Anthropic's own guidance for Console-created WIF resources is
  to import rather than recreate. Every resource implements `ImportState`.
- **Immutable fields** are marked `RequiresReplace`: issuer/rule linkage
  (`issuer_id`, `service_account_id`, `workspace_id`, `applies_to_all_workspaces`),
  service account `organization_role`, invite `email`/`role`.

## 3. Data sources

| Data source | Lookup |
|---|---|
| `claudeplatform_organization` | `GET /v1/organizations/me` |
| `claudeplatform_workspace` | by `id` or by `name` |
| `claudeplatform_workspaces` | list (optionally include archived) |
| `claudeplatform_user` | by `email` |
| `claudeplatform_service_account` / `_federation_issuer` / `_federation_rule` (roadmap) | by `id` or `name` (names are unique per resource type) |
| `claudeplatform_api_keys` (roadmap) | filter by workspace/status |

## 4. Implementation

- **Language/stack:** Go, `terraform-plugin-framework` (protocol v6), `terraform-plugin-docs`
  for generated docs, `goreleaser` for registry publishing.
- **API client:** hand-rolled thin client in `internal/client` — no official Go SDK covers
  the WIF admin endpoints. Headers: `anthropic-version: 2023-06-01` + either
  `authorization: Bearer` or `x-api-key`. Bearer wins when both are configured.
- **Two pagination schemes:**
  - classic admin endpoints: `limit`/`after_id` request, `{data, has_more, first_id, last_id}` response
  - WIF endpoints: `limit`/`page` request, `{data, next_page}` response (`include_archived=true` to see archived)
- **Retries:** 429 (honoring `retry-after`) and 5xx, capped exponential backoff, 4 attempts.
- **Errors:** standard envelope `{"type":"error","error":{"type","message"},"request_id"}` →
  typed `APIError` carrying `request_id` for supportability.
- **Token exchange:** `POST /v1/oauth/token` RFC 7523 `jwt-bearer` grant, done once at
  provider Configure. Identity-token file is read at Configure time (projected tokens
  rotate; a plan/apply is short-lived relative to rule lifetimes).

### Field-level notes

- `federation_rule.match`: at least one of `subject_prefix`/`claims`/`condition` must be
  set (an audience-only match is rejected server-side); we validate client-side too.
  `claims` is exact-string-match only — nested/typed claims need `condition` (CEL).
- `federation_issuer.jwks.keys_json` is a JSON string (the `keys` array of a JWKS doc).
  Read uses semantic JSON comparison against state to avoid whitespace-only drift.
  In `inline` mode there is **no automatic key refresh** — rotation requires updating
  this resource (good fit for IaC, called out in docs).
- `token_lifetime_seconds`: 60–86400, server default 3600 → Optional+Computed.
- Names for WIF resources must match `^[a-z0-9-]+$` (validated client-side).
- `workspace.tags`: keys must not begin with `anthropic` (server-enforced).

## 5. Not covered (deliberately, for now)

- Usage & Cost / Claude Code Analytics / Rate limit / Compliance APIs — read-only
  reporting, better served by scripts or future data sources.
- Claude Platform on AWS: only workspace endpoints exist there; the provider targets
  the first-party API (`base_url` is configurable regardless).
- CMEK (`external_key_id`, write-once) and `data_residency` on workspaces — roadmap;
  both are create-time/immutable-ish fields that need careful plan modifiers.

## 6. Testing

- Acceptance tests (`TF_ACC=1`) require a real org + `ANTHROPIC_OAUTH_TOKEN` with
  `org:admin`. WIF tests create against a throwaway issuer (GitHub Actions issuer URL
  with discovery is safe to register).
- Unit tests for the client use `httptest` fixtures for both pagination schemes and the
  error envelope.
