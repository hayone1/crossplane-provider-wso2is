package config

import (
	"regexp"
	"strings"

	"github.com/crossplane/upjet/v2/pkg/config"
)

// The descriptions in the CRDs come from the Terraform provider's documentation. The
// rewrites below turn the parts that only make sense for Terraform into Crossplane terms:
// Terraform resource names become kinds, "destroy" becomes "delete the managed resource",
// and import instructions describe the external-name annotation.

// docReplacements are exact text replacements, applied in order.
var docReplacements = []struct{ old, new string }{
	// Secrets: Crossplane reads them from Kubernetes Secrets and has no Terraform state.
	{" Terraform stores it in the state. Use value_wo instead to keep it out of the state.", " The provider reads it from a Kubernetes Secret, so it never appears in the managed resource."},
	{" Terraform stores it in the state. Use password_wo instead to keep it out of the state.", " The provider reads it from a Kubernetes Secret, so it never appears in the managed resource."},
	{"Use password_wo (Terraform 1.11 or later) to keep the password out of the state, or ask_password to let the user choose a password.", "Set ask_password to let the user choose a password instead."},
	{", so the credentials appear in the Terraform state.", ". The provider does not copy them into the managed resource."},
	{"changed outside Terraform", "changed outside Crossplane"},
	{"made outside Terraform", "made outside Crossplane"},
	// Provider configuration: Crossplane uses one ProviderConfig per tenant or organization.
	{"The provider that creates the tenant must manage the super tenant (tenant_domain = \"carbon.super\").", "The ProviderConfig that creates the tenant must manage the super tenant (tenantDomain carbon.super)."},
	{"To manage objects inside the tenant, add another provider block with an alias. Set tenant_domain = wso2is_tenant.", "To manage objects inside the tenant, create another ProviderConfig and set its tenantDomain to the tenant's domain."},
	{"To manage objects inside it, add a provider block with an alias that sets organization_id.", "To manage objects inside it, create a ProviderConfig that sets organizationID to the organization's ID."},
	{"To create a nested organization, use a provider alias whose organization_id is the parent.", "To create a nested organization, use a ProviderConfig whose organizationID is the parent."},
	{"Add depends_on for that resource, so that Terraform also removes the domains before it turns discovery off.", "Delete the OrganizationDiscovery resources before that resource, so that the provider removes the domains before it turns discovery off."},
	{"Terraform resources that perform the operation, such as wso2is_user, then fail", "Managed resources that perform the operation, such as a User, then fail"},
	// Life cycle: Crossplane deletes managed resources instead of destroying them.
	{"When you destroy the resource,", "When you delete the managed resource,"},
	{"When you destroy this resource,", "When you delete the managed resource,"},
	{"when the resource is destroyed", "when the managed resource is deleted"},
	{"when you destroy the resource", "when you delete the managed resource"},
	{"to the server defaults on destroy", "to the server defaults when the managed resource is deleted"},
	{"and Terraform stops managing them", "and Crossplane stops managing them"},
	// Terraform functions and data sources.
	{"To find a management API, use the wso2is_api_resource data source.", "To find the ID of a management API, call GET /api/server/v1/api-resources on the server."},
	{"To see a connector's properties, use the wso2is_governance_connector data source.", "To see a connector's properties, call GET /api/server/v1/identity-governance on the server."},
	{" (use jsonencode)", " (a JSON document as a string)"},
	{"You can design the flow in the Console and copy it with the wso2is_flow data source, or write it with jsonencode.", "You can design the flow in the Console and copy its JSON from that API, or write the JSON yourself."},
}

var (
	reFileFunction = regexp.MustCompile(`,? \(?for example file\("[^"]*"\)\)?`)
	// The import sentence is always the last sentence, and IDs in it can contain periods.
	reImportSentence   = regexp.MustCompile(`\s*To import.*$`)
	reTerraformName    = regexp.MustCompile(`\bwso2is_[a-z0-9_]+\b`)
	reMultipleSpaces   = regexp.MustCompile(`  +`)
	reSpaceBeforePunct = regexp.MustCompile(` +([.,])`)
)

// idFormats describes the IDs of the created kinds whose ID is not a server-assigned UUID.
var idFormats = map[string]string{
	"wso2is_application_authorized_api": "<application ID>/<API resource ID>",
	"wso2is_push_sender":                "<provider name>_PushPublisher, for example FCM_PushPublisher",
	"wso2is_claim":                      "the base64url encoding of the claim URI, without padding",
	"wso2is_claim_dialect":              "the base64url encoding of the dialect URI, without padding, or local for the local dialect",
	"wso2is_external_claim":             "<dialect ID>/<claim ID>, where both parts are base64url encodings of the URIs, without padding",
	"wso2is_custom_authenticator":       "the base64url encoding of the name, without padding",
	"wso2is_userstore":                  "the base64url encoding of the domain name, without padding",
	"wso2is_oidc_scope":                 "the scope name",
	"wso2is_script_library":             "the library name",
	"wso2is_secret":                     "<type>/<name>",
	"wso2is_notification_template_type": "<channel>/<name>",
}

// importSentence returns the sentence that explains how a managed resource finds an
// existing object.
func importSentence(name string) string {
	switch {
	case isSingleton(name):
		return "The setting exists once per tenant, so manage it with only one managed resource."
	case isDerived(name):
		return "If the object already exists, the managed resource takes it over."
	case idFormats[name] != "":
		return "To manage an existing object, set the crossplane.io/external-name annotation to its ID, which is " + idFormats[name] + "."
	default:
		return "To manage an existing object, set the crossplane.io/external-name annotation to its ID."
	}
}

// crossplaneDocs rewrites the descriptions of a resource and its arguments.
func crossplaneDocs(r *config.Resource) {
	if r.MetaResource == nil {
		return
	}
	r.MetaResource.Description = rewriteDoc(r.MetaResource.Description)
	if reImportSentence.MatchString(r.MetaResource.Description) {
		r.MetaResource.Description = reImportSentence.ReplaceAllString(r.MetaResource.Description, " "+importSentence(r.Name))
	} else {
		r.MetaResource.Description = strings.TrimSpace(r.MetaResource.Description) + " " + importSentence(r.Name)
	}
	r.MetaResource.Description = tidy(r.MetaResource.Description)
	for k, v := range r.MetaResource.ArgumentDocs {
		r.MetaResource.ArgumentDocs[k] = tidy(rewriteDoc(v))
	}
}

func rewriteDoc(s string) string {
	for _, rep := range docReplacements {
		s = strings.ReplaceAll(s, rep.old, rep.new)
	}
	s = reFileFunction.ReplaceAllString(s, "")
	return reTerraformName.ReplaceAllStringFunc(s, func(name string) string {
		if k, ok := kinds[name]; ok {
			return k.kind
		}
		return name
	})
}

func tidy(s string) string {
	s = reMultipleSpaces.ReplaceAllString(s, " ")
	s = reSpaceBeforePunct.ReplaceAllString(s, "$1")
	return strings.TrimSpace(s)
}

func isSingleton(name string) bool {
	_, ok := singletonIDs[name]
	return ok
}

func isDerived(name string) bool {
	_, ok := derivedNames[name]
	return ok
}
