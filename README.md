<img src="extensions/icons/icon.svg" alt="" width="96" align="right">

# provider-wso2is

`provider-wso2is` is a [Crossplane](https://crossplane.io) provider for
[WSO2 Identity Server](https://wso2.com/identity-server/) 7.3. You describe applications, users,
roles, identity providers, organizations and server settings as Kubernetes objects, and
Crossplane keeps the server in line with them.

> This provider is an independent community project. It is not affiliated with, sponsored or
> endorsed by WSO2. WSO2 is a trademark of its owner, and this page uses it only to name the
> product that the provider works with.

[Upjet](https://github.com/crossplane/upjet) generated the provider from the
[WSO2 Identity Server Terraform provider](https://registry.terraform.io/providers/hayone1/wso2is/latest)
(version 0.1.2). The provider runs the Terraform provider inside its own process, so the package
needs no Terraform binary, and the resources behave like their Terraform counterparts.

## What you can manage

The provider has 52 managed resources in 17 API groups:

| Area | Kinds |
|------|-------|
| Applications | Application, AuthorizedAPI, ApplicationShare, APIResource |
| Users, groups and roles | User, Group, GroupMembers, Role, RoleAssignment, UserStore |
| Attributes | Claim, ClaimDialect, ExternalClaim, OIDCScope |
| Connections | IdentityProvider, CustomAuthenticator, IdentityVerificationProvider |
| Organizations (B2B) | Organization, OrganizationDiscovery, OrganizationDiscoveryConfiguration, UserShare |
| Tenants | Tenant |
| Notifications | EmailSender, SMSSender, PushSender, NotificationTemplate, NotificationTemplateType |
| Sign-in experience | BrandingPreference, BrandingText, Flow, Action, ScriptLibrary, ScriptSecret |
| Policies | GovernanceConnector, ValidationRules, Workflow, WorkflowAssociation |
| Events and credentials | Webhook, VCTemplate |
| Tenant-wide settings | ServerConfiguration, CORSConfiguration, DCRConfiguration, OAuth2Configuration, OAuth2IssuerUsageScope, SAMLIdPConfiguration, PassiveSTSConfiguration, ImpersonationConfiguration, InboundProvisioningConfiguration, JWTKeyValidatorConfiguration, FraudDetectionConfiguration, AdminAdvisoryBanner, RemoteLogging |

Every kind exists twice. The cluster-scoped variant uses the API groups
`*.wso2is.crossplane.io`. The namespaced variant, which Crossplane v2 introduced, uses
`*.wso2is.m.crossplane.io`. [Managed resources](docs/resources.md) lists every kind with its
group, its Terraform resource and its external name.

## What you need

* Kubernetes with Crossplane v2. The provider was tested with Crossplane 2.4.
* WSO2 Identity Server 7.3. Version 7.2 works for most kinds, and 7.1 is not supported.
* An administrator account or an M2M (machine-to-machine) application on the server.

## Install the provider

1. Install Crossplane, if your cluster does not have it yet.

   ```shell
   helm repo add crossplane-stable https://charts.crossplane.io/stable
   helm install crossplane crossplane-stable/crossplane --namespace crossplane-system --create-namespace
   ```

2. Install the provider package. Replace the version with the latest release.

   ```yaml
   apiVersion: pkg.crossplane.io/v1
   kind: Provider
   metadata:
     name: provider-wso2is
   spec:
     package: ghcr.io/hayone1/provider-wso2is:v0.1.0
   ```

3. Wait until the provider is healthy.

   ```shell
   kubectl wait provider.pkg/provider-wso2is --for=condition=Healthy --timeout=5m
   ```

## Connect the provider to your server

Put the server URL and the sign-in details into a Secret, and create a provider configuration
that points at it.

```shell
kubectl create secret generic wso2is-creds -n crossplane-system \
  --from-literal=credentials='{"base_url": "https://id.example.com", "username": "admin", "password": "change-me"}'
```

```yaml
# For namespaced managed resources. Every namespace can use it.
apiVersion: wso2is.m.crossplane.io/v1beta1
kind: ClusterProviderConfig
metadata:
  name: default
spec:
  credentials:
    source: Secret
    secretRef:
      namespace: crossplane-system
      name: wso2is-creds
      key: credentials
---
# For cluster-scoped managed resources.
apiVersion: wso2is.crossplane.io/v1beta1
kind: ProviderConfig
metadata:
  name: default
spec:
  credentials:
    source: Secret
    secretRef:
      namespace: crossplane-system
      name: wso2is-creds
      key: credentials
```

[Connect the provider](docs/authentication.md) explains the other sign-in methods, tenants,
organizations, private certificates and namespaced ProviderConfigs.

## Create your first resources

This manifest creates a group and a role, and gives the role to the group.

```yaml
apiVersion: identity.wso2is.m.crossplane.io/v1alpha1
kind: Group
metadata:
  name: support
  namespace: default
spec:
  forProvider:
    name: support
---
apiVersion: identity.wso2is.m.crossplane.io/v1alpha1
kind: Role
metadata:
  name: support
  namespace: default
spec:
  forProvider:
    name: support
    permissions:
      - internal_user_mgt_view
      - internal_user_mgt_list
---
apiVersion: identity.wso2is.m.crossplane.io/v1alpha1
kind: RoleAssignment
metadata:
  name: support
  namespace: default
spec:
  forProvider:
    roleIdRef:
      name: support
    groupIdsRefs:
      - name: support
```

Run `kubectl get managed -n default` to watch the resources become ready. The
[getting started guide](docs/getting-started.md) builds a complete application step by step.
[examples/](examples) has a cluster-scoped and a namespaced manifest for every kind.

## Documentation

* [Getting started](docs/getting-started.md): a tutorial that creates an application, a user and
  a role, and then cleans up.
* [Connect the provider](docs/authentication.md): every ProviderConfig setting, with examples.
* [Organizations (B2B)](docs/organizations.md): manage the objects inside sub-organizations.
* [Managed resources](docs/resources.md): every kind, its references, its secrets and its
  connection details.
* [Known behaviour and limitations](docs/behaviour.md): what the server and Crossplane do in
  special cases, and what you should do about it.
* [Troubleshooting](docs/troubleshooting.md): how to find out why a managed resource is not ready.
* [Development](contributing/development.md): how to build, test and release the provider.

## How the provider differs from the Terraform provider

* **Crossplane never replaces an object.** Terraform replaces an object when you change a
  setting that the server accepts only at creation. The provider reports an error instead.
* **Know which kinds take existing objects over.** As in Terraform, a GovernanceConnector, a
  Flow, a GroupMembers and similar kinds take over an object that already exists, because they
  manage something that always exists or belongs to another object. The settings kinds, such
  as ServerConfiguration and EmailSender, always manage the one setting of the tenant. The
  other kinds fail when the object exists. See
  [Known behaviour](docs/behaviour.md#how-a-managed-resource-finds-its-object).
* **Settings that you leave out stay under the server's control.** The provider does not copy
  server values into `spec.forProvider`. You can read them in `status.atProvider`.
* **Secrets come from Kubernetes Secrets.** The write-only arguments of the Terraform provider are
  not available, because `...SecretRef` fields already keep the values out of the managed
  resource.
* **Identity provider secrets use three top-level fields.** See
  [Known behaviour](docs/behaviour.md#secrets-of-identity-provider-authenticators).
* **There are no data sources.** Look IDs up with the server's API, or read them from
  `status.atProvider` of another managed resource.

## Development

The repository uses the `make`-based build of the Upjet provider template. The most common
commands are:

```shell
make submodules            # once, to fetch the build scripts
make generate              # regenerate the APIs, controllers and CRDs
go test ./config/... ./internal/...
test/e2e/run.sh            # test the examples against a real server (see the script)
make build                 # build the provider binary, the image and the package
```

[Development](contributing/development.md) has the details.

## License

The provider is licensed under the [Apache License 2.0](LICENSE), like the Terraform provider
that it uses.
