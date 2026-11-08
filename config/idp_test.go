package config

import (
	"context"
	"testing"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/crossplane/upjet/v2/pkg/resource"
	"github.com/google/go-cmp/cmp"

	"github.com/hayone1/crossplane-provider-wso2is/apis/cluster/idp/v1alpha1"
)

type fakeSecrets map[string]map[string][]byte

func (f fakeSecrets) GetSecretData(_ context.Context, ref *xpv2.SecretReference) (map[string][]byte, error) {
	return f[ref.Name], nil
}

func (f fakeSecrets) GetSecretValue(_ context.Context, sel xpv2.SecretKeySelector) ([]byte, error) {
	return f[sel.Name][sel.Key], nil
}

// The provider's top-level secret references of an identity provider must reach the
// right authenticators, through Upjet's secret handling and the configuration injector,
// as they do in a reconcile.
func TestIdentityProviderSecretsReachTheirAuthenticators(t *testing.T) {
	idp := &v1alpha1.IdentityProvider{}
	idp.Spec.ForProvider.FederatedAuthenticatorSecretPropertiesSecretRef = &xpv2.SecretReference{Name: "authenticators", Namespace: "ns"}
	idp.Spec.ForProvider.OutboundProvisioningConnectorSecretPropertiesSecretRef = &xpv2.SecretReference{Name: "connectors", Namespace: "ns"}
	secrets := fakeSecrets{
		"authenticators": {"GoogleOIDCAuthenticator.ClientSecret": []byte("google"), "FacebookAuthenticator.ClientSecret": []byte("facebook")},
		"connectors":     {"scim2.scim2-password": []byte("scim")},
	}
	tf := map[string]any{
		"federated_authenticators": map[string]any{
			"GoogleOIDCAuthenticator": map[string]any{"enabled": true},
			"FacebookAuthenticator":   map[string]any{"enabled": true},
		},
		"outbound_provisioning_connectors": map[string]any{"scim2": map[string]any{}},
	}
	if err := resource.GetSensitiveParameters(context.Background(), secrets, idp, tf, idp.GetConnectionDetailsMapping()); err != nil {
		t.Fatal(err)
	}
	if err := injectIdentityProviderSecrets(nil, tf); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"federated_authenticators": map[string]any{
			"GoogleOIDCAuthenticator": map[string]any{"enabled": true, "secret_properties": map[string]any{"ClientSecret": "google"}},
			"FacebookAuthenticator":   map[string]any{"enabled": true, "secret_properties": map[string]any{"ClientSecret": "facebook"}},
		},
		"outbound_provisioning_connectors": map[string]any{"scim2": map[string]any{"secret_properties": map[string]any{"scim2-password": "scim"}}},
	}
	if diff := cmp.Diff(want, tf); diff != "" {
		t.Errorf("-want +got:\n%s", diff)
	}
	// The broken nested mappings must be gone, so that Upjet never writes to "*".
	for tfPath := range idp.GetConnectionDetailsMapping() {
		if tfPath != "federated_authenticator_secret_properties" &&
			tfPath != "federated_authenticator_endpoint_authentication_properties" &&
			tfPath != "outbound_provisioning_connector_secret_properties" {
			t.Errorf("unexpected secret mapping %q", tfPath)
		}
	}
}
