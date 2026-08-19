# Publishing to the Terraform Registry

The provider source string is `homecloudlab/homecloud`
(`registry.terraform.io/homecloudlab/homecloud`).

GitHub repo (must stay this name — HashiCorp requires the prefix):
https://github.com/HomeCloudLab/terraform-provider-homecloud

The repo is **public**. CI and release workflows live in `.github/workflows/`.
Releases are signed with the GPG key whose public half is
[`docs/signing-key.asc`](docs/signing-key.asc).

Fingerprint: `8A98220ACEEF8018FCEDCEBE4B8BCFED1A615BA9`

GitHub Releases: signed **v0.1.0** (first listing) and **v0.1.1** (CLI credentials file):
https://github.com/HomeCloudLab/terraform-provider-homecloud/releases
(`SHA256SUMS` + `SHA256SUMS.sig` + zips + manifest).

GitHub Actions secrets already set on the repo: `GPG_PRIVATE_KEY`, `PASSPHRASE`.
`.github/workflows/ci.yml` and `release.yml` are on `main`. Later tags `v*`
reuse those workflows.

## Already in this repo

- `LICENSE` — MPL-2.0 (OSI-approved; Registry requires a license)
- `docs/` — Registry documentation (English)
- `README.he.md` — Hebrew overview
- `terraform-registry-manifest.json` — protocol 6.0
- `.goreleaser.yml` — zip + SHA256SUMS + GPG-signed checksums
- `.github/workflows/ci.yml` and `release.yml`

## Listed (2026-08-19)

https://registry.terraform.io/providers/homecloudlab/homecloud/latest

`terraform init` installs the latest listed version (**v0.1.1**). Existing lock
files need `terraform init -upgrade`. Terraform CLI reports the GPG key as
**self-signed** (`key ID 4B8BCFED1A615BA9`) — expected for a community provider.

Later versions: tag `v*` on this repo; GoReleaser + Registry pick them up.

Do **not** rename the GitHub repo away from `terraform-provider-homecloud`.
