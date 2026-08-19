// Package credentials reads the same ~/.homecloud/credentials JSON as the CLI/SDK.
package credentials

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const defaultProfileName = "default"

type Profile struct {
	Name             string
	AccessKeyID      string
	SecretAccessKey  string
	Apex             string
	DefaultAccountID string
	Path             string
}

type Chain struct {
	AccessKey string
	SecretKey string
	RoleARN   string
	Apex      string
	AccountID string
	Profile   string
	Source    string
}

func envFirst(names ...string) string {
	for _, name := range names {
		if v := strings.TrimSpace(os.Getenv(name)); v != "" {
			return v
		}
	}
	return ""
}

// Path is HOMECLOUD_CREDENTIALS_FILE / HC_CREDENTIALS_FILE, else
// {HOMECLOUD_CONFIG_DIR|HC_CONFIG_DIR}/credentials, else ~/.homecloud/credentials.
func Path() string {
	if p := envFirst("HOMECLOUD_CREDENTIALS_FILE", "HC_CREDENTIALS_FILE"); p != "" {
		return p
	}
	dir := envFirst("HOMECLOUD_CONFIG_DIR", "HC_CONFIG_DIR")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			return ""
		}
		dir = filepath.Join(home, ".homecloud")
	}
	return filepath.Join(dir, "credentials")
}

func ProfileName(explicit string) string {
	return firstNonEmpty(explicit, envFirst("HOMECLOUD_PROFILE", "HC_PROFILE"))
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

type rawFile struct {
	Version          int                   `json:"version"`
	DefaultProfile   string                `json:"default_profile"`
	Profiles         map[string]rawProfile `json:"profiles"`
	AccessKeyID      string                `json:"access_key_id"`
	SecretAccessKey  string                `json:"secret_access_key"`
	Apex             string                `json:"apex"`
	DefaultAccountID string                `json:"default_account_id"`
}

type rawProfile struct {
	AccessKeyID      string `json:"access_key_id"`
	SecretAccessKey  string `json:"secret_access_key"`
	Apex             string `json:"apex"`
	DefaultAccountID string `json:"default_account_id"`
}

func Load(path, profileHint string) (Profile, error) {
	rawBytes, err := os.ReadFile(path)
	if err != nil {
		return Profile{}, err
	}
	var raw rawFile
	if err := json.Unmarshal(rawBytes, &raw); err != nil {
		return Profile{}, fmt.Errorf("invalid credentials file %s: %w", path, err)
	}

	profiles := raw.Profiles
	if len(profiles) == 0 {
		profiles = map[string]rawProfile{
			defaultProfileName: {
				AccessKeyID:      raw.AccessKeyID,
				SecretAccessKey:  raw.SecretAccessKey,
				Apex:             raw.Apex,
				DefaultAccountID: raw.DefaultAccountID,
			},
		}
	}

	name := firstNonEmpty(profileHint, raw.DefaultProfile, defaultProfileName)
	entry, ok := profiles[name]
	if !ok {
		return Profile{}, fmt.Errorf("profile %q not found in %s", name, path)
	}
	return Profile{
		Name:             name,
		AccessKeyID:      strings.TrimSpace(entry.AccessKeyID),
		SecretAccessKey:  strings.TrimSpace(entry.SecretAccessKey),
		Apex:             strings.TrimSpace(strings.TrimRight(entry.Apex, "/")),
		DefaultAccountID: strings.TrimSpace(entry.DefaultAccountID),
		Path:             path,
	}, nil
}

// ApplyFileFallback fills keys from the CLI credentials file when HCL/env did
// not set them and role_arn is unset (OIDC must not lose to a leftover file).
func ApplyFileFallback(in Chain) (Chain, error) {
	out := in
	if strings.TrimSpace(out.AccessKey) != "" || strings.TrimSpace(out.SecretKey) != "" {
		out.Source = "env"
		return out, nil
	}
	if strings.TrimSpace(out.RoleARN) != "" {
		out.Source = "oidc"
		return out, nil
	}

	path := Path()
	if path == "" {
		return out, fmt.Errorf("cannot resolve home directory for ~/.homecloud/credentials")
	}
	profile, err := Load(path, ProfileName(in.Profile))
	if err != nil {
		if os.IsNotExist(err) {
			return out, fmt.Errorf("no Access Key in the environment and credentials file not found (%s); run `homecloud configure`, or set HC_ACCESS_KEY_ID / HC_SECRET_ACCESS_KEY, or HC_ROLE_ARN for GitHub OIDC", path)
		}
		return out, err
	}
	if profile.AccessKeyID == "" || profile.SecretAccessKey == "" {
		return out, fmt.Errorf("profile %q in %s has no access_key_id/secret_access_key; run `homecloud configure`", profile.Name, path)
	}
	out.AccessKey = profile.AccessKeyID
	out.SecretKey = profile.SecretAccessKey
	if strings.TrimSpace(out.Apex) == "" && profile.Apex != "" {
		out.Apex = profile.Apex
	}
	if strings.TrimSpace(out.AccountID) == "" && profile.DefaultAccountID != "" {
		out.AccountID = profile.DefaultAccountID
	}
	out.Profile = profile.Name
	out.Source = "file:" + profile.Name
	return out, nil
}
