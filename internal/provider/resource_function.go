package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/homecloudlab/terraform-provider-homecloud/internal/client"
)

func NewFunctionResource() resource.Resource {
	return &functionResource{}
}

type functionResource struct {
	client *client.Client
}

type functionModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Runtime        types.String `tfsdk:"runtime"`
	Handler        types.String `tfsdk:"handler"`
	MemoryLimitMB  types.Int64  `tfsdk:"memory_limit_mb"`
	TimeoutSeconds types.Int64  `tfsdk:"timeout_seconds"`
	IamARN         types.String `tfsdk:"iam_arn"`
	Status         types.String `tfsdk:"status"`
	InvokeURL      types.String `tfsdk:"invoke_url"`
}

func (r *functionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_function"
}

func (r *functionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Managed function (`POST /api/v1/accounts/{id}/functions`). Does not manage IDE workspace files or deploys.",
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
			"runtime": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"handler": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"memory_limit_mb": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"timeout_seconds": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"iam_arn": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status":     schema.StringAttribute{Computed: true},
			"invoke_url": schema.StringAttribute{Computed: true},
		},
	}
}

func (r *functionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *functionResource) flatten(fn *client.Function) functionModel {
	return functionModel{
		ID:             types.StringValue(fn.ID),
		Name:           types.StringValue(fn.Name),
		Runtime:        types.StringValue(fn.Runtime),
		Handler:        types.StringValue(fn.Handler),
		MemoryLimitMB:  types.Int64Value(fn.MemoryLimitMB),
		TimeoutSeconds: types.Int64Value(fn.TimeoutSeconds),
		IamARN:         types.StringValue(fn.IamARN),
		Status:         types.StringValue(fn.Status),
		InvokeURL:      types.StringValue(fn.InvokeURL),
	}
}

func (r *functionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan functionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	created, err := r.client.CreateFunction(ctx, accountID, client.FunctionCreate{
		Name:           plan.Name.ValueString(),
		Runtime:        plan.Runtime.ValueString(),
		Handler:        plan.Handler.ValueString(),
		MemoryLimitMB:  plan.MemoryLimitMB.ValueInt64(),
		TimeoutSeconds: plan.TimeoutSeconds.ValueInt64(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Create function", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, r.flatten(created))...)
}

func (r *functionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state functionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	got, err := r.client.GetFunction(ctx, accountID, state.Name.ValueString())
	if err != nil {
		if apiErr, ok := err.(*client.APIError); ok && apiErr.NotFound() {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Read function", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, r.flatten(got))...)
}

func (r *functionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan functionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	mem := plan.MemoryLimitMB.ValueInt64()
	timeout := plan.TimeoutSeconds.ValueInt64()
	updated, err := r.client.UpdateFunction(ctx, accountID, plan.Name.ValueString(), client.FunctionUpdate{
		Runtime:        plan.Runtime.ValueString(),
		Handler:        plan.Handler.ValueString(),
		MemoryLimitMB:  &mem,
		TimeoutSeconds: &timeout,
	})
	if err != nil {
		resp.Diagnostics.AddError("Update function", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, r.flatten(updated))...)
}

func (r *functionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state functionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	if err := r.client.DeleteFunction(ctx, accountID, state.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError("Delete function", err.Error())
	}
}

func (r *functionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), strings.TrimSpace(req.ID))...)
}
