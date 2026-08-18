# Publishing to the Terraform Registry

The provider source string is `homecloudlab/homecloud`
(`registry.terraform.io/homecloudlab/homecloud`).

GitHub repo (must stay this name — HashiCorp requires the prefix):
https://github.com/HomeCloudLab/terraform-provider-homecloud

The repo is **public**. CI and release workflows live in `.github/workflows/`.
Releases are signed with the GPG key whose public half is
[`docs/signing-key.asc`](docs/signing-key.asc).

Fingerprint: `8A98220ACEEF8018FCEDCEBE4B8BCFED1A615BA9`

GitHub Releases already has a signed **v0.1.0**:
https://github.com/HomeCloudLab/terraform-provider-homecloud/releases/tag/v0.1.0
(`SHA256SUMS` + `SHA256SUMS.sig` + zips + manifest).

GitHub Actions secrets already set on the repo: `GPG_PRIVATE_KEY`, `PASSPHRASE`.
Later tags `v*` need `.github/workflows/release.yml` on `main` (pushing that
file requires `gh auth refresh --scopes repo,workflow`).

## Already in this repo

- `LICENSE` — MPL-2.0 (OSI-approved; Registry requires a license)
- `docs/` — Registry documentation (English)
- `README.he.md` — Hebrew overview
- `terraform-registry-manifest.json` — protocol 6.0
- `.goreleaser.yml` — zip + SHA256SUMS + GPG-signed checksums
- `.github/workflows/ci.yml` and `release.yml`

## Last human step (cannot be done from this machine)

HashiCorp has no API that replaces the Registry website. Someone with access
to the **HomeCloudLab** GitHub org (owner: David-Abravanel; repo admin is
enough for the repo itself, but **org OAuth** for the Terraform Registry app
usually needs an org owner) must:

1. Open https://registry.terraform.io and **Sign in with GitHub**.
2. Grant the Terraform Registry GitHub App access to the **HomeCloudLab**
   organization if GitHub asks.
3. **User Settings → Signing Keys** — paste [`docs/signing-key.asc`](docs/signing-key.asc)
   onto the `homecloudlab` namespace.
4. **Publish → Provider** — choose `HomeCloudLab/terraform-provider-homecloud`.
5. Accept the Registry terms.

Do **not** rename the GitHub repo away from `terraform-provider-homecloud`.

Until that listing is live, keep using `dev_overrides` and skip `terraform init`.
