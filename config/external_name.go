package config

import (
	"context"
	"encoding/base64"
	"fmt"
	"sort"

	"github.com/crossplane/upjet/v2/pkg/config"
	"github.com/crossplane/upjet/v2/pkg/terraform"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/pkg/errors"
)

// The external name of every managed resource is the Terraform `id` of the object. The
// configurations below fall into three groups:
//
//   - Crossplane creates the object and then learns its ID. The external name starts
//     empty. To take over an existing object, set the crossplane.io/external-name
//     annotation to its ID.
//   - The ID follows from the arguments, and the kind manages something that always exists
//     or belongs to another object, for example the rules of a field or the members of a
//     group. Before the first create, the provider computes the ID from spec.forProvider,
//     so a managed resource takes over the existing object. Once the external name is set,
//     the provider uses it, so a changed argument is reported as a change that needs a
//     replacement instead of creating a second object.
//   - Singletons manage settings that always exist once per tenant. Their external name
//     is a fixed string.

// Terraform IDs of the singleton resources.
const (
	idEmailPublisher = "EmailPublisher"
	idSMSPublisher   = "SMSPublisher"
)

// Default values of arguments that are part of a computed ID. They must match the
// Terraform provider's schema defaults.
const (
	defaultTenantDomain       = "carbon.super"
	defaultBrandingType       = "ORG"
	defaultBrandingLocale     = "en-US"
	defaultNotificationLocale = "en_US"
)

// createdNames are the resources whose object Crossplane creates first and then records
// the ID of. The server assigns most of these IDs. For the last group below, the ID follows
// from the arguments, but taking over an existing object of these kinds could take over, and
// later delete, an object that belongs to something else: the OIDC claim dialect, the openid
// scope, the AGENT user store, a system notification type, or the Console's authorization for
// an API. A create of an object that already exists therefore
// fails, as it does with Terraform, and you take an object over only on purpose, with the
// crossplane.io/external-name annotation.
var createdNames = []string{
	"wso2is_action",
	"wso2is_api_resource",
	"wso2is_application",
	"wso2is_group",
	"wso2is_identity_provider",
	"wso2is_identity_verification_provider",
	"wso2is_organization",
	"wso2is_role",
	"wso2is_tenant",
	"wso2is_user",
	"wso2is_vc_template",
	"wso2is_webhook",
	"wso2is_workflow",
	"wso2is_workflow_association",
	// The ID follows from the arguments (see docs/resources.md for each format).
	"wso2is_application_authorized_api",
	"wso2is_claim",
	"wso2is_claim_dialect",
	"wso2is_custom_authenticator",
	"wso2is_external_claim",
	"wso2is_notification_template_type",
	"wso2is_oidc_scope",
	"wso2is_push_sender",
	"wso2is_script_library",
	"wso2is_secret",
	"wso2is_userstore",
}

// derivedNames maps the resources whose ID follows from their arguments, and whose objects
// a managed resource takes over, to their external name configuration. Taking over is what
// these kinds do: they manage settings that always exist (governance connectors, validation
// rules, flows, remote logging, branding, notification templates) or the complete member
// list or sharing of another object.
var derivedNames = map[string]config.ExternalName{
	"wso2is_governance_connector": derived(func(p params, _ setup) (string, error) {
		return p.base64("name")
	}, "name"),
	"wso2is_flow":             derived(paramID("flow_type"), "flow_type"),
	"wso2is_remote_logging":   derived(paramID("log_type"), "log_type"),
	"wso2is_validation_rules": derived(paramID("field"), "field"),
	"wso2is_notification_template": derived(func(p params, _ setup) (string, error) {
		id, err := p.join("channel", "type")
		if err != nil {
			return "", err
		}
		id += "/" + p.strOr("locale", defaultNotificationLocale)
		if app := p.strOr("application_id", ""); app != "" {
			id += "/" + app
		}
		return id, nil
	}, "channel", "type"),
	"wso2is_branding_preference": derived(func(p params, s setup) (string, error) {
		return brandingID(p, s, "")
	}),
	"wso2is_branding_text": derived(func(p params, s setup) (string, error) {
		screen, err := p.str("screen")
		if err != nil {
			return "", err
		}
		return brandingID(p, s, screen)
	}, "screen"),
	// These resources manage one part of a parent object, and their ID is the parent's ID.
	// The parent ID usually comes from a reference, so it is not an identifier field: an
	// identifier field is always required, and the reference could not fill it.
	"wso2is_application_share":      derived(paramID("application_id")),
	"wso2is_group_members":          derived(paramID("group_id")),
	"wso2is_role_assignment":        derived(paramID("role_id")),
	"wso2is_user_share":             derived(paramID("user_id")),
	"wso2is_organization_discovery": derived(paramID("organization_id")),
}

