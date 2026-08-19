package provider

import (
	"context"
	"testing"

	"github.com/homecloudlab/terraform-provider-homecloud/internal/client"
	"github.com/homecloudlab/terraform-provider-homecloud/internal/credentials"
)

// Hits the live console if this machine already ran `homecloud configure`.
func TestLiveWhoamiFromCLICredentialsFile(t *testing.T) {
	path := credentials.Path()
	profile, err := credentials.Load(path, credentials.ProfileName(""))
	if err != nil {
		t.Skipf("no CLI credentials file (%s): %v", path, err)
	}
	if profile.AccessKeyID == "" || profile.SecretAccessKey == "" {
		t.Skip("credentials profile has no Access Key")
	}
	apex := profile.Apex
	if apex == "" {
		apex = "holab.abrdns.com"
	}
	c := &client.Client{
		Endpoint:    "https://console." + apex,
		AccessKeyID: profile.AccessKeyID,
		Secret:      profile.SecretAccessKey,
		AccountID:   profile.DefaultAccountID,
	}
	id, err := c.ResolveAccountID(context.Background())
	if err != nil {
		t.Fatalf("whoami with ~/.homecloud/credentials failed: %v", err)
	}
	if id == "" {
		t.Fatal("whoami returned empty account id")
	}
}
