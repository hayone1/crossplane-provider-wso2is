# Known behaviour and limitations

WSO2 Identity Server and Crossplane both have rules that affect how you write managed
resources. The provider handles most of them for you. This page lists the rules that you still
need to know.

Each section says what happens, what the provider does about it, and what you should do.

## Supported server versions

The provider supports **WSO2 Identity Server 7.3.x**.

* **7.2.0 mostly works.** Features that WSO2 added in 7.3 fail on 7.2.0 with HTTP 400
  `UE-10000` or HTTP 404. These features are EmailSender, the `authentication` settings of
  SMSSender, PushSender, UserShare, VCTemplate, notifications in workflows,
  OAuth2Configuration, FraudDetectionConfiguration, and `oidc.tokenIssuerOrganizationId` on
  applications.
* **7.1.0 is not supported.** Most areas fail on 7.1.0.

## How a managed resource finds its object

Every managed resource records the ID of its object in the `crossplane.io/external-name`
annotation. The kinds fall into three groups. [Managed resources](resources.md) shows the group
of each kind.

* **Crossplane creates the object.** Examples are Application, User, Group, Role,
  Organization, Claim, ClaimDialect, OIDCScope and UserStore. Crossplane creates the object and
  then records its ID. If an object with the same name already exists, the server rejects the
  create, and the managed resource shows the error. To manage an existing object, set the
  annotation to the object's ID before you create the managed resource.
* **The managed resource takes the object over.** These kinds manage something that always
  exists, or that belongs to another object: GovernanceConnector, ValidationRules, Flow,
  RemoteLogging, BrandingPreference, BrandingText, NotificationTemplate, ApplicationShare,
  GroupMembers, RoleAssignment, UserShare and OrganizationDiscovery. The provider computes the
  ID from the spec. If the object exists, the managed resource takes it over and changes it to
  match the spec.
* **Singletons** manage settings that exist once per tenant, such as ServerConfiguration or
  CORSConfiguration. Their external name is fixed.

Deleting a managed resource deletes or resets its object, also when the managed resource took the
object over. For example, a RoleAssignment for an existing role replaces the role's members when
you create it, and removes all of them when you delete it.

**What you should do:**

* Before you create a managed resource that takes an object over, decide whether Crossplane
  should own the object.
* To only watch an object, set `spec.managementPolicies: ["Observe"]`.
* If you watch an object first and allow changes later, copy the values from
  `status.atProvider` into `spec.forProvider` before you add `Update` to the management
  policies. Otherwise the first update applies the spec as it is. For a GroupMembers or a
  RoleAssignment with no member list, that update removes every member.

## Deleting a managed resource

When you delete a managed resource, the provider deletes its object on the server. Some kinds
behave differently, because their object cannot be deleted:

