package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func New() provider.Provider { return &lrcicdProvider{} }

type lrcicdProvider struct{}

type providerModel struct {
	CacheURL types.String `tfsdk:"cache_url"`
	Token    types.String `tfsdk:"token"`
	DryRun   types.Bool   `tfsdk:"dry_run"`
}

func (p *lrcicdProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "lrcicd"
}

func (p *lrcicdProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"cache_url": schema.StringAttribute{
				Optional:    true,
				Description: "Cache service base URL (env TCD_CACHE_URL).",
			},
			"token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Bearer token for cache API (env TCD_CACHE_TOKEN).",
			},
			"dry_run": schema.BoolAttribute{
				Optional:    true,
				Description: "Skip mutating cache calls; plan/apply still succeed. Terraform plan is the real dry-run.",
			},
		},
	}
}

func (p *lrcicdProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	base := cfg.CacheURL.ValueString()
	if base == "" {
		base = os.Getenv("TCD_CACHE_URL")
	}
	token := cfg.Token.ValueString()
	if token == "" {
		token = os.Getenv("TCD_CACHE_TOKEN")
	}
	if base == "" {
		resp.Diagnostics.AddError("missing cache_url", "set provider cache_url or TCD_CACHE_URL")
		return
	}
	c := &Client{BaseURL: base, Token: token, DryRun: cfg.DryRun.ValueBool()}
	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *lrcicdProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{NewRepoResource}
}

func (p *lrcicdProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}
