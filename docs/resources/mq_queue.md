---
page_title: "homecloud_mq_queue Resource - homecloud"
subcategory: "MQ"
description: |-
  Managed MQ queue.
---

# homecloud_mq_queue

Managed MQ queue (`POST /api/v1/accounts/{id}/queues`). Import by name or UUID.

## Example Usage

```terraform
resource "homecloud_mq_queue" "jobs" {
  name = "jobs"
}
```

## Schema

### Required

- `name` (String) Queue name. Changing this forces a new resource.

### Optional

- `max_message_size` (Number)
- `visibility_timeout_seconds` (Number)
- `max_receive_count` (Number)
- `message_retention_seconds` (Number)

### Read-Only

- `id` (String)
- `iam_arn` (String) `arn:homecloud:mq::{account}:queue/{name}`
- `status` (String)
- `queue_url` (String)
