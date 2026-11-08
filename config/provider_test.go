package config

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	ujconfig "github.com/crossplane/upjet/v2/pkg/config"
	"github.com/crossplane/upjet/v2/pkg/terraform"
	"github.com/google/go-cmp/cmp"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"sigs.k8s.io/yaml"
)

func schemaResourceNames(t *testing.T) []string {
	t.Helper()
	var doc struct {
		ProviderSchemas map[string]struct {
			ResourceSchemas map[string]json.RawMessage `json:"resource_schemas"`
		} `json:"provider_schemas"`
	}
	if err := json.Unmarshal([]byte(providerSchema), &doc); err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, ps := range doc.ProviderSchemas {
		for name := range ps.ResourceSchemas {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// Every resource of the Terraform provider must be generated, in both API variants, with
// a kind, a group and a framework resource.
func TestProviders_configureEveryResource(t *testing.T) {
	names := schemaResourceNames(t)
	if len(names) != 52 {
		t.Fatalf("schema.json has %d resources, want 52", len(names))
	}
	for scope, pc := range map[string]*ujconfig.Provider{"cluster": GetProvider(), "namespaced": GetProviderNamespaced()} {
		seenKinds := map[string]string{}
		for _, name := range names {
			r, ok := pc.Resources[name]
			if !ok {
				t.Errorf("%s: %s is not configured", scope, name)
				continue
			}
			k, ok := kinds[name]
			if !ok {
				t.Errorf("%s has no group and kind", name)
				continue
			}
			if r.Kind != k.kind || r.ShortGroup != k.group {
				t.Errorf("%s: %s has kind %s.%s, want %s.%s", scope, name, r.ShortGroup, r.Kind, k.group, k.kind)
			}
			if other, dup := seenKinds[r.Kind]; dup {
				t.Errorf("kind %s is used by %s and %s", r.Kind, other, name)
			}
			seenKinds[r.Kind] = name
			if r.TerraformPluginFrameworkResource == nil {
				t.Errorf("%s: %s has no framework resource", scope, name)
			}
			// User stores, tenants and shares take up to two minutes. Upjet runs asynchronous
			// operations outside the reconcile, with a one-hour deadline, so they do not
			// hit the three-minute reconcile timeout.
			if !r.UseAsync {
				t.Errorf("%s: %s is not asynchronous", scope, name)
			}
		}
		if len(pc.Resources) != len(names) {
			t.Errorf("%s: %d resources configured, want %d", scope, len(pc.Resources), len(names))
		}
	}
	if len(ExternalNameConfigs) != len(names) || len(kinds) != len(names) {
		t.Errorf("ExternalNameConfigs has %d entries and kinds has %d, want %d each", len(ExternalNameConfigs), len(kinds), len(names))
	}
}

// The external names computed before the first create must be the IDs that the
// Terraform provider gives the objects.
func TestDerivedExternalNames(t *testing.T) {
	// Upjet passes the map of terraform.Setup, whose configuration has a named map type.
	providerConfig := func(cfg map[string]any) map[string]any { return terraform.Setup{Configuration: cfg}.Map() }
	cases := []struct {
		resource string
		params   map[string]any
		setup    map[string]any
		want     string
		wantErr  bool
	}{
		{"wso2is_governance_connector", map[string]any{"name": "passwordHistory"}, nil, "cGFzc3dvcmRIaXN0b3J5", false},
		{"wso2is_flow", map[string]any{"flow_type": "REGISTRATION"}, nil, "REGISTRATION", false},
		{"wso2is_notification_template", map[string]any{"channel": "email", "type": "AccountConfirmation"}, nil, "email/AccountConfirmation/en_US", false},
		{"wso2is_notification_template", map[string]any{"channel": "sms", "type": "SMSOTP", "locale": "fr_FR", "application_id": "app-1"}, nil, "sms/SMSOTP/fr_FR/app-1", false},
		{"wso2is_branding_preference", map[string]any{}, nil, "ORG/carbon.super/en-US", false},
		{"wso2is_branding_preference", map[string]any{}, providerConfig(map[string]any{"tenant_domain": "acme.com"}), "ORG/acme.com/en-US", false},
		{"wso2is_branding_preference", map[string]any{"locale": "fr-FR"}, providerConfig(map[string]any{"tenant_domain": "acme.com", "organization_id": "org-1"}), "ORG/org-1/fr-FR", false},
		{"wso2is_branding_preference", map[string]any{"type": "APP", "application_id": "app-1"}, nil, "APP/app-1/en-US", false},
		{"wso2is_branding_preference", map[string]any{"type": "APP"}, nil, "", true},
		{"wso2is_branding_text", map[string]any{"screen": "login"}, nil, "ORG/carbon.super/en-US/login", false},
		{"wso2is_group_members", map[string]any{"group_id": "g"}, nil, "g", false},
		{"wso2is_flow", map[string]any{}, nil, "", true},
		{"wso2is_validation_rules", map[string]any{"field": "password"}, nil, "password", false},
		{"wso2is_remote_logging", map[string]any{"log_type": "AUDIT"}, nil, "AUDIT", false},
		{"wso2is_role_assignment", map[string]any{"role_id": "r"}, nil, "r", false},
		{"wso2is_user_share", map[string]any{"user_id": "u"}, nil, "u", false},
		{"wso2is_application_share", map[string]any{"application_id": "a"}, nil, "a", false},
		{"wso2is_organization_discovery", map[string]any{"organization_id": "o"}, nil, "o", false},
	}
	for _, tc := range cases {
		t.Run(tc.resource+"/"+tc.want, func(t *testing.T) {
			e := ExternalNameConfigs[tc.resource]
			got, err := e.GetIDFn(context.Background(), "", tc.params, tc.setup)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("GetIDFn() = %q, want an error", got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("GetIDFn() = %q, want %q", got, tc.want)
			}
			// Once Crossplane knows the external name, it keeps using it.
			if got, _ := e.GetIDFn(context.Background(), "recorded", tc.params, tc.setup); got != "recorded" {
				t.Errorf("GetIDFn() with an external name = %q, want %q", got, "recorded")
			}
		})
	}
}

// The defaults that the derived IDs use must match the Terraform provider's schema.
func TestDerivedExternalNames_defaultsMatchTerraform(t *testing.T) {
	pc := GetProvider()
	cases := []struct{ resource, attribute, want string }{
		{"wso2is_branding_preference", "type", defaultBrandingType},
		{"wso2is_branding_preference", "locale", defaultBrandingLocale},
		{"wso2is_branding_text", "locale", defaultBrandingLocale},
		{"wso2is_notification_template", "locale", defaultNotificationLocale},
	}
	for _, tc := range cases {
		res := pc.Resources[tc.resource].TerraformPluginFrameworkResource
		var resp fwresource.SchemaResponse
		res.Schema(context.Background(), fwresource.SchemaRequest{}, &resp)
		a, ok := resp.Schema.Attributes[tc.attribute].(schema.StringAttribute)
		if !ok || a.Default == nil {
			t.Errorf("%s.%s has no string default", tc.resource, tc.attribute)
			continue
		}
		var dr defaults.StringResponse
		a.Default.DefaultString(context.Background(), defaults.StringRequest{}, &dr)
		if got := dr.PlanValue.ValueString(); got != tc.want {
			t.Errorf("%s.%s default is %q, the provider assumes %q", tc.resource, tc.attribute, got, tc.want)
		}
	}
}

// Kinds whose objects Crossplane creates never compute an ID, so that a managed resource
// never takes over an existing object of these kinds, such as a built-in claim dialect.
func TestCreatedExternalNames(t *testing.T) {
	params := map[string]any{"claim_uri": "http://wso2.org/claims/x", "dialect_uri": "http://wso2.org/oidc/claim", "name": "openid", "channel": "email", "type": "ADAPTIVE_AUTH_CALL_CHOREO"}
	for _, name := range createdNames {
		e := ExternalNameConfigs[name]
		if got, err := e.GetIDFn(context.Background(), "", params, nil); err != nil || got != "" {
			t.Errorf("%s: GetIDFn() without an external name = %q, %v; want an empty ID", name, got, err)
		}
		if got, _ := e.GetIDFn(context.Background(), "recorded", params, nil); got != "recorded" {
			t.Errorf("%s: GetIDFn() with an external name = %q, want %q", name, got, "recorded")
		}
	}
}

func TestSingletonExternalNames(t *testing.T) {
	for name, id := range singletonIDs {
		e := ExternalNameConfigs[name]
		if got, _ := e.GetIDFn(context.Background(), "something-else", nil, nil); got != id {
			t.Errorf("%s: GetIDFn() = %q, want %q", name, got, id)
		}
		if got, _ := e.GetExternalNameFn(map[string]any{"id": "x"}); got != id {
			t.Errorf("%s: GetExternalNameFn() = %q, want %q", name, got, id)
		}
	}
}

func TestStateHasNoID(t *testing.T) {
	objType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{"id": tftypes.String, "name": tftypes.String}}
	obj := func(id tftypes.Value) tftypes.Value {
		return tftypes.NewValue(objType, map[string]tftypes.Value{"id": id, "name": tftypes.NewValue(tftypes.String, "x")})
	}
	cases := map[string]struct {
		v    tftypes.Value
		want bool
	}{
		"NullState":  {tftypes.NewValue(objType, nil), true},
		"NullID":     {obj(tftypes.NewValue(tftypes.String, nil)), true},
		"UnknownID":  {obj(tftypes.NewValue(tftypes.String, tftypes.UnknownValue)), true},
		"EmptyID":    {obj(tftypes.NewValue(tftypes.String, "")), true},
		"ID":         {obj(tftypes.NewValue(tftypes.String, "5f6b")), false},
		"NoIDInType": {tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{"name": tftypes.String}}, map[string]tftypes.Value{"name": tftypes.NewValue(tftypes.String, "x")}), false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := stateHasNoID(context.Background(), tc.v, schema.Schema{})
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("stateHasNoID() = %v, want %v", got, tc.want)
			}
		})
	}
}

