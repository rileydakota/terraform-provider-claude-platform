variable "name" {
  type        = string
  description = "Team name; used for the workspace and derived WIF resource names."

  validation {
    condition     = can(regex("^[a-z0-9-]{1,200}$", var.name))
    error_message = "name must be lowercase alphanumeric plus hyphens (WIF resource names require ^[a-z0-9-]+$)."
  }
}

variable "admin_emails" {
  type        = list(string)
  description = "Emails of existing organization members to grant workspace_admin (they self-manage members and API keys in the Console). Members must already be in the org — invite separately."
  default     = []
}

variable "developer_emails" {
  type        = list(string)
  description = "Emails of existing organization members to grant workspace_developer."
  default     = []
}

variable "issuer_id" {
  type        = string
  description = "Federation issuer ID (fdis_...) shared across teams, e.g. GitHub Actions."
}

variable "ci_subject_prefix" {
  type        = string
  description = "JWT subject the team's CI rule matches. Pin to a protected ref, e.g. repo:my-org/team-repo:ref:refs/heads/main. A trailing * is a prefix match — avoid broad wildcards, they also match fork-triggered PR runs."
}

variable "ci_match_claims" {
  type        = map(string)
  description = "Additional exact-match claims for the CI rule (e.g. { repository_owner = \"my-org\" })."
  default     = {}
}

variable "oauth_scope" {
  type        = string
  description = "Scope for tokens the team's CI mints: workspace:developer (full non-admin API in the workspace) or workspace:inference (Messages/Models only)."
  default     = "workspace:developer"

  validation {
    condition     = contains(["workspace:developer", "workspace:inference"], var.oauth_scope)
    error_message = "Only workspace:developer and workspace:inference are API-manageable rule scopes."
  }
}

variable "token_lifetime_seconds" {
  type        = number
  description = "Lifetime of tokens minted under the team's CI rule (60-86400)."
  default     = 3600
}

variable "tags" {
  type        = map(string)
  description = "Extra workspace tags (merged with team/managed_by defaults). Keys may not begin with \"anthropic\"."
  default     = {}
}
