---
page_title: "homecloud_application Resource - homecloud"
subcategory: "Applications"
description: |-
  Application spec. Create stays draft — no provision or deploy.
---

# homecloud_application

Application spec (`POST /api/v1/accounts/{id}/applications`). Create stays `draft` —
no provision, deploy, scale, or YAML apply. Import by slug.

## Schema

### Required

- `name` (String)
- `slug` (String)

### Optional

- `template` (String) `api-only` (default), `fullstack`, `static-site`, or `worker`.
- `project_id` (String)

### Read-Only

- `id` (String)
- `iam_arn` (String)
- `status` (String)
