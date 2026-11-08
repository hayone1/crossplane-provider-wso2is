package config

import (
	"encoding/json"

	ujconfig "github.com/crossplane/upjet/v2/pkg/config"
	"github.com/pkg/errors"
)

// apiGroup and kind of every managed resource. The kinds are unique across the groups,
// so that `kubectl get <kind>` is never ambiguous within this provider.
type kindInfo struct {
	group string
	kind  string
}

var kinds = map[string]kindInfo{
	// Applications and the APIs that they call.
	"wso2is_application":                {"application", "Application"},
	"wso2is_application_authorized_api": {"application", "AuthorizedAPI"},
	"wso2is_application_share":          {"application", "ApplicationShare"},
	"wso2is_api_resource":               {"application", "APIResource"},

	// Actions (extension points that call your own service).
	"wso2is_action": {"action", "Action"},

	// Connections: identity providers and authenticators.
	"wso2is_identity_provider":              {"idp", "IdentityProvider"},
	"wso2is_custom_authenticator":           {"idp", "CustomAuthenticator"},
	"wso2is_identity_verification_provider": {"idp", "IdentityVerificationProvider"},

	// Branding of the login and account pages.
	"wso2is_branding_preference": {"branding", "BrandingPreference"},
	"wso2is_branding_text":       {"branding", "BrandingText"},

	// Attributes (claims), dialects and OpenID Connect scopes.
	"wso2is_claim":          {"claim", "Claim"},
	"wso2is_claim_dialect":  {"claim", "ClaimDialect"},
	"wso2is_external_claim": {"claim", "ExternalClaim"},
	"wso2is_oidc_scope":     {"claim", "OIDCScope"},

	// Tenant-wide settings.
	"wso2is_admin_advisory_banner":              {"config", "AdminAdvisoryBanner"},
	"wso2is_cors_configuration":                 {"config", "CORSConfiguration"},
	"wso2is_dcr_configuration":                  {"config", "DCRConfiguration"},
	"wso2is_fraud_detection_configuration":      {"config", "FraudDetectionConfiguration"},
	"wso2is_impersonation_configuration":        {"config", "ImpersonationConfiguration"},
	"wso2is_inbound_provisioning_configuration": {"config", "InboundProvisioningConfiguration"},
	"wso2is_jwt_key_validator_configuration":    {"config", "JWTKeyValidatorConfiguration"},
	"wso2is_oauth2_configuration":               {"config", "OAuth2Configuration"},
	"wso2is_oauth2_issuer_usage_scope":          {"config", "OAuth2IssuerUsageScope"},
	"wso2is_passive_sts_configuration":          {"config", "PassiveSTSConfiguration"},
	"wso2is_remote_logging":                     {"config", "RemoteLogging"},
	"wso2is_saml_idp_configuration":             {"config", "SAMLIdPConfiguration"},
	"wso2is_server_configuration":               {"config", "ServerConfiguration"},

	// Registration and recovery flows.
	"wso2is_flow": {"flow", "Flow"},

	// Account policies (governance connectors) and input validation.
	"wso2is_governance_connector": {"governance", "GovernanceConnector"},
	"wso2is_validation_rules":     {"governance", "ValidationRules"},

	// Users, groups and roles.
	"wso2is_user":            {"identity", "User"},
	"wso2is_group":           {"identity", "Group"},
	"wso2is_group_members":   {"identity", "GroupMembers"},
	"wso2is_role":            {"identity", "Role"},
	"wso2is_role_assignment": {"identity", "RoleAssignment"},

	// Email, SMS and push notifications.
	"wso2is_email_sender":               {"notification", "EmailSender"},
	"wso2is_sms_sender":                 {"notification", "SMSSender"},
	"wso2is_push_sender":                {"notification", "PushSender"},
	"wso2is_notification_template":      {"notification", "NotificationTemplate"},
	"wso2is_notification_template_type": {"notification", "NotificationTemplateType"},

	// Organizations (B2B).
	"wso2is_organization":                         {"organization", "Organization"},
	"wso2is_organization_discovery":               {"organization", "OrganizationDiscovery"},
	"wso2is_organization_discovery_configuration": {"organization", "OrganizationDiscoveryConfiguration"},
	"wso2is_user_share":                           {"organization", "UserShare"},

	// Conditional authentication scripts.
	"wso2is_script_library": {"script", "ScriptLibrary"},
	"wso2is_secret":         {"script", "ScriptSecret"},

	// Tenants, user stores, verifiable credentials, webhooks and approval workflows.
	"wso2is_tenant":               {"tenant", "Tenant"},
	"wso2is_userstore":            {"userstore", "UserStore"},
	"wso2is_vc_template":          {"vc", "VCTemplate"},
	"wso2is_webhook":              {"webhook", "Webhook"},
	"wso2is_workflow":             {"workflow", "Workflow"},
	"wso2is_workflow_association": {"workflow", "WorkflowAssociation"},
}