// Late initialization is off for every argument of every resource.
func TestLateInitializationIsOff(t *testing.T) {
	for name, r := range GetProvider().Resources {
		ignored := map[string]bool{}
		for _, f := range r.LateInitializer.IgnoredFields {
			ignored[f] = true
		}
		for arg, s := range r.TerraformResource.Schema {
			if (s.Optional || s.Required) && !ignored[arg] {
				t.Errorf("%s: late initialization is on for %s", name, arg)
			}
		}
	}
}

// The CRD descriptions must not talk about Terraform.
func TestDescriptionsUseCrossplaneTerms(t *testing.T) {
	forbidden := []string{"Terraform", "terraform", "jsonencode", "To import", "data source", "provider block", "provider alias", "depends_on", "file(", "wso2is_"}
	for name, r := range GetProvider().Resources {
		texts := map[string]string{"description": r.MetaResource.Description}
		for k, v := range r.MetaResource.ArgumentDocs {
			if _, ok := r.TerraformResource.Schema[k]; ok {
				texts[k] = v
			}
		}
		for field, text := range texts {
			// restore_on_destroy is an argument name; other uses of "destroy" are Terraform terms.
			if strings.Contains(strings.ReplaceAll(text, "restore_on_destroy", ""), "destroy") {
				t.Errorf("%s %s says destroy: %s", name, field, text)
			}
			for _, f := range forbidden {
				if strings.Contains(text, f) {
					t.Errorf("%s %s contains %q: %s", name, field, f, text)
				}
			}
		}
		if !strings.HasSuffix(r.MetaResource.Description, importSentence(name)) {
			t.Errorf("%s description does not end with %q: %s", name, importSentence(name), r.MetaResource.Description)
		}
	}
}

