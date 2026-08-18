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

func NewIAMRoleResource() resource.Resource {
	return &iamRoleResource{}
}

type iamRoleResource struct {
	client *client.Client
}

type iamRoleModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	ARN           types.String `tfsdk:"arn"`
	Description   types.String `tfsdk:"description"`
	TrustDocument types.String `tfsdk:"trust_document"`
}

func (r *iamRoleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iam_role"
}

func (r *iamRoleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "IAM role (`POST /api/v1/accounts/{id}/iam/roles`). Attach managed/custom policies with `homecloud_iam_policy_attachment`. Requires console role owner/admin.",
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
			"arn": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"trust_document": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Assume-role trust JSON. Defaults to the account `functions` service account.",
			},
		},
	}
}

func (r *iamRoleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *iamRoleResource) flatten(role *client.IAMRole, priorTrust string) iamRoleModel {
	desc := types.StringNull()
	if role.Description != nil && strings.TrimSpace(*role.Description) != "" {
		desc = types.StringValue(*role.Description)
	}
	trust := compactJSON(role.TrustDocument)
	if priorTrust != "" && jsonEqual(priorTrust, trust) {
		trust = priorTrust
	}
	return iamRoleModel{
		ID:            types.StringValue(role.ID),
		Name:          types.StringValue(role.Name),
		ARN:           types.StringValue(role.ARN),
		Description:   desc,
		TrustDocument: types.StringValue(trust),
	}
}

func (r *iamRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan iamRoleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	var trust []byte
	if !plan.TrustDocument.IsNull() && !plan.TrustDocument.IsUnknown() && plan.TrustDocument.ValueString() != "" {
		trust = []byte(plan.TrustDocument.ValueString())
	}
	created, err := r.client.CreateIAMRole(ctx, accountID, plan.Name.ValueString(), plan.Description.ValueString(), trust)
	if err != nil {
		resp.Diagnostics.AddError("Create IAM role", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, r.flatten(created, plan.TrustDocument.ValueString()))...)
}

func (r *iamRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state iamRoleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	got, err := r.client.GetIAMRole(ctx, accountID, state.Name.ValueString())
	if err != nil {
		if apiErr, ok := err.(*client.APIError); ok && apiErr.NotFound() {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Read IAM role", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, r.flatten(got, state.TrustDocument.ValueString()))...)
}

func (r *iamRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan iamRoleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	updated, err := r.client.UpdateIAMRoleTrust(ctx, accountID, plan.Name.ValueString(), []byte(plan.TrustDocument.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Update IAM role", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, r.flatten(updated, plan.TrustDocument.ValueString()))...)
}

func (r *iamRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state iamRoleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	if err := r.client.DeleteIAMRole(ctx, accountID, state.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError("Delete IAM role", err.Error())
	}
}

func (r *iamRoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), strings.TrimSpace(req.ID))...)
}
