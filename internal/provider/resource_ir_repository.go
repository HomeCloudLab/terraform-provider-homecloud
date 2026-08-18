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

func NewIRRepositoryResource() resource.Resource {
	return &irRepositoryResource{}
}

type irRepositoryResource struct {
	client *client.Client
}

type irRepositoryModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	IamARN         types.String `tfsdk:"iam_arn"`
	Status         types.String `tfsdk:"status"`
	ZotNamespace   types.String `tfsdk:"zot_namespace"`
	ImageRefPrefix types.String `tfsdk:"image_ref_prefix"`
}

func (r *irRepositoryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ir_repository"
}

func (r *irRepositoryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Image Registry repository (`POST /api/v1/accounts/{id}/registry/repositories`). Image tags are not a Terraform resource.",
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
			"iam_arn": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status":           schema.StringAttribute{Computed: true},
			"zot_namespace":    schema.StringAttribute{Computed: true},
			"image_ref_prefix": schema.StringAttribute{Computed: true},
		},
	}
}

func (r *irRepositoryResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *irRepositoryResource) flatten(repo *client.Repository) irRepositoryModel {
	return irRepositoryModel{
		ID:             types.StringValue(repo.ID),
		Name:           types.StringValue(repo.Name),
		IamARN:         types.StringValue(repo.IamARN),
		Status:         types.StringValue(repo.Status),
		ZotNamespace:   types.StringValue(repo.ZotNamespace),
		ImageRefPrefix: types.StringValue(repo.ImageRefPrefix),
	}
}

func (r *irRepositoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan irRepositoryModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	created, err := r.client.CreateRepository(ctx, accountID, client.RepositoryCreate{Name: plan.Name.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError("Create IR repository", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, r.flatten(created))...)
}

func (r *irRepositoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state irRepositoryModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	got, err := r.client.GetRepository(ctx, accountID, state.Name.ValueString())
	if err != nil {
		if apiErr, ok := err.(*client.APIError); ok && apiErr.NotFound() {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Read IR repository", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, r.flatten(got))...)
}

func (r *irRepositoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "IR repository name requires replace.")
}

func (r *irRepositoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state irRepositoryModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	if err := r.client.DeleteRepository(ctx, accountID, state.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError("Delete IR repository", err.Error())
	}
}

func (r *irRepositoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), strings.TrimSpace(req.ID))...)
}
