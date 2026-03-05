// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/rschlenker/qlik-terraform-provider/internal/sdk/qlikapps"
)

// Ensure QlikProvider satisfies the provider interface.
var _ provider.Provider = &QlikProvider{}

// QlikProvider defines the provider implementation.
type QlikProvider struct {
	version string
}

// QlikProviderModel describes the provider data model.
type QlikProviderModel struct {
	TenantId types.String `tfsdk:"tenant_id"`
	Region   types.String `tfsdk:"region"`
	ApiKey   types.String `tfsdk:"api_key"`
}

// ProviderClient holds the API clients shared with resources and data sources.
type ProviderClient struct {
	Apps *qlikapps.ClientWithResponses
	// Spaces and Automations added in T3
}

func (p *QlikProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "qlik"
	resp.Version = p.version
}

func (p *QlikProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"tenant_id": schema.StringAttribute{
				MarkdownDescription: "Qlik Cloud tenant subdomain (e.g. `pflege-abc`). Can also be set via `QLIK_TENANT_ID`.",
				Optional:            true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "Qlik Cloud region code (e.g. `de`, `eu`, `us`). Can also be set via `QLIK_REGION`.",
				Optional:            true,
			},
			"api_key": schema.StringAttribute{
				MarkdownDescription: "Qlik Cloud API key (used as a Bearer token). Can also be set via `QLIK_API_KEY`.",
				Optional:            true,
				Sensitive:           true,
			},
		},
	}
}

func (p *QlikProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data QlikProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tenantId := data.TenantId.ValueString()
	if tenantId == "" {
		tenantId = os.Getenv("QLIK_TENANT_ID")
	}

	region := data.Region.ValueString()
	if region == "" {
		region = os.Getenv("QLIK_REGION")
	}

	apiKey := data.ApiKey.ValueString()
	if apiKey == "" {
		apiKey = os.Getenv("QLIK_API_KEY")
	}

	if tenantId == "" {
		resp.Diagnostics.AddError(
			"Missing provider configuration",
			"tenant_id must be set in the provider block or via QLIK_TENANT_ID",
		)
	}
	if region == "" {
		resp.Diagnostics.AddError(
			"Missing provider configuration",
			"region must be set in the provider block or via QLIK_REGION",
		)
	}
	if apiKey == "" {
		resp.Diagnostics.AddError(
			"Missing provider configuration",
			"api_key must be set in the provider block or via QLIK_API_KEY",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	baseURL := fmt.Sprintf("https://%s.%s.qlikcloud.com", tenantId, region)

	client, err := qlikapps.NewClientWithResponses(
		baseURL,
		qlikapps.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
			req.Header.Set("Authorization", "Bearer "+apiKey)
			return nil
		}),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Qlik API client", err.Error())
		return
	}

	pc := &ProviderClient{Apps: client}
	resp.DataSourceData = pc
	resp.ResourceData = pc
}

func (p *QlikProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewAppResource,
		NewAppScriptResource,
	}
}

func (p *QlikProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &QlikProvider{
			version: version,
		}
	}
}
