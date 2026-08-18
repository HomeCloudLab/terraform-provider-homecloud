---
page_title: "homecloud_function_url Resource - homecloud"
subcategory: "Functions"
description: |-
  HTTP Function URL. Delete disables the URL.
---

# homecloud_function_url

Enable the data-plane Function URL. Import by function name.

## Schema

### Required

- `function_name` (String)

### Optional

- `public_url_enabled` (Boolean)

### Read-Only

- `id` (String)
- `function_url` (String)
