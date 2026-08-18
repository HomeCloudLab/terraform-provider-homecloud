---
page_title: "homecloud_domain Resource - homecloud"
subcategory: "DNS"
description: |-
  Account domain. Verification stays out of band.
---

# homecloud_domain

BYOD domain. Create stays `pending_verification` until you add the TXT record
out of band. Terraform does not wait for verify.

## Schema

### Required

- `hostname` (String)

### Optional

- `dns_mode` (String) `external` (default) or `homecloud`.

### Read-Only

- `id` (String)
- `iam_arn` (String)
- `status` (String)
- `fqdn` (String)
- `verified` (Boolean)
