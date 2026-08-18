# GitHub OIDC for Terraform

Exchange a GitHub Actions OIDC JWT for short-lived HomeCloud SigV1 credentials.
No long-lived `HC_SECRET_ACCESS_KEY` in GitHub Secrets.

The sample workflow is **documentation** (`workflow.yml` in this folder). Copy
it into **your** infra repo as `.github/workflows/terraform.yml`. The provider
repo itself already has CI (`ci.yml`) and signed-release (`release.yml`)
workflows; this folder is not that pipeline.

## 1. IAM role trust

Create a role (console owner/admin, or a User-bound Terraform apply) with trust
like:

```json
{
  "Version": "2026-07-24",
  "Statement": [{
    "Effect": "Allow",
    "Principal": {
      "Federated": "arn:homecloud:iam::ACCOUNT_NUMBER:oidc-provider/token.actions.githubusercontent.com"
    },
    "Action": "sts:AssumeRole",
    "Condition": {
      "StringEquals": {
        "token.actions.githubusercontent.com:aud": "https://console.holab.abrdns.com"
      },
      "StringLike": {
        "token.actions.githubusercontent.com:sub": "repo:YOUR_ORG/YOUR_REPO:*"
      }
    }
  }]
}
```

Attach managed policies for the resources CI manages (`MQAdmin`,
`SOBucketAdmin`, `SecretsAdmin`). Assumed-role sessions cannot call unmapped
console routes (`403 iam.management_role_not_enabled`).

## 2. GitHub Actions

Copy `workflow.yml` into your infra repo as `.github/workflows/terraform.yml`.
Set repository variable `HC_ROLE_ARN` to the role ARN.

```hcl
provider "homecloud" {
  apex     = "holab.abrdns.com"
  role_arn = var.ci_role_arn # or env HC_ROLE_ARN
}
```

In Actions the provider reads `ACTIONS_ID_TOKEN_REQUEST_URL` /
`ACTIONS_ID_TOKEN_REQUEST_TOKEN` when `HC_WEB_IDENTITY_TOKEN` is unset.
Request audience `https://console.{apex}` (`HC_OIDC_AUDIENCE`).
