// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/rschlenker/qlik-terraform-provider/internal/sdk/qlikapps"
)

var _ resource.Resource = &AppScriptResource{}

func NewAppScriptResource() resource.Resource {
	return &AppScriptResource{}
}

type AppScriptResource struct {
	client *qlikapps.ClientWithResponses
}

type AppScriptResourceModel struct {
	AppId          types.String `tfsdk:"app_id"`
	Script         types.String `tfsdk:"script"`
	VersionMessage types.String `tfsdk:"version_message"`
	ScriptId       types.String `tfsdk:"script_id"`
	ModifiedTime   types.String `tfsdk:"modified_time"`
	ModifierId     types.String `tfsdk:"modifier_id"`
}

func (r *AppScriptResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_script"
}

func (r *AppScriptResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a script version for a Qlik Cloud application.",
		Attributes: map[string]schema.Attribute{
			"app_id": schema.StringAttribute{
				MarkdownDescription: "The identifier of the parent application.",
				Required:            true,
			},
			"script": schema.StringAttribute{
				MarkdownDescription: "The raw script text. Use the `file()` function to load from disk.",
				Required:            true,
			},
			"version_message": schema.StringAttribute{
				MarkdownDescription: "An optional description of this script version.",
				Optional:            true,
			},
			"script_id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the script.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"modified_time": schema.StringAttribute{
				MarkdownDescription: "The last modification time of the script version.",
				Computed:            true,
			},
			"modifier_id": schema.StringAttribute{
				MarkdownDescription: "The identifier of the user who last modified the script.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *AppScriptResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*qlikapps.ClientWithResponses)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *qlikapps.ClientWithResponses, got: %T", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *AppScriptResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AppScriptResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	scriptVersion := qlikapps.ScriptVersion{
		Script: strPtr(data.Script.ValueString()),
	}
	if !data.VersionMessage.IsNull() && !data.VersionMessage.IsUnknown() {
		scriptVersion.VersionMessage = strPtr(data.VersionMessage.ValueString())
	}

	body, err := json.Marshal(scriptVersion)
	if err != nil {
		resp.Diagnostics.AddError("Failed to marshal script request", err.Error())
		return
	}

	apiResp, err := r.client.PostApiV1AppsAppIdScriptsWithBodyWithResponse(
		ctx,
		data.AppId.ValueString(),
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create script", err.Error())
		return
	}

	if apiResp.StatusCode() != http.StatusOK && apiResp.StatusCode() != http.StatusCreated {
		resp.Diagnostics.AddError(
			"API error creating script",
			fmt.Sprintf("Status: %s, Body: %s", apiResp.Status(), string(apiResp.Body)),
		)
		return
	}

	r.fetchAndUpdateScriptMetadata(ctx, data.AppId.ValueString(), &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AppScriptResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AppScriptResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := r.client.GetApiV1AppsAppIdScriptsIdWithResponse(ctx, data.AppId.ValueString(), "current")
	if err != nil {
		resp.Diagnostics.AddError("Failed to read script", err.Error())
		return
	}

	if apiResp.StatusCode() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}

	if apiResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(
			"API error reading script",
			fmt.Sprintf("Status: %s, Body: %s", apiResp.Status(), string(apiResp.Body)),
		)
		return
	}

	if apiResp.JSON200 != nil {
		if apiResp.JSON200.Script != nil && *apiResp.JSON200.Script != data.Script.ValueString() {
			data.Script = types.StringValue(*apiResp.JSON200.Script)
		}
		if apiResp.JSON200.VersionMessage != nil {
			data.VersionMessage = types.StringValue(*apiResp.JSON200.VersionMessage)
		} else {
			data.VersionMessage = types.StringNull()
		}

		r.fetchAndUpdateScriptMetadata(ctx, data.AppId.ValueString(), &data, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AppScriptResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state AppScriptResourceModel
	var plan AppScriptResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.AppId.ValueString() != plan.AppId.ValueString() {
		resp.Diagnostics.AddError(
			"app_id is immutable",
			fmt.Sprintf("Cannot change app_id from %q to %q", state.AppId.ValueString(), plan.AppId.ValueString()),
		)
		return
	}

	appId := plan.AppId.ValueString()

	scriptVersion := qlikapps.ScriptVersion{
		Script: strPtr(plan.Script.ValueString()),
	}
	if !plan.VersionMessage.IsNull() && !plan.VersionMessage.IsUnknown() {
		scriptVersion.VersionMessage = strPtr(plan.VersionMessage.ValueString())
	}

	body, err := json.Marshal(scriptVersion)
	if err != nil {
		resp.Diagnostics.AddError("Failed to marshal script request", err.Error())
		return
	}

	apiResp, err := r.client.PostApiV1AppsAppIdScriptsWithBodyWithResponse(
		ctx,
		appId,
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update script", err.Error())
		return
	}

	if apiResp.StatusCode() != http.StatusOK && apiResp.StatusCode() != http.StatusCreated {
		resp.Diagnostics.AddError(
			"API error updating script",
			fmt.Sprintf("Status: %s, Body: %s", apiResp.Status(), string(apiResp.Body)),
		)
		return
	}

	r.fetchAndUpdateScriptMetadata(ctx, appId, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AppScriptResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Warn(ctx, "qlik_app_script does not support deletion; removing from state only")
}

func (r *AppScriptResource) fetchAndUpdateScriptMetadata(
	ctx context.Context,
	appId string,
	model *AppScriptResourceModel,
	diags *diag.Diagnostics,
) {
	apiResp, err := r.client.GetApiV1AppsAppIdScriptsWithResponse(
		ctx,
		appId,
		&qlikapps.GetApiV1AppsAppIdScriptsParams{
			Limit: strPtr("1"),
		},
	)
	if err != nil {
		diags.AddError("Failed to fetch script metadata", err.Error())
		return
	}

	if apiResp.StatusCode() != http.StatusOK {
		diags.AddError(
			"API error fetching script metadata",
			fmt.Sprintf("Status: %s, Body: %s", apiResp.Status(), string(apiResp.Body)),
		)
		return
	}

	if apiResp.JSON200 == nil || apiResp.JSON200.Scripts == nil || len(*apiResp.JSON200.Scripts) == 0 {
		diags.AddError(
			"No script versions found",
			"Expected at least one script version after creation/update",
		)
		return
	}

	meta := (*apiResp.JSON200.Scripts)[0]

	if meta.ScriptId != nil {
		model.ScriptId = types.StringValue(*meta.ScriptId)
	}
	if meta.ModifiedTime != nil {
		model.ModifiedTime = types.StringValue(*meta.ModifiedTime)
	}
	if meta.ModifierId != nil {
		model.ModifierId = types.StringValue(*meta.ModifierId)
	}
}

