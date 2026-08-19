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
	"github.com/homecloudlab/terraform-provider-homecloud/internal/credentials"
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
	AccessKey        types.String `tfsdk:"access_key"`
	SecretKey        types.String `tfsdk:"secret_key"`
	SessionToken     types.String `tfsdk:"session_token"`
	RoleARN          types.String `tfsdk:"role_arn"`
	WebIdentityToken types.String `tfsdk:"web_identity_token"`
	OIDCAudience     types.String `tfsdk:"oidc_audience"`
	Profile          types.String `tfsdk:"profile"`
	Apex             types.String `tfsdk:"apex"`
	Endpoint         types.String `tfsdk:"endpoint"`
	AccountID        types.String `tfsdk:"account_id"`
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
			"profile": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Named profile in `~/.homecloud/credentials` (same file as `homecloud configure`). Env: `HC_PROFILE` / `HOMECLOUD_PROFILE`. Ignored when Access Keys are set in HCL/env, and when `role_arn` is set (OIDC).",
			},
			"session_token": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "STS session token for assumed-role credentials. Env: `HC_SESSION_TOKEN`.",
			},
			"role_arn": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "IAM role ARN for GitHub OIDC (`sts:AssumeRoleWithWebIdentity`). Env: `HC_ROLE_ARN`.",
			},
			"web_identity_token": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "GitHub OIDC JWT. Env: `HC_WEB_IDENTITY_TOKEN`. In GitHub Actions the provider can fetch it from `ACTIONS_ID_TOKEN_REQUEST_*`.",
			},
			"oidc_audience": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Audience requested for the GitHub OIDC token. Default `https://console.{apex}`. Env: `HC_OIDC_AUDIENCE`.",
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
	sessionToken := firstNonEmpty(config.SessionToken.ValueString(), os.Getenv("HC_SESSION_TOKEN"))
	roleARN := firstNonEmpty(config.RoleARN.ValueString(), os.Getenv("HC_ROLE_ARN"))
	webToken := firstNonEmpty(config.WebIdentityToken.ValueString(), os.Getenv("HC_WEB_IDENTITY_TOKEN"))
	apex := firstNonEmpty(config.Apex.ValueString(), os.Getenv("HC_APEX"), os.Getenv("HOMECLOUD_APEX"))
	endpoint := firstNonEmpty(config.Endpoint.ValueString(), os.Getenv("HC_ENDPOINT"))
	accountID := firstNonEmpty(config.AccountID.ValueString(), os.Getenv("HC_ACCOUNT_ID"), os.Getenv("HOMECLOUD_ACCOUNT_ID"))
	audience := firstNonEmpty(config.OIDCAudience.ValueString(), os.Getenv("HC_OIDC_AUDIENCE"))

	chain, err := credentials.ApplyFileFallback(credentials.Chain{
		AccessKey: accessKey,
		SecretKey: secretKey,
		RoleARN:   roleARN,
		Apex:      apex,
		AccountID: accountID,
		Profile:   config.Profile.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Missing HomeCloud credentials", err.Error())
		return
	}
	accessKey = chain.AccessKey
	secretKey = chain.SecretKey
	apex = firstNonEmpty(chain.Apex, "holab.abrdns.com")
	accountID = chain.AccountID

	if endpoint == "" {
		endpoint = "https://console." + apex
	}
	if audience == "" {
		audience = strings.TrimRight(endpoint, "/")
	}

	c := &client.Client{
		Endpoint:     endpoint,
		AccessKeyID:  accessKey,
		Secret:       secretKey,
		SessionToken: sessionToken,
		AccountID:    accountID,
	}

	if roleARN != "" && accessKey == "" {
		if webToken == "" {
			tok, err := client.FetchGitHubOIDCToken(ctx, audience)
			if err != nil {
				resp.Diagnostics.AddError("HomeCloud GitHub OIDC token missing", err.Error())
				return
			}
			webToken = tok
		}
		out, err := c.AssumeRoleWithWebIdentity(ctx, client.AssumeRoleWithWebIdentityInput{
			RoleARN:          roleARN,
			WebIdentityToken: webToken,
			SessionName:      "terraform",
		})
		if err != nil {
			resp.Diagnostics.AddError("HomeCloud OIDC AssumeRole failed", err.Error())
			return
		}
		c.AccessKeyID = out.AccessKeyID
		c.Secret = out.SecretAccessKey
		c.SessionToken = out.SessionToken
		if c.AccountID == "" {
			c.AccountID = out.AccountID
		}
	}

	if c.AccessKeyID == "" || c.Secret == "" {
		resp.Diagnostics.AddError(
			"Missing HomeCloud credentials",
			"Set access_key/secret_key, HC_ACCESS_KEY_ID / HC_SECRET_ACCESS_KEY, run `homecloud configure` (~/.homecloud/credentials), or set role_arn / HC_ROLE_ARN for GitHub OIDC.",
		)
		return
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
		NewMDBInstanceResource,
		NewMDBUserResource,
		NewRedisInstanceResource,
		NewFunctionResource,
		NewFunctionURLResource,
		NewIRRepositoryResource,
		NewDomainResource,
		NewComputeMachineResource,
		NewSSHKeyResource,
		NewApplicationResource,
	}
}

func (p *homecloudProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewAccountDataSource,
		NewIAMServiceAccountDataSource,
	}
}
