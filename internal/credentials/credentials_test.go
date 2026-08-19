package credentials

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProfilesAndLegacyFlat(t *testing.T) {
	dir := t.TempDir()
	multi := filepath.Join(dir, "multi.json")
	if err := os.WriteFile(multi, []byte(`{
  "version": 2,
  "default_profile": "work",
  "profiles": {
    "work": {"access_key_id": "HCAKWORK", "secret_access_key": "swork", "apex": "holab.abrdns.com"},
    "lab": {"access_key_id": "HCAKLAB", "secret_access_key": "slab"}
  }
}`), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := Load(multi, "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "work" || got.AccessKeyID != "HCAKWORK" {
		t.Fatalf("default profile: %+v", got)
	}

	lab, err := Load(multi, "lab")
	if err != nil {
		t.Fatal(err)
	}
	if lab.AccessKeyID != "HCAKLAB" {
		t.Fatalf("lab: %+v", lab)
	}

	flat := filepath.Join(dir, "flat.json")
	if err := os.WriteFile(flat, []byte(`{"access_key_id":"HCAKFLAT","secret_access_key":"sflat"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := Load(flat, "")
	if err != nil {
		t.Fatal(err)
	}
	if f.AccessKeyID != "HCAKFLAT" || f.Name != "default" {
		t.Fatalf("flat: %+v", f)
	}
}

func TestApplyFileFallbackSkipsFileWhenRoleARNSet(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials")
	if err := os.WriteFile(path, []byte(`{"access_key_id":"HCAKFILE","secret_access_key":"sfile"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOMECLOUD_CREDENTIALS_FILE", path)

	out, err := ApplyFileFallback(Chain{RoleARN: "arn:homecloud:iam::1:role/ci"})
	if err != nil {
		t.Fatal(err)
	}
	if out.AccessKey != "" || out.Source != "oidc" {
		t.Fatalf("OIDC must ignore leftover credentials file: %+v", out)
	}
}

func TestApplyFileFallbackReadsFileWhenNoEnvOrOIDC(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials")
	if err := os.WriteFile(path, []byte(`{
  "default_profile": "dev",
  "profiles": {"dev": {"access_key_id": "HCAKDEV", "secret_access_key": "sdev", "default_account_id": "acc-1"}}
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOMECLOUD_CREDENTIALS_FILE", path)

	out, err := ApplyFileFallback(Chain{})
	if err != nil {
		t.Fatal(err)
	}
	if out.AccessKey != "HCAKDEV" || out.SecretKey != "sdev" || out.AccountID != "acc-1" || out.Source != "file:dev" {
		t.Fatalf("got %+v", out)
	}
}

func TestApplyFileFallbackEnvWinsOverFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials")
	if err := os.WriteFile(path, []byte(`{"access_key_id":"HCAKFILE","secret_access_key":"sfile"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOMECLOUD_CREDENTIALS_FILE", path)

	out, err := ApplyFileFallback(Chain{AccessKey: "HCAKENV", SecretKey: "senv"})
	if err != nil {
		t.Fatal(err)
	}
	if out.AccessKey != "HCAKENV" || out.Source != "env" {
		t.Fatalf("got %+v", out)
	}
}
