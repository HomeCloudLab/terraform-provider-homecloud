package credentials

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeCreds(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadProfilesAndLegacyFlat(t *testing.T) {
	dir := t.TempDir()
	multi := writeCreds(t, dir, "multi.json", `{
  "version": 2,
  "default_profile": "work",
  "profiles": {
    "work": {"access_key_id": "HCAKWORK", "secret_access_key": "swork", "apex": "holab.abrdns.com"},
    "lab": {"access_key_id": "HCAKLAB", "secret_access_key": "slab"}
  }
}`)

	got, err := Load(multi, "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "work" || got.AccessKeyID != "HCAKWORK" || got.Apex != "holab.abrdns.com" {
		t.Fatalf("default profile: %+v", got)
	}

	lab, err := Load(multi, "lab")
	if err != nil {
		t.Fatal(err)
	}
	if lab.AccessKeyID != "HCAKLAB" {
		t.Fatalf("lab: %+v", lab)
	}

	flat := writeCreds(t, dir, "flat.json", `{"access_key_id":"HCAKFLAT","secret_access_key":"sflat"}`)
	f, err := Load(flat, "")
	if err != nil {
		t.Fatal(err)
	}
	if f.AccessKeyID != "HCAKFLAT" || f.Name != "default" {
		t.Fatalf("flat: %+v", f)
	}
}

func TestLoadTrimsWhitespaceAndApexSlash(t *testing.T) {
	dir := t.TempDir()
	path := writeCreds(t, dir, "trim.json", `{
  "profiles": {
    "default": {
      "access_key_id": "  HCAKTRIM  ",
      "secret_access_key": " secret ",
      "apex": "holab.abrdns.com/",
      "default_account_id": " acc-9 "
    }
  }
}`)
	got, err := Load(path, "default")
	if err != nil {
		t.Fatal(err)
	}
	if got.AccessKeyID != "HCAKTRIM" || got.SecretAccessKey != "secret" {
		t.Fatalf("trim keys: %+v", got)
	}
	if got.Apex != "holab.abrdns.com" || got.DefaultAccountID != "acc-9" {
		t.Fatalf("trim apex/account: %+v", got)
	}
}

func TestLoadUnknownProfile(t *testing.T) {
	dir := t.TempDir()
	path := writeCreds(t, dir, "p.json", `{"profiles":{"default":{"access_key_id":"A","secret_access_key":"B"}}}`)
	_, err := Load(path, "missing")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("want profile not found, got %v", err)
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := writeCreds(t, dir, "bad.json", `{not json`)
	_, err := Load(path, "")
	if err == nil || !strings.Contains(err.Error(), "invalid credentials file") {
		t.Fatalf("want invalid JSON error, got %v", err)
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "nope"), "")
	if err == nil || !os.IsNotExist(err) {
		t.Fatalf("want not exist, got %v", err)
	}
}

