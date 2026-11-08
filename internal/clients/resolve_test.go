package clients

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	fwprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	wso2is "github.com/hayone1/terraform-provider-wso2is/provider"

	namespacedidentity "github.com/hayone1/crossplane-provider-wso2is/apis/namespaced/identity/v1alpha1"
	namespacedv1beta1 "github.com/hayone1/crossplane-provider-wso2is/apis/namespaced/v1beta1"
)

func newScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	if err := namespacedv1beta1.SchemeBuilder.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	if err := namespacedidentity.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	return s
}

func namespacedUser(kind, name string) *namespacedidentity.User {
	u := &namespacedidentity.User{ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "alice", UID: "uid-1"}}
	u.SetProviderConfigReference(&xpv2.ProviderConfigReference{Kind: kind, Name: name})
	return u
}

func TestResolveModern(t *testing.T) {
	secretRef := func(ns string) *xpv2.SecretKeySelector {
		return &xpv2.SecretKeySelector{SecretReference: xpv2.SecretReference{Namespace: ns, Name: "creds"}, Key: "credentials"}
	}
	cases := map[string]struct {
		objects       []client.Object
		mg            resource.ModernManaged
		wantNamespace string
		wantErr       string
	}{
		"NamespacedProviderConfigReadsOnlyItsOwnNamespace": {
			objects: []client.Object{&namespacedv1beta1.ProviderConfig{
				ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "default"},
				Spec:       namespacedv1beta1.ProviderConfigSpec{Credentials: namespacedv1beta1.ProviderCredentials{Source: xpv2.CredentialsSourceSecret, CommonCredentialSelectors: xpv2.CommonCredentialSelectors{SecretRef: secretRef("crossplane-system")}}},
			}},
			mg:            namespacedUser("ProviderConfig", "default"),
			wantNamespace: "team-a",
		},
		"NamespacedProviderConfigRejectsTheEnvironmentSource": {
			objects: []client.Object{&namespacedv1beta1.ProviderConfig{
				ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "default"},
				Spec:       namespacedv1beta1.ProviderConfigSpec{Credentials: namespacedv1beta1.ProviderCredentials{Source: xpv2.CredentialsSourceEnvironment, CommonCredentialSelectors: xpv2.CommonCredentialSelectors{Env: &xpv2.EnvSelector{Name: "HOME"}}}},
			}},
			mg:      namespacedUser("ProviderConfig", "default"),
			wantErr: "supports only the Secret credentials source",
		},
		"NamespacedProviderConfigRejectsTheFilesystemSource": {
			objects: []client.Object{&namespacedv1beta1.ProviderConfig{
				ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "default"},
				Spec:       namespacedv1beta1.ProviderConfigSpec{Credentials: namespacedv1beta1.ProviderCredentials{Source: xpv2.CredentialsSourceFilesystem, CommonCredentialSelectors: xpv2.CommonCredentialSelectors{Fs: &xpv2.FsSelector{Path: "/var/run/secrets/kubernetes.io/serviceaccount/token"}}}},
			}},
			mg:      namespacedUser("ProviderConfig", "default"),
			wantErr: "supports only the Secret credentials source",
		},
		"NamespacedProviderConfigOfAnotherNamespaceIsNotFound": {
			objects: []client.Object{&namespacedv1beta1.ProviderConfig{
				ObjectMeta: metav1.ObjectMeta{Namespace: "team-b", Name: "default"},
				Spec:       namespacedv1beta1.ProviderConfigSpec{Credentials: namespacedv1beta1.ProviderCredentials{Source: xpv2.CredentialsSourceSecret, CommonCredentialSelectors: xpv2.CommonCredentialSelectors{SecretRef: secretRef("team-b")}}},
			}},
			mg:      namespacedUser("ProviderConfig", "default"),
			wantErr: errGetProviderConfig,
		},
		"ClusterProviderConfigKeepsItsSecretNamespace": {
			objects: []client.Object{&namespacedv1beta1.ClusterProviderConfig{
				ObjectMeta: metav1.ObjectMeta{Name: "default"},
				Spec:       namespacedv1beta1.ProviderConfigSpec{Credentials: namespacedv1beta1.ProviderCredentials{Source: xpv2.CredentialsSourceSecret, CommonCredentialSelectors: xpv2.CommonCredentialSelectors{SecretRef: secretRef("crossplane-system")}}},
			}},
			mg:            namespacedUser("ClusterProviderConfig", "default"),
			wantNamespace: "crossplane-system",
		},
		"UnknownKind": {
			mg:      namespacedUser("SomethingElse", "default"),
			wantErr: "unknown provider config kind",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// The API server ignores the namespace of a cluster-scoped kind. The fake client
			// does the same when its REST mapper knows the scope.
			mapper := meta.NewDefaultRESTMapper(nil)
			gv := namespacedv1beta1.SchemeGroupVersion
			mapper.Add(gv.WithKind("ProviderConfig"), meta.RESTScopeNamespace)
			mapper.Add(gv.WithKind("ProviderConfigUsage"), meta.RESTScopeNamespace)
			mapper.Add(gv.WithKind("ClusterProviderConfig"), meta.RESTScopeRoot)
			kube := fake.NewClientBuilder().WithScheme(newScheme(t)).WithRESTMapper(mapper).WithObjects(tc.objects...).Build()
			pc, err := resolveModern(context.Background(), kube, tc.mg)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("resolveModern() error = %v, want one that contains %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got := pc.spec.Credentials.SecretRef.Namespace; got != tc.wantNamespace {
				t.Errorf("secret namespace = %q, want %q", got, tc.wantNamespace)
			}
		})
	}
}

