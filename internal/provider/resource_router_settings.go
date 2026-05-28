package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &RouterSettingsResource{}
var _ resource.ResourceWithImportState = &RouterSettingsResource{}

func NewRouterSettingsResource() resource.Resource {
	return &RouterSettingsResource{}
}

type RouterSettingsResource struct {
	client *Client
}

type RouterSettingsResourceModel struct {
	ID                     types.String  `tfsdk:"id"`
	RoutingStrategy        types.String  `tfsdk:"routing_strategy"`
	RoutingStrategyArgs    types.String  `tfsdk:"routing_strategy_args"`
	ModelGroupRetryPolicy  types.String  `tfsdk:"model_group_retry_policy"`
	AllowedFails           types.Int64   `tfsdk:"allowed_fails"`
	CooldownTime           types.Float64 `tfsdk:"cooldown_time"`
	NumRetries             types.Int64   `tfsdk:"num_retries"`
	Timeout                types.Float64 `tfsdk:"timeout"`
	MaxRetries             types.Int64   `tfsdk:"max_retries"`
	RetryAfter             types.Float64 `tfsdk:"retry_after"`
	EnablePreCallChecks    types.Bool    `tfsdk:"enable_pre_call_checks"`
	ModelGroupAlias        types.String  `tfsdk:"model_group_alias"`
}

func (r *RouterSettingsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_router_settings"
}

func (r *RouterSettingsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages global LiteLLM router settings. This is a singleton resource — only one instance should exist per LiteLLM deployment. Destroying this resource removes it from state only; router settings have no meaningful API delete.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Always 'router_settings'.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"routing_strategy": schema.StringAttribute{
				Description: "The routing strategy to use. Options: 'simple-shuffle', 'least-busy', 'latency-based-routing', 'usage-based-routing', 'cost-based-routing'.",
				Optional:    true,
				Computed:    true,
			},
			"routing_strategy_args": schema.StringAttribute{
				Description: "JSON-encoded map of additional arguments for the routing strategy.",
				Optional:    true,
				Computed:    true,
			},
			"model_group_retry_policy": schema.StringAttribute{
				Description: "JSON-encoded map of per-model-group retry policies.",
				Optional:    true,
				Computed:    true,
			},
			"allowed_fails": schema.Int64Attribute{
				Description: "Number of failures allowed before a model is put in cooldown.",
				Optional:    true,
				Computed:    true,
			},
			"cooldown_time": schema.Float64Attribute{
				Description: "Seconds a model stays in cooldown after exceeding allowed_fails.",
				Optional:    true,
				Computed:    true,
			},
			"num_retries": schema.Int64Attribute{
				Description: "Number of retries per request before marking as failed.",
				Optional:    true,
				Computed:    true,
			},
			"timeout": schema.Float64Attribute{
				Description: "Request timeout in seconds.",
				Optional:    true,
				Computed:    true,
			},
			"max_retries": schema.Int64Attribute{
				Description: "Maximum total retries across all fallbacks.",
				Optional:    true,
				Computed:    true,
			},
			"retry_after": schema.Float64Attribute{
				Description: "Seconds to wait before retrying a failed request.",
				Optional:    true,
				Computed:    true,
			},
			"enable_pre_call_checks": schema.BoolAttribute{
				Description: "Whether to run pre-call checks (e.g. context window validation) before routing.",
				Optional:    true,
				Computed:    true,
			},
			"model_group_alias": schema.StringAttribute{
				Description: "JSON-encoded map of model group aliases (e.g. {\"my-alias\": \"gpt-4\"}).",
				Optional:    true,
				Computed:    true,
			},
		},
	}
}

func (r *RouterSettingsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *Client, got: %T.", req.ProviderData),
		)
		return
	}
	r.client = client
}

