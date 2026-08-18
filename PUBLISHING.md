# Publishing to the Terraform Registry

The provider source string is already `homecloudlab/homecloud`
(`registry.terraform.io/homecloudlab/homecloud`). Console copy-HCL uses that
source. The listing is **not live** until the steps below are done.

## Already in this repo

- `docs/` — Registry documentation (English)
- `README.he.md` — Hebrew overview
- `terraform-registry-manifest.json` — protocol 6.0
- `.goreleaser.yml` — zip + SHA256SUMS + GPG-signed checksums
- GitHub repo name `terraform-provider-homecloud` (required by HashiCorp)

## Still needs a human (cannot be faked)

1. **HashiCorp Terraform Registry** — sign in, claim namespace `homecloudlab`,
   and publish this GitHub repository.
2. **GPG key** — create a signing key, upload the public key to the Registry
   publisher, and set `GPG_FINGERPRINT` when cutting a release.
3. **GitHub Release** — tag `v0.1.0` (or later) and run GoReleaser so the
   release includes `*_SHA256SUMS` + `*_SHA256SUMS.sig` + zips + manifest.
4. **Optional CI** — a release workflow needs the GitHub `workflow` token
   scope (`gh auth refresh --scopes repo,workflow`). Until then, run
   GoReleaser locally:

   ```bash
   export GPG_FINGERPRINT=...
   goreleaser release --clean
   ```

Do not rename the GitHub repo away from `terraform-provider-homecloud`; the
Registry requires that prefix.

Until the listing is live, keep using `dev_overrides` and skip `terraform init`.
