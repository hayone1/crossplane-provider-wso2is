package clients

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	wso2is "github.com/hayone1/terraform-provider-wso2is/provider"
)

// Every resource of the wrapped Terraform provider must report an object without an ID as
// missing, without calling the server. Upjet reads an object with an empty ID before it
// creates it; terraform-provider-wso2is handles this since v0.1.2. The provider points at an address that cannot
// resolve, so a read that calls the server fails the test.
func TestConfigCheck_readWithoutIDIsNotFound(t *testing.T) {
	ctx := context.Background()
	server := providerserver.NewProtocol6(WithConfigChecks(wso2is.New("test"))())()
	schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := Configuration([]byte(`{"base_url": "https://wso2is.invalid", "username": "admin", "password": "admin", "max_retries": 0}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	cfgValue := dynamicValue(t, schemas.Provider, cfg)
	if resp, err := server.ConfigureProvider(ctx, &tfprotov6.ConfigureProviderRequest{Config: cfgValue}); err != nil || hasError(resp.Diagnostics) {
		t.Fatalf("configure: %v %v", err, resp.Diagnostics)
	}
	if len(schemas.ResourceSchemas) != 52 {
		t.Fatalf("the wrapped provider has %d resources, want 52", len(schemas.ResourceSchemas))
	}
	for name, s := range schemas.ResourceSchemas {
		t.Run(name, func(t *testing.T) {
			attrs := map[string]any{"id": ""}
			for _, a := range s.Block.Attributes {
				if a.Required && a.Type != nil && a.Type.Is(tftypes.String) {
					// "{}" is also valid for JSON arguments.
					attrs[a.Name] = "{}"
				}
			}
			state := dynamicValue(t, s, attrs)
			resp, err := server.ReadResource(ctx, &tfprotov6.ReadResourceRequest{TypeName: name, CurrentState: state})
			if err != nil {
				t.Fatal(err)
			}
			if hasError(resp.Diagnostics) {
				t.Fatalf("Read returned an error: %v", resp.Diagnostics[0])
			}
			got, err := resp.NewState.Unmarshal(s.ValueType())
			if err != nil {
				t.Fatal(err)
			}
			if !got.IsNull() {
				t.Fatal("Read kept the object; want it reported as missing")
			}
		})
	}
}

// fakeResource records the calls that reach it. Its configuration is invalid when
// invalid is set.
type fakeResource struct {
	reads, modifyPlans, configures, creates, updates int
	invalid                                          bool
}

func (f *fakeResource) ValidateConfig(_ context.Context, _ resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	if f.invalid {
		resp.Diagnostics.AddError("Invalid configuration", "the fake resource rejects this configuration")
	}
}

func (f *fakeResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "test_fake"
}

func (f *fakeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Attributes: map[string]schema.Attribute{"id": schema.StringAttribute{Computed: true}}}
}
func (f *fakeResource) Create(context.Context, resource.CreateRequest, *resource.CreateResponse) {
	f.creates++
}
func (f *fakeResource) Read(context.Context, resource.ReadRequest, *resource.ReadResponse) { f.reads++ }
func (f *fakeResource) Update(context.Context, resource.UpdateRequest, *resource.UpdateResponse) {
	f.updates++
}
func (f *fakeResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {}
func (f *fakeResource) ModifyPlan(context.Context, resource.ModifyPlanRequest, *resource.ModifyPlanResponse) {
	f.modifyPlans++
}

func (f *fakeResource) Configure(context.Context, resource.ConfigureRequest, *resource.ConfigureResponse) {
	f.configures++
}

func TestConfigCheck_forwardsOtherCalls(t *testing.T) {
	ctx := context.Background()
	inner := &fakeResource{}
	g := &configCheck{inner: inner}
	var sresp resource.SchemaResponse
	g.Schema(ctx, resource.SchemaRequest{}, &sresp)
	typ := sresp.Schema.Type().TerraformType(ctx)

	read := func(id any) *resource.ReadResponse {
		state := tfsdk.State{Schema: sresp.Schema, Raw: tftypes.NewValue(typ, map[string]tftypes.Value{"id": tftypes.NewValue(tftypes.String, id)})}
		resp := &resource.ReadResponse{State: state}
		g.Read(ctx, resource.ReadRequest{State: state}, resp)
		return resp
	}
	if read("5f6b"); inner.reads != 1 {
		t.Errorf("a read did not reach the resource")
	}
	g.ModifyPlan(ctx, resource.ModifyPlanRequest{}, &resource.ModifyPlanResponse{})
	g.Configure(ctx, resource.ConfigureRequest{}, &resource.ConfigureResponse{})
	if inner.modifyPlans != 1 || inner.configures != 1 {
		t.Errorf("ModifyPlan or Configure did not reach the resource")
	}
	// Writes run the resource's configuration checks first.
	var cresp resource.CreateResponse
	g.Create(ctx, resource.CreateRequest{}, &cresp)
	var uresp resource.UpdateResponse
	g.Update(ctx, resource.UpdateRequest{}, &uresp)
	if inner.creates != 1 || inner.updates != 1 || cresp.Diagnostics.HasError() || uresp.Diagnostics.HasError() {
		t.Errorf("a valid Create or Update did not reach the resource")
	}
	inner.invalid = true
	cresp, uresp = resource.CreateResponse{}, resource.UpdateResponse{}
	g.Create(ctx, resource.CreateRequest{}, &cresp)
	g.Update(ctx, resource.UpdateRequest{}, &uresp)
	if inner.creates != 1 || inner.updates != 1 || !cresp.Diagnostics.HasError() || !uresp.Diagnostics.HasError() {
		t.Errorf("an invalid Create or Update reached the resource, or reported no error")
	}

	var iresp resource.ImportStateResponse
	g.ImportState(ctx, resource.ImportStateRequest{}, &iresp)
	if !iresp.Diagnostics.HasError() {
		t.Errorf("ImportState of a resource without import support did not fail")
	}
}

func dynamicValue(t *testing.T, s *tfprotov6.Schema, values map[string]any) *tfprotov6.DynamicValue {
	t.Helper()
	objType := s.ValueType().(tftypes.Object)
	attrs := map[string]tftypes.Value{}
	for k, typ := range objType.AttributeTypes {
		attrs[k] = tftypes.NewValue(typ, nil)
	}
	for k, v := range values {
		typ := objType.AttributeTypes[k]
		switch val := v.(type) {
		case []string:
			elems := make([]tftypes.Value, 0, len(val))
			for _, e := range val {
				elems = append(elems, tftypes.NewValue(tftypes.String, e))
			}
			attrs[k] = tftypes.NewValue(typ, elems)
		case int64:
			attrs[k] = tftypes.NewValue(typ, val)
		default:
			attrs[k] = tftypes.NewValue(typ, val)
		}
	}
	dv, err := tfprotov6.NewDynamicValue(objType, tftypes.NewValue(objType, attrs))
	if err != nil {
		t.Fatal(err)
	}
	return &dv
}

func hasError(diags []*tfprotov6.Diagnostic) bool {
	for _, d := range diags {
		if d.Severity == tfprotov6.DiagnosticSeverityError {
			return true
		}
	}
	return false
}

// Through the plugin protocol, as Upjet calls it, an invalid PushSender must fail with the
// resource's own check before the provider calls the server, and a valid one must get as far
// as the server. The provider points at an address that cannot resolve.
func TestConfigCheck_checksConfigurationBeforeCreate(t *testing.T) {
	ctx := context.Background()
	server := providerserver.NewProtocol6(WithConfigChecks(wso2is.New("test"))())()
	schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := Configuration([]byte(`{"base_url": "https://wso2is.invalid", "username": "admin", "password": "admin", "max_retries": 0, "request_timeout": 5}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp, err := server.ConfigureProvider(ctx, &tfprotov6.ConfigureProviderRequest{Config: dynamicValue(t, schemas.Provider, cfg)}); err != nil || hasError(resp.Diagnostics) {
		t.Fatalf("configure: %v %v", err, resp.Diagnostics)
	}
	s := schemas.ResourceSchemas["wso2is_push_sender"]
	objType := s.ValueType().(tftypes.Object)
	fcmType := objType.AttributeTypes["fcm"].(tftypes.Object)

	apply := func(fcm tftypes.Value) []*tfprotov6.Diagnostic {
		attrs := map[string]tftypes.Value{}
		for k, typ := range objType.AttributeTypes {
			attrs[k] = tftypes.NewValue(typ, nil)
		}
		attrs["provider_name"] = tftypes.NewValue(tftypes.String, "FCM")
		attrs["fcm"] = fcm
		config, err := tfprotov6.NewDynamicValue(objType, tftypes.NewValue(objType, attrs))
		if err != nil {
			t.Fatal(err)
		}
		attrs["id"] = tftypes.NewValue(tftypes.String, tftypes.UnknownValue)
		attrs["default"] = tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue)
		planned, err := tfprotov6.NewDynamicValue(objType, tftypes.NewValue(objType, attrs))
		if err != nil {
			t.Fatal(err)
		}
		prior, err := tfprotov6.NewDynamicValue(objType, tftypes.NewValue(objType, nil))
		if err != nil {
			t.Fatal(err)
		}
		resp, err := server.ApplyResourceChange(ctx, &tfprotov6.ApplyResourceChangeRequest{
			TypeName: "wso2is_push_sender", PriorState: &prior, PlannedState: &planned, Config: &config,
		})
		if err != nil {
			t.Fatal(err)
		}
		return resp.Diagnostics
	}
	summaries := func(diags []*tfprotov6.Diagnostic) string {
		out := make([]string, 0, len(diags))
		for _, d := range diags {
			out = append(out, d.Summary+": "+d.Detail)
		}
		return strings.Join(out, "; ")
	}

	invalid := apply(tftypes.NewValue(fcmType, nil))
	if got := summaries(invalid); !strings.Contains(got, "Missing FCM settings") {
		t.Errorf("an FCM push sender without fcm: got %q, want the resource's check \"Missing FCM settings\"", got)
	}
	valid := apply(tftypes.NewValue(fcmType, map[string]tftypes.Value{
		"service_account_json": tftypes.NewValue(tftypes.String, `{"type": "service_account"}`),
	}))
	if got := summaries(valid); strings.Contains(got, "Missing FCM settings") || !hasError(valid) {
		t.Errorf("a valid push sender: got %q, want an error from the unreachable server only", got)
	}
}
