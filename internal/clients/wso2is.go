// Package clients turns a ProviderConfig into the configuration of the in-process WSO2
// Identity Server Terraform provider.
package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/crossplane/upjet/v2/pkg/terraform"
	fwprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	clusterv1beta1 "github.com/hayone1/crossplane-provider-wso2is/apis/cluster/v1beta1"
	namespacedv1beta1 "github.com/hayone1/crossplane-provider-wso2is/apis/namespaced/v1beta1"
)

// Keys of the credentials JSON object. They are the argument names of the Terraform
// provider's configuration.
const (
	keyBaseURL            = "base_url"
	keyTenantDomain       = "tenant_domain"
	keyOrganizationID     = "organization_id"
	keyUsername           = "username"
	keyPassword           = "password"
	keyClientID           = "client_id"
	keyClientSecret       = "client_secret"
	keyScopes             = "scopes"
	keyAccessToken        = "access_token"
	keyInsecureSkipVerify = "insecure_skip_verify"
	keyCACertificate      = "ca_certificate"
	keyRequestTimeout     = "request_timeout"
	keyMaxRetries         = "max_retries"
)

// Defaults of the Terraform provider. The provider passes them explicitly, so that the
// Terraform provider never falls back to WSO2IS_* environment variables.
const (
	defaultRequestTimeout = 60
	defaultMaxRetries     = 4
)

// trimmedKeys are the string settings whose surrounding spaces the provider removes.
var trimmedKeys = map[string]bool{keyBaseURL: true, keyTenantDomain: true, keyOrganizationID: true}

var stringKeys = []string{
	keyBaseURL, keyTenantDomain, keyOrganizationID, keyUsername, keyPassword,
	keyClientID, keyClientSecret, keyAccessToken, keyCACertificate,
}

// EnvironmentVariables are the variables that the Terraform provider reads when its
// configuration leaves an argument empty. The provider always passes every argument, so
// these variables have no effect.
var EnvironmentVariables = []string{
	"WSO2IS_BASE_URL", "WSO2IS_TENANT_DOMAIN", "WSO2IS_ORGANIZATION_ID", "WSO2IS_USERNAME",
	"WSO2IS_PASSWORD", "WSO2IS_CLIENT_ID", "WSO2IS_CLIENT_SECRET", "WSO2IS_SCOPES",
	"WSO2IS_ACCESS_TOKEN", "WSO2IS_INSECURE_SKIP_VERIFY", "WSO2IS_CA_CERTIFICATE",
	"WSO2IS_REQUEST_TIMEOUT", "WSO2IS_MAX_RETRIES",
}

const (
	errNoProviderConfig   = "no providerConfigRef provided"
	errGetProviderConfig  = "cannot get referenced ProviderConfig"
	errTrackUsage         = "cannot track ProviderConfig usage"
	errExtractCredentials = "cannot extract credentials"
)

// TerraformSetupBuilder returns a terraform.SetupFn that configures the in-process
// ("no-fork") WSO2 Identity Server Terraform provider with the settings of the
// ProviderConfig that a managed resource references. newProvider creates the provider.
// One provider instance serves every ProviderConfig, because the Terraform provider keeps
// each configuration's API client in its configure response, not in the instance.
func TerraformSetupBuilder(newProvider func() fwprovider.Provider) terraform.SetupFn {
	p := newProvider()
	return func(ctx context.Context, kube client.Client, mg resource.Managed) (terraform.Setup, error) {
		pc, err := resolveProviderConfig(ctx, kube, mg)
		if err != nil {
			return terraform.Setup{}, errors.Wrap(err, "cannot resolve provider config")
		}
		data, err := resource.CommonCredentialExtractor(ctx, pc.spec.Credentials.Source, kube, pc.spec.Credentials.CommonCredentialSelectors)
		if err != nil {
			return terraform.Setup{}, errors.Wrap(err, errExtractCredentials)
		}
		cfg, err := Configuration(data, pc.spec)
		if err != nil {
			return terraform.Setup{}, errors.Wrapf(err, "invalid credentials of %s", pc.name)
		}
		return terraform.Setup{
			Configuration:     cfg,
			FrameworkProvider: p,
		}, nil
	}
}

