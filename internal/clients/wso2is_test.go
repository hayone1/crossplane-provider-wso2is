package clients

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	namespacedv1beta1 "github.com/hayone1/crossplane-provider-wso2is/apis/namespaced/v1beta1"
)

func TestConfiguration(t *testing.T) {
	// defaults is the configuration of the minimal credentials. Every argument of the
	// Terraform provider is present, so that it never reads environment variables.
	defaults := func(changes map[string]any) map[string]any {
		cfg := map[string]any{
			"base_url":             "https://id.example.com",
			"tenant_domain":        "",
			"organization_id":      "",
			"username":             "admin",
			"password":             "secret",
			"client_id":            "",
			"client_secret":        "",
			"access_token":         "",
			"ca_certificate":       "",
			"scopes":               []string{},
			"insecure_skip_verify": false,
			"request_timeout":      int64(60),
			"max_retries":          int64(4),
		}
		for k, v := range changes {
			cfg[k] = v
		}
		return cfg
	}
	basic := `{"base_url": "https://id.example.com", "username": "admin", "password": "secret"}`

	cases := map[string]struct {
		creds   string
		spec    namespacedv1beta1.ProviderConfigSpec
		want    map[string]any
		wantErr string
	}{
		"BasicAuthentication": {
			creds: basic,
			want:  defaults(nil),
		},
		"ClientCredentialsWithAllSettings": {
			creds: `{"base_url": "https://id.example.com/", "tenant_domain": "acme.com", "organization_id": "org-1",
				"client_id": "id", "client_secret": "cs", "scopes": ["SYSTEM", "internal_org_user_mgt_view"],
				"insecure_skip_verify": true, "ca_certificate": "-----BEGIN CERTIFICATE-----\nx\n-----END CERTIFICATE-----",
				"request_timeout": 30, "max_retries": 0}`,
			want: defaults(map[string]any{
				"tenant_domain": "acme.com", "organization_id": "org-1", "username": "", "password": "",
				"client_id": "id", "client_secret": "cs", "scopes": []string{"SYSTEM", "internal_org_user_mgt_view"},
				"insecure_skip_verify": true, "ca_certificate": "-----BEGIN CERTIFICATE-----\nx\n-----END CERTIFICATE-----",
				"request_timeout": int64(30), "max_retries": int64(0),
			}),
		},
		"AccessToken": {
			creds: `{"base_url": "https://id.example.com", "access_token": "tok"}`,
			want:  defaults(map[string]any{"username": "", "password": "", "access_token": "tok"}),
		},
		"StringValuesForBooleansAndNumbers": {
			creds: `{"base_url": "https://id.example.com", "username": "admin", "password": "secret",
				"insecure_skip_verify": "true", "request_timeout": "90", "max_retries": "2", "scopes": "SYSTEM  openid"}`,
			want: defaults(map[string]any{"insecure_skip_verify": true, "request_timeout": int64(90), "max_retries": int64(2), "scopes": []string{"SYSTEM", "openid"}}),
		},
		"SpecSetsTheServerTenantAndOrganization": {
			creds: `{"client_id": "id", "client_secret": "cs"}`,
			spec:  namespacedv1beta1.ProviderConfigSpec{BaseURL: "https://id.example.com/", TenantDomain: "acme.com", OrganizationID: "org-1"},
			want: defaults(map[string]any{"tenant_domain": "acme.com", "organization_id": "org-1", "username": "", "password": "",
				"client_id": "id", "client_secret": "cs"}),
		},
		"SpecRepeatsTheCredentialsValue": {
			creds: basic,
			spec:  namespacedv1beta1.ProviderConfigSpec{BaseURL: "https://id.example.com/"},
			want:  defaults(nil),
		},
		"SpecConflictsWithTheCredentials": {
			creds:   basic,
			spec:    namespacedv1beta1.ProviderConfigSpec{BaseURL: "https://other.example.com"},
			wantErr: "different values for base_url",
		},
		"SecretsKeepTheirSpaces": {
			creds: `{"base_url": " https://id.example.com/ ", "tenant_domain": " acme.com ", "username": "admin", "password": " secret "}`,
			want:  defaults(map[string]any{"tenant_domain": "acme.com", "password": " secret "}),
		},
		"MissingServerURL": {
			creds:   `{"username": "admin", "password": "secret"}`,
			wantErr: "server URL is missing",
		},
		"UnknownKey": {
			creds:   `{"base_url": "https://id.example.com", "clientId": "id", "client_secret": "cs"}`,
			wantErr: "unknown keys clientId",
		},
		"Empty": {
			creds:   "  ",
			wantErr: "credentials are empty",
		},
		"NotAnObject": {
			creds:   "my-token",
			wantErr: "must be a JSON object",
		},
		"InvalidJSON": {
			creds:   `{"base_url": "https://id.example.com", "password": "hunter2",`,
			wantErr: "not a valid JSON object",
		},
		"NoSignInMethod": {
			creds:   `{"base_url": "https://id.example.com"}`,
			wantErr: "no sign-in method",
		},
		"TwoSignInMethods": {
			creds:   `{"base_url": "https://id.example.com", "username": "admin", "password": "secret", "access_token": "tok"}`,
			wantErr: "more than one sign-in method",
		},
		"UsernameWithoutPassword": {
			creds:   `{"base_url": "https://id.example.com", "username": "admin"}`,
			wantErr: "needs both username and password",
		},
		"ClientIDWithoutSecret": {
			creds:   `{"base_url": "https://id.example.com", "client_id": "id"}`,
			wantErr: "needs both client_id and client_secret",
		},
		"OrganizationWithBasicAuthentication": {
			creds:   basic,
			spec:    namespacedv1beta1.ProviderConfigSpec{OrganizationID: "org-1"},
			wantErr: "reject basic authentication",
		},
		"CACertificateFilePath": {
			creds:   `{"base_url": "https://id.example.com", "username": "admin", "password": "secret", "ca_certificate": "/etc/ssl/ca.pem"}`,
			wantErr: "must hold PEM text",
		},
		"WrongTypeForAString": {
			creds:   `{"base_url": "https://id.example.com", "username": "admin", "password": 1234}`,
			wantErr: `"password" must be a string`,
		},
		"WrongScopes": {
			creds:   `{"base_url": "https://id.example.com", "username": "admin", "password": "secret", "scopes": [1]}`,
			wantErr: "scopes must be a list of strings",
		},
		"WrongBoolean": {
			creds:   `{"base_url": "https://id.example.com", "username": "admin", "password": "secret", "insecure_skip_verify": "yes please"}`,
			wantErr: "must be true or false",
		},
		"ZeroTimeout": {
			creds:   `{"base_url": "https://id.example.com", "username": "admin", "password": "secret", "request_timeout": 0}`,
			wantErr: "positive number of seconds",
		},
		"FractionalTimeout": {
			creds:   `{"base_url": "https://id.example.com", "username": "admin", "password": "secret", "request_timeout": 1.5}`,
			wantErr: "must be a whole number",
		},
		"NegativeRetries": {
			creds:   `{"base_url": "https://id.example.com", "username": "admin", "password": "secret", "max_retries": -1}`,
			wantErr: "zero or a positive number",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := Configuration([]byte(tc.creds), &tc.spec)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("Configuration() error = %v, want an error that contains %q", err, tc.wantErr)
				}
				// Error messages must never repeat secret values.
				for _, secret := range []string{"secret", "hunter2", "cs", "tok"} {
					if strings.Contains(err.Error(), `"`+secret+`"`) || strings.Contains(err.Error(), " "+secret+" ") {
						t.Fatalf("Configuration() error %q contains a secret value", err)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("Configuration() unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Configuration() -want +got:\n%s", diff)
			}
		})
	}
}
