---
page_title: "homecloud_mdb_user Resource - homecloud"
subcategory: "Database"
description: |-
  Managed database user. password is write-only.
---

# homecloud_mdb_user

Managed application user on an MDB instance. Import id: `instance_name/username`.
`password` is write-only and never on GET.

## Schema

### Required

- `instance_name` (String)
- `username` (String)

### Optional

- `password` (String, Sensitive, Write-only)
- `role` (String)

### Read-Only

- `id` (String)
- `iam_arn` (String)
- `phase` (String)