func (r *RouterSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RouterSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DoRequestWithResponse(ctx, "POST", "/router/settings", r.buildRequest(&data), nil); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create router settings: %s", err))
		return
	}

	data.ID = types.StringValue("router_settings")

	if err := r.readRouterSettings(ctx, &data); err != nil {
		resp.Diagnostics.AddWarning("Read Error", fmt.Sprintf("Router settings created but failed to read back: %s", err))
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RouterSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RouterSettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.readRouterSettings(ctx, &data); err != nil {
		if IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read router settings: %s", err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RouterSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data RouterSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DoRequestWithResponse(ctx, "POST", "/router/settings", r.buildRequest(&data), nil); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update router settings: %s", err))
		return
	}

	if err := r.readRouterSettings(ctx, &data); err != nil {
		resp.Diagnostics.AddWarning("Read Error", fmt.Sprintf("Router settings updated but failed to read back: %s", err))
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete removes the resource from state only. Router settings are global config with no meaningful delete.
func (r *RouterSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
}

func (r *RouterSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	data := RouterSettingsResourceModel{
		ID: types.StringValue("router_settings"),
	}
	if err := r.readRouterSettings(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to read router settings after import: %s", err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RouterSettingsResource) buildRequest(data *RouterSettingsResourceModel) map[string]interface{} {
	req := map[string]interface{}{}
	if !data.RoutingStrategy.IsNull() && !data.RoutingStrategy.IsUnknown() {
		req["routing_strategy"] = data.RoutingStrategy.ValueString()
	}
	if !data.RoutingStrategyArgs.IsNull() && !data.RoutingStrategyArgs.IsUnknown() {
		req["routing_strategy_args"] = data.RoutingStrategyArgs.ValueString()
	}
	if !data.ModelGroupRetryPolicy.IsNull() && !data.ModelGroupRetryPolicy.IsUnknown() {
		req["model_group_retry_policy"] = data.ModelGroupRetryPolicy.ValueString()
	}
	if !data.AllowedFails.IsNull() && !data.AllowedFails.IsUnknown() {
		req["allowed_fails"] = data.AllowedFails.ValueInt64()
	}
	if !data.CooldownTime.IsNull() && !data.CooldownTime.IsUnknown() {
		req["cooldown_time"] = data.CooldownTime.ValueFloat64()
	}
	if !data.NumRetries.IsNull() && !data.NumRetries.IsUnknown() {
		req["num_retries"] = data.NumRetries.ValueInt64()
	}
	if !data.Timeout.IsNull() && !data.Timeout.IsUnknown() {
		req["timeout"] = data.Timeout.ValueFloat64()
	}
	if !data.MaxRetries.IsNull() && !data.MaxRetries.IsUnknown() {
		req["max_retries"] = data.MaxRetries.ValueInt64()
	}
	if !data.RetryAfter.IsNull() && !data.RetryAfter.IsUnknown() {
		req["retry_after"] = data.RetryAfter.ValueFloat64()
	}
	if !data.EnablePreCallChecks.IsNull() && !data.EnablePreCallChecks.IsUnknown() {
		req["enable_pre_call_checks"] = data.EnablePreCallChecks.ValueBool()
	}
	if !data.ModelGroupAlias.IsNull() && !data.ModelGroupAlias.IsUnknown() {
		req["model_group_alias"] = data.ModelGroupAlias.ValueString()
	}
	return req
}

func (r *RouterSettingsResource) readRouterSettings(ctx context.Context, data *RouterSettingsResourceModel) error {
	var result map[string]interface{}
	if err := r.client.DoRequestWithResponse(ctx, "GET", "/router/settings", nil, &result); err != nil {
		return err
	}

	// LiteLLM returns {"current_values": {...}, "fields": [...]}
	values, ok := result["current_values"].(map[string]interface{})
	if !ok {
		// Some versions return the fields at the top level
		values = result
	}

	if v, ok := values["routing_strategy"].(string); ok {
		data.RoutingStrategy = types.StringValue(v)
	}
	if v, ok := values["routing_strategy_args"]; ok && v != nil {
		data.RoutingStrategyArgs = types.StringValue(fmt.Sprintf("%v", v))
	}
	if v, ok := values["model_group_retry_policy"]; ok && v != nil {
		data.ModelGroupRetryPolicy = types.StringValue(fmt.Sprintf("%v", v))
	}
	if v, ok := values["allowed_fails"]; ok && v != nil {
		if f, ok := toFloat64(v); ok {
			data.AllowedFails = types.Int64Value(int64(f))
		}
	}
	if v, ok := values["cooldown_time"]; ok && v != nil {
		if f, ok := toFloat64(v); ok {
			data.CooldownTime = types.Float64Value(f)
		}
	}
	if v, ok := values["num_retries"]; ok && v != nil {
		if f, ok := toFloat64(v); ok {
			data.NumRetries = types.Int64Value(int64(f))
		}
	}
	if v, ok := values["timeout"]; ok && v != nil {
		if f, ok := toFloat64(v); ok {
			data.Timeout = types.Float64Value(f)
		}
	}
	if v, ok := values["max_retries"]; ok && v != nil {
		if f, ok := toFloat64(v); ok {
			data.MaxRetries = types.Int64Value(int64(f))
		}
	}
	if v, ok := values["retry_after"]; ok && v != nil {
		if f, ok := toFloat64(v); ok {
			data.RetryAfter = types.Float64Value(f)
		}
	}
	if v, ok := values["enable_pre_call_checks"].(bool); ok {
		data.EnablePreCallChecks = types.BoolValue(v)
	}
	if v, ok := values["model_group_alias"]; ok && v != nil {
		data.ModelGroupAlias = types.StringValue(fmt.Sprintf("%v", v))
	}

	data.ID = types.StringValue("router_settings")
	return nil
}

// toFloat64 converts JSON number types (float64, int, int64) to float64.
func toFloat64(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	}
	return 0, false
}
