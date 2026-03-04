// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"

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

func (p *QlikProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "qlik"
	resp.Version = p.version
}

func (p *QlikProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"tenant_id": schema.StringAttribute{
				MarkdownDescription: "Qlik Cloud tenant subdomain (e.g. `pflege-abc`).",
				Required:            true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "Qlik Cloud region code (e.g. `de`, `eu`, `us`).",
				Required:            true,
			},
			"api_key": schema.StringAttribute{
				MarkdownDescription: "Qlik Cloud API key (used as a Bearer token).",
				Required:            true,
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

	baseURL := fmt.Sprintf("https://%s.%s.qlikcloud.com", data.TenantId.ValueString(), data.Region.ValueString())
	apiKey := data.ApiKey.ValueString()

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

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *QlikProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewAppResource,
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