// Configuration builds the Terraform provider configuration from the credentials data and
// the ProviderConfig spec. Every argument of the Terraform provider is set, so that the
// Terraform provider never reads WSO2IS_* environment variables of the provider pod.
func Configuration(data []byte, spec *namespacedv1beta1.ProviderConfigSpec) (map[string]any, error) {
	if spec == nil {
		spec = &namespacedv1beta1.ProviderConfigSpec{}
	}
	creds, err := parseCredentials(data)
	if err != nil {
		return nil, err
	}
	cfg, err := stringSettings(creds)
	if err != nil {
		return nil, err
	}
	if err := applySpec(cfg, spec); err != nil {
		return nil, err
	}
	if err := checkStringSettings(cfg); err != nil {
		return nil, err
	}
	if err := otherSettings(creds, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// stringSettings returns every string argument of the Terraform provider, with an empty
// string for each one that the credentials do not set.
func stringSettings(creds map[string]any) (map[string]any, error) {
	cfg := map[string]any{}
	for _, k := range stringKeys {
		cfg[k] = ""
		v, ok := creds[k]
		if !ok {
			continue
		}
		s, ok := v.(string)
		if !ok {
			return nil, errors.Errorf("the credentials key %q must be a string", k)
		}
		// Secrets are used exactly as given, as the Terraform provider does. Only the
		// settings that name the server, the tenant and the organization are trimmed.
		if trimmedKeys[k] {
			s = strings.TrimSpace(s)
		}
		cfg[k] = s
	}
	return cfg, nil
}

// applySpec adds the server, tenant and organization from the ProviderConfig spec. A spec
// value that differs from the credentials is an error.
func applySpec(cfg map[string]any, spec *namespacedv1beta1.ProviderConfigSpec) error {
	for _, setting := range []struct{ key, value string }{
		{keyBaseURL, spec.BaseURL}, {keyTenantDomain, spec.TenantDomain}, {keyOrganizationID, spec.OrganizationID},
	} {
		value := strings.TrimSpace(setting.value)
		if value == "" {
			continue
		}
		if cfg[setting.key] != "" && !sameSetting(setting.key, cfg[setting.key].(string), value) {
			return errors.Errorf("the ProviderConfig spec and the credentials set different values for %s: set it in only one place", setting.key)
		}
		cfg[setting.key] = value
	}
	cfg[keyBaseURL] = strings.TrimRight(cfg[keyBaseURL].(string), "/")
	return nil
}

// checkStringSettings checks the server URL, the sign-in method and the CA certificate.
func checkStringSettings(cfg map[string]any) error {
	if cfg[keyBaseURL] == "" {
		return errors.New("the server URL is missing: set spec.baseURL or the credentials key base_url")
	}
	if err := checkSignIn(cfg); err != nil {
		return err
	}
	if ca := cfg[keyCACertificate].(string); ca != "" && !strings.Contains(ca, "-----BEGIN") {
		// The Terraform provider would read a value without PEM markers as a file path in
		// the provider pod. Crossplane accepts only PEM text.
		return errors.New("the credentials key ca_certificate must hold PEM text (-----BEGIN CERTIFICATE----- ...), not a file path")
	}
	return nil
}

// otherSettings adds the scopes, the TLS switch, the request timeout and the retries.
func otherSettings(creds, cfg map[string]any) error {
	var err error
	if cfg[keyScopes], err = scopes(creds[keyScopes]); err != nil {
		return err
	}
	if cfg[keyInsecureSkipVerify], err = boolValue(creds, keyInsecureSkipVerify, false); err != nil {
		return err
	}
	timeout, err := intValue(creds, keyRequestTimeout, defaultRequestTimeout)
	if err != nil {
		return err
	}
	if timeout <= 0 {
		return errors.New("the credentials key request_timeout must be a positive number of seconds")
	}
	cfg[keyRequestTimeout] = timeout
	retries, err := intValue(creds, keyMaxRetries, defaultMaxRetries)
	if err != nil {
		return err
	}
	if retries < 0 {
		return errors.New("the credentials key max_retries must be zero or a positive number")
	}
	cfg[keyMaxRetries] = retries
	return nil
}

// parseCredentials reads the credentials JSON object and rejects unknown keys, so that a
// typing mistake such as "clientId" does not go unnoticed.
func parseCredentials(data []byte) (map[string]any, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return nil, errors.New("the credentials are empty")
	}
	if data[0] != '{' {
		return nil, errors.New("the credentials must be a JSON object, for example {\"base_url\": \"https://id.example.com\", \"username\": \"admin\", \"password\": \"...\"}")
	}
	creds := map[string]any{}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := dec.Decode(&creds); err != nil {
		// Do not include the data in the message: it holds secrets.
		return nil, errors.New("the credentials are not a valid JSON object")
	}
	known := map[string]bool{keyScopes: true, keyInsecureSkipVerify: true, keyRequestTimeout: true, keyMaxRetries: true}
	for _, k := range stringKeys {
		known[k] = true
	}
	var unknown []string
	for k := range creds {
		if !known[k] {
			unknown = append(unknown, k)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		allowed := make([]string, 0, len(known))
		for k := range known {
			allowed = append(allowed, k)
		}
		sort.Strings(allowed)
		return nil, errors.Errorf("the credentials contain unknown keys %s: the allowed keys are %s", strings.Join(unknown, ", "), strings.Join(allowed, ", "))
	}
	return creds, nil
}

// checkSignIn makes sure that the credentials hold exactly one complete sign-in method.
// The Terraform provider checks this too. This check gives messages that name the
// credentials keys.
func checkSignIn(cfg map[string]any) error {
	set := func(k string) bool { return cfg[k] != "" }
	basic := set(keyUsername) || set(keyPassword)
	clientCreds := set(keyClientID) || set(keyClientSecret)
	n := 0
	for _, b := range []bool{basic, clientCreds, set(keyAccessToken)} {
		if b {
			n++
		}
	}
	switch {
	case n == 0:
		return errors.New("no sign-in method is set: put username and password, client_id and client_secret, or access_token into the credentials")
	case n > 1:
		return errors.New("more than one sign-in method is set: put only one of username and password, client_id and client_secret, or access_token into the credentials")
	}
	return checkSignInComplete(set, basic, clientCreds)
}

// checkSignInComplete checks that the chosen sign-in method has all of its keys.
func checkSignInComplete(set func(string) bool, basic, clientCreds bool) error {
	switch {
	case basic && (!set(keyUsername) || !set(keyPassword)):
		return errors.New("basic authentication needs both username and password")
	case clientCreds && (!set(keyClientID) || !set(keyClientSecret)):
		return errors.New("the client credentials grant needs both client_id and client_secret")
	case basic && set(keyOrganizationID):
		return errors.New("the organization APIs reject basic authentication: use client_id and client_secret of an application that is shared with the organization, or an access_token issued for the organization")
	}
	return nil
}

// sameSetting reports whether two values of a setting mean the same thing.
func sameSetting(key, a, b string) bool {
	if key == keyBaseURL {
		return strings.TrimRight(a, "/") == strings.TrimRight(b, "/")
	}
	return a == b
}

// scopes accepts a JSON list of strings or a string with space-separated scopes.
func scopes(v any) ([]string, error) {
	out := []string{}
	switch s := v.(type) {
	case nil:
	case string:
		out = append(out, strings.Fields(s)...)
	case []any:
		for _, e := range s {
			str, ok := e.(string)
			if !ok {
				return nil, errors.New("the credentials key scopes must be a list of strings")
			}
			if str = strings.TrimSpace(str); str != "" {
				out = append(out, str)
			}
		}
	default:
		return nil, errors.New("the credentials key scopes must be a list of strings or a string with space-separated scopes")
	}
	return out, nil
}

func boolValue(creds map[string]any, key string, def bool) (bool, error) {
	switch v := creds[key].(type) {
	case nil:
		return def, nil
	case bool:
		return v, nil
	case string:
		b, err := strconv.ParseBool(strings.TrimSpace(v))
		if err != nil {
			return false, errors.Errorf("the credentials key %s must be true or false", key)
		}
		return b, nil
	default:
		return false, errors.Errorf("the credentials key %s must be true or false", key)
	}
}

func intValue(creds map[string]any, key string, def int64) (int64, error) {
	var s string
	switch v := creds[key].(type) {
	case nil:
		return def, nil
	case json.Number:
		s = v.String()
	case string:
		s = strings.TrimSpace(v)
	default:
		return 0, errors.Errorf("the credentials key %s must be a whole number", key)
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, errors.Errorf("the credentials key %s must be a whole number", key)
	}
	return n, nil
}

// SetEnvironmentVariables returns the Terraform provider variables that are set in the
// provider's environment. They have no effect, and the provider logs a message about them.
func SetEnvironmentVariables() []string {
	var set []string
	for _, v := range EnvironmentVariables {
		if _, ok := os.LookupEnv(v); ok {
			set = append(set, v)
		}
	}
	return set
}

// providerConfig is a resolved ProviderConfig or ClusterProviderConfig.
type providerConfig struct {
	name string
	spec *namespacedv1beta1.ProviderConfigSpec
}

func resolveProviderConfig(ctx context.Context, crClient client.Client, mg resource.Managed) (*providerConfig, error) {
	switch managed := mg.(type) {
	case resource.LegacyManaged: //nolint:staticcheck // the cluster-scoped API is still supported
		return resolveLegacy(ctx, crClient, managed)
	case resource.ModernManaged:
		return resolveModern(ctx, crClient, managed)
	default:
		return nil, errors.New("resource is not a managed resource")
	}
}

func resolveLegacy(ctx context.Context, kube client.Client, mg resource.LegacyManaged) (*providerConfig, error) { //nolint:staticcheck // the cluster-scoped API is still supported
	configRef := mg.GetProviderConfigReference()
	if configRef == nil {
		return nil, errors.New(errNoProviderConfig)
	}
	pc := &clusterv1beta1.ProviderConfig{}
	if err := kube.Get(ctx, types.NamespacedName{Name: configRef.Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetProviderConfig)
	}
	t := resource.NewLegacyProviderConfigUsageTracker(kube, &clusterv1beta1.ProviderConfigUsage{})
	if err := t.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackUsage)
	}
	spec, err := toSharedPCSpec(pc)
	if err != nil {
		return nil, err
	}
	return &providerConfig{name: fmt.Sprintf("ProviderConfig %s", pc.Name), spec: spec}, nil
}

