package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/homecloudlab/terraform-provider-homecloud/internal/client"
)

func NewMDBInstanceResource() resource.Resource {
	return &mdbInstanceResource{}
}

type mdbInstanceResource struct {
	client *client.Client
}

type mdbInstanceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Engine           types.String `tfsdk:"engine"`
	InstanceClass    types.String `tfsdk:"instance_class"`
	EngineVersion    types.String `tfsdk:"engine_version"`
	StorageGi        types.Int64  `tfsdk:"storage_gi"`
	Database         types.String `tfsdk:"database"`
	Owner            types.String `tfsdk:"owner"`
	IamARN           types.String `tfsdk:"iam_arn"`
	Status           types.String `tfsdk:"status"`
	Endpoint         types.String `tfsdk:"endpoint"`
	InternalEndpoint types.String `tfsdk:"internal_endpoint"`
	Port             types.Int64  `tfsdk:"port"`
}

func (r *mdbInstanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mdb_instance"
}

func (r *mdbInstanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Managed database instance (`POST /api/v1/accounts/{id}/databases`). Create waits until `status=active`.",
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
			"engine": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "`postgresql`, `mysql`, or `mongodb`.",
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
			"engine_version": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"storage_gi": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"database": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"owner": schema.StringAttribute{
				Optional: true,
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
			"status":            schema.StringAttribute{Computed: true},
			"endpoint":          schema.StringAttribute{Computed: true},
			"internal_endpoint": schema.StringAttribute{Computed: true},
			"port":              schema.Int64Attribute{Computed: true},
		},
	}
}

func (r *mdbInstanceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *mdbInstanceResource) flatten(d *client.Database, prior mdbInstanceModel) mdbInstanceModel {
	storage := types.Int64Null()
	if d.StorageGi != nil {
		storage = types.Int64Value(*d.StorageGi)
	}
	dbName := types.StringValue(d.Connection.Database)
	if prior.Database.ValueString() != "" && d.Connection.Database == "" {
		dbName = prior.Database
	}
	return mdbInstanceModel{
		ID:               types.StringValue(d.ID),
		Name:             types.StringValue(d.Name),
		Engine:           types.StringValue(d.Engine),
		InstanceClass:    types.StringValue(d.InstanceClass),
		EngineVersion:    types.StringValue(d.EngineVersion),
		StorageGi:        storage,
		Database:         dbName,
		Owner:            prior.Owner,
		IamARN:           types.StringValue(d.IamARN),
		Status:           types.StringValue(d.Status),
		Endpoint:         types.StringValue(d.Connection.Endpoint),
		InternalEndpoint: types.StringValue(d.Connection.InternalEndpoint),
		Port:             types.Int64Value(d.Connection.Port),
	}
}

func (r *mdbInstanceResource) waitActive(ctx context.Context, accountID, name string) (*client.Database, error) {
	var latest *client.Database
	err := r.client.WaitUntilActive(ctx, func() (string, error) {
		got, err := r.client.GetDatabase(ctx, accountID, name)
		if err != nil {
			return "", err
		}
		latest = got
		return got.Status, nil
	}, 20*time.Minute)
	return latest, err
}

func (r *mdbInstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan mdbInstanceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	body := client.DatabaseCreate{
		Name:          plan.Name.ValueString(),
		Engine:        plan.Engine.ValueString(),
		InstanceClass: plan.InstanceClass.ValueString(),
		EngineVersion: plan.EngineVersion.ValueString(),
		Database:      plan.Database.ValueString(),
		Owner:         plan.Owner.ValueString(),
	}
	if !plan.StorageGi.IsNull() && !plan.StorageGi.IsUnknown() {
		v := plan.StorageGi.ValueInt64()
		body.StorageGi = &v
	}
	created, err := r.client.CreateDatabase(ctx, accountID, body)
	if err != nil {
		resp.Diagnostics.AddError("Create MDB instance", err.Error())
		return
	}
	ready := created
	if created.Status != "active" {
		ready, err = r.waitActive(ctx, accountID, created.Name)
		if err != nil {
			resp.Diagnostics.AddError("Wait for MDB instance", err.Error())
			return
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, r.flatten(ready, plan))...)
}

func (r *mdbInstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state mdbInstanceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	got, err := r.client.GetDatabase(ctx, accountID, state.Name.ValueString())
	if err != nil {
		if apiErr, ok := err.(*client.APIError); ok && apiErr.NotFound() {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Read MDB instance", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, r.flatten(got, state))...)
}

func (r *mdbInstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "MDB instance fields require replace.")
}

func (r *mdbInstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state mdbInstanceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	if err := r.client.DeleteDatabase(ctx, accountID, state.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError("Delete MDB instance", err.Error())
	}
}

func (r *mdbInstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), strings.TrimSpace(req.ID))...)
}
