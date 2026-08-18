package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/homecloudlab/terraform-provider-homecloud/internal/client"
)

func NewSecretResource() resource.Resource {
	return &secretResource{}
}

type secretResource struct {
	client *client.Client
}

type secretModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	IamARN      types.String `tfsdk:"iam_arn"`
	Status      types.String `tfsdk:"status"`
	Version     types.Int64  `tfsdk:"version"`
	HasValue    types.Bool   `tfsdk:"has_value"`
	KeyNames    types.List   `tfsdk:"key_names"`
	Values      types.Map    `tfsdk:"values"`
}

func (r *secretResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret"
}

func (r *secretResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Managed account secret (`POST /api/v1/accounts/{id}/secrets`). Values are write-only (Terraform 1.11+) and never stored in state or returned by GET.",
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
			"description": schema.StringAttribute{
				Optional: true,
			},
			"iam_arn": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "IAM-canonical ARN (`arn:homecloud:secrets::{account_number}:secret/{name}`).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"version": schema.Int64Attribute{
				Computed: true,
			},
			"has_value": schema.BoolAttribute{
				Computed: true,
			},
			"key_names": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
			"values": schema.MapAttribute{
				Optional:            true,
				Sensitive:           true,
				WriteOnly:           true,
				ElementType:         types.StringType,
				MarkdownDescription: "Secret payload. Write-only; never in state. Requires Terraform 1.11+.",
			},
		},
	}
}

func (r *secretResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func mapStringValues(ctx context.Context, m types.Map) (map[string]string, diag.Diagnostics) {
	out := map[string]string{}
	if m.IsNull() || m.IsUnknown() {
		return out, nil
	}
	var diags diag.Diagnostics
	diags.Append(m.ElementsAs(ctx, &out, false)...)
	return out, diags
}

func (r *secretResource) flatten(ctx context.Context, s *client.Secret) (secretModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	names := s.KeyNames
	if names == nil {
		names = []string{}
	}
	keyNames, d := types.ListValueFrom(ctx, types.StringType, names)
	diags.Append(d...)
	desc := types.StringNull()
	if s.Description != nil && strings.TrimSpace(*s.Description) != "" {
		desc = types.StringValue(*s.Description)
	}
	return secretModel{
		ID:          types.StringValue(s.ID),
		Name:        types.StringValue(s.Name),
		Description: desc,
		IamARN:      types.StringValue(s.IamARN),
		Status:      types.StringValue(s.Status),
		Version:     types.Int64Value(s.Version),
		HasValue:    types.BoolValue(s.HasValue),
		KeyNames:    keyNames,
		Values:      types.MapNull(types.StringType),
	}, diags
}

func (r *secretResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var config secretModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	var plan secretModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	body := client.SecretCreate{Name: plan.Name.ValueString()}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		body.Description = plan.Description.ValueString()
	}
	created, err := r.client.CreateSecret(ctx, accountID, body)
	if err != nil {
		resp.Diagnostics.AddError("Create secret", err.Error())
		return
	}
	values, d := mapStringValues(ctx, config.Values)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(values) > 0 {
		created, err = r.client.PutSecretValue(ctx, accountID, created.Name, values)
		if err != nil {
			resp.Diagnostics.AddError("Put secret value", err.Error())
			return
		}
	}
	state, d := r.flatten(ctx, created)
	resp.Diagnostics.Append(d...)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *secretResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state secretModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	got, err := r.client.GetSecret(ctx, accountID, state.Name.ValueString())
	if err != nil {
		if apiErr, ok := err.(*client.APIError); ok && apiErr.NotFound() {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Read secret", err.Error())
		return
	}
	next, d := r.flatten(ctx, got)
	resp.Diagnostics.Append(d...)
	if !state.Description.IsNull() && next.Description.IsNull() {
		next.Description = state.Description
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, next)...)
}

func (r *secretResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var config secretModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	var plan secretModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	name := plan.Name.ValueString()
	updated, err := r.client.UpdateSecret(ctx, accountID, name, plan.Description.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Update secret", err.Error())
		return
	}
	values, d := mapStringValues(ctx, config.Values)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(values) > 0 {
		updated, err = r.client.PutSecretValue(ctx, accountID, name, values)
		if err != nil {
			resp.Diagnostics.AddError("Put secret value", err.Error())
			return
		}
	}
	state, d := r.flatten(ctx, updated)
	resp.Diagnostics.Append(d...)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *secretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state secretModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	if err := r.client.DeleteSecret(ctx, accountID, state.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError("Delete secret", err.Error())
	}
}

func (r *secretResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id := strings.TrimSpace(req.ID)
	if id == "" {
		resp.Diagnostics.AddError("Invalid import id", "expected secret name")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), id)...)
}