// singletonIDs maps the singleton resources to their fixed Terraform ID.
var singletonIDs = map[string]string{
	"wso2is_admin_advisory_banner":                "admin_advisory_banner",
	"wso2is_cors_configuration":                   "cors_configuration",
	"wso2is_dcr_configuration":                    "dcr_configuration",
	"wso2is_fraud_detection_configuration":        "fraud_detection_configuration",
	"wso2is_impersonation_configuration":          "impersonation_configuration",
	"wso2is_inbound_provisioning_configuration":   "inbound_provisioning_configuration",
	"wso2is_jwt_key_validator_configuration":      "jwt_key_validator_configuration",
	"wso2is_oauth2_configuration":                 "oauth2_configuration",
	"wso2is_oauth2_issuer_usage_scope":            "oauth2_issuer_usage_scope",
	"wso2is_organization_discovery_configuration": "organization_discovery_configuration",
	"wso2is_passive_sts_configuration":            "passive_sts_configuration",
	"wso2is_saml_idp_configuration":               "saml_idp_configuration",
	"wso2is_server_configuration":                 "server_configuration",
	"wso2is_email_sender":                         idEmailPublisher,
	"wso2is_sms_sender":                           idSMSPublisher,
}

// ExternalNameConfigs holds the external name configuration of every resource. Only the
// resources listed here are generated.
var ExternalNameConfigs = func() map[string]config.ExternalName {
	m := make(map[string]config.ExternalName, len(createdNames)+len(derivedNames)+len(singletonIDs))
	for _, name := range createdNames {
		m[name] = config.IdentifierFromProvider
	}
	for name, e := range derivedNames {
		m[name] = e
	}
	for name, id := range singletonIDs {
		m[name] = singleton(id)
	}
	return m
}()

// ExternalNameConfigurations applies the external name configuration of each resource. It
// also makes Crossplane treat a state without an ID as "the object does not exist yet".
func ExternalNameConfigurations() config.ResourceOption {
	return func(r *config.Resource) {
		if e, ok := ExternalNameConfigs[r.Name]; ok {
			r.ExternalName = e
		}
		r.TerraformPluginFrameworkIsStateEmptyFn = stateHasNoID
		r.LateInitializer = config.LateInitializer{IgnoredFields: topLevelArguments(r)}
		crossplaneDocs(r)
	}
}

