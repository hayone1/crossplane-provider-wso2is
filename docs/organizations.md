# Organizations (B2B)

Organizations let one tenant serve many business customers. Each **organization** has its own
users, groups, roles and sub-organizations. The users of an organization sign in to the
applications that you share with that organization.

In this guide, you build a complete B2B setup with Crossplane. You create:

* an organization called "Partner Inc",
* an application that Crossplane uses to make changes inside the organization,
* a ProviderConfig for the organization, and
* a user, a group and a role inside the organization, with the user in the group and the role
  given to the group.

## What you need

* A cluster with Crossplane and this provider. [Getting started](getting-started.md) shows how
  to install both.
* A ProviderConfig named `default` that manages the root organization with basic
  authentication. See [Connect the provider](authentication.md).
* `kubectl`, `curl` and `jq`.

## How the provider reaches into an organization

A ProviderConfig normally manages the root organization of a tenant. To manage objects inside an
organization, you create a second ProviderConfig that sets `organizationID`.

This second ProviderConfig cannot use a username and password, because the organization APIs
reject basic authentication. Instead, it signs in with an application of the root organization.
The application must meet three conditions:

1. The application has the `client_credentials` and `organization_switch` grants.
2. The application may call the organization APIs that you need. The identifiers of these APIs
   start with `/o/`.
3. The application is shared with the organization.

The provider first gets a token for the root organization. Then it trades that token for an
organization token with the `organization_switch` grant. Finally, it calls the organization APIs
with the organization token.

## Step 1: Create the organization

Save this manifest as `organization.yaml`, and apply it with `kubectl apply -f organization.yaml`.

```yaml
apiVersion: organization.wso2is.crossplane.io/v1alpha1
kind: Organization
metadata:
  name: partner
spec:
  forProvider:
    name: Partner Inc
    description: Reseller in the EU
```

Wait until the organization is ready, and note its ID:

```shell
kubectl wait organization.organization.wso2is.crossplane.io/partner --for=condition=Ready --timeout=2m
kubectl get organization.organization.wso2is.crossplane.io/partner -o jsonpath='{.status.atProvider.id}'
```

## Step 2: Create the administration application

The application gets the two grants and writes its client credentials into a Secret.

```yaml
apiVersion: application.wso2is.crossplane.io/v1alpha1
kind: Application
metadata:
  name: b2b-admin
spec:
  forProvider:
    name: Organization administration
    oidc:
      grantTypes:
        - client_credentials
        - organization_switch
  writeConnectionSecretToRef:
    namespace: crossplane-system
    name: b2b-admin-client
```

The Secret `b2b-admin-client` gets three keys: `client_id`, `client_secret`, and
`credentials`. The `credentials` key holds both values as a JSON object, in the format that a
ProviderConfig reads.

## Step 3: Allow the application to call the organization APIs

The management APIs already exist on the server, but their IDs differ from server to server.
Look up the ID of each API that you need. This example uses the users, groups and roles APIs.

```shell
for api in /o/scim2/Users /o/scim2/Groups /o/scim2/Roles; do
  curl -s -u admin:change-me \
    "https://id.example.com/api/server/v1/api-resources?filter=identifier+eq+$api" |
    jq -r '.apiResources[] | "\(.identifier) \(.id)"'
done
```

