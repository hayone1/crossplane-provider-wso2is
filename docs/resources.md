# Managed resources

The provider has 52 managed resources. Each one exists in two API variants:

* **Cluster-scoped:** the API group is `<group>.wso2is.crossplane.io`, and the version is
  `v1alpha1`. For example, an Application is `application.wso2is.crossplane.io/v1alpha1`.
* **Namespaced:** the API group is `<group>.wso2is.m.crossplane.io`. Crossplane v2 introduced
  this variant. For example, a namespaced Application is
  `application.wso2is.m.crossplane.io/v1alpha1`.

Every kind matches one resource of the Terraform provider. The arguments in `spec.forProvider`
are the Terraform arguments in camel case. For example, `claim_uri` becomes `claimUri`. Run
`kubectl explain <kind>.spec.forProvider` to see every field with its description.

The last column says where the external name comes from:

* **Created:** Crossplane creates the object and then records its ID. A create fails if the
  object already exists. To manage an existing object, set the `crossplane.io/external-name`
  annotation to its ID in the format that the column shows. "base64url" means the URL-safe
  base64 encoding without the trailing `=` padding. For example,
  `printf '%s' 'http://wso2.org/claims/loyaltyTier' | base64 | tr '+/' '-_' | tr -d '=\n'`
  prints the ID of that claim.
* **Takes over:** the provider computes the ID from the spec. If the object already exists, the
  managed resource takes it over.
* **Fixed:** the setting exists once per tenant.

