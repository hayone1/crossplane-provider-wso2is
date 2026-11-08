package config

import (
	"sort"
	"strings"

	ujconfig "github.com/crossplane/upjet/v2/pkg/config"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
)

// Upjet v2.5.1 cannot pass secret maps that are nested in a map of objects. It writes
// the values of federatedAuthenticators[*].secretPropertiesSecretRef to an authenticator
// literally named "*", and the secrets of the last authenticator overwrite the others.
// The identity provider has three such fields, so the provider replaces them with
// top-level secret references. The keys of their Secrets have the form
// "<authenticator or connector name>.<property>", for example
// "GoogleOIDCAuthenticator.ClientSecret". A configuration injector moves each value to
// its nested place before the Terraform provider sees the configuration.
// Reported upstream: https://github.com/crossplane/upjet/issues/810

// idpNestedSecret describes one nested secret map of wso2is_identity_provider and the
// top-level argument that replaces it.
type idpNestedSecret struct {
	// topLevel is the name of the replacement argument.
	topLevel string
	// parent is the map of objects, for example "federated_authenticators".
	parent string
	// path leads from an element of the parent to the secret map.
	path []string
	// description of the replacement argument.
	description string
}

var idpNestedSecrets = []idpNestedSecret{
	{
		topLevel: "federated_authenticator_secret_properties",
		parent:   "federated_authenticators",
		path:     []string{"secret_properties"},
		description: "Secret properties of the federated authenticators, such as client secrets. " +
			"Each key of the Secret has the form <authenticator>.<property>, for example GoogleOIDCAuthenticator.ClientSecret. " +
			"The authenticator must be listed in federatedAuthenticators.",
	},
	{
		topLevel: "federated_authenticator_endpoint_authentication_properties",
		parent:   "federated_authenticators",
		path:     []string{"endpoint", "authentication_properties"},
		description: "Credentials that the server sends to the endpoints of custom federated authenticators. " +
			"Each key of the Secret has the form <authenticator>.<property>, for example MyAuthenticator.password. " +
			"The authenticator must be listed in federatedAuthenticators and have an endpoint.",
	},
	{
		topLevel: "outbound_provisioning_connector_secret_properties",
		parent:   "outbound_provisioning_connectors",
		path:     []string{"secret_properties"},
		description: "Secret properties of the outbound provisioning connectors. " +
			"Each key of the Secret has the form <connector>.<property>, for example scim2.scim2-password. " +
			"The connector must be listed in outboundProvisioningConnectors.",
	},
}

func configureIdentityProvider(p *ujconfig.Provider) {
	p.AddResourceConfigurator("wso2is_identity_provider", func(r *ujconfig.Resource) {
		for _, s := range idpNestedSecrets {
			parent := r.TerraformResource.Schema[s.parent]
			elem := parent.Elem.(*schema.Resource)
			for _, step := range s.path[:len(s.path)-1] {
				elem = elem.Schema[step].Elem.(*schema.Resource)
			}
			delete(elem.Schema, s.path[len(s.path)-1])

			r.TerraformResource.Schema[s.topLevel] = &schema.Schema{
				Type:        schema.TypeMap,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Optional:    true,
				Sensitive:   true,
				Description: s.description,
			}
			if r.MetaResource != nil {
				r.MetaResource.ArgumentDocs[s.topLevel] = s.description
			}
			r.LateInitializer.IgnoredFields = append(r.LateInitializer.IgnoredFields, s.topLevel)
		}
		r.TerraformConfigurationInjector = injectIdentityProviderSecrets
	})
}

// injectIdentityProviderSecrets moves the values of the top-level secret arguments of an
// identity provider into the nested secret maps of the Terraform configuration.
func injectIdentityProviderSecrets(_ map[string]any, tfMap map[string]any) error {
	for _, s := range idpNestedSecrets {
		values, ok := tfMap[s.topLevel].(map[string]any)
		delete(tfMap, s.topLevel)
		if !ok || len(values) == 0 {
			continue
		}
		parent, _ := tfMap[s.parent].(map[string]any)
		keys := make([]string, 0, len(values))
		for k := range values {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, key := range keys {
			target, property, err := s.target(parent, key)
			if err != nil {
				return err
			}
			target[property] = values[key]
		}
	}
	return nil
}

// target returns the nested secret map that a key of the Secret belongs to, and the
// property name inside that map. The key has the form "<name>.<property>".
func (s idpNestedSecret) target(parent map[string]any, key string) (map[string]any, string, error) {
	name, property, found := strings.Cut(key, ".")
	if !found || name == "" || property == "" {
		return nil, "", errors.Errorf("the key %q of the Secret for %s must have the form <name>.<property>", key, s.topLevel)
	}
	elem, ok := parent[name].(map[string]any)
	if !ok {
		return nil, "", errors.Errorf("the key %q of the Secret for %s names %q, which %s does not list", key, s.topLevel, name, s.parent)
	}
	for _, step := range s.path[:len(s.path)-1] {
		next, ok := elem[step].(map[string]any)
		if !ok {
			return nil, "", errors.Errorf("the key %q of the Secret for %s needs %s.%s.%s to be set", key, s.topLevel, s.parent, name, step)
		}
		elem = next
	}
	last := s.path[len(s.path)-1]
	m, ok := elem[last].(map[string]any)
	if !ok {
		m = map[string]any{}
		elem[last] = m
	}
	return m, property, nil
}
