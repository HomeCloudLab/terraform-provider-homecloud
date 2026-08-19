# Changelog

## 0.1.1 (2026-08-19)

Local `terraform apply` after `homecloud configure` — no env vars required.

- Read `~/.homecloud/credentials` (same JSON as the CLI), including named profiles
- `profile` / `HC_PROFILE` / `HOMECLOUD_PROFILE`; file `default_profile`
- Path aliases: `HOMECLOUD_CREDENTIALS_FILE` / `HC_CREDENTIALS_FILE` / `HC_CONFIG_DIR`
- If `role_arn` / `HC_ROLE_ARN` is set and no explicit Access Key, skip the file (leftover credentials on a CI runner must not beat OIDC)
- Incomplete env (only an access key) does not mix a secret from the file

Upgrade an existing workspace: `terraform init -upgrade`.

## 0.1.0 (2026-08-19)

First Terraform Registry listing (`homecloudlab/homecloud`).
