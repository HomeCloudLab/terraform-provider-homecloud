package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/homecloudlab/terraform-provider-homecloud/internal/client"
)

func NewRedisInstanceResource() resource.Resource {
	return &redisInstanceResource{}
}

type redisInstanceResource struct {
	client *client.Client
}

type redisInstanceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	InstanceClass     types.String `tfsdk:"instance_class"`
	RedisVersion      types.String `tfsdk:"redis_version"`
	IamARN            types.String `tfsdk:"iam_arn"`
	Status            types.String `tfsdk:"status"`
	Endpoint          types.String `tfsdk:"endpoint"`
	InternalEndpoint  types.String `tfsdk:"internal_endpoint"`
	Port              types.Int64  `tfsdk:"port"`
	CredentialsSecret types.String `tfsdk:"credentials_secret"`
}

func (r *redisInstanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_redis_instance"
}

func (r *redisInstanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Managed Redis cache (`POST /api/v1/accounts/{id}/caches`). Create waits until `status=active`. Password lives in `credentials_secret`, not in this resource.",
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
			"instance_class": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"redis_version": schema.StringAttribute{
				Optional: true,
				Computed: true,
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
			"status":             schema.StringAttribute{Computed: true},
			"endpoint":           schema.StringAttribute{Computed: true},
			"internal_endpoint":  schema.StringAttribute{Computed: true},
			"port":               schema.Int64Attribute{Computed: true},
			"credentials_secret": schema.StringAttribute{Computed: true},
		},
	}
}

func (r *redisInstanceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *redisInstanceResource) flatten(cache *client.Cache) redisInstanceModel {
	return redisInstanceModel{
		ID:                types.StringValue(cache.ID),
		Name:              types.StringValue(cache.Name),
		InstanceClass:     types.StringValue(cache.InstanceClass),
		RedisVersion:      types.StringValue(cache.RedisVersion),
		IamARN:            types.StringValue(cache.IamARN),
		Status:            types.StringValue(cache.Status),
		Endpoint:          types.StringValue(cache.Connection.Endpoint),
		InternalEndpoint:  types.StringValue(cache.Connection.InternalEndpoint),
		Port:              types.Int64Value(cache.Connection.Port),
		CredentialsSecret: types.StringValue(cache.Connection.CredentialsSecret),
	}
}

func (r *redisInstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan redisInstanceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	created, err := r.client.CreateCache(ctx, accountID, client.CacheCreate{
		Name:          plan.Name.ValueString(),
		InstanceClass: plan.InstanceClass.ValueString(),
		RedisVersion:  plan.RedisVersion.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Create Redis instance", err.Error())
		return
	}
	ready := created
	if created.Status != "active" {
		var latest *client.Cache
		err = r.client.WaitUntilActive(ctx, func() (string, error) {
			got, err := r.client.GetCache(ctx, accountID, created.Name)
			if err != nil {
				return "", err
			}
			latest = got
			return got.Status, nil
		}, 10*time.Minute)
		if err != nil {
			resp.Diagnostics.AddError("Wait for Redis instance", err.Error())
			return
		}
		ready = latest
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, r.flatten(ready))...)
}

func (r *redisInstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state redisInstanceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	got, err := r.client.GetCache(ctx, accountID, state.Name.ValueString())
	if err != nil {
		if apiErr, ok := err.(*client.APIError); ok && apiErr.NotFound() {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Read Redis instance", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, r.flatten(got))...)
}

func (r *redisInstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "Redis instance fields require replace.")
}

func (r *redisInstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state redisInstanceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	if err := r.client.DeleteCache(ctx, accountID, state.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError("Delete Redis instance", err.Error())
	}
}

func (r *redisInstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), strings.TrimSpace(req.ID))...)
}
