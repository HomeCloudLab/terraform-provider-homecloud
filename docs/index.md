---
page_title: "HomeCloud Provider"
description: |-
  Manage HomeCloud account resources (queues, buckets, IAM, functions, compute) through the console API with SigV1 Access Keys.
---

# HomeCloud Provider

The HomeCloud Terraform / OpenTofu provider talks to `console.{apex}/api/v1` with
**SigV1 Access Keys**. It does **not** manage K3s, Helm, or data-plane bytes.

Install with `terraform init` from
[`registry.terraform.io/providers/homecloudlab/homecloud`](https://registry.terraform.io/providers/homecloudlab/homecloud/latest)
(**v0.1.1**). Existing lock files: `terraform init -upgrade`. Community listings show as **self-signed** (key ID `4B8BCFED1A615BA9`).
To hack this repo, use `provider_installation.dev_overrides` and skip `terraform init`.
See the repository README and `PUBLISHING.md`.

## Example Usage

```terraform
terraform {
  required_providers {
    homecloud = {
      source = "homecloudlab/homecloud"
    }
  }
}

provider "homecloud" {
  apex = "holab.abrdns.com"
}

data "homecloud_account" "this" {}

resource "homecloud_mq_queue" "jobs" {
  name = "jobs"
}
```

## Authentication

Local: `homecloud configure`, then `terraform apply`. The provider reads
`~/.homecloud/credentials` (same JSON as the CLI). `profile` / `HC_PROFILE`
selects another account. Do not put secrets in `.tf` files.

CI: `HC_ACCESS_KEY_ID` + `HC_SECRET_ACCESS_KEY`, or `HC_ROLE_ARN` +
`permissions: id-token: write` (OIDC → temporary SigV1). If `HC_ROLE_ARN` is
set, a leftover credentials file is ignored. See `examples/github-oidc`.
Optional: `HC_OIDC_AUDIENCE`, `HC_SESSION_TOKEN`.

IAM mutations need a console role of **owner or admin**. Unmapped Service Account
console routes return `403 iam.management_sa_not_enabled`. Unmapped assumed-role
sessions return `403 iam.management_role_not_enabled`.

## Schema

### Optional

- `profile` (String) Named profile in `~/.homecloud/credentials`. Env: `HC_PROFILE`.
- `access_key` (String, Sensitive) Access Key ID. Env: `HC_ACCESS_KEY_ID`.
- `secret_key` (String, Sensitive) Secret Access Key. Env: `HC_SECRET_ACCESS_KEY`.
- `apex` (String) Platform apex. Default `holab.abrdns.com`. Env: `HC_APEX`.
- `endpoint` (String) Override console base URL (tests). Env: `HC_ENDPOINT`.
- `account_id` (String) Account UUID. Default from Access Key whoami. Env: `HC_ACCOUNT_ID`.
- `role_arn` (String) IAM role ARN for GitHub OIDC. Env: `HC_ROLE_ARN`.
- `web_identity_token` (String, Sensitive) GitHub OIDC JWT. Env: `HC_WEB_IDENTITY_TOKEN`.
- `oidc_audience` (String) Audience for the GitHub OIDC token. Default `https://console.{apex}`. Env: `HC_OIDC_AUDIENCE`.
- `session_token` (String, Sensitive) STS session token. Env: `HC_SESSION_TOKEN`.
