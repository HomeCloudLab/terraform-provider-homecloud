---
page_title: "homecloud_iam_role Resource - homecloud"
subcategory: "IAM"
description: |-
  IAM role. Attach policies with homecloud_iam_policy_attachment.
---

# homecloud_iam_role

IAM role. Requires owner/admin. Attach managed/custom policies with
`homecloud_iam_policy_attachment`.

## Schema

### Required

- `name` (String)

### Optional

- `description` (String)
- `trust_document` (String) Assume-role trust JSON. Defaults to the account `functions` service account.

### Read-Only

- `id` (String)
- `arn` (String)
