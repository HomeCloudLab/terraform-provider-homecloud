package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/homecloudlab/terraform-provider-homecloud/internal/client"
)

func NewComputeMachineResource() resource.Resource {
	return &computeMachineResource{}
}

type computeMachineResource struct {
	client *client.Client
}

type computeMachineModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	MachineClass types.String `tfsdk:"machine_class"`
	ImageID      types.String `tfsdk:"image_id"`
	RegionCode   types.String `tfsdk:"region_code"`
	AzCode       types.String `tfsdk:"az_code"`
	SSHKeyIDs    types.List   `tfsdk:"ssh_key_ids"`
	IamARN       types.String `tfsdk:"iam_arn"`
	Status       types.String `tfsdk:"status"`
	PublicIPv4   types.String `tfsdk:"public_ipv4"`
	OperationID  types.String `tfsdk:"operation_id"`
}

func (r *computeMachineResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_compute_machine"
}

func (r *computeMachineResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Compute machine (`POST /api/v1/accounts/{id}/compute/machines`). Create waits for the Operations API (`SUCCEEDED`/`FAILED`). Does not manage firewall, volumes, exec, or files.",
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
			"machine_class": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "`basic` or `standard`. Sent as JSON `class`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"image_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"region_code": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"az_code": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ssh_key_ids": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"iam_arn": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status":      schema.StringAttribute{Computed: true},
			"public_ipv4": schema.StringAttribute{Computed: true},
			"operation_id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *computeMachineResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *computeMachineResource) flatten(m *client.Machine, prior computeMachineModel, operationID string) computeMachineModel {
	op := operationID
	if op == "" {
		op = prior.OperationID.ValueString()
	}
	return computeMachineModel{
		ID:           types.StringValue(m.ID),
		Name:         types.StringValue(m.Name),
		MachineClass: types.StringValue(m.Class),
		ImageID:      types.StringValue(m.ImageID),
		RegionCode:   types.StringValue(m.RegionCode),
		AzCode:       types.StringValue(m.AzCode),
		SSHKeyIDs:    prior.SSHKeyIDs,
		IamARN:       types.StringValue(m.IamARN),
		Status:       types.StringValue(m.Status),
		PublicIPv4:   types.StringValue(m.Nic.PublicIPv4),
		OperationID:  types.StringValue(op),
	}
}

func (r *computeMachineResource) waitOperation(ctx context.Context, accountID, operationID string) error {
	if strings.TrimSpace(operationID) == "" {
		return nil
	}
	return r.client.WaitUntilSucceeded(ctx, func() (string, error) {
		op, err := r.client.GetOperation(ctx, accountID, operationID)
		if err != nil {
			return "", err
		}
		if op.Status == "FAILED" && op.Error != "" {
			return op.Status, fmt.Errorf("operation failed: %s", op.Error)
		}
		return op.Status, nil
	}, 20*time.Minute)
}

func (r *computeMachineResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan computeMachineModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	body := client.MachineCreate{
		Name:       plan.Name.ValueString(),
		Class:      plan.MachineClass.ValueString(),
		ImageID:    plan.ImageID.ValueString(),
		RegionCode: plan.RegionCode.ValueString(),
		AzCode:     plan.AzCode.ValueString(),
	}
	if !plan.SSHKeyIDs.IsNull() && !plan.SSHKeyIDs.IsUnknown() {
		resp.Diagnostics.Append(plan.SSHKeyIDs.ElementsAs(ctx, &body.SSHKeyIDs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	created, err := r.client.CreateMachine(ctx, accountID, body)
	if err != nil {
		resp.Diagnostics.AddError("Create compute machine", err.Error())
		return
	}
	if err := r.waitOperation(ctx, accountID, created.OperationID); err != nil {
		resp.Diagnostics.AddError("Wait for compute machine", err.Error())
		return
	}
	got, err := r.client.GetMachine(ctx, accountID, plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read compute machine", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, r.flatten(got, plan, created.OperationID))...)
}

func (r *computeMachineResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state computeMachineModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	got, err := r.client.GetMachine(ctx, accountID, state.Name.ValueString())
	if err != nil {
		if apiErr, ok := err.(*client.APIError); ok && apiErr.NotFound() {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Read compute machine", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, r.flatten(got, state, ""))...)
}

func (r *computeMachineResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "Compute machine fields require replace.")
}

func (r *computeMachineResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state computeMachineModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	deleted, err := r.client.DeleteMachine(ctx, accountID, state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Delete compute machine", err.Error())
		return
	}
	if deleted != nil {
		if err := r.waitOperation(ctx, accountID, deleted.OperationID); err != nil {
			resp.Diagnostics.AddError("Wait for compute machine delete", err.Error())
		}
	}
}

func (r *computeMachineResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), strings.TrimSpace(req.ID))...)
}
