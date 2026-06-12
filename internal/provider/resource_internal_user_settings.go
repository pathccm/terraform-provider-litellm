package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &InternalUserSettingsResource{}

func NewInternalUserSettingsResource() resource.Resource {
	return &InternalUserSettingsResource{}
}

type InternalUserSettingsResource struct {
	client *Client
}

type InternalUserSettingsResourceModel struct {
	ID             types.String  `tfsdk:"id"`
	UserRole       types.String  `tfsdk:"user_role"`
	MaxBudget      types.Float64 `tfsdk:"max_budget"`
	BudgetDuration types.String  `tfsdk:"budget_duration"`
	Models         types.List    `tfsdk:"models"`
}

func (r *InternalUserSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_internal_user_settings"
}

func (r *InternalUserSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the default settings applied to new internal users on creation via /update/internal_user_settings. This is a singleton resource — only one should exist per LiteLLM deployment.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Singleton identifier, always \"internal_user_settings\".",
				Computed:    true,
			},
			"user_role": schema.StringAttribute{
				Description: "Default role assigned to new users. One of: internal_user, internal_user_viewer, proxy_admin, proxy_admin_viewer.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("internal_user_viewer"),
			},
			"max_budget": schema.Float64Attribute{
				Description: "Default maximum budget (USD) for new users.",
				Optional:    true,
			},
			"budget_duration": schema.StringAttribute{
				Description: "Default budget reset cadence for new users (e.g. '1mo', 'daily', 'weekly').",
				Optional:    true,
			},
			"models": schema.ListAttribute{
				Description: "Default list of models (or access group names) new users can access.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *InternalUserSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *Client, got: %T.", req.ProviderData))
		return
	}
	r.client = client
}

func (r *InternalUserSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data InternalUserSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.applySettings(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update internal user settings: %s", err))
		return
	}

	data.ID = types.StringValue("internal_user_settings")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *InternalUserSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data InternalUserSettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result := make(map[string]interface{})
	if err := r.client.DoRequestWithResponse(ctx, "GET", "/get/internal_user_settings", nil, &result); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read internal user settings: %s", err))
		return
	}

	if v, ok := result["user_role"].(string); ok {
		data.UserRole = types.StringValue(v)
	}
	if v, ok := result["max_budget"].(float64); ok {
		data.MaxBudget = types.Float64Value(v)
	} else {
		data.MaxBudget = types.Float64Null()
	}
	if v, ok := result["budget_duration"].(string); ok {
		data.BudgetDuration = types.StringValue(v)
	} else {
		data.BudgetDuration = types.StringNull()
	}
	if models, ok := result["models"].([]interface{}); ok {
		elems := make([]types.String, len(models))
		for i, m := range models {
			elems[i] = types.StringValue(fmt.Sprintf("%v", m))
		}
		listVal, diags := types.ListValueFrom(ctx, types.StringType, elems)
		resp.Diagnostics.Append(diags...)
		data.Models = listVal
	} else if data.Models.IsUnknown() {
		data.Models, _ = types.ListValue(types.StringType, []attr.Value{})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *InternalUserSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data InternalUserSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.applySettings(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update internal user settings: %s", err))
		return
	}

	data.ID = types.StringValue("internal_user_settings")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete is a no-op — there is no API to delete global default settings.
func (r *InternalUserSettingsResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
}

func (r *InternalUserSettingsResource) applySettings(ctx context.Context, data *InternalUserSettingsResourceModel) error {
	body := map[string]interface{}{}

	if !data.UserRole.IsNull() && !data.UserRole.IsUnknown() {
		body["user_role"] = data.UserRole.ValueString()
	}
	if !data.MaxBudget.IsNull() && !data.MaxBudget.IsUnknown() {
		body["max_budget"] = data.MaxBudget.ValueFloat64()
	}
	if !data.BudgetDuration.IsNull() && !data.BudgetDuration.IsUnknown() {
		body["budget_duration"] = data.BudgetDuration.ValueString()
	}
	if !data.Models.IsNull() && !data.Models.IsUnknown() {
		var models []string
		data.Models.ElementsAs(ctx, &models, false)
		body["models"] = models
	}

	return r.client.DoRequestWithResponse(ctx, "PATCH", "/update/internal_user_settings", body, nil)
}
