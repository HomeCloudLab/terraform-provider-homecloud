package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/homecloudlab/terraform-provider-homecloud/internal/client"
)

func NewFunctionURLResource() resource.Resource {
	return &functionURLResource{}
}

type functionURLResource struct {
	client *client.Client
}

type functionURLModel struct {
	ID               types.String `tfsdk:"id"`
	FunctionName     types.String `tfsdk:"function_name"`
	PublicURLEnabled types.Bool   `tfsdk:"public_url_enabled"`
	FunctionURL      types.String `tfsdk:"function_url"`
}

func (r *functionURLResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_function_url"
}

func (r *functionURLResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "HTTP URL for a function (`POST .../functions/{name}/url/enable`). Delete disables the URL.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"function_name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"public_url_enabled": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"function_url": schema.StringAttribute{Computed: true},
		},
	}
}

func (r *functionURLResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *functionURLResource) flatten(name string, info *client.FunctionURL) functionURLModel {
	return functionURLModel{
		ID:               types.StringValue(name),
		FunctionName:     types.StringValue(name),
		PublicURLEnabled: types.BoolValue(info.PublicURLEnabled),
		FunctionURL:      types.StringValue(info.FunctionURL),
	}
}

func (r *functionURLResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan functionURLModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	info, err := r.client.EnableFunctionURL(ctx, accountID, plan.FunctionName.ValueString(), client.FunctionURLEnable{
		PublicURLEnabled: plan.PublicURLEnabled.ValueBool(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Enable function URL", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, r.flatten(plan.FunctionName.ValueString(), info))...)
}

func (r *functionURLResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state functionURLModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	info, err := r.client.GetFunctionURL(ctx, accountID, state.FunctionName.ValueString())
	if err != nil {
		if apiErr, ok := err.(*client.APIError); ok && apiErr.NotFound() {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Read function URL", err.Error())
		return
	}
	if !info.FunctionURLEnabled {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, r.flatten(state.FunctionName.ValueString(), info))...)
}

func (r *functionURLResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan functionURLModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	info, err := r.client.EnableFunctionURL(ctx, accountID, plan.FunctionName.ValueString(), client.FunctionURLEnable{
		PublicURLEnabled: plan.PublicURLEnabled.ValueBool(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Update function URL", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, r.flatten(plan.FunctionName.ValueString(), info))...)
}

func (r *functionURLResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state functionURLModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	if err := r.client.DisableFunctionURL(ctx, accountID, state.FunctionName.ValueString()); err != nil {
		resp.Diagnostics.AddError("Disable function URL", err.Error())
	}
}

func (r *functionURLResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("function_name"), strings.TrimSpace(req.ID))...)
}
