---
page_title: "homecloud_mdb_instance Resource - homecloud"
subcategory: "Database"
description: |-
  Managed database instance. Create waits until status=active.
---

# homecloud_mdb_instance

Managed database (`POST /api/v1/accounts/{id}/databases`). Create waits until `status=active`.

## Schema

### Required

- `name` (String)
- `engine` (String) `postgresql`, `mysql`, or `mongodb`.

### Optional

- `instance_class` (String)
- `engine_version` (String)
- `storage_gi` (Number)
- `database` (String)
- `owner` (String)

### Read-Only

- `id` (String)
- `iam_arn` (String)
- `status` (String)
- `endpoint` (String)
- `internal_endpoint` (String)
- `port` (Number)
