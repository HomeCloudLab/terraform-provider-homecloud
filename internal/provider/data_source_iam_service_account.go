package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/homecloudlab/terraform-provider-homecloud/internal/client"
)

func NewIAMServiceAccountDataSource() datasource.DataSource {
	return &iamServiceAccountDataSource{}
}

type iamServiceAccountDataSource struct {
	client *client.Client
}

type iamServiceAccountModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	ARN  types.String `tfsdk:"arn"`
}

func (d *iamServiceAccountDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iam_service_account"
}

func (d *iamServiceAccountDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Account IAM service account (`GET /iam/service-accounts/{name}`). Default runtime SA is `functions`.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
			},
			"id": schema.StringAttribute{
				Computed: true,
			},
			"arn": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (d *iamServiceAccountDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected data source configure type", fmt.Sprintf("%T", req.ProviderData))
		return
	}
	d.client = c
}

func (d *iamServiceAccountDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config iamServiceAccountModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := d.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	sa, err := d.client.GetIAMServiceAccount(ctx, accountID, config.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read IAM service account", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, iamServiceAccountModel{
		ID:   types.StringValue(sa.ID),
		Name: types.StringValue(sa.Name),
		ARN:  types.StringValue(sa.ARN),
	})...)
}
