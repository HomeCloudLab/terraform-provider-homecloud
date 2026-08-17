# terraform-provider-homecloud

User-facing Terraform / OpenTofu provider for HomeCloud account resources.
Talks to `console.{apex}/api/v1` with **SigV1 Access Keys** (ADR-049). It does **not**
manage K3s, Helm, or data-plane bytes.

## P1 resources

| Resource | API |
|----------|-----|
| `homecloud_mq_queue` | `/api/v1/accounts/{id}/queues` |
| `homecloud_so_bucket` | `/api/v1/accounts/{id}/storage/buckets` |
| `data.homecloud_account` | whoami + `GET /accounts/{id}` |

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
  access_key = var.hc_access_key
  secret_key = var.hc_secret_key
  apex       = "holab.abrdns.com"
}
```

Environment: `HC_ACCESS_KEY_ID`, `HC_SECRET_ACCESS_KEY`, `HC_APEX`, optional `HC_ACCOUNT_ID` / `HC_ENDPOINT`.

Create a dedicated IAM user (`developer` or `admin`) and an Access Key bound to that user.
Service Account keys are rejected on mutating console routes until P0b.

## Develop

```bash
go test ./...
go build -o terraform-provider-homecloud
```

Acceptance tests (live console):

```bash
export TF_ACC=1 HC_ACCESS_KEY_ID=... HC_SECRET_ACCESS_KEY=... HC_APEX=holab.abrdns.com
go test ./internal/provider -count=1 -timeout 20m
```
