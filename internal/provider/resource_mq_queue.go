package provider

import (
	"context"
	"encoding/json"
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

func NewMQQueueResource() resource.Resource {
	return &mqQueueResource{}
}

type mqQueueResource struct {
	client *client.Client
}

type mqQueueModel struct {
	ID                       types.String `tfsdk:"id"`
	Name                     types.String `tfsdk:"name"`
	IamARN                   types.String `tfsdk:"iam_arn"`
	Status                   types.String `tfsdk:"status"`
	QueueURL                 types.String `tfsdk:"queue_url"`
	MaxMessageSize           types.Int64  `tfsdk:"max_message_size"`
	VisibilityTimeoutSeconds types.Int64  `tfsdk:"visibility_timeout_seconds"`
	MaxReceiveCount          types.Int64  `tfsdk:"max_receive_count"`
	MessageRetentionSeconds  types.Int64  `tfsdk:"message_retention_seconds"`
}

func (r *mqQueueResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mq_queue"
}

func (r *mqQueueResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Managed MQ queue (`POST /api/v1/accounts/{id}/queues`).",
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
				Computed:            true,
				MarkdownDescription: "IAM-canonical ARN (`arn:homecloud:mq::{account_number}:queue/{name}`).",
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
			"queue_url": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"max_message_size": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"visibility_timeout_seconds": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"max_receive_count": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"message_retention_seconds": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *mqQueueResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func optionalInt(v types.Int64) *int64 {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	n := v.ValueInt64()
	return &n
}

func intOrNull(v *int64) types.Int64 {
	if v == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*v)
}

func (r *mqQueueResource) flatten(q *client.Queue) mqQueueModel {
	return mqQueueModel{
		ID:                       types.StringValue(q.ID),
		Name:                     types.StringValue(q.Name),
		IamARN:                   types.StringValue(q.IamARN),
		Status:                   types.StringValue(q.Status),
		QueueURL:                 types.StringValue(q.QueueURL),
		MaxMessageSize:           intOrNull(q.MaxMessageSize),
		VisibilityTimeoutSeconds: intOrNull(q.VisibilityTimeoutSeconds),
		MaxReceiveCount:          intOrNull(q.MaxReceiveCount),
		MessageRetentionSeconds:  intOrNull(q.MessageRetentionSeconds),
	}
}

func (r *mqQueueResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan mqQueueModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	created, err := r.client.CreateQueue(ctx, accountID, client.QueueCreate{
		Name:                     plan.Name.ValueString(),
		MaxMessageSize:           optionalInt(plan.MaxMessageSize),
		VisibilityTimeoutSeconds: optionalInt(plan.VisibilityTimeoutSeconds),
		MaxReceiveCount:          optionalInt(plan.MaxReceiveCount),
		MessageRetentionSeconds:  optionalInt(plan.MessageRetentionSeconds),
	})
	if err != nil {
		resp.Diagnostics.AddError("Create queue", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, r.flatten(created))...)
}

func (r *mqQueueResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state mqQueueModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	got, err := r.client.GetQueue(ctx, accountID, state.Name.ValueString())
	if err != nil {
		if apiErr, ok := err.(*client.APIError); ok && apiErr.NotFound() {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Read queue", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, r.flatten(got))...)
}

func (r *mqQueueResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan mqQueueModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	updated, err := r.client.UpdateQueue(ctx, accountID, plan.Name.ValueString(), client.QueueUpdate{
		MaxMessageSize:           optionalInt(plan.MaxMessageSize),
		VisibilityTimeoutSeconds: optionalInt(plan.VisibilityTimeoutSeconds),
		MaxReceiveCount:          optionalInt(plan.MaxReceiveCount),
		MessageRetentionSeconds:  optionalInt(plan.MessageRetentionSeconds),
	})
	if err != nil {
		resp.Diagnostics.AddError("Update queue", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, r.flatten(updated))...)
}

func (r *mqQueueResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state mqQueueModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	if err := r.client.DeleteQueue(ctx, accountID, state.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError("Delete queue", err.Error())
	}
}

func (r *mqQueueResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id := strings.TrimSpace(req.ID)
	if id == "" {
		resp.Diagnostics.AddError("Invalid import id", "expected queue name or resource UUID")
		return
	}
	if strings.Contains(id, "-") && len(id) == 36 {
		accountID, err := r.client.ResolveAccountID(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Resolve account", err.Error())
			return
		}
		raw, _, err := r.client.Do(ctx, "GET", "/api/v1/accounts/"+accountID+"/queues", accountID, "", nil)
		if err != nil {
			resp.Diagnostics.AddError("List queues", err.Error())
			return
		}
		var listing struct {
			Items []client.Queue `json:"items"`
		}
		if err := json.Unmarshal(raw, &listing); err != nil {
			resp.Diagnostics.AddError("Parse queue list", err.Error())
			return
		}
		for _, item := range listing.Items {
			if item.ID == id {
				resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), item.Name)...)
				resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), item.ID)...)
				return
			}
		}
		resp.Diagnostics.AddError("Queue not found", "no queue with id "+id)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), id)...)
}
