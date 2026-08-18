---
page_title: "homecloud_ir_repository Resource - homecloud"
subcategory: "Registry"
description: |-
  Image Registry repository. Image tags are not Terraform resources.
---

# homecloud_ir_repository

IR repository (`POST /api/v1/accounts/{id}/registry/repositories`). Tags are not Terraform resources.

## Schema

### Required

- `name` (String)

### Read-Only

- `id` (String)
- `iam_arn` (String)
- `status` (String)
- `zot_namespace` (String)
- `image_ref_prefix` (String)
