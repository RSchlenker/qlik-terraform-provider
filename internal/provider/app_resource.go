// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/rschlenker/qlik-terraform-provider/internal/sdk/qlikapps"
)

var _ resource.Resource = &AppResource{}
var _ resource.ResourceWithImportState = &AppResource{}

func NewAppResource() resource.Resource {
	return &AppResource{}
}

type AppResource struct {
	client *qlikapps.ClientWithResponses
}

type AppResourceModel struct {
	Id           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Description  types.String `tfsdk:"description"`
	SpaceId      types.String `tfsdk:"space_id"`
	OwnerId      types.String `tfsdk:"owner_id"`
	Locale       types.String `tfsdk:"locale"`
	CreatedDate  types.String `tfsdk:"created_date"`
	ModifiedDate types.String `tfsdk:"modified_date"`
	Published    types.Bool   `tfsdk:"published"`
	PublishTime  types.String `tfsdk:"publish_time"`
	Thumbnail    types.String `tfsdk:"thumbnail"`
}

func (r *AppResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app"
}

func (r *AppResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Qlik Cloud application.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the app.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name (title) of the app.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A description of the app.",
				Optional:            true,
			},
			"space_id": schema.StringAttribute{
				MarkdownDescription: "The space ID the app belongs to.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"owner_id": schema.StringAttribute{
				MarkdownDescription: "The identifier of the app owner.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"locale": schema.StringAttribute{
				MarkdownDescription: "The locale setting of the app.",
				Computed:            true,
			},
			"created_date": schema.StringAttribute{
				MarkdownDescription: "The date and time (RFC3339) when the app was created.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"modified_date": schema.StringAttribute{
				MarkdownDescription: "The date and time (RFC3339) when the app was last modified.",
				Computed:            true,
			},
			"published": schema.BoolAttribute{
				MarkdownDescription: "Whether the app has been published.",
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"publish_time": schema.StringAttribute{
				MarkdownDescription: "The date and time (RFC3339) when the app was published, empty if unpublished.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"thumbnail": schema.StringAttribute{
				MarkdownDescription: "App thumbnail URL.",
				Computed:            true,
			},
		},
	}
}

func (r *AppResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	pc, ok := req.ProviderData.(*ProviderClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *ProviderClient, got: %T", req.ProviderData),
		)
		return
	}

	r.client = pc.Apps
}