[Known behaviour](behaviour.md#how-a-managed-resource-finds-its-object) explains what each choice
means for objects that already exist.

| Kind | Group | Terraform resource | External name |
|------|-------|--------------------|---------------|
| `APIResource` | `application` | `wso2is_api_resource` | Created: the server assigns a UUID |
| `Application` | `application` | `wso2is_application` | Created: the server assigns a UUID |
| `ApplicationShare` | `application` | `wso2is_application_share` | Takes over: `applicationId` |
| `AuthorizedAPI` | `application` | `wso2is_application_authorized_api` | Created: `<applicationId>/<apiResourceId>` |
| `Action` | `action` | `wso2is_action` | Created: the server assigns a UUID |
| `CustomAuthenticator` | `idp` | `wso2is_custom_authenticator` | Created: base64url of the name |
| `IdentityProvider` | `idp` | `wso2is_identity_provider` | Created: the server assigns a UUID |
| `IdentityVerificationProvider` | `idp` | `wso2is_identity_verification_provider` | Created: the server assigns a UUID |
| `BrandingPreference` | `branding` | `wso2is_branding_preference` | Takes over: `<type>/<name>/<locale>` |
| `BrandingText` | `branding` | `wso2is_branding_text` | Takes over: `<type>/<name>/<locale>/<screen>` |
| `Claim` | `claim` | `wso2is_claim` | Created: base64url of the claim URI |
| `ClaimDialect` | `claim` | `wso2is_claim_dialect` | Created: base64url of the dialect URI, or `local` |
| `ExternalClaim` | `claim` | `wso2is_external_claim` | Created: `<dialect ID>/<claim ID>` (both base64url) |
| `OIDCScope` | `claim` | `wso2is_oidc_scope` | Created: the scope name |
| `AdminAdvisoryBanner` | `config` | `wso2is_admin_advisory_banner` | Fixed: `admin_advisory_banner` |
| `CORSConfiguration` | `config` | `wso2is_cors_configuration` | Fixed: `cors_configuration` |
| `DCRConfiguration` | `config` | `wso2is_dcr_configuration` | Fixed: `dcr_configuration` |
| `FraudDetectionConfiguration` | `config` | `wso2is_fraud_detection_configuration` | Fixed: `fraud_detection_configuration` |
| `ImpersonationConfiguration` | `config` | `wso2is_impersonation_configuration` | Fixed: `impersonation_configuration` |
| `InboundProvisioningConfiguration` | `config` | `wso2is_inbound_provisioning_configuration` | Fixed: `inbound_provisioning_configuration` |
| `JWTKeyValidatorConfiguration` | `config` | `wso2is_jwt_key_validator_configuration` | Fixed: `jwt_key_validator_configuration` |
| `OAuth2Configuration` | `config` | `wso2is_oauth2_configuration` | Fixed: `oauth2_configuration` |
| `OAuth2IssuerUsageScope` | `config` | `wso2is_oauth2_issuer_usage_scope` | Fixed: `oauth2_issuer_usage_scope` |
| `PassiveSTSConfiguration` | `config` | `wso2is_passive_sts_configuration` | Fixed: `passive_sts_configuration` |
| `RemoteLogging` | `config` | `wso2is_remote_logging` | Takes over: `logType` |
| `SAMLIdPConfiguration` | `config` | `wso2is_saml_idp_configuration` | Fixed: `saml_idp_configuration` |
| `ServerConfiguration` | `config` | `wso2is_server_configuration` | Fixed: `server_configuration` |
| `Flow` | `flow` | `wso2is_flow` | Takes over: `flowType` |
| `GovernanceConnector` | `governance` | `wso2is_governance_connector` | Takes over: base64url of `name` |
| `ValidationRules` | `governance` | `wso2is_validation_rules` | Takes over: `field` |
| `Group` | `identity` | `wso2is_group` | Created: the server assigns a UUID |
| `GroupMembers` | `identity` | `wso2is_group_members` | Takes over: `groupId` |
| `Role` | `identity` | `wso2is_role` | Created: the server assigns a UUID |
| `RoleAssignment` | `identity` | `wso2is_role_assignment` | Takes over: `roleId` |
| `User` | `identity` | `wso2is_user` | Created: the server assigns a UUID |
| `EmailSender` | `notification` | `wso2is_email_sender` | Fixed: `EmailPublisher` |
| `NotificationTemplate` | `notification` | `wso2is_notification_template` | Takes over: `<channel>/<type>/<locale>[/<applicationId>]` |
| `NotificationTemplateType` | `notification` | `wso2is_notification_template_type` | Created: `<channel>/<name>` |
| `PushSender` | `notification` | `wso2is_push_sender` | Created: `<providerName>_PushPublisher` |
| `SMSSender` | `notification` | `wso2is_sms_sender` | Fixed: `SMSPublisher` |
| `Organization` | `organization` | `wso2is_organization` | Created: the server assigns a UUID |
| `OrganizationDiscovery` | `organization` | `wso2is_organization_discovery` | Takes over: `organizationId` |
| `OrganizationDiscoveryConfiguration` | `organization` | `wso2is_organization_discovery_configuration` | Fixed: `organization_discovery_configuration` |
| `UserShare` | `organization` | `wso2is_user_share` | Takes over: `userId` |
| `ScriptLibrary` | `script` | `wso2is_script_library` | Created: the library name |
| `ScriptSecret` | `script` | `wso2is_secret` | Created: `<type>/<name>` |
| `Tenant` | `tenant` | `wso2is_tenant` | Created: the server assigns a UUID |
| `UserStore` | `userstore` | `wso2is_userstore` | Created: base64url of the domain name |
| `VCTemplate` | `vc` | `wso2is_vc_template` | Created: the server assigns a UUID |
| `Webhook` | `webhook` | `wso2is_webhook` | Created: the server assigns a UUID |
| `Workflow` | `workflow` | `wso2is_workflow` | Created: the server assigns a UUID |
| `WorkflowAssociation` | `workflow` | `wso2is_workflow_association` | Created: the server assigns a UUID |

## References between resources

Instead of an ID, you can name another managed resource in a field that ends in `Ref`, or select
one by labels in a field that ends in `Selector`. Crossplane fills in the value as soon as the
other managed resource has it. For an ID, that happens after the provider has created the other
object. For a value such as a claim URI, it happens after the provider has observed the other
object once.

| Kind | Field | Refers to |
|------|-------|-----------|
| Application | `associatedRoles.roleIds` | Role |
| Application | `advancedConfiguration.discoverableGroups.groupIds` | Group |
| AuthorizedAPI | `applicationId`, `apiResourceId` | Application, APIResource |
| ApplicationShare | `applicationId`, `organizationIds` | Application, Organization |
| BrandingPreference, BrandingText, NotificationTemplate | `applicationId` | Application |
| User, Group | `userStore` | UserStore (its `name`) |
| GroupMembers | `groupId`, `userIds` | Group, User |
| Role | `audienceId` | Application |
| RoleAssignment | `roleId`, `userIds`, `groupIds` | Role, User, Group |
| ExternalClaim | `dialectUri`, `mappedLocalClaimUri` | ClaimDialect (its `dialectUri`), Claim (its `claimUri`) |
| OIDCScope | `claims` | ExternalClaim (its `claimUri`) |
| OrganizationDiscovery | `organizationId` | Organization |
| UserShare | `userId`, `organizations.organizationId` | User, Organization |
| Workflow | `steps.roleIds`, `steps.userIds` | Role, User |
| WorkflowAssociation | `workflowId` | Workflow |

## Secrets

A field that the Terraform provider marks as sensitive becomes a field that ends in
`SecretRef`. It points at a key of a Kubernetes Secret (for example `passwordSecretRef`), or at
a whole Secret for a map (for example `secretPropertiesSecretRef`). The identity provider is
an exception with three top-level fields. [Known behaviour](behaviour.md#secrets-of-identity-provider-authenticators)
explains them.

The Terraform provider's write-only arguments `password_wo` (User) and `value_wo` (ScriptSecret)
are not available. Use `passwordSecretRef` and `valueSecretRef` instead. They also keep the
value out of the managed resource.

## Connection details

A managed resource with `spec.writeConnectionSecretToRef` writes its sensitive values into that
Secret. Upjet names these keys `attribute.<field path>`, for example
`attribute.oidc.client_secret` for an Application, or `attribute.password` for a User. An
Application also writes these keys:

| Key | Value |
|-----|-------|
| `client_id` | The OAuth2 client ID of the OIDC application |
| `client_secret` | The OAuth2 client secret |
| `credentials` | Both values as a JSON object that a ProviderConfig can read |
