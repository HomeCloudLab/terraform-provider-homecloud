---
page_title: "HomeCloud Provider"
description: |-
  Manage HomeCloud account resources (queues, buckets, IAM, functions, compute) through the console API with SigV1 Access Keys.
---

# HomeCloud Provider

The HomeCloud Terraform / OpenTofu provider talks to `console.{apex}/api/v1` with
**SigV1 Access Keys**. It does **not** manage K3s, Helm, or data-plane bytes.

Until this provider is listed on the Terraform Registry, build from source and
use `provider_installation.dev_overrides` (skip `terraform init`). See the
repository README.

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

Set `HC_ACCESS_KEY_ID` and `HC_SECRET_ACCESS_KEY` (User-bound Access Key).
Do not put secrets in `.tf` files. Optional: `HC_APEX`, `HC_ACCOUNT_ID`, `HC_ENDPOINT`.

IAM mutations need a console role of **owner or admin**. Unmapped Service Account
console routes return `403 iam.management_sa_not_enabled`.

## Schema

### Optional

- `access_key` (String, Sensitive) Access Key ID. Env: `HC_ACCESS_KEY_ID`.
- `secret_key` (String, Sensitive) Secret Access Key. Env: `HC_SECRET_ACCESS_KEY`.
- `apex` (String) Platform apex. Default `holab.abrdns.com`. Env: `HC_APEX`.
- `endpoint` (String) Override console base URL (tests). Env: `HC_ENDPOINT`.
- `account_id` (String) Account UUID. Default from Access Key whoami. Env: `HC_ACCOUNT_ID`.
