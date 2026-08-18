---
page_title: "homecloud_iam_policy_attachment Resource - homecloud"
subcategory: "IAM"
description: |-
  Attach a policy to a user, role, or service account.
---

# homecloud_iam_policy_attachment

Attach a policy ARN to a principal. Import id: `principal_type:principal_id:policy_arn`.

## Schema

### Required

- `policy_arn` (String)
- `principal_type` (String) `user`, `role`, or `service_account`.
- `principal_id` (String) Principal UUID (for example `data.homecloud_iam_service_account.functions.id`).

### Read-Only

- `id` (String)