func TestInjectIdentityProviderSecrets(t *testing.T) {
	cases := map[string]struct {
		tf      map[string]any
		want    map[string]any
		wantErr string
	}{
		"MovesEachSecretToItsAuthenticator": {
			tf: map[string]any{
				"federated_authenticators": map[string]any{
					"GoogleOIDCAuthenticator": map[string]any{"properties": map[string]any{"ClientId": "id"}},
					"FacebookAuthenticator":   map[string]any{"secret_properties": map[string]any{"Existing": "kept"}},
					"MyAuthenticator":         map[string]any{"endpoint": map[string]any{"uri": "https://auth.example.com"}},
				},
				"outbound_provisioning_connectors": map[string]any{"scim2": map[string]any{}},
				"federated_authenticator_secret_properties": map[string]any{
					"GoogleOIDCAuthenticator.ClientSecret": "google",
					"FacebookAuthenticator.ClientSecret":   "facebook",
				},
				"federated_authenticator_endpoint_authentication_properties": map[string]any{"MyAuthenticator.password": "pw"},
				"outbound_provisioning_connector_secret_properties":          map[string]any{"scim2.scim2-password": "scim"},
			},
			want: map[string]any{
				"federated_authenticators": map[string]any{
					"GoogleOIDCAuthenticator": map[string]any{"properties": map[string]any{"ClientId": "id"}, "secret_properties": map[string]any{"ClientSecret": "google"}},
					"FacebookAuthenticator":   map[string]any{"secret_properties": map[string]any{"Existing": "kept", "ClientSecret": "facebook"}},
					"MyAuthenticator":         map[string]any{"endpoint": map[string]any{"uri": "https://auth.example.com", "authentication_properties": map[string]any{"password": "pw"}}},
				},
				"outbound_provisioning_connectors": map[string]any{"scim2": map[string]any{"secret_properties": map[string]any{"scim2-password": "scim"}}},
			},
		},
		"PropertyNamesMayContainDots": {
			tf: map[string]any{
				"federated_authenticators":                  map[string]any{"A": map[string]any{}},
				"federated_authenticator_secret_properties": map[string]any{"A.client.secret": "x"},
			},
			want: map[string]any{"federated_authenticators": map[string]any{"A": map[string]any{"secret_properties": map[string]any{"client.secret": "x"}}}},
		},
		"NothingToInject": {
			tf:   map[string]any{"name": "x"},
			want: map[string]any{"name": "x"},
		},
		"KeyWithoutProperty": {
			tf: map[string]any{
				"federated_authenticators":                  map[string]any{"A": map[string]any{}},
				"federated_authenticator_secret_properties": map[string]any{"ClientSecret": "x"},
			},
			wantErr: "must have the form <name>.<property>",
		},
		"UnknownAuthenticator": {
			tf: map[string]any{
				"federated_authenticators":                  map[string]any{"A": map[string]any{}},
				"federated_authenticator_secret_properties": map[string]any{"B.ClientSecret": "x"},
			},
			wantErr: `names "B", which federated_authenticators does not list`,
		},
		"AuthenticatorWithoutEndpoint": {
			tf: map[string]any{
				"federated_authenticators":                                   map[string]any{"A": map[string]any{}},
				"federated_authenticator_endpoint_authentication_properties": map[string]any{"A.password": "x"},
			},
			wantErr: "needs federated_authenticators.A.endpoint to be set",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := injectIdentityProviderSecrets(nil, tc.tf)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %v, want one that contains %q", err, tc.wantErr)
				}
				if strings.Contains(err.Error(), `"x"`) {
					t.Fatalf("error %q contains the secret value", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tc.want, tc.tf); diff != "" {
				t.Errorf("-want +got:\n%s", diff)
			}
		})
	}
}

