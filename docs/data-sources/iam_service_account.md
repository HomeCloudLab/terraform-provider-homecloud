---
page_title: "homecloud_iam_service_account Data Source - homecloud"
subcategory: "IAM"
description: |-
  Built-in IAM service account by name (for example functions).
---

# homecloud_iam_service_account

Look up a built-in service account by name (`functions`, …). Use `.id` as
`principal_id` on `homecloud_iam_policy_attachment`.

## Schema

### Required

- `name` (String)

### Read-Only

- `id` (String)
- `arn` (String)