func (r *AppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AppResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	attrs := qlikapps.AppAttributes{
		Name: strPtr(data.Name.ValueString()),
	}
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		attrs.Description = strPtr(data.Description.ValueString())
	}
	if !data.SpaceId.IsNull() && !data.SpaceId.IsUnknown() {
		attrs.SpaceId = strPtr(data.SpaceId.ValueString())
	}

	body, err := json.Marshal(qlikapps.CreateApp{Attributes: &attrs})
	if err != nil {
		resp.Diagnostics.AddError("Failed to marshal create request", err.Error())
		return
	}

	apiResp, err := r.client.PostApiV1AppsWithBodyWithResponse(ctx, "application/json", bytes.NewReader(body))
	if err != nil {
		resp.Diagnostics.AddError("Failed to create app", err.Error())
		return
	}

	if apiResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(
			"API error creating app",
			fmt.Sprintf("Status: %s, Body: %s", apiResp.Status(), string(apiResp.Body)),
		)
		return
	}

	mapNxAppToModel(apiResp.JSON200, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AppResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := r.client.GetApiV1AppsAppIdWithResponse(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read app", err.Error())
		return
	}

	if apiResp.StatusCode() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}

	if apiResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(
			"API error reading app",
			fmt.Sprintf("Status: %s, Body: %s", apiResp.Status(), string(apiResp.Body)),
		)
		return
	}

	mapNxAppToModel(apiResp.JSON200, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AppResourceModel
	var state AppResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appId := state.Id.ValueString()

	updateAttrs := qlikapps.AppUpdateAttributes{
		Name: strPtr(plan.Name.ValueString()),
	}
	if !plan.Description.IsUnknown() {
		if !plan.Description.IsNull() {
			updateAttrs.Description = strPtr(plan.Description.ValueString())
		} else {
			// Explicitly set description to empty string when removed from config
			// to clear it via API rather than omitting it (which causes drift)
			updateAttrs.Description = strPtr("")
		}
	}

	body, err := json.Marshal(qlikapps.UpdateApp{Attributes: &updateAttrs})
	if err != nil {
		resp.Diagnostics.AddError("Failed to marshal update request", err.Error())
		return
	}

	putResp, err := r.client.PutApiV1AppsAppIdWithBodyWithResponse(ctx, appId, "application/json", bytes.NewReader(body))
	if err != nil {
		resp.Diagnostics.AddError("Failed to update app", err.Error())
		return
	}

	if putResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(
			"API error updating app",
			fmt.Sprintf("Status: %s, Body: %s", putResp.Status(), string(putResp.Body)),
		)
		return
	}

	mapNxAppToModel(putResp.JSON200, &plan)

	if !plan.SpaceId.Equal(state.SpaceId) {
		if !plan.SpaceId.IsNull() && plan.SpaceId.ValueString() != "" {
			spaceBody, err := json.Marshal(qlikapps.UpdateSpace{SpaceId: strPtr(plan.SpaceId.ValueString())})
			if err != nil {
				resp.Diagnostics.AddError("Failed to marshal space update request", err.Error())
				return
			}

			spaceResp, err := r.client.PutApiV1AppsAppIdSpaceWithBodyWithResponse(ctx, appId, "application/json", bytes.NewReader(spaceBody))
			if err != nil {
				resp.Diagnostics.AddError("Failed to update app space", err.Error())
				return
			}

			if spaceResp.StatusCode() != http.StatusOK {
				resp.Diagnostics.AddError(
					"API error updating app space",
					fmt.Sprintf("Status: %s, Body: %s", spaceResp.Status(), string(spaceResp.Body)),
				)
				return
			}

			mapNxAppToModel(spaceResp.JSON200, &plan)
		} else {
			spaceResp, err := r.client.DeleteApiV1AppsAppIdSpaceWithResponse(ctx, appId)
			if err != nil {
				resp.Diagnostics.AddError("Failed to remove app space", err.Error())
				return
			}

			if spaceResp.StatusCode() != http.StatusOK {
				resp.Diagnostics.AddError(
					"API error removing app space",
					fmt.Sprintf("Status: %s, Body: %s", spaceResp.Status(), string(spaceResp.Body)),
				)
				return
			}

			mapNxAppToModel(spaceResp.JSON200, &plan)
			plan.SpaceId = types.StringNull()
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AppResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AppResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := r.client.DeleteApiV1AppsAppIdWithResponse(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete app", err.Error())
		return
	}

	if apiResp.StatusCode() != http.StatusOK && apiResp.StatusCode() != http.StatusNotFound {
		resp.Diagnostics.AddError(
			"API error deleting app",
			fmt.Sprintf("Status: %s, Body: %s", apiResp.Status(), string(apiResp.Body)),
		)
	}
}

func (r *AppResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func mapNxAppToModel(app *qlikapps.NxApp, model *AppResourceModel) {
	if app == nil || app.Attributes == nil {
		return
	}

	attrs := app.Attributes

	if attrs.Id != nil {
		model.Id = types.StringValue(*attrs.Id)
	}
	if attrs.Name != nil {
		model.Name = types.StringValue(*attrs.Name)
	}
	if attrs.Description != nil {
		model.Description = types.StringValue(*attrs.Description)
	} else {
		model.Description = types.StringNull()
	}
	if attrs.OwnerId != nil {
		model.OwnerId = types.StringValue(*attrs.OwnerId)
	} else {
		model.OwnerId = types.StringNull()
	}
	// SpaceId is not returned in NxAttributes. If it was provided in the plan it is
	// already a known value; resolve any remaining unknown (no space configured) to null.
	if model.SpaceId.IsUnknown() {
		model.SpaceId = types.StringNull()
	}
	// Locale is only accepted as input and is never echoed back in the response.
	model.Locale = types.StringNull()
	if attrs.CreatedDate != nil {
		model.CreatedDate = types.StringValue(attrs.CreatedDate.Format("2006-01-02T15:04:05Z07:00"))
	}
	if attrs.ModifiedDate != nil {
		model.ModifiedDate = types.StringValue(attrs.ModifiedDate.Format("2006-01-02T15:04:05Z07:00"))
	}
	if attrs.Published != nil {
		model.Published = types.BoolValue(*attrs.Published)
	} else {
		model.Published = types.BoolValue(false)
	}
	if attrs.PublishTime != nil && *attrs.PublishTime != "" {
		model.PublishTime = types.StringValue(*attrs.PublishTime)
	} else {
		model.PublishTime = types.StringNull()
	}
	if attrs.Thumbnail != nil {
		model.Thumbnail = types.StringValue(*attrs.Thumbnail)
	} else {
		model.Thumbnail = types.StringNull()
	}
}

func strPtr(s string) *string {
	return &s
}