func resolveModern(ctx context.Context, crClient client.Client, mg resource.ModernManaged) (*providerConfig, error) {
	configRef := mg.GetProviderConfigReference()
	if configRef == nil {
		return nil, errors.New(errNoProviderConfig)
	}
	pcRuntimeObj, err := crClient.Scheme().New(namespacedv1beta1.SchemeGroupVersion.WithKind(configRef.Kind))
	if err != nil {
		return nil, errors.Wrapf(err, "unknown provider config kind %q", configRef.Kind)
	}
	pcObj, ok := pcRuntimeObj.(client.Object)
	if !ok {
		// This indicates a programming error: the types are not generated properly.
		return nil, errors.Errorf("%s is not a Kubernetes object", configRef.Kind)
	}
	key := types.NamespacedName{Name: configRef.Name, Namespace: mg.GetNamespace()}
	if _, cluster := pcObj.(*namespacedv1beta1.ClusterProviderConfig); cluster {
		key.Namespace = ""
	}
	if err := crClient.Get(ctx, key, pcObj); err != nil {
		return nil, errors.Wrap(err, errGetProviderConfig)
	}

	var pc providerConfig
	switch obj := pcObj.(type) {
	case *namespacedv1beta1.ProviderConfig:
		spec, err := namespacedSpec(obj, mg.GetNamespace())
		if err != nil {
			return nil, err
		}
		pc = providerConfig{name: fmt.Sprintf("ProviderConfig %s/%s", obj.Namespace, obj.Name), spec: spec}
	case *namespacedv1beta1.ClusterProviderConfig:
		pc = providerConfig{name: fmt.Sprintf("ClusterProviderConfig %s", obj.Name), spec: obj.Spec.DeepCopy()}
	default:
		return nil, errors.New("unknown provider config type")
	}
	t := resource.NewProviderConfigUsageTracker(crClient, &namespacedv1beta1.ProviderConfigUsage{})
	if err := t.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackUsage)
	}
	return &pc, nil
}

// namespacedSpec returns the spec of a namespaced ProviderConfig. A namespaced
// ProviderConfig may only read a Secret in its own namespace. The Environment and Filesystem
// sources would let users of one namespace make the provider read its own environment or
// files, so they need a ClusterProviderConfig.
func namespacedSpec(pc *namespacedv1beta1.ProviderConfig, namespace string) (*namespacedv1beta1.ProviderConfigSpec, error) {
	spec := pc.Spec.DeepCopy()
	if spec.Credentials.Source != xpv2.CredentialsSourceSecret {
		return nil, errors.Errorf("ProviderConfig %s/%s: a namespaced ProviderConfig supports only the Secret credentials source; use a ClusterProviderConfig for the %s source", pc.Namespace, pc.Name, spec.Credentials.Source)
	}
	if spec.Credentials.SecretRef != nil {
		spec.Credentials.SecretRef.Namespace = namespace
	}
	return spec, nil
}

func toSharedPCSpec(pc *clusterv1beta1.ProviderConfig) (*namespacedv1beta1.ProviderConfigSpec, error) {
	data, err := json.Marshal(pc.Spec)
	if err != nil {
		return nil, err
	}
	var spec namespacedv1beta1.ProviderConfigSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, err
	}
	return &spec, nil
}
