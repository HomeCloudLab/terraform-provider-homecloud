package provider

import (
	"context"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/homecloudlab/terraform-provider-homecloud/internal/client"
)

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &homecloudProvider{version: version}
	}
}

type homecloudProvider struct {
	version string
}

type providerModel struct {
	AccessKey types.String `tfsdk:"access_key"`
	SecretKey types.String `tfsdk:"secret_key"`
	Apex      types.String `tfsdk:"apex"`
	Endpoint  types.String `tfsdk:"endpoint"`
	AccountID types.String `tfsdk:"account_id"`
}

func (p *homecloudProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "homecloud"
	resp.Version = p.version
}

func (p *homecloudProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "HomeCloud Terraform provider. Talks to `console.{apex}/api/v1` with SigV1 Access Keys (ADR-049).",
		Attributes: map[string]schema.Attribute{
			"access_key": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "Access Key ID. Env: `HC_ACCESS_KEY_ID` / `HOMECLOUD_ACCESS_KEY_ID`.",
			},
			"secret_key": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "Secret Access Key. Env: `HC_SECRET_ACCESS_KEY` / `HOMECLOUD_SECRET_ACCESS_KEY`.",
			},
			"apex": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Platform apex domain. Default `holab.abrdns.com`. Env: `HC_APEX`.",
			},
			"endpoint": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Override console base URL (tests). Env: `HC_ENDPOINT`.",
			},
			"account_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Account UUID. Default from Access Key whoami. Env: `HC_ACCOUNT_ID`.",
			},
		},
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func (p *homecloudProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	accessKey := firstNonEmpty(config.AccessKey.ValueString(), os.Getenv("HC_ACCESS_KEY_ID"), os.Getenv("HOMECLOUD_ACCESS_KEY_ID"))
	secretKey := firstNonEmpty(config.SecretKey.ValueString(), os.Getenv("HC_SECRET_ACCESS_KEY"), os.Getenv("HOMECLOUD_SECRET_ACCESS_KEY"))
	apex := firstNonEmpty(config.Apex.ValueString(), os.Getenv("HC_APEX"), "holab.abrdns.com")
	endpoint := firstNonEmpty(config.Endpoint.ValueString(), os.Getenv("HC_ENDPOINT"))
	accountID := firstNonEmpty(config.AccountID.ValueString(), os.Getenv("HC_ACCOUNT_ID"))

	if accessKey == "" || secretKey == "" {
		resp.Diagnostics.AddError(
			"Missing HomeCloud credentials",
			"Set access_key/secret_key or HC_ACCESS_KEY_ID and HC_SECRET_ACCESS_KEY.",
		)
		return
	}
	if endpoint == "" {
		endpoint = "https://console." + apex
	}

	c := &client.Client{
		Endpoint:    endpoint,
		AccessKeyID: accessKey,
		Secret:      secretKey,
		AccountID:   accountID,
	}
	if _, err := c.ResolveAccountID(ctx); err != nil {
		resp.Diagnostics.AddError("HomeCloud whoami failed", err.Error())
		return
	}
	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *homecloudProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewMQQueueResource,
		NewSOBucketResource,
		NewSecretResource,
		NewIAMPolicyResource,
		NewIAMRoleResource,
		NewIAMPolicyAttachmentResource,
	}
}

func (p *homecloudProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewAccountDataSource,
		NewIAMServiceAccountDataSource,
	}
}