func TestPathAliases(t *testing.T) {
	dir := t.TempDir()
	file := writeCreds(t, dir, "creds", `{}`)
	t.Setenv("HOMECLOUD_CREDENTIALS_FILE", file)
	if Path() != file {
		t.Fatalf("HOMECLOUD_CREDENTIALS_FILE: %s", Path())
	}
	t.Setenv("HOMECLOUD_CREDENTIALS_FILE", "")
	t.Setenv("HC_CREDENTIALS_FILE", file)
	if Path() != file {
		t.Fatalf("HC_CREDENTIALS_FILE: %s", Path())
	}
	t.Setenv("HC_CREDENTIALS_FILE", "")
	cfg := filepath.Join(dir, "cfg")
	if err := os.Mkdir(cfg, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HC_CONFIG_DIR", cfg)
	want := filepath.Join(cfg, "credentials")
	if Path() != want {
		t.Fatalf("HC_CONFIG_DIR: got %s want %s", Path(), want)
	}
}

func TestProfileNameEnv(t *testing.T) {
	t.Setenv("HOMECLOUD_PROFILE", "from-long")
	t.Setenv("HC_PROFILE", "from-short")
	if ProfileName("") != "from-long" {
		t.Fatalf("HOMECLOUD_PROFILE should win: %q", ProfileName(""))
	}
	if ProfileName("explicit") != "explicit" {
		t.Fatalf("explicit should win: %q", ProfileName("explicit"))
	}
}

func TestApplyFileFallbackSkipsFileWhenRoleARNSet(t *testing.T) {
	dir := t.TempDir()
	path := writeCreds(t, dir, "credentials", `{"access_key_id":"HCAKFILE","secret_access_key":"sfile"}`)
	t.Setenv("HOMECLOUD_CREDENTIALS_FILE", path)

	out, err := ApplyFileFallback(Chain{RoleARN: "arn:homecloud:iam::1:role/ci"})
	if err != nil {
		t.Fatal(err)
	}
	if out.AccessKey != "" || out.Source != "oidc" {
		t.Fatalf("OIDC must ignore leftover credentials file: %+v", out)
	}
}

func TestApplyFileFallbackEnvWinsOverRoleARNAndFile(t *testing.T) {
	dir := t.TempDir()
	path := writeCreds(t, dir, "credentials", `{"access_key_id":"HCAKFILE","secret_access_key":"sfile"}`)
	t.Setenv("HOMECLOUD_CREDENTIALS_FILE", path)

	out, err := ApplyFileFallback(Chain{
		AccessKey: "HCAKENV",
		SecretKey: "senv",
		RoleARN:   "arn:homecloud:iam::1:role/ci",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.AccessKey != "HCAKENV" || out.Source != "env" {
		t.Fatalf("explicit keys must beat OIDC and file: %+v", out)
	}
}

func TestApplyFileFallbackDoesNotMixPartialEnvWithFile(t *testing.T) {
	dir := t.TempDir()
	path := writeCreds(t, dir, "credentials", `{"access_key_id":"HCAKFILE","secret_access_key":"sfile"}`)
	t.Setenv("HOMECLOUD_CREDENTIALS_FILE", path)

	out, err := ApplyFileFallback(Chain{AccessKey: "HCAKONLY"})
	if err != nil {
		t.Fatal(err)
	}
	if out.AccessKey != "HCAKONLY" || out.SecretKey != "" || out.Source != "env" {
		t.Fatalf("must not fill secret from file when access key is already set: %+v", out)
	}
}

func TestApplyFileFallbackReadsFileWhenNoEnvOrOIDC(t *testing.T) {
	dir := t.TempDir()
	path := writeCreds(t, dir, "credentials", `{
  "default_profile": "dev",
  "profiles": {"dev": {"access_key_id": "HCAKDEV", "secret_access_key": "sdev", "default_account_id": "acc-1", "apex": "example.test"}}
}`)
	t.Setenv("HOMECLOUD_CREDENTIALS_FILE", path)

	out, err := ApplyFileFallback(Chain{})
	if err != nil {
		t.Fatal(err)
	}
	if out.AccessKey != "HCAKDEV" || out.SecretKey != "sdev" || out.AccountID != "acc-1" || out.Apex != "example.test" || out.Source != "file:dev" {
		t.Fatalf("got %+v", out)
	}
}

func TestApplyFileFallbackKeepsExplicitApexAndAccount(t *testing.T) {
	dir := t.TempDir()
	path := writeCreds(t, dir, "credentials", `{
  "profiles": {"default": {"access_key_id": "HCAK", "secret_access_key": "s", "apex": "from-file.test", "default_account_id": "from-file"}}
}`)
	t.Setenv("HOMECLOUD_CREDENTIALS_FILE", path)

	out, err := ApplyFileFallback(Chain{Apex: "holab.abrdns.com", AccountID: "from-hcl"})
	if err != nil {
		t.Fatal(err)
	}
	if out.Apex != "holab.abrdns.com" || out.AccountID != "from-hcl" {
		t.Fatalf("explicit apex/account must win: %+v", out)
	}
}

func TestApplyFileFallbackHCProfile(t *testing.T) {
	dir := t.TempDir()
	path := writeCreds(t, dir, "credentials", `{
  "default_profile": "work",
  "profiles": {
    "work": {"access_key_id": "HCAKWORK", "secret_access_key": "sw"},
    "lab": {"access_key_id": "HCAKLAB", "secret_access_key": "sl"}
  }
}`)
	t.Setenv("HOMECLOUD_CREDENTIALS_FILE", path)
	t.Setenv("HC_PROFILE", "lab")

	out, err := ApplyFileFallback(Chain{})
	if err != nil {
		t.Fatal(err)
	}
	if out.AccessKey != "HCAKLAB" || out.Source != "file:lab" {
		t.Fatalf("HC_PROFILE=lab: %+v", out)
	}
}

func TestApplyFileFallbackEnvWinsOverFile(t *testing.T) {
	dir := t.TempDir()
	path := writeCreds(t, dir, "credentials", `{"access_key_id":"HCAKFILE","secret_access_key":"sfile"}`)
	t.Setenv("HOMECLOUD_CREDENTIALS_FILE", path)

	out, err := ApplyFileFallback(Chain{AccessKey: "HCAKENV", SecretKey: "senv"})
	if err != nil {
		t.Fatal(err)
	}
	if out.AccessKey != "HCAKENV" || out.Source != "env" {
		t.Fatalf("got %+v", out)
	}
}

func TestApplyFileFallbackMissingFile(t *testing.T) {
	t.Setenv("HOMECLOUD_CREDENTIALS_FILE", filepath.Join(t.TempDir(), "missing"))
	_, err := ApplyFileFallback(Chain{})
	if err == nil || !strings.Contains(err.Error(), "homecloud configure") {
		t.Fatalf("want configure hint, got %v", err)
	}
}

func TestApplyFileFallbackEmptyKeys(t *testing.T) {
	dir := t.TempDir()
	path := writeCreds(t, dir, "credentials", `{"profiles":{"default":{"access_key_id":"","secret_access_key":""}}}`)
	t.Setenv("HOMECLOUD_CREDENTIALS_FILE", path)
	_, err := ApplyFileFallback(Chain{})
	if err == nil || !strings.Contains(err.Error(), "no access_key_id") {
		t.Fatalf("want empty key error, got %v", err)
	}
}