// Extractors for references whose value is an argument of the referenced resource
// instead of its external name. They read the argument from status.atProvider, so a
// reference resolves only after Crossplane has observed the referenced object.
const (
	extractClaimURI   = `github.com/crossplane/upjet/v2/pkg/resource.ExtractParamPath("claim_uri",true)`
	extractDialectURI = `github.com/crossplane/upjet/v2/pkg/resource.ExtractParamPath("dialect_uri",true)`
	extractName       = `github.com/crossplane/upjet/v2/pkg/resource.ExtractParamPath("name",true)`
)

// configurators are applied to both the cluster-scoped and the namespaced provider.
// References are declared by Terraform resource name, so the same configuration resolves
// to the right Go types in either API variant.
var configurators = []func(*ujconfig.Provider){
	configureKinds,
	configureApplications,
	configureIdentityProvider,
	configureIdentity,
	configureClaims,
	configureOrganizations,
	configureOthers,
}

func configureKinds(p *ujconfig.Provider) {
	for name, k := range kinds {
		p.AddResourceConfigurator(name, func(r *ujconfig.Resource) {
			r.ShortGroup = k.group
			r.Kind = k.kind
		})
	}
}

func configureApplications(p *ujconfig.Provider) {
	p.AddResourceConfigurator("wso2is_application", func(r *ujconfig.Resource) {
		r.References["associated_roles.role_ids"] = ujconfig.Reference{TerraformName: "wso2is_role"}
		r.References["advanced_configuration.discoverable_groups.group_ids"] = ujconfig.Reference{TerraformName: "wso2is_group"}
		// Other workloads read the OAuth2 client of an OIDC application from the
		// connection secret.
		r.Sensitive.AdditionalConnectionDetailsFn = applicationConnectionDetails
	})
	p.AddResourceConfigurator("wso2is_application_authorized_api", func(r *ujconfig.Resource) {
		r.References["application_id"] = ujconfig.Reference{TerraformName: "wso2is_application"}
		r.References["api_resource_id"] = ujconfig.Reference{TerraformName: "wso2is_api_resource"}
	})
	p.AddResourceConfigurator("wso2is_application_share", func(r *ujconfig.Resource) {
		r.References["application_id"] = ujconfig.Reference{TerraformName: "wso2is_application"}
		r.References["organization_ids"] = ujconfig.Reference{TerraformName: "wso2is_organization"}
	})
	for _, name := range []string{"wso2is_branding_preference", "wso2is_branding_text", "wso2is_notification_template"} {
		p.AddResourceConfigurator(name, func(r *ujconfig.Resource) {
			r.References["application_id"] = ujconfig.Reference{TerraformName: "wso2is_application"}
		})
	}
}

func configureIdentity(p *ujconfig.Provider) {
	p.AddResourceConfigurator("wso2is_user", func(r *ujconfig.Resource) {
		// Crossplane does not support write-only arguments. passwordSecretRef already
		// keeps the password out of the managed resource.
		delete(r.TerraformResource.Schema, "password_wo")
		delete(r.TerraformResource.Schema, "password_wo_version")
		r.References["user_store"] = ujconfig.Reference{TerraformName: "wso2is_userstore", Extractor: extractName}
	})
	p.AddResourceConfigurator("wso2is_group", func(r *ujconfig.Resource) {
		r.References["user_store"] = ujconfig.Reference{TerraformName: "wso2is_userstore", Extractor: extractName}
	})
	p.AddResourceConfigurator("wso2is_group_members", func(r *ujconfig.Resource) {
		r.References["group_id"] = ujconfig.Reference{TerraformName: "wso2is_group"}
		r.References["user_ids"] = ujconfig.Reference{TerraformName: "wso2is_user"}
	})
	p.AddResourceConfigurator("wso2is_role", func(r *ujconfig.Resource) {
		r.References["audience_id"] = ujconfig.Reference{TerraformName: "wso2is_application"}
	})
	p.AddResourceConfigurator("wso2is_role_assignment", func(r *ujconfig.Resource) {
		r.References["role_id"] = ujconfig.Reference{TerraformName: "wso2is_role"}
		r.References["user_ids"] = ujconfig.Reference{TerraformName: "wso2is_user"}
		r.References["group_ids"] = ujconfig.Reference{TerraformName: "wso2is_group"}
	})
}

