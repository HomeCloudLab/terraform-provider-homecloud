package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/homecloudlab/terraform-provider-homecloud/internal/client"
)

func NewAccountDataSource() datasource.DataSource {
	return &accountDataSource{}
}

type accountDataSource struct {
	client *client.Client
}

type accountModel struct {
	ID            types.String `tfsdk:"id"`
	ShortID       types.String `tfsdk:"short_id"`
	AccountNumber types.String `tfsdk:"account_number"`
	Name          types.String `tfsdk:"name"`
	Slug          types.String `tfsdk:"slug"`
	Status        types.String `tfsdk:"status"`
}

func (d *accountDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_account"
}

func (d *accountDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Current HomeCloud account (from Access Key whoami, or explicit `id`).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Account UUID. Defaults to the Access Key's account.",
			},
			"short_id":       schema.StringAttribute{Computed: true},
			"account_number": schema.StringAttribute{Computed: true},
			"name":           schema.StringAttribute{Computed: true},
			"slug":           schema.StringAttribute{Computed: true},
			"status":         schema.StringAttribute{Computed: true},
		},
	}
}

func (d *accountDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *accountDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config accountModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID := config.ID.ValueString()
	if accountID == "" {
		resolved, err := d.client.ResolveAccountID(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Resolve account", err.Error())
			return
		}
		accountID = resolved
	}
	acct, err := d.client.GetAccount(ctx, accountID)
	if err != nil {
		resp.Diagnostics.AddError("Read account", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, accountModel{
		ID:            types.StringValue(acct.ID),
		ShortID:       types.StringValue(acct.ShortID),
		AccountNumber: types.StringValue(acct.AccountNumber),
		Name:          types.StringValue(acct.Name),
		Slug:          types.StringValue(acct.Slug),
		Status:        types.StringValue(acct.Status),
	})...)
}