// topLevelArguments returns the names of all top-level arguments of a resource. The
// provider turns late initialization off for all of them. The Terraform provider already
// treats an optional argument that is not set as "keep the server's value". Copying the
// server's values into spec.forProvider would change that meaning: a settings resource
// would take over every setting instead of only the listed ones, and a list with an empty
// default, such as the users of a role assignment, would keep members that the managed
// resource means to remove. The current values are in status.atProvider.
func topLevelArguments(r *config.Resource) []string {
	names := make([]string, 0, len(r.TerraformResource.Schema))
	for name, s := range r.TerraformResource.Schema {
		if s.Optional || s.Required {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// ExternalNameConfigured returns the include list of the Terraform plugin framework
// resources: every resource with an external name configuration.
func ExternalNameConfigured() []string {
	l := make([]string, 0, len(ExternalNameConfigs))
	for name := range ExternalNameConfigs {
		// The list holds regular expressions. "^" and "$" make each one match exactly.
		l = append(l, "^"+name+"$")
	}
	sort.Strings(l)
	return l
}

// stateHasNoID reports a state without an `id` as empty. Before the first create, the
// external name of a resource with a server-assigned ID is empty, and Upjet reads the
// object with an empty ID. terraform-provider-wso2is reports such an object as missing
// since v0.1.2 (v0.1.1 answered some of these reads with a state, for example for
// applications, users and roles). This check stays as a second line of defense: without
// it, a state without an ID would make Crossplane treat the object as existing and never
// create it.
func stateHasNoID(_ context.Context, v tftypes.Value, _ schema.Schema) (bool, error) {
	if v.IsNull() || !v.IsKnown() {
		return true, nil
	}
	var attrs map[string]tftypes.Value
	if err := v.As(&attrs); err != nil {
		return false, errors.Wrap(err, "cannot read the Terraform state as an object")
	}
	id, ok := attrs["id"]
	if !ok {
		// Every resource of this provider has an `id`. A state without one is not
		// something this check can judge.
		return false, nil
	}
	if id.IsNull() || !id.IsKnown() {
		return true, nil
	}
	var s string
	if err := id.As(&s); err != nil {
		return false, errors.Wrap(err, "cannot read the id attribute of the Terraform state")
	}
	return s == "", nil
}

// params are the Terraform arguments of a managed resource (spec.forProvider merged with
// spec.initProvider, after references are resolved).
type params map[string]any

// setup is the map that config.Setup.Map returns. Its "configuration" key holds the
// Terraform provider configuration of the ProviderConfig.
type setup map[string]any

func (p params) str(key string) (string, error) {
	v, ok := p[key].(string)
	if !ok || v == "" {
		return "", errors.Errorf("cannot compute the external name: the argument %q is not set", key)
	}
	return v, nil
}

func (p params) strOr(key, def string) string {
	if v, ok := p[key].(string); ok && v != "" {
		return v
	}
	return def
}

func (p params) base64(key string) (string, error) {
	v, err := p.str(key)
	return base64.RawURLEncoding.EncodeToString([]byte(v)), err
}

func (p params) join(keys ...string) (string, error) {
	id := ""
	for i, k := range keys {
		v, err := p.str(k)
		if err != nil {
			return "", err
		}
		if i > 0 {
			id += "/"
		}
		id += v
	}
	return id, nil
}

func (s setup) providerString(key string) string {
	var cfg map[string]any
	switch c := s["configuration"].(type) {
	case terraform.ProviderConfiguration:
		cfg = c
	case map[string]any:
		cfg = c
	}
	v, _ := cfg[key].(string)
	return v
}

func paramID(key string) func(params, setup) (string, error) {
	return func(p params, _ setup) (string, error) { return p.str(key) }
}

// brandingID returns `<type>/<name>/<locale>[/<screen>]`. The name is the application ID
// for application branding. For organization branding, the name is the organization ID
// of the ProviderConfig, or else its tenant domain.
func brandingID(p params, s setup, screen string) (string, error) {
	typ := p.strOr("type", defaultBrandingType)
	var name string
	if typ == "APP" {
		var err error
		if name, err = p.str("application_id"); err != nil {
			return "", err
		}
	} else {
		name = s.providerString("organization_id")
		if name == "" {
			name = s.providerString("tenant_domain")
		}
		if name == "" {
			name = defaultTenantDomain
		}
	}
	id := fmt.Sprintf("%s/%s/%s", typ, name, p.strOr("locale", defaultBrandingLocale))
	if screen != "" {
		id += "/" + screen
	}
	return id, nil
}

// derived returns the configuration of a resource whose ID follows from its arguments.
// identifiers are the arguments that make up the ID. They must not have references.
func derived(id func(params, setup) (string, error), identifiers ...string) config.ExternalName {
	e := config.IdentifierFromProvider
	e.GetIDFn = func(_ context.Context, externalName string, parameters map[string]any, terraformSetup map[string]any) (string, error) {
		if externalName != "" {
			return externalName, nil
		}
		return id(parameters, terraformSetup)
	}
	e.IdentifierFields = identifiers
	return e
}

// singleton returns the configuration of a resource that manages a setting with a fixed
// ID. The provider ignores any other external name.
func singleton(id string) config.ExternalName {
	e := config.IdentifierFromProvider
	e.GetIDFn = func(context.Context, string, map[string]any, map[string]any) (string, error) {
		return id, nil
	}
	e.GetExternalNameFn = func(map[string]any) (string, error) {
		return id, nil
	}
	return e
}
