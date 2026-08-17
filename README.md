# terraform-provider-homecloud

User-facing Terraform / OpenTofu provider for HomeCloud account resources.
Talks to `console.{apex}/api/v1` with **SigV1 Access Keys** (ADR-049). It does **not**
manage K3s, Helm, or data-plane bytes.

The provider is **not** on the Terraform Registry yet. Build it locally and use
`dev_overrides` (see below).

## P1 resources

| Resource | API |
|----------|-----|
| `homecloud_mq_queue` | `/api/v1/accounts/{id}/queues` |
| `homecloud_so_bucket` | `/api/v1/accounts/{id}/storage/buckets` |
| `data.homecloud_account` | whoami + `GET /accounts/{id}` |

Bucket schema is `name` only. Versioning / lifecycle / website are later sibling
resources, not attributes on `homecloud_so_bucket`. There is no `region` until
the control plane defines one.

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

Create a dedicated IAM user (`developer` or `admin`) and an Access Key bound to that user.
Service Account keys are rejected on mutating console routes until P0b.

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
