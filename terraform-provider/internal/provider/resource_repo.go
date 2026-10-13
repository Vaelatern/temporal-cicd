package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewRepoResource() resource.Resource { return &repoResource{} }

type repoResource struct {
	client *Client
}

type repoModel struct {
	ID                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	URL                  types.String `tfsdk:"url"`
	SSHReadingPrivateKey types.String `tfsdk:"ssh_reading_private_key"`
	SSHReadingPublicKey  types.String `tfsdk:"ssh_reading_public_key"`
}

func (r *repoResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repo"
}

func (r *repoResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Registers a git repo with the lrcicd cache (PUT/POST /sync/{name}).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Internal repo alias. ForceNew — rename = replace.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"url": schema.StringAttribute{
				Required:    true,
				Description: "Upstream git URL (ssh preferred).",
			},
			"ssh_reading_private_key": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "SSH private key used by cache to fetch the upstream.",
			},
			"ssh_reading_public_key": schema.StringAttribute{
				Computed:    true,
				Description: "Public key derived from the configured private key (from GET /sync).",
			},
		},
	}
}

func (r *repoResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("bad provider data", "expected *Client")
		return
	}
	r.client = c
}

func (r *repoResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan repoModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body := RepoRequest{URL: plan.URL.ValueString(), SSHReadingPrivateKey: plan.SSHReadingPrivateKey.ValueString()}
	if err := r.client.PutRepo(plan.Name.ValueString(), body); err != nil {
		resp.Diagnostics.AddError("create failed", err.Error())
		return
	}
	plan.ID = plan.Name
	if st, err := r.client.GetRepo(plan.Name.ValueString()); err == nil && st != nil {
		plan.SSHReadingPublicKey = types.StringValue(st.SSHReadingPublicKey)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *repoResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state repoModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	st, err := r.client.GetRepo(state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("read failed", err.Error())
		return
	}
	if st == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	state.URL = types.StringValue(st.URL)
	state.SSHReadingPublicKey = types.StringValue(st.SSHReadingPublicKey)
	state.ID = state.Name
	// private key is write-only / hashed server-side — keep state value
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *repoResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan repoModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body := RepoRequest{URL: plan.URL.ValueString(), SSHReadingPrivateKey: plan.SSHReadingPrivateKey.ValueString()}
	if err := r.client.PostRepo(plan.Name.ValueString(), body); err != nil {
		resp.Diagnostics.AddError("update failed", err.Error())
		return
	}
	plan.ID = plan.Name
	if st, err := r.client.GetRepo(plan.Name.ValueString()); err == nil && st != nil {
		plan.SSHReadingPublicKey = types.StringValue(st.SSHReadingPublicKey)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *repoResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state repoModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteRepo(state.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError("delete failed", err.Error())
	}
}

func (r *repoResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), req.ID)...)
}
