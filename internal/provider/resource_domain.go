package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/homecloudlab/terraform-provider-homecloud/internal/client"
)

func NewDomainResource() resource.Resource {
	return &domainResource{}
}

type domainResource struct {
	client *client.Client
}

type domainModel struct {
	ID       types.String `tfsdk:"id"`
	Hostname types.String `tfsdk:"hostname"`
	DNSMode  types.String `tfsdk:"dns_mode"`
	IamARN   types.String `tfsdk:"iam_arn"`
	Status   types.String `tfsdk:"status"`
	FQDN     types.String `tfsdk:"fqdn"`
	Verified types.Bool   `tfsdk:"verified"`
}

func (r *domainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (r *domainResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Custom hostname (`POST /api/v1/accounts/{id}/domains`). Create returns `pending_verification`; DNS/TXT verification stays out of band. Does not attach the domain to a function or app.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"hostname": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"dns_mode": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "`external` (default) or `homecloud`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"iam_arn": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status":   schema.StringAttribute{Computed: true},
			"fqdn":     schema.StringAttribute{Computed: true},
			"verified": schema.BoolAttribute{Computed: true},
		},
	}
}

func (r *domainResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected resource configure type", fmt.Sprintf("%T", req.ProviderData))
		return
	}
	r.client = c
}

func (r *domainResource) flatten(d *client.Domain, hostname string) domainModel {
	name := d.Name
	if name == "" {
		name = hostname
	}
	return domainModel{
		ID:       types.StringValue(d.ID),
		Hostname: types.StringValue(name),
		DNSMode:  types.StringValue(d.DNSMode),
		IamARN:   types.StringValue(d.IamARN),
		Status:   types.StringValue(d.Status),
		FQDN:     types.StringValue(d.FQDN),
		Verified: types.BoolValue(d.Verified),
	}
}

func (r *domainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan domainModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	created, err := r.client.CreateDomain(ctx, accountID, client.DomainCreate{
		Hostname: plan.Hostname.ValueString(),
		DNSMode:  plan.DNSMode.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Create domain", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, r.flatten(created, plan.Hostname.ValueString()))...)
}

func (r *domainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state domainModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	got, err := r.client.GetDomain(ctx, accountID, state.Hostname.ValueString())
	if err != nil {
		if apiErr, ok := err.(*client.APIError); ok && apiErr.NotFound() {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Read domain", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, r.flatten(got, state.Hostname.ValueString()))...)
}

func (r *domainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "Domain hostname and dns_mode require replace.")
}

func (r *domainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state domainModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	if err := r.client.DeleteDomain(ctx, accountID, state.Hostname.ValueString()); err != nil {
		resp.Diagnostics.AddError("Delete domain", err.Error())
	}
}

func (r *domainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("hostname"), strings.TrimSpace(req.ID))...)
}