| Kind | What happens on delete |
|------|-----------------------|
| The `*Configuration` kinds, AdminAdvisoryBanner, OAuth2IssuerUsageScope | The provider either restores the server defaults or leaves the current values. The description of each kind says which. |
| GovernanceConnector, ValidationRules | The provider restores the server defaults, unless `restoreOnDestroy` is `false`. |
| Flow | The provider turns the flow off and deletes it. By default, WSO2 IS 7.3.0 does not allow the API to delete the root organization's flow. The flow then stays on the server, turned off. See [Flows](#flows). |
| Tenant | The server deletes a tenant only when its `deployment.toml` allows it. Otherwise the tenant is only deactivated. See [Tenants](#tenants). |
| WorkflowAssociation | The server refuses to delete the last association of a workflow. The provider then turns the association off. It goes away with the workflow. |
| NotificationTemplate | The server goes back to the system template. |
| EmailSender, SMSSender, PushSender | The provider removes the sender. |

**What you should do:** To keep the object on the server when you delete the managed resource,
set `spec.deletionPolicy: Orphan` (cluster-scoped resources) or leave `Delete` out of
`spec.managementPolicies`.

## Delete objects in the right order

Crossplane deletes the managed resources of a manifest at the same time. Some objects should go
before the objects that they belong to:

* **OrganizationDiscovery before its OrganizationDiscoveryConfiguration.** The server accepts
  the removal of email domains only while discovery is on.
* **Objects inside an organization or a tenant before the Organization or the Tenant, and
  before its ProviderConfig.**

To make Crossplane keep this order, create a Usage. A Usage stops the deletion of one managed
resource while another managed resource uses it. With `replayDeletion: true`, Crossplane
deletes the used resource as soon as the user is gone. This example needs Crossplane v2 and
namespaced managed resources. For cluster-scoped managed resources, use a ClusterUsage of the
same API group instead:

```yaml
apiVersion: protection.crossplane.io/v1beta1
kind: Usage
metadata:
  name: discovery-is-used-by-partner-domains
  namespace: default
spec:
  replayDeletion: true
  of:
    apiVersion: organization.wso2is.m.crossplane.io/v1alpha1
    kind: OrganizationDiscoveryConfiguration
    resourceRef:
      name: discovery
  by:
    apiVersion: organization.wso2is.m.crossplane.io/v1alpha1
    kind: OrganizationDiscovery
    resourceRef:
      name: domains-partner
```

## Changes that need a new object

The server accepts some settings only when it creates an object. For example, an
Application's `isManagementApp`, `isB2bSelfServiceApp`, custom OIDC client ID, custom client
secret and FAPI setting are fixed at creation. Terraform would replace the object. Crossplane
never replaces objects on its own. Instead, the managed resource shows the condition
`Synced=False` with the message "diff contains fields that require resource replacement".

The same happens when you change a setting that is part of the object's ID, for example the
`claimUri` of a Claim or the `groupId` of a GroupMembers. The provider keeps the recorded ID, so
it never creates a second object by accident.

**What you should do:** Change the setting back, or create a new managed resource with the new
value and then delete the old one.

## Settings that you leave out

The provider does not copy the server's values into `spec.forProvider`. Crossplane calls that
copy *late initialization*, and this provider turns it off for every field. A setting that you
leave out stays under the server's control:

* A settings kind, such as ServerConfiguration, only changes the settings that you list.
  Someone can still change the other settings in the Console.
* A list with an empty default, such as `userIds` and `groupIds` of a RoleAssignment, means
  "none". A RoleAssignment with only `userIds` removes every group from the role.

You can read the current value of every setting in `status.atProvider`.

## Secrets

* **Secrets come from Kubernetes Secrets.** Each secret field has a `...SecretRef` field, for
  example `passwordSecretRef` on a User. The value never appears in the managed resource.
* **Most APIs never send secrets back.** The provider uses the value from the Secret. It cannot
  notice when someone changes the secret in the Console. To change a secret, change the
  Kubernetes Secret, and the provider sends the new value with the next reconcile.
* **Change secrets while the provider runs.** The provider remembers the secret values that it
  sent only while it runs. After a restart, it assumes that the server has the values from the
  Secrets. A Secret that you changed while the provider was stopped therefore does not reach the
  server. To send it, change the Secret to a new value while the provider runs.
* **Some APIs do send secrets back.** These are the SMS sender with basic authentication, push
  senders, and the secrets of identity provider authenticators. The provider does not copy them
  into `status.atProvider`.
* **Applications publish their OAuth2 client.** Set `spec.writeConnectionSecretToRef` on an
  Application. The Secret gets `client_id`, `client_secret`, and `credentials`. The
  `credentials` key holds both values in the format of a ProviderConfig Secret. Like every
  managed resource, the Application also writes its sensitive values as `attribute.*` keys.
  See [Connection details](resources.md#connection-details).
* **A User's password is sent again only when the Secret changes.** The server never returns
  passwords, so the provider compares the Secret with the value that it sent last. A user can
  therefore change their own password in My Account, and Crossplane leaves the new password
  alone. To let the user choose the first password, set `askPassword: true` and leave
  `passwordSecretRef` out.

### Secrets of identity provider authenticators

Upjet, the code generator behind this provider, cannot read secret maps that sit inside
`federatedAuthenticators` or `outboundProvisioningConnectors`. It would put the values of all
authenticators under one wrong name
([crossplane/upjet#810](https://github.com/crossplane/upjet/issues/810)). The provider
therefore offers three top-level fields instead:

| Field | Secret key format | Example key |
|-------|-------------------|-------------|
| `federatedAuthenticatorSecretPropertiesSecretRef` | `<authenticator>.<property>` | `GoogleOIDCAuthenticator.ClientSecret` |
| `federatedAuthenticatorEndpointAuthenticationPropertiesSecretRef` | `<authenticator>.<property>` | `MyAuthenticator.password` |
| `outboundProvisioningConnectorSecretPropertiesSecretRef` | `<connector>.<property>` | `scim2.scim2-password` |

Each field points at a whole Secret. The provider splits each key at the first period. The part
before the period must name an entry of `federatedAuthenticators` or
`outboundProvisioningConnectors`. Otherwise the managed resource reports an error that names
the key.

These fields have two limits:

* Kubernetes allows only letters, digits, `-`, `_` and `.` in the keys of a Secret. A property
  with another character in its name cannot be set this way.
* After the provider restarts, it updates each identity provider with these secrets once more,
  with the current values of the Secrets. Upjet cannot carry these three fields into the state
  that it rebuilds after a restart, so the first comparison sees a difference. For identity
  providers, a Secret that changed while the provider was stopped therefore does reach the
  server, unlike the rule in [Secrets](#secrets).

## Which values the provider checks

The provider checks the rules that connect several fields before it creates or updates an
object. For example, it rejects a PushSender that sets both `fcm` and `amazonSns`, or a
ValidationRules resource with a regular expression for usernames. The error appears in the
condition `LastAsyncOperation`. These messages use the Terraform names of the fields, such as
`provider_name` for `providerName`.

The provider does not check single values, such as the allowed values of a field or the length
of a name. Crossplane's Upjet skips that part of the Terraform provider's validation, so the
server checks these values instead. The managed resource then shows the server's error message,
for example `HTTP 400 FM-10001: Invalid flow type.`

## Changes that take time

The server finishes some changes in the background. The provider waits for them before it
reports success:

* **User stores.** A background task deploys a new user store 10 to 20 seconds after the
  request.
* **User sharing.** If the server cannot find a role that a UserShare names, it skips the role
  without an error. The provider then stops waiting after two minutes and reports an error.
  Check the role names and audiences.
* **Tenants.** A new tenant needs a short time before its APIs answer.

The provider runs every create, update and delete in the background, with a limit of one hour.
While the provider creates or deletes an object, the managed resource shows `Ready=False` with
the reason `Creating` or `Deleting`. The condition `LastAsyncOperation` shows the result of the
last create, update or delete that finished.

## A restart during a create

The provider creates, updates and deletes objects in the background. If the provider stops while
it creates an object, for example during an upgrade, it may not have recorded the new object's
ID yet. After the restart, the provider tries to create the object again:

* If the object's name must be unique, the server rejects the second create with HTTP 409.
  Set the `crossplane.io/external-name` annotation to the ID of the object that exists.
  [Managed resources](resources.md) shows the ID format of each kind.
* If the name does not need to be unique, for example for an Organization, the server creates
  a second object. Delete the extra object in the Console.

Upgrade the provider when no managed resource shows `Ready=False` with the reason `Creating`
or `Deleting`.

## Run one provider replica

The server stores many tenant-wide settings in one record. Two writes at the same time can
overwrite each other. The provider therefore makes writes to tenant-wide settings wait for each
other, but only inside one process.

**What you should do:** Keep leader election on, which is the default. With leader election,
only one replica works at a time, also during an upgrade, when Kubernetes starts the new pod
before it stops the old one. Turn it off (`--no-leader-election`) only when you run the provider
outside a cluster. Manage each tenant with only one Crossplane installation.

## Two managed resources for the same object

Two managed resources that manage the same object overwrite each other's changes. This happens
easily with these kinds:

* every singleton, for example two ServerConfiguration resources in the same tenant,
* GroupMembers and RoleAssignment, which manage the *complete* member list of one group or role,
* ApplicationShare and UserShare, which manage the complete sharing of one application or user,
* every kind that takes its object over, for example two Flows with the same `flowType`.

**What you should do:** Create exactly one managed resource for each of these objects.

## Tenants

* Create tenants with a ProviderConfig that uses **basic authentication** in the super tenant.
  The server fails to create a tenant with a client credentials token (`TM-65002`).
* The server deletes tenants only when `deployment.toml` sets
  `[tenant_mgt] tenant_deletion = true`. Without this setting, deleting a Tenant only
  deactivates it. A deactivated tenant keeps its domain.
* A tenant that reuses the domain of a recently deleted tenant does not work for about 15
  minutes, because the server keeps cached data about the deleted tenant.
* The super tenant's email provider is set in `deployment.toml` (`[output_adapter.email]`).
  EmailSender does not work in the super tenant.

## Organizations

* To manage a sub-organization, the provider must sign in with an application's client
  credentials and the `organization_switch` grant. See [Organizations](organizations.md).
* Sub-organizations are created with the ProviderConfig of their parent organization.
* Email domain discovery (OrganizationDiscovery) works only after an
  OrganizationDiscoveryConfiguration sets `emailDomainEnabled: true`.

## Flows

WSO2 IS 7.3.0 has no access control rule for `DELETE /api/server/v1/flow`, so the server
answers HTTP 403. When you delete a Flow of the root organization, the provider turns the flow
off instead. The managed resource is then deleted, but the flow stays on the server, turned off. To allow deletion, add this rule to `deployment.toml` and
restart the server:

```toml
[[resource.access_control]]
context = "(.*)/api/server/v1/flow(.*)"
secure = true
http_method = "DELETE"
scopes = ["internal_flow_update"]
```

## Approval workflows

A WorkflowAssociation makes an operation, such as `ADD_USER`, wait for approval. While it
waits, the object does not exist yet, so the managed resource that performs the operation stays
not ready and retries. Do not attach workflows to operations that Crossplane performs.

## Settings that change how other resources behave

**Inbound provisioning user store.** While `provisioningUserstore` of the
InboundProvisioningConfiguration is set, the server puts *every* user that is created through
SCIM into that user store. The server ignores the User's `userStore` and refuses to update
users in other user stores. The provider notices when the server creates a user in the wrong
user store. It then deletes that user and reports why.

## Rules for names and values

* **Group names** must be 3 to 30 characters long and cannot contain spaces. This rule is the
  default of the user store.
* **Scope names** must be unique in the whole tenant, across all API resources.
* **The User Sharing API comes with its version 2.** When you authorize an application for the
  User Sharing API (`/o/api/server/v1/users`), the server also authorizes it for the User
  Sharing API V2 (`/o/api/server/v2/users`). An AuthorizedAPI for version 2 then fails with
  HTTP 409 `APP-60509`. Leave version 2 out, or take the authorization over with the external
  name `<applicationId>/<apiResourceId>`.
* **Role mappings of identity providers** must name local roles as `Internal/<role>`.
* **Regular expression rules** (ValidationRules `regex`) only work for passwords.
* **VCTemplate has no description.** The 7.3.0 API accepts a description when it creates a
  template, but the server does not store the description.

## Settings that only exist in deployment.toml

The server reads some settings only from its `deployment.toml` file. The provider cannot change
them. Examples are token persistence, the hashing of secrets, keystores, the super tenant's
email provider and access control rules.

## What the provider does not manage

* **Data sources.** The Terraform provider has twelve data sources, for example to look up a
  management API by its identifier. Crossplane has no data sources. To find an ID, call the
  server's API or read `status.atProvider` of a managed resource.
* **APIs without a resource.** Certificate revocation checks, tenant keystore certificates, the
  deprecated B2B self-service preferences API, guest invitations and the consent management v2
  API have no managed resource.
* **Runtime operations.** User sessions, credentials that users enrol, and workflow approvals
  belong in the Console and the APIs.
