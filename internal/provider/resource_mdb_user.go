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

func NewMDBUserResource() resource.Resource {
	return &mdbUserResource{}
}

type mdbUserResource struct {
	client *client.Client
}

type mdbUserModel struct {
	ID           types.String `tfsdk:"id"`
	InstanceName types.String `tfsdk:"instance_name"`
	Username     types.String `tfsdk:"username"`
	Password     types.String `tfsdk:"password"`
	Role         types.String `tfsdk:"role"`
	Database     types.String `tfsdk:"database"`
	Phase        types.String `tfsdk:"phase"`
}

func (r *mdbUserResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mdb_user"
}

func (r *mdbUserResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Managed database login (`POST /databases/{instance}/users`). `password` is write-only (Terraform 1.11+) and never stored in state. Instance must already be `active`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"instance_name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"username": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"password": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				WriteOnly:           true,
				MarkdownDescription: "Login password. Write-only; never in state. Required on create.",
			},
			"role": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "`readwrite` (default) or `readonly`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"database": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"phase": schema.StringAttribute{Computed: true},
		},
	}
}

func (r *mdbUserResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *mdbUserResource) flatten(instance, username string, user *client.DatabaseUser, database types.String) mdbUserModel {
	role := types.StringNull()
	if user.Role != "" {
		role = types.StringValue(user.Role)
	}
	return mdbUserModel{
		ID:           types.StringValue(instance + "/" + username),
		InstanceName: types.StringValue(instance),
		Username:     types.StringValue(username),
		Password:     types.StringNull(),
		Role:         role,
		Database:     database,
		Phase:        types.StringValue(user.Phase),
	}
}

func (r *mdbUserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var config mdbUserModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	var plan mdbUserModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	password := config.Password.ValueString()
	if password == "" {
		resp.Diagnostics.AddError("Missing password", "homecloud_mdb_user.password is required on create.")
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	role := plan.Role.ValueString()
	if role == "" {
		role = "readwrite"
	}
	created, err := r.client.CreateDatabaseUser(
		ctx,
		accountID,
		plan.InstanceName.ValueString(),
		plan.Username.ValueString(),
		password,
		role,
		plan.Database.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Create MDB user", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, r.flatten(plan.InstanceName.ValueString(), plan.Username.ValueString(), created, plan.Database))...)
}

func (r *mdbUserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state mdbUserModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	got, err := r.client.GetDatabaseUser(ctx, accountID, state.InstanceName.ValueString(), state.Username.ValueString())
	if err != nil {
		if apiErr, ok := err.(*client.APIError); ok && apiErr.NotFound() {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Read MDB user", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, r.flatten(state.InstanceName.ValueString(), state.Username.ValueString(), got, state.Database))...)
}

func (r *mdbUserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var config mdbUserModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	var plan mdbUserModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	password := config.Password.ValueString()
	if password == "" {
		resp.Diagnostics.AddError("Missing password", "Set password to rotate the login.")
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	updated, err := r.client.RotateDatabaseUser(ctx, accountID, plan.InstanceName.ValueString(), plan.Username.ValueString(), password)
	if err != nil {
		resp.Diagnostics.AddError("Rotate MDB user", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, r.flatten(plan.InstanceName.ValueString(), plan.Username.ValueString(), updated, plan.Database))...)
}

func (r *mdbUserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state mdbUserModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := r.client.ResolveAccountID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Resolve account", err.Error())
		return
	}
	if err := r.client.DeleteDatabaseUser(ctx, accountID, state.InstanceName.ValueString(), state.Username.ValueString()); err != nil {
		resp.Diagnostics.AddError("Delete MDB user", err.Error())
	}
}

func (r *mdbUserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(strings.TrimSpace(req.ID), "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import id", "expected instance_name/username")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("username"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
