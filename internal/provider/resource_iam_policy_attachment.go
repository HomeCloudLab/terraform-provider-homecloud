package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/homecloudlab/terraform-provider-homecloud/internal/client"
)

func NewIAMPolicyAttachmentResource() resource.Resource {
	return &iamPolicyAttachmentResource{}
}

type iamPolicyAttachmentResource struct {
	client *client.Client
}

type iamPolicyAttachmentModel struct {
	ID            types.String `tfsdk:"id"`
	PolicyARN     types.String `tfsdk:"policy_arn"`
	PrincipalType types.String `tfsdk:"principal_type"`
	PrincipalID   types.String `tfsdk:"principal_id"`
}

func (r *iamPolicyAttachmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iam_policy_attachment"
}

func (r *iamPolicyAttachmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Attach an IAM policy to a user, role, or service account (`POST /iam/principals/attachments`).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"policy_arn": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"principal_type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "`user`, `role`, or `service_account`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"principal_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *iamPolicyAttachmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *iamPolicyAttachmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan iamPolicyAttachmentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	policyARN := plan.PolicyARN.ValueString()
	principalType := plan.PrincipalType.ValueString()
	principalID := plan.PrincipalID.ValueString()
	if err := r.client.AttachIAMPolicy(ctx, accountID, policyARN, principalType, principalID); err != nil {
		resp.Diagnostics.AddError("Attach IAM policy", err.Error())
		return
	}
	items, err := r.client.ListIAMAttachments(ctx, accountID, principalType, principalID)
	if err != nil {
		resp.Diagnostics.AddError("List IAM attachments", err.Error())
		return
	}
	id := principalType + ":" + principalID + ":" + policyARN
	for _, item := range items {
		if item.PolicyARN == policyARN {
			id = item.ID
			break
		}
	}
	plan.ID = types.StringValue(id)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *iamPolicyAttachmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state iamPolicyAttachmentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	items, err := r.client.ListIAMAttachments(ctx, accountID, state.PrincipalType.ValueString(), state.PrincipalID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read IAM attachment", err.Error())
		return
	}
	found := false
	for _, item := range items {
		if item.PolicyARN == state.PolicyARN.ValueString() {
			found = true
			state.ID = types.StringValue(item.ID)
			break
		}
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *iamPolicyAttachmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "IAM policy attachments are replaced on change.")
}

func (r *iamPolicyAttachmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state iamPolicyAttachmentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	if err := r.client.DetachIAMPolicy(ctx, accountID, state.PolicyARN.ValueString(), state.PrincipalType.ValueString(), state.PrincipalID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Detach IAM policy", err.Error())
	}
}

func (r *iamPolicyAttachmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(strings.TrimSpace(req.ID), ":", 3)
	if len(parts) != 3 {
		resp.Diagnostics.AddError("Invalid import id", "expected principal_type:principal_id:policy_arn")
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, iamPolicyAttachmentModel{
		ID:            types.StringValue(req.ID),
		PrincipalType: types.StringValue(parts[0]),
		PrincipalID:   types.StringValue(parts[1]),
		PolicyARN:     types.StringValue(parts[2]),
	})...)
}
