package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

// A ProviderConfigSpec defines how the provider signs in to a WSO2 Identity Server.
type ProviderConfigSpec struct {
	// Credentials holds the sign-in settings. The provider reads a JSON object with the
	// same keys as the Terraform provider's configuration. Set base_url and one of three
	// sign-in methods: username and password, client_id and client_secret, or
	// access_token. The other keys are tenant_domain, organization_id, scopes,
	// insecure_skip_verify, ca_certificate, request_timeout and max_retries.
	Credentials ProviderCredentials `json:"credentials"`

	// BaseURL is the origin of the server, for example https://id.example.com. Do not
	// include a path. Set it here or as base_url in the credentials, but not in both
	// places with different values.
	// +optional
	// +kubebuilder:validation:Pattern=`^https?://[^/]+/?$`
	BaseURL string `json:"baseURL,omitempty"`

	// TenantDomain is the tenant (root organization) that the managed resources belong
	// to. The default is carbon.super. Set it here or as tenant_domain in the
	// credentials, but not in both places with different values.
	// +optional
	TenantDomain string `json:"tenantDomain,omitempty"`

	// OrganizationID is the ID of a sub-organization. Set it to manage objects inside
	// that organization instead of the root organization. The organization APIs need
	// client_id and client_secret of an application that is shared with the
	// organization, or an access_token issued for the organization. Set it here or as
	// organization_id in the credentials, but not in both places with different values.
	// +optional
	OrganizationID string `json:"organizationID,omitempty"`
}

// ProviderCredentials says where the provider finds the credentials.
type ProviderCredentials struct {
	// Source of the credentials. Use Secret to read a key of a Kubernetes Secret.
	// +kubebuilder:validation:Enum=Secret;Environment;Filesystem
	Source xpv2.CredentialsSource `json:"source"`

	xpv2.CommonCredentialSelectors `json:",inline"`
}

// A ProviderConfigStatus reflects the observed state of a ProviderConfig.
type ProviderConfigStatus struct {
	xpv2.ProviderConfigStatus `json:",inline"`
}

// +kubebuilder:object:root=true

// A ProviderConfig configures how the provider signs in to a WSO2 Identity Server.
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="SECRET-NAME",type="string",JSONPath=".spec.credentials.secretRef.name",priority=1
// +kubebuilder:resource:scope=Cluster,categories={crossplane,provider,wso2is}
type ProviderConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProviderConfigSpec   `json:"spec"`
	Status ProviderConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProviderConfigList contains a list of ProviderConfig.
type ProviderConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProviderConfig `json:"items"`
}

// +kubebuilder:object:root=true

// A ProviderConfigUsage indicates that a resource is using a ProviderConfig.
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="CONFIG-NAME",type="string",JSONPath=".providerConfigRef.name"
// +kubebuilder:printcolumn:name="RESOURCE-KIND",type="string",JSONPath=".resourceRef.kind"
// +kubebuilder:printcolumn:name="RESOURCE-NAME",type="string",JSONPath=".resourceRef.name"
// +kubebuilder:resource:scope=Cluster,categories={crossplane,provider,wso2is}
type ProviderConfigUsage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	xpv2.ProviderConfigUsage `json:",inline"`
}

// +kubebuilder:object:root=true

// ProviderConfigUsageList contains a list of ProviderConfigUsage
type ProviderConfigUsageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProviderConfigUsage `json:"items"`
}