Create one AuthorizedAPI for each of the three APIs. The example below shows the one for the
users API. Create two more in the same way, named `b2b-admin-org-groups` for `/o/scim2/Groups`
and `b2b-admin-org-roles` for `/o/scim2/Roles`, with their own IDs and scopes. Replace the `apiResourceId` with the ID that you found, and list the
scopes that the application may use. The scope names of an API are in the response of
`GET /api/server/v1/api-resources/<id>`. To authorize every organization API at once, list them
with the filter `type eq ORGANIZATION` instead. Leave out `/o/api/server/v2/users` from that
list: the server authorizes it together with `/o/api/server/v1/users`, and a separate
AuthorizedAPI for it fails with HTTP 409 (see [Rules for names and values](behaviour.md#rules-for-names-and-values)).

```yaml
apiVersion: application.wso2is.crossplane.io/v1alpha1
kind: AuthorizedAPI
metadata:
  name: b2b-admin-org-users
spec:
  forProvider:
    applicationIdRef:
      name: b2b-admin
    apiResourceId: 0f4b7c66-3d37-4b6f-9b1f-6c2b1a3d9e01
    scopes:
      - internal_org_user_mgt_view
      - internal_org_user_mgt_list
      - internal_org_user_mgt_create
      - internal_org_user_mgt_update
      - internal_org_user_mgt_delete
```

## Step 4: Share the application with the organization

```yaml
apiVersion: application.wso2is.crossplane.io/v1alpha1
kind: ApplicationShare
metadata:
  name: b2b-admin
spec:
  forProvider:
    applicationIdRef:
      name: b2b-admin
    organizationIdsRefs:
      - name: partner
```

Share the application only after the AuthorizedAPI resources are ready. A token from an
application that was shared too early may lack the new scopes until the server issues a new
token.

## Step 5: Create the ProviderConfig for the organization

Set `organizationID` to the ID from Step 1. The ProviderConfig reads the `credentials` key of
the application's Secret, and it sets the server URL in its spec.

```yaml
apiVersion: wso2is.crossplane.io/v1beta1
kind: ProviderConfig
metadata:
  name: partner
spec:
  baseURL: https://id.example.com
  organizationID: 2f2c0e6a-6a5b-4f53-9c6a-8a3f5d1e2b7c
  credentials:
    source: Secret
    secretRef:
      namespace: crossplane-system
      name: b2b-admin-client
      key: credentials
```

## Step 6: Create objects inside the organization

Every managed resource below names the `partner` ProviderConfig, so the provider creates it
inside the organization.

```yaml
apiVersion: v1
kind: Secret
metadata:
  namespace: crossplane-system
  name: partner-pat-password
stringData:
  password: Change-Me-Passw0rd!
---
apiVersion: identity.wso2is.crossplane.io/v1alpha1
kind: User
metadata:
  name: partner-pat
spec:
  providerConfigRef:
    name: partner
  forProvider:
    username: pat
    email: pat@partner.example
    passwordSecretRef:
      namespace: crossplane-system
      name: partner-pat-password
      key: password
---
apiVersion: identity.wso2is.crossplane.io/v1alpha1
kind: GroupMembers
metadata:
  name: partner-admins
spec:
  providerConfigRef:
    name: partner
  forProvider:
    groupIdRef:
      name: partner-admins
    userIdsRefs:
      - name: partner-pat
---
apiVersion: identity.wso2is.crossplane.io/v1alpha1
kind: Group
metadata:
  name: partner-admins
spec:
  providerConfigRef:
    name: partner
  forProvider:
    name: administrators
---
apiVersion: identity.wso2is.crossplane.io/v1alpha1
kind: Role
metadata:
  name: partner-user-managers
spec:
  providerConfigRef:
    name: partner
  forProvider:
    name: user-managers
    permissions:
      - internal_org_user_mgt_view
      - internal_org_user_mgt_list
---
apiVersion: identity.wso2is.crossplane.io/v1alpha1
kind: RoleAssignment
metadata:
  name: partner-user-managers
spec:
  providerConfigRef:
    name: partner
  forProvider:
    roleIdRef:
      name: partner-user-managers
    groupIdsRefs:
      - name: partner-admins
```

A role inside an organization uses the organization's own permission names, such as
`internal_org_user_mgt_view`. Leave out `audienceType` to create such a role.

## Check the result

1. Run `kubectl get managed`. Every resource should show `READY` and `SYNCED` as `True`.
2. In the WSO2 Console, switch to the "Partner Inc" organization. The user `pat`, the group
   `administrators` with `pat` in it, and the role `user-managers` are there.

## Clean up

Delete the objects inside the organization first, and the organization last. The server deletes
an organization only after its sub-organizations and its own objects are gone.

```shell
kubectl delete roleassignment.identity.wso2is.crossplane.io/partner-user-managers \
  role.identity.wso2is.crossplane.io/partner-user-managers \
  groupmembers.identity.wso2is.crossplane.io/partner-admins \
  group.identity.wso2is.crossplane.io/partner-admins \
  user.identity.wso2is.crossplane.io/partner-pat
kubectl delete providerconfig.wso2is.crossplane.io/partner
kubectl delete applicationshare.application.wso2is.crossplane.io/b2b-admin \
  authorizedapi.application.wso2is.crossplane.io/b2b-admin-org-users \
  authorizedapi.application.wso2is.crossplane.io/b2b-admin-org-groups \
  authorizedapi.application.wso2is.crossplane.io/b2b-admin-org-roles
kubectl delete application.application.wso2is.crossplane.io/b2b-admin
kubectl delete organization.organization.wso2is.crossplane.io/partner
```

Crossplane does not delete a ProviderConfig while managed resources still use it.

## More things you can do

* **Share an application with every organization.** Set `shareWithAllChildren: true` on an
  ApplicationShare instead of listing organizations. This choice also covers organizations
  that you create later.
* **Send users to their organization by email domain.** Turn on email domain discovery with an
  OrganizationDiscoveryConfiguration (`emailDomainEnabled: true`). Then list each
  organization's domains in an OrganizationDiscovery. Delete the OrganizationDiscovery
  resources before the configuration, so that the server accepts the removal of the domains.
* **Give a root user access to organizations.** A UserShare gives a user of the root
  organization access to organizations, with roles there.
* **Create a sub-organization of an organization.** Create the Organization with the parent's
  ProviderConfig. The server rejects the request from the root organization with `ORG-60021`.

## Limitations

* **The ProviderConfig needs the organization ID as text.** A ProviderConfig cannot refer to an
  Organization by name. Copy the ID from `status.atProvider.id`, or let a Composition patch it
  into the ProviderConfig.
* **An organization with sub-organizations cannot be deleted.** Delete the sub-organizations
  first.