// The configuration check wrapper forwards only the optional interfaces that it
// implements. If a new release of the Terraform provider adds another optional interface
// to a resource or to the provider, the wrapper would hide it, and Upjet (which checks the
// unwrapped resource) and the protocol server (which sees the wrapper) would disagree. This
// test fails first.
func TestConfigCheck_hidesNoOptionalInterface(t *testing.T) {
	guarded := reflect.TypeOf(&configCheck{})
	optional := []reflect.Type{
		reflect.TypeFor[fwresource.ResourceWithConfigure](),
		reflect.TypeFor[fwresource.ResourceWithModifyPlan](),
		reflect.TypeFor[fwresource.ResourceWithImportState](),
		reflect.TypeFor[fwresource.ResourceWithValidateConfig](),
		reflect.TypeFor[fwresource.ResourceWithConfigValidators](),
		reflect.TypeFor[fwresource.ResourceWithUpgradeState](),
		reflect.TypeFor[fwresource.ResourceWithMoveState](),
		reflect.TypeFor[fwresource.ResourceWithIdentity](),
		reflect.TypeFor[fwresource.ResourceWithUpgradeIdentity](),
	}
	p := wso2is.New("test")()
	for _, newResource := range p.Resources(context.Background()) {
		r := newResource()
		var meta fwresource.MetadataResponse
		r.Metadata(context.Background(), fwresource.MetadataRequest{ProviderTypeName: "wso2is"}, &meta)
		for _, iface := range optional {
			if reflect.TypeOf(r).Implements(iface) && !guarded.Implements(iface) {
				t.Errorf("%s implements %s, which the configuration check wrapper does not forward", meta.TypeName, iface)
			}
		}
	}
	for _, iface := range []reflect.Type{
		reflect.TypeFor[fwprovider.ProviderWithConfigValidators](),
		reflect.TypeFor[fwprovider.ProviderWithValidateConfig](),
		reflect.TypeFor[fwprovider.ProviderWithFunctions](),
		reflect.TypeFor[fwprovider.ProviderWithEphemeralResources](),
		reflect.TypeFor[fwprovider.ProviderWithMetaSchema](),
		reflect.TypeFor[fwprovider.ProviderWithListResources](),
		reflect.TypeFor[fwprovider.ProviderWithActions](),
		reflect.TypeFor[fwprovider.ProviderWithStateStores](),
	} {
		if reflect.TypeOf(p).Implements(iface) {
			t.Errorf("the Terraform provider implements %s, which the configuration check wrapper's provider does not forward", iface)
		}
	}
}
