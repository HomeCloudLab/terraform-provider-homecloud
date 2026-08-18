---
page_title: "homecloud_iam_policy Resource - homecloud"
subcategory: "IAM"
description: |-
  Custom IAM policy. Requires owner/admin Access Key.
---

# homecloud_iam_policy

Custom IAM policy. Console role must be **owner or admin**. `document` is a JSON string
(use `jsonencode`). Policy JSON `Version` is `2026-07-24`, not AWS `2012-10-17`.

## Example Usage

```terraform
resource "homecloud_iam_policy" "mq" {
  name = "mq-send"
  document = jsonencode({
    Version = "2026-07-24"
    Statement = [{
      Effect   = "Allow"
      Action   = ["mq:SendMessage"]
      Resource = "*"
    }]
  })
}
```

## Schema

### Required

- `name` (String)
- `document` (String) IAM policy document JSON.

### Optional

- `description` (String)

### Read-Only

- `id` (String)
- `arn` (String)
