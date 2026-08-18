---
page_title: "homecloud_account Data Source - homecloud"
subcategory: "Account"
description: |-
  Current HomeCloud account from Access Key whoami.
---

# homecloud_account

Current account (from Access Key whoami, or explicit `id`).

## Schema

### Optional

- `id` (String) Account UUID. Defaults to the Access Key's account.

### Read-Only

- `short_id` (String)
- `account_number` (String)
- `name` (String)
- `slug` (String)
- `status` (String)
