---
page_title: "homecloud_function Resource - homecloud"
subcategory: "Functions"
description: |-
  Function spec. Does not manage workspace files or deploys.
---

# homecloud_function

Managed function spec. Does **not** manage IDE workspace files or deploys.
Delete requires owner/admin (`function.delete`).

## Schema

### Required

- `name` (String)

### Optional

- `runtime` (String)
- `handler` (String)
- `memory_limit_mb` (Number)
- `timeout_seconds` (Number)

### Read-Only

- `id` (String)
- `iam_arn` (String)
- `status` (String)
- `invoke_url` (String)