func TestApplicationConnectionDetails(t *testing.T) {
	cases := map[string]struct {
		attr map[string]any
		want map[string][]byte
	}{
		"OIDC": {map[string]any{"oidc": map[string]any{"client_id": "cid", "client_secret": "cs"}}, map[string][]byte{
			"client_id": []byte("cid"), "client_secret": []byte("cs"),
			"credentials": []byte(`{"client_id":"cid","client_secret":"cs"}`),
		}},
		"WrappedOIDC": {map[string]any{"oidc": []any{map[string]any{"client_id": "cid"}}}, map[string][]byte{"client_id": []byte("cid")}},
		"NoOIDC":      {map[string]any{"name": "x"}, map[string][]byte{}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := applicationConnectionDetails(tc.attr)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("-want +got:\n%s", diff)
			}
		})
	}
}

// docs/resources.md must list every kind with its group and Terraform resource.
func TestResourcesDocListsEveryKind(t *testing.T) {
	doc, err := os.ReadFile("../docs/resources.md")
	if err != nil {
		t.Fatal(err)
	}
	for name, k := range kinds {
		row := fmt.Sprintf("| `%s` | `%s` | `%s` |", k.kind, k.group, name)
		if !strings.Contains(string(doc), row) {
			t.Errorf("docs/resources.md has no row that starts with %s", row)
		}
	}
}

// Every kind needs a cluster-scoped and a namespaced example, so that users can copy one
// and the end-to-end tests cover it.
func TestEveryKindHasExamples(t *testing.T) {
	for scope, suffix := range map[string]string{"cluster": "wso2is.crossplane.io", "namespaced": "wso2is.m.crossplane.io"} {
		found := map[string]bool{}
		err := filepath.WalkDir(filepath.Join("..", "examples", scope), func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".yaml") {
				return err
			}
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			for _, doc := range strings.Split(string(b), "\n---\n") {
				var obj struct {
					APIVersion string `json:"apiVersion"`
					Kind       string `json:"kind"`
				}
				if err := yaml.Unmarshal([]byte(doc), &obj); err != nil {
					return fmt.Errorf("%s: %w", path, err)
				}
				found[obj.APIVersion+"/"+obj.Kind] = true
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		for _, k := range kinds {
			if !found[k.group+"."+suffix+"/v1alpha1/"+k.kind] {
				t.Errorf("examples/%s has no %s (%s.%s)", scope, k.kind, k.group, suffix)
			}
		}
	}
}
