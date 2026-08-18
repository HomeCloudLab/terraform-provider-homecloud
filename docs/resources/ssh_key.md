---
page_title: "homecloud_ssh_key Resource - homecloud"
subcategory: "Compute"
description: |-
  Compute SSH key. private_key is returned once on create and never on GET.
---

# homecloud_ssh_key

Account SSH key. `private_key` is sensitive, returned once on create, never on GET.

## Schema

### Required

- `name` (String)

### Read-Only

- `id` (String)
- `iam_arn` (String)
- `fingerprint` (String)
- `key_type` (String)
- `public_key` (String)
- `private_key` (String, Sensitive)
