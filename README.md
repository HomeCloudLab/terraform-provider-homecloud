# terraform-provider-homecloud

User-facing Terraform / OpenTofu provider for HomeCloud account resources.
Talks to `console.{apex}/api/v1` with **SigV1 Access Keys** (ADR-049). It does **not**
manage K3s, Helm, or data-plane bytes.

The provider is **not** on the Terraform Registry yet. Build it locally and use
`dev_overrides` (see below). **User guide (all resources, examples, import):**
[`docs/guides/getting-started.md`](docs/guides/getting-started.md) · Hebrew
[`README.he.md`](README.he.md). Registry listing still needs a HashiCorp
publisher + GPG — see [`PUBLISHING.md`](PUBLISHING.md).

Skip `terraform init` until the listing is live.

## P1 / P1b / P2 resources

| Resource | API |
|----------|-----|
| `homecloud_mq_queue` | `/api/v1/accounts/{id}/queues` |
| `homecloud_so_bucket` | `/api/v1/accounts/{id}/storage/buckets` |
| `homecloud_secret` | `/api/v1/accounts/{id}/secrets` |
| `data.homecloud_account` | whoami + `GET /accounts/{id}` |
| `homecloud_iam_policy` | `/api/v1/accounts/{id}/iam/policies` |
| `homecloud_iam_role` | `/api/v1/accounts/{id}/iam/roles` |
| `homecloud_iam_policy_attachment` | `/iam/principals/attachments` |
| `data.homecloud_iam_service_account` | `GET /iam/service-accounts/{name}` |

Bucket schema is `name` only. Versioning / lifecycle / website are later sibling
resources, not attributes on `homecloud_so_bucket`. There is no `region` until
the control plane defines one.

`homecloud_secret.values` is **write-only** (Terraform 1.11+). GET never returns
payloads; they are not stored in Terraform state.

IAM resources require a User-bound Access Key whose console role is **owner or
admin** (`iam.manage`). Policy `arn` is already IAM-canonical — there is no
separate `iam_arn`. Attachments use `principal_id` (UUID), e.g. `data.homecloud_iam_service_account.functions.id`.
Document JSON uses `Version = "2026-07-24"` (not AWS `2012-10-17`). Example:
`examples/iam`.

## P3 MDB / Redis

| Resource | API |
|----------|-----|
| `homecloud_mdb_instance` | `/api/v1/accounts/{id}/databases` |
| `homecloud_mdb_user` | `/databases/{instance}/users` |
| `homecloud_redis_instance` | `/api/v1/accounts/{id}/caches` |

Create waits until `status=active` (or errors on `failed`). GET is by name or UUID.
`iam_arn` is `arn:homecloud:mdb::{account}:instance/{name}` and
`arn:homecloud:redis::{account}:cache/{name}`. User `password` is write-only and
never on GET. Example: `examples/mdb`. Redis credentials stay in the
`credentials_secret` HomeCloud secret, not in the Redis resource.

## P4 Functions / IR / Domains

| Resource | API |
|----------|-----|
| `homecloud_function` | `/api/v1/accounts/{id}/functions` |
| `homecloud_function_url` | `.../functions/{name}/url/enable` |
| `homecloud_ir_repository` | `/api/v1/accounts/{id}/registry/repositories` |
| `homecloud_domain` | `/api/v1/accounts/{id}/domains` |

`homecloud_function` is spec-only: no workspace files, deploys, or invoke. Function **delete** needs an owner/admin Access Key (`function.delete`). Create/update is developer. Example: `examples/p4`. Domain create stays `pending_verification` until you add the TXT record out of band. IR image tags are not Terraform resources.

## P5 Compute / SSH / Applications

| Resource | API |
|----------|-----|
| `homecloud_compute_machine` | `/api/v1/accounts/{id}/compute/machines` |
| `homecloud_ssh_key` | `/api/v1/accounts/{id}/compute/ssh-keys` |
| `homecloud_application` | `/api/v1/accounts/{id}/applications` |

`homecloud_compute_machine` waits on `GET .../operations/{id}` until `SUCCEEDED` or `FAILED`. `homecloud_application` is spec-only (`draft` — no provision/deploy). `homecloud_ssh_key.private_key` is sensitive, returned once on create, never on GET. Example: `examples/p5`. Machine acc tests also need `HC_TF_ACC_COMPUTE=1` (provisions a VM).

## Configure

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

Environment: `HC_ACCESS_KEY_ID`, `HC_SECRET_ACCESS_KEY`, `HC_APEX`, optional `HC_ACCOUNT_ID` / `HC_ENDPOINT`.

Create a dedicated IAM user (`developer` or `admin`) and an Access Key bound to that user,
or a Service Account key with IAM policies that allow the mapped actions (queue/bucket/secret
Create/Delete/Get, plus `secrets:PutSecretValue` to set values). Unmapped console routes still return `403 iam.management_sa_not_enabled`.

## Local run (Windows)

```powershell
cd terraform-provider-homecloud
go build -o terraform-provider-homecloud.exe .
Copy-Item dev.tfrc.example dev.tfrc
# Edit dev.tfrc: set the path to this directory (forward slashes are fine).
$env:TF_CLI_CONFIG_FILE = "$PWD\dev.tfrc"
$env:HC_ACCESS_KEY_ID = "HCAK..."
$env:HC_SECRET_ACCESS_KEY = "..."
$env:HC_APEX = "holab.abrdns.com"

cd examples\mvp
# Skip `terraform init` — overrides do not use the Registry (init will 404).
terraform apply -var="name_suffix=$env:USERNAME"
terraform plan    # expect: No changes
terraform destroy -var="name_suffix=$env:USERNAME"
```

IAM example (`examples/iam`) needs an **owner/admin** Access Key. Same `TF_CLI_CONFIG_FILE`; skip `terraform init`.

P4 example (`examples/p4`) also needs owner/admin to **destroy** functions (`function.delete`). Skip `terraform init`.

P5 example (`examples/p5`) creates an SSH key and a draft application. Uncomment the machine resource only if you want a VM. Skip `terraform init`.

`terraform apply` warns that development overrides are in effect. That is expected.

## Develop

```bash
go test ./...
go build -o terraform-provider-homecloud
```

On Windows the binary is `terraform-provider-homecloud.exe` (`make build`).

Acceptance tests (live console):

```bash
export TF_ACC=1 HC_ACCESS_KEY_ID=... HC_SECRET_ACCESS_KEY=... HC_APEX=holab.abrdns.com
go test ./internal/provider -count=1 -timeout 20m
```

IAM acc tests also need `HC_TF_ACC_IAM=1` and an owner/admin Access Key (`iam.manage`).
MDB/Redis acc tests also need `HC_TF_ACC_MDB=1` (create waiters can take several minutes).
Function / IR / domain acc tests also need `HC_TF_ACC_P4=1` (function delete needs owner/admin).
SSH key / application acc tests also need `HC_TF_ACC_P5=1`. Compute machine acc tests also need `HC_TF_ACC_COMPUTE=1`.
