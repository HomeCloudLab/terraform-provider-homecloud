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

func NewApplicationResource() resource.Resource {
	return &applicationResource{}
}

type applicationResource struct {
	client *client.Client
}

type applicationModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Slug      types.String `tfsdk:"slug"`
	Template  types.String `tfsdk:"template"`
	ProjectID types.String `tfsdk:"project_id"`
	IamARN    types.String `tfsdk:"iam_arn"`
	Status    types.String `tfsdk:"status"`
}

func (r *applicationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application"
}

func (r *applicationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Application spec (`POST /api/v1/accounts/{id}/applications`). Create stays `draft` — no provision, deploy, scale, or YAML apply.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"slug": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"template": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "`api-only` (default), `fullstack`, `static-site`, or `worker`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Optional: true,
				Computed: true,
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
			"status": schema.StringAttribute{Computed: true},
		},
	}
}

func (r *applicationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *applicationResource) flatten(app *client.Application) applicationModel {
	return applicationModel{
		ID:        types.StringValue(app.ID),
		Name:      types.StringValue(app.Name),
		Slug:      types.StringValue(app.Slug),
		Template:  types.StringValue(app.Template),
		ProjectID: types.StringValue(app.ProjectID),
		IamARN:    types.StringValue(app.IamARN),
		Status:    types.StringValue(app.Status),
	}
}

func (r *applicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan applicationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	created, err := r.client.CreateApplication(ctx, accountID, client.ApplicationCreate{
		Name:      plan.Name.ValueString(),
		Slug:      plan.Slug.ValueString(),
		Template:  plan.Template.ValueString(),
		ProjectID: plan.ProjectID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Create application", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, r.flatten(created))...)
}

func (r *applicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state applicationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	got, err := r.client.GetApplication(ctx, accountID, state.Slug.ValueString())
	if err != nil {
		if apiErr, ok := err.(*client.APIError); ok && apiErr.NotFound() {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Read application", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, r.flatten(got))...)
}

func (r *applicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "Application slug and template require replace.")
}

func (r *applicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state applicationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	if err := r.client.DeleteApplication(ctx, accountID, state.Slug.ValueString()); err != nil {
		resp.Diagnostics.AddError("Delete application", err.Error())
	}
}

func (r *applicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("slug"), strings.TrimSpace(req.ID))...)
}
