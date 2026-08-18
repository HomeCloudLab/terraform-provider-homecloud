---
page_title: "homecloud_secret Resource - homecloud"
subcategory: "Secrets"
description: |-
  Secret metadata. Values are write-only and never returned on GET.
---

# homecloud_secret

Secret (`POST /api/v1/accounts/{id}/secrets`). `values` is write-only (Terraform 1.11+).
GET never returns payloads; they are not stored in state.

## Example Usage

```terraform
resource "homecloud_secret" "db" {
  name = "db-creds"
  values = {
    username = "app"
    password = var.db_password
  }
}
```

## Schema

### Required

- `name` (String)

### Optional

- `values` (Map of String, Write-only, Sensitive)

### Read-Only

- `id` (String)
- `iam_arn` (String)
- `status` (String)
