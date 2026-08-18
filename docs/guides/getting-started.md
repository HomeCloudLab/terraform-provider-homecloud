# Terraform / OpenTofu for HomeCloud

User guide for the `homecloud` provider (`source = "homecloudlab/homecloud"`).
Hebrew: [`README.he.md`](../../README.he.md) in this repo, or
[`terraform.he.md`](https://github.com/HomeCloudLab/homecloud-infra/blob/main/docs/guides/terraform.he.md).

This provider manages **account resources** on `console.{apex}/api/v1` with
**SigV1 Access Keys**. It does **not** manage the homelab itself (K3s, Helm,
Traefik, GitOps).

The provider is **not on the Terraform Registry yet**. Build it from source and
skip `terraform init` until it is listed (see [Install](#install)).

Repo: [`terraform-provider-homecloud`](https://github.com/HomeCloudLab/terraform-provider-homecloud)
(GitHub may show a rename hint; keep the `terraform-provider-*` name for Registry).

---

## Auth

Create a **User-bound Access Key** in the console (IAM → Access keys). Put the
secret in the environment — never in `.tf` files.

| Variable | Meaning |
|----------|---------|
| `HC_ACCESS_KEY_ID` | Access Key ID |
| `HC_SECRET_ACCESS_KEY` | Secret |
| `HC_APEX` | Platform apex (default `holab.abrdns.com`) |
| `HC_ACCOUNT_ID` | Optional account UUID (default: whoami) |
| `HC_ENDPOINT` | Optional console URL override (tests) |

IAM create/update/delete needs a console role of **owner or admin**. Function
**delete** also needs owner/admin. Queue/bucket/secret Create/Delete/Get can use
a Service Account key with the matching IAM actions. Unmapped SA console routes
return `403 iam.management_sa_not_enabled`.

---

## Install (local, until Registry)

```powershell
cd terraform-provider-homecloud
go build -o terraform-provider-homecloud.exe .
Copy-Item dev.tfrc.example dev.tfrc
# Edit dev.tfrc: absolute path to this directory (forward slashes are fine).
$env:TF_CLI_CONFIG_FILE = "$PWD\dev.tfrc"
$env:HC_ACCESS_KEY_ID = "HCAK..."
$env:HC_SECRET_ACCESS_KEY = "..."
$env:HC_APEX = "holab.abrdns.com"
```

`dev_overrides` ignore the Registry. **`terraform init` will 404** — skip it.
`terraform apply` warns that development overrides are in effect. That is expected.

```hcl
terraform {
  required_providers {
    homecloud = {
      source = "homecloudlab/homecloud"
    }
  }
}

provider "homecloud" {
  apex = "holab.abrdns.com"
}
```

OpenTofu works the same (`tofu apply`).

---

## Resource catalog

Import almost everything by **name** (UUID also works where the API accepts it).
After import, `terraform plan` should be empty if the HCL matches the live object.

### `data.homecloud_account`

Current account from the Access Key (or explicit `id`).

```hcl
data "homecloud_account" "this" {}

output "account_number" {
  value = data.homecloud_account.this.account_number
}
```

Read-only: `id`, `short_id`, `account_number`, `name`, `slug`, `status`.

### `homecloud_mq_queue`

Managed MQ queue. Optional: `max_message_size`, `visibility_timeout_seconds`,
`max_receive_count`, `message_retention_seconds`. Computed: `iam_arn`, `queue_url`,
`status`.

```hcl
resource "homecloud_mq_queue" "jobs" {
  name = "jobs"
}

# terraform import homecloud_mq_queue.jobs jobs
```

### `homecloud_so_bucket`

Object Storage bucket. Schema is **`name` only**. Versioning, lifecycle, website,
and bucket policy stay in the console — they are not attributes on this resource.

```hcl
resource "homecloud_so_bucket" "assets" {
  name = "assets"
}

# terraform import homecloud_so_bucket.assets assets
```

### `homecloud_secret`

Secret metadata. `values` is **write-only** (Terraform 1.11+). GET never returns
payloads; they are not stored in state.

```hcl
resource "homecloud_secret" "db" {
  name = "db-creds"
  values = {
    username = "app"
    password = var.db_password
  }
}

# terraform import homecloud_secret.db db-creds
```

Import does not restore `values`. Set them again on the next apply if you need
Terraform to own the payload.

### IAM

Requires owner/admin. Policy JSON `Version` is **`2026-07-24`** (not AWS
`2012-10-17`). Policy `arn` is already IAM-canonical (no separate `iam_arn`).

```hcl
data "homecloud_iam_service_account" "functions" {
  name = "functions"
}

resource "homecloud_iam_policy" "mq" {
  name        = "mq-send"
  description = "Send to all queues"
  document = jsonencode({
    Version = "2026-07-24"
    Statement = [{
      Effect   = "Allow"
      Action   = ["mq:SendMessage"]
      Resource = "arn:homecloud:mq::${data.homecloud_account.this.account_number}:queue/*"
    }]
  })
}

resource "homecloud_iam_role" "ci" {
  name = "ci"
}

resource "homecloud_iam_policy_attachment" "functions_mq" {
  policy_arn     = homecloud_iam_policy.mq.arn
  principal_type = "service_account" # user | role | service_account
  principal_id   = data.homecloud_iam_service_account.functions.id
}

# terraform import homecloud_iam_policy.mq mq-send
# terraform import homecloud_iam_role.ci ci
# terraform import homecloud_iam_policy_attachment.functions_mq service_account:<sa-uuid>:<policy-arn>
```

`trust_document` on a role is optional JSON (defaults to the account `functions`
service account). Attachments use **principal UUID**, not the display name.

Worked example: `examples/iam`.

### Managed database (`homecloud_mdb_instance` / `homecloud_mdb_user`)

Create **waits** until `status=active` (fails on `failed`). Engine:
`postgresql`, `mysql`, or `mongodb`. Optional: `instance_class`, `engine_version`,
`storage_gi`, `database`, `owner`.

User `password` is write-only. Import users as `instance_name/username`.

```hcl
resource "homecloud_mdb_instance" "app" {
  name           = "app-pg"
  engine         = "postgresql"
  instance_class = "micro"
}

resource "homecloud_mdb_user" "ci" {
  instance_name = homecloud_mdb_instance.app.name
  username      = "ci"
  password      = var.db_password
  role          = "readwrite"
}

# terraform import homecloud_mdb_instance.app app-pg
# terraform import homecloud_mdb_user.ci app-pg/ci
```

Worked example: `examples/mdb`.

### Redis (`homecloud_redis_instance`)

Create waits until `status=active`. Password lives in `credentials_secret`
(a HomeCloud secret), **not** on this resource.

```hcl
resource "homecloud_redis_instance" "cache" {
  name           = "app-redis"
  instance_class = "micro"
}

# terraform import homecloud_redis_instance.cache app-redis
```

### Functions (`homecloud_function` / `homecloud_function_url`)

**Spec only:** runtime/handler/memory/timeout. Terraform does **not** manage
workspace files, deploys, layers, or invoke. Delete requires owner/admin
(`function.delete`).

```hcl
resource "homecloud_function" "hello" {
  name    = "hello"
  handler = "main.handler"
}

resource "homecloud_function_url" "hello" {
  function_name      = homecloud_function.hello.name
  public_url_enabled = false
}

# terraform import homecloud_function.hello hello
# terraform import homecloud_function_url.hello hello
```

Worked example: `examples/p4`.

### Image Registry (`homecloud_ir_repository`)

Repository record only. **Image tags are not Terraform resources.**

```hcl
resource "homecloud_ir_repository" "app" {
  name = "app"
}

# terraform import homecloud_ir_repository.app app
```

### Domain (`homecloud_domain`)

Create stays `pending_verification` until you add the TXT record **out of band**.
Terraform does not wait for verify.

```hcl
resource "homecloud_domain" "site" {
  hostname = "app.example.com"
  dns_mode = "external" # or homecloud
}

# terraform import homecloud_domain.site app.example.com
```

### Compute (`homecloud_compute_machine` / `homecloud_ssh_key`)

Machine create waits on the Operations API (`SUCCEEDED` / `FAILED`).
Does **not** manage firewall, volumes, exec, or files.

`homecloud_ssh_key.private_key` is sensitive, returned **once on create**, never
on GET. Save it from apply output if you need it.

```hcl
resource "homecloud_ssh_key" "ci" {
  name = "ci"
}

resource "homecloud_compute_machine" "web" {
  name          = "web-1"
  machine_class = "basic" # or standard
  image_id      = "ubuntu-24.04"
  ssh_key_ids   = [homecloud_ssh_key.ci.id]
}

# terraform import homecloud_ssh_key.ci ci
# terraform import homecloud_compute_machine.web web-1
```

`machine_class` is sent to the API as JSON `class`. Images: `ubuntu-24.04`,
`debian-12`, `almalinux-9`.

Worked example: `examples/p5` (machine resource is commented — it provisions a VM).

### Application (`homecloud_application`)

**Spec / `draft` only.** No provision, deploy, scale, or YAML apply.

```hcl
resource "homecloud_application" "api" {
  name     = "Shop"
  slug     = "shop"
  template = "api-only" # fullstack | static-site | worker
}

# terraform import homecloud_application.api shop
```

Import by **slug**.

---

## What stays in the console

These are **not** Terraform resources (nested AWS-style siblings, later):

- Bucket versioning / lifecycle / website / policy
- Function code, versions, deploys, layers, invoke
- IR tags and lifecycle rules
- Domain DNS records and verify
- Machine firewall, disks, snapshots, agent exec
- Application provision / deploy / HPA / custom domains

The console stays fully writable. Drift is `terraform plan`, `import`, or
`lifecycle.ignore_changes`. There is no `managed_by=terraform` lock.

---

## Worked examples in the provider repo

| Directory | What it creates |
|-----------|-----------------|
| `examples/mvp` | Queue + bucket + account data |
| `examples/secret` | Secret with write-only values |
| `examples/iam` | Policy + role + SA attachment (owner/admin) |
| `examples/mdb` | PostgreSQL + user + Redis |
| `examples/p4` | Function + URL + IR repo + domain |
| `examples/p5` | SSH key + draft application (machine commented) |

---

## Registry listing

Docs, `terraform-registry-manifest.json`, and GoReleaser live in the provider
repo. A live listing on `registry.terraform.io` still needs a HashiCorp
publisher for namespace `homecloudlab` and a GPG-signed GitHub Release. See
[`PUBLISHING.md`](https://github.com/HomeCloudLab/terraform-provider-homecloud/blob/main/PUBLISHING.md)
in the provider repo.
