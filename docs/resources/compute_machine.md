---
page_title: "homecloud_compute_machine Resource - homecloud"
subcategory: "Compute"
description: |-
  IaaS machine. Create waits on the Operations API.
---

# homecloud_compute_machine

Compute machine. Create waits for `GET .../operations/{id}` until `SUCCEEDED` or `FAILED`.
Does not manage firewall, volumes, exec, or files.

## Schema

### Required

- `name` (String)
- `machine_class` (String) `basic` or `standard`. Sent as JSON `class`.
- `image_id` (String)

### Optional

- `region_code` (String)
- `az_code` (String)
- `ssh_key_ids` (List of String)

### Read-Only

- `id` (String)
- `iam_arn` (String)
- `status` (String)
- `public_ipv4` (String)
- `operation_id` (String)