func configureClaims(p *ujconfig.Provider) {
	p.AddResourceConfigurator("wso2is_external_claim", func(r *ujconfig.Resource) {
		r.References["mapped_local_claim_uri"] = ujconfig.Reference{TerraformName: "wso2is_claim", Extractor: extractClaimURI}
		r.References["dialect_uri"] = ujconfig.Reference{TerraformName: "wso2is_claim_dialect", Extractor: extractDialectURI}
	})
	p.AddResourceConfigurator("wso2is_oidc_scope", func(r *ujconfig.Resource) {
		r.References["claims"] = ujconfig.Reference{TerraformName: "wso2is_external_claim", Extractor: extractClaimURI}
	})
}

func configureOrganizations(p *ujconfig.Provider) {
	p.AddResourceConfigurator("wso2is_organization_discovery", func(r *ujconfig.Resource) {
		r.References["organization_id"] = ujconfig.Reference{TerraformName: "wso2is_organization"}
	})
	p.AddResourceConfigurator("wso2is_user_share", func(r *ujconfig.Resource) {
		r.References["user_id"] = ujconfig.Reference{TerraformName: "wso2is_user"}
		r.References["organizations.organization_id"] = ujconfig.Reference{TerraformName: "wso2is_organization"}
	})
}

func configureOthers(p *ujconfig.Provider) {
	p.AddResourceConfigurator("wso2is_secret", func(r *ujconfig.Resource) {
		// Crossplane does not support write-only arguments. valueSecretRef already keeps
		// the value out of the managed resource.
		delete(r.TerraformResource.Schema, "value_wo")
		delete(r.TerraformResource.Schema, "value_wo_version")
	})
	p.AddResourceConfigurator("wso2is_workflow", func(r *ujconfig.Resource) {
		r.References["steps.role_ids"] = ujconfig.Reference{TerraformName: "wso2is_role"}
		r.References["steps.user_ids"] = ujconfig.Reference{TerraformName: "wso2is_user"}
	})
	p.AddResourceConfigurator("wso2is_workflow_association", func(r *ujconfig.Resource) {
		r.References["workflow_id"] = ujconfig.Reference{TerraformName: "wso2is_workflow"}
	})
}

// applicationConnectionDetails publishes the OAuth2 client of an OIDC application as the
// connection secret keys `client_id` and `client_secret`. The key `credentials` holds both
// as a JSON object in the format of a ProviderConfig Secret, so that a ProviderConfig can
// sign in with an application that Crossplane created. The ProviderConfig then sets
// baseURL, and organizationID for an organization, in its spec.
func applicationConnectionDetails(attr map[string]any) (map[string][]byte, error) {
	out := map[string][]byte{}
	oidc, ok := firstObject(attr["oidc"])
	if !ok {
		return out, nil
	}
	creds := map[string]string{}
	for _, k := range []string{"client_id", "client_secret"} {
		if v, ok := oidc[k].(string); ok && v != "" {
			out[k] = []byte(v)
			creds[k] = v
		}
	}
	if len(creds) == 2 {
		b, err := json.Marshal(creds)
		if err != nil {
			return nil, errors.Wrap(err, "cannot encode the application's client credentials")
		}
		out["credentials"] = b
	}
	return out, nil
}

// firstObject returns a nested object of a Terraform state map. A single nested attribute
// is a map, and Upjet sometimes wraps it in a list with one element.
func firstObject(v any) (map[string]any, bool) {
	switch o := v.(type) {
	case map[string]any:
		return o, true
	case []any:
		if len(o) == 1 {
			m, ok := o[0].(map[string]any)
			return m, ok
		}
	}
	return nil, false
}
