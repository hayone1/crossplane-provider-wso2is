package clients

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	fwprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
)

// WithConfigChecks wraps the Terraform provider so that each resource runs its own
// configuration checks (ValidateConfig and ConfigValidators) before Create and Update.
// Terraform runs them at plan time, but Upjet never calls ValidateResourceConfig, so without
// the wrapper a spec that breaks a rule, such as a PushSender with both fcm and amazonSns,
// would reach the server. The checks run only before writes, so a managed resource with an
// invalid spec can still be observed and deleted.
func WithConfigChecks(newProvider func() fwprovider.Provider) func() fwprovider.Provider {
	return func() fwprovider.Provider {
		return configCheckProvider{Provider: newProvider()}
	}
}

// configCheckProvider forwards everything to the Terraform provider and wraps its resources.
// terraform-provider-wso2is implements no optional provider interfaces, so embedding the
// interface hides nothing.
type configCheckProvider struct {
	fwprovider.Provider
}

func (p configCheckProvider) Resources(ctx context.Context) []func() resource.Resource {
	inner := p.Provider.Resources(ctx)
	out := make([]func() resource.Resource, 0, len(inner))
	for _, newResource := range inner {
		out = append(out, func() resource.Resource { return &configCheck{inner: newResource()} })
	}
	return out
}

// configCheck forwards every call to the wrapped resource, and checks the configuration
// before Create and Update. It implements each optional interface that the resources of
// terraform-provider-wso2is use. A method whose interface the wrapped resource does not
// implement behaves as the framework does when the interface is missing.
type configCheck struct {
	inner resource.Resource
}

var (
	_ resource.ResourceWithConfigure        = (*configCheck)(nil)
	_ resource.ResourceWithModifyPlan       = (*configCheck)(nil)
	_ resource.ResourceWithImportState      = (*configCheck)(nil)
	_ resource.ResourceWithValidateConfig   = (*configCheck)(nil)
	_ resource.ResourceWithConfigValidators = (*configCheck)(nil)
)

func (g *configCheck) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	g.inner.Metadata(ctx, req, resp)
}

func (g *configCheck) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	g.inner.Schema(ctx, req, resp)
}

func (g *configCheck) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if g.validate(ctx, req.Config, &resp.Diagnostics); resp.Diagnostics.HasError() {
		return
	}
	g.inner.Create(ctx, req, resp)
}

func (g *configCheck) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	g.inner.Read(ctx, req, resp)
}

func (g *configCheck) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if g.validate(ctx, req.Config, &resp.Diagnostics); resp.Diagnostics.HasError() {
		return
	}
	g.inner.Update(ctx, req, resp)
}

// validate runs the configuration checks of the wrapped resource, as the framework does
// for ValidateResourceConfig. The schema's attribute validators are not part of it; the
// server checks those values.
func (g *configCheck) validate(ctx context.Context, cfg tfsdk.Config, diags *diag.Diagnostics) {
	req := resource.ValidateConfigRequest{Config: cfg}
	if r, ok := g.inner.(resource.ResourceWithValidateConfig); ok {
		var resp resource.ValidateConfigResponse
		r.ValidateConfig(ctx, req, &resp)
		diags.Append(resp.Diagnostics...)
	}
	if r, ok := g.inner.(resource.ResourceWithConfigValidators); ok {
		for _, v := range r.ConfigValidators(ctx) {
			var resp resource.ValidateConfigResponse
			v.ValidateResource(ctx, req, &resp)
			diags.Append(resp.Diagnostics...)
		}
	}
}

func (g *configCheck) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	g.inner.Delete(ctx, req, resp)
}

func (g *configCheck) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if r, ok := g.inner.(resource.ResourceWithConfigure); ok {
		r.Configure(ctx, req, resp)
	}
}

func (g *configCheck) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if r, ok := g.inner.(resource.ResourceWithModifyPlan); ok {
		r.ModifyPlan(ctx, req, resp)
	}
}

func (g *configCheck) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if r, ok := g.inner.(resource.ResourceWithImportState); ok {
		r.ImportState(ctx, req, resp)
		return
	}
	resp.Diagnostics.AddError("Resource Import Not Implemented", "This resource does not support import.")
}

func (g *configCheck) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	if r, ok := g.inner.(resource.ResourceWithValidateConfig); ok {
		r.ValidateConfig(ctx, req, resp)
	}
}

func (g *configCheck) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	if r, ok := g.inner.(resource.ResourceWithConfigValidators); ok {
		return r.ConfigValidators(ctx)
	}
	return nil
}
