---
page_title: "homecloud_redis_instance Resource - homecloud"
subcategory: "Cache"
description: |-
  Managed Redis cache. Password lives in credentials_secret, not this resource.
---

# homecloud_redis_instance

Managed Redis (`POST /api/v1/accounts/{id}/caches`). Create waits until `status=active`.

## Schema

### Required

- `name` (String)

### Optional

- `instance_class` (String)
- `redis_version` (String)

### Read-Only

- `id` (String)
- `iam_arn` (String)
- `status` (String)
- `endpoint` (String)
- `internal_endpoint` (String)
- `port` (Number)
- `credentials_secret` (String)
