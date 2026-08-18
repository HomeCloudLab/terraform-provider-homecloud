---
page_title: "homecloud_so_bucket Resource - homecloud"
subcategory: "Storage"
description: |-
  Object Storage bucket. Schema is name only.
---

# homecloud_so_bucket

Managed SO bucket (`POST /api/v1/accounts/{id}/storage/buckets`). Schema is **name only**.
Versioning, lifecycle, and website are console sibling settings, not attributes here.

## Example Usage

```terraform
resource "homecloud_so_bucket" "assets" {
  name = "assets"
}
```

## Schema

### Required

- `name` (String) Bucket name. Changing this forces a new resource.

### Read-Only

- `id` (String)
- `iam_arn` (String)
- `status` (String)
- `created_at` (String)
