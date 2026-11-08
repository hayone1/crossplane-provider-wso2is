# Connect the provider to WSO2 Identity Server

Every managed resource names a **ProviderConfig**. The ProviderConfig tells the provider which
server to call, which tenant or organization to manage, and how to sign in. This guide explains
each setting and shows a complete example for each sign-in method.

## Where the settings go

A ProviderConfig has two parts:

* **A Kubernetes Secret** holds a JSON object with the sign-in details. The keys are the same as
  the arguments of the Terraform provider for WSO2 Identity Server.
* **The ProviderConfig's `spec`** points at the Secret. The spec can also set the server URL,
  the tenant and the organization. Put these three settings in the spec when several
  ProviderConfigs share one Secret, for example one ProviderConfig for each organization.

You can set `baseURL`, `tenantDomain` and `organizationID` in the spec or in the Secret. If you
set one of them in both places, the two values must be the same. Otherwise the provider reports
an error, so that you never wonder which value wins.

### Keys of the credentials JSON object

| Key | What it does |
|-----|--------------|
| `base_url` | The address of the server without a path, for example `https://id.example.com`. Every ProviderConfig needs this setting, in the Secret or as `spec.baseURL`. |
| `tenant_domain` | The tenant to manage. The default is `carbon.super`, the super tenant. |
| `organization_id` | The ID of a sub-organization to manage instead of the tenant's root organization. |
| `username`, `password` | Basic authentication with a user of the tenant. |
| `client_id`, `client_secret` | The OAuth2 client credentials of an application that may call the management APIs. |
| `scopes` | The scopes that the provider asks for with the client credentials. The default is `["SYSTEM"]`, which gives the application every scope that it is allowed to have. You can write a JSON list or a string with spaces between the scopes. |
| `access_token` | A ready-made access token. The provider uses it as it is and never renews it. |
| `insecure_skip_verify` | Set it to `true` to skip the check of the server's TLS certificate. Use it only for a test server with the default self-signed certificate. |
| `ca_certificate` | One or more PEM certificates of a private certificate authority (CA) that signed the server's certificate. Write the PEM text itself, not a file path. |
| `request_timeout` | The time limit for one HTTP request, in seconds. The default is `60`. |
| `max_retries` | How often the provider repeats a request that failed for a temporary reason. The default is `4`. |

Choose **one** sign-in method, and put only its keys into the Secret:

* `username` and `password`, or
* `client_id` and `client_secret`, or
* `access_token`.

The provider rejects a Secret with two methods, a method with a missing half (for example a
`username` without a `password`), and keys that it does not know. The last check catches typing
mistakes such as `clientId`.

## Example: basic authentication

Basic authentication is the simplest method. It is also the only method that can create tenants.

1. Create the Secret. Replace the values with your own.

   ```shell
   kubectl create secret generic wso2is-admin -n crossplane-system \
     --from-literal=credentials='{"base_url": "https://id.example.com", "username": "admin", "password": "change-me"}'
   ```

2. Create a ProviderConfig that points at the Secret.

   ```yaml
   apiVersion: wso2is.crossplane.io/v1beta1
   kind: ProviderConfig
   metadata:
     name: default
   spec:
     credentials:
       source: Secret
       secretRef:
         namespace: crossplane-system
         name: wso2is-admin
         key: credentials
   ```

3. Check that the ProviderConfig exists.

   ```shell
   kubectl get providerconfigs.wso2is.crossplane.io
   ```

   The provider reads the Secret when it reconciles a managed resource. A wrong Secret
   therefore shows up as an error on the first managed resource that uses the ProviderConfig.
   Run `kubectl describe` on that managed resource to see the message.

## Example: client credentials

Client credentials belong to an M2M (machine-to-machine) application. An administrator can turn
the application's access off at any time. Crossplane also never needs the password of a person.

1. In the WSO2 Console, create an M2M application. On its **API Authorization** tab, allow the
   management APIs that Crossplane needs, for example the SCIM2 Users API.
2. Copy the application's client ID and client secret into a Secret.

   ```shell
   kubectl create secret generic wso2is-m2m -n crossplane-system \
     --from-literal=credentials='{"base_url": "https://id.example.com", "client_id": "abc", "client_secret": "xyz"}'
   ```

3. Create a ProviderConfig that points at the Secret, as in the first example.

You can also create the M2M application with Crossplane itself. The `Application` managed
resource writes its `client_id` and `client_secret` into a connection Secret. See
[Organizations](organizations.md) for a complete example.

## Example: another tenant

To manage a tenant other than the super tenant, set `tenantDomain` in the spec, and use the
credentials of a user or an application of that tenant.

```yaml
apiVersion: wso2is.crossplane.io/v1beta1
kind: ProviderConfig
metadata:
  name: acme
spec:
  tenantDomain: acme.com
  credentials:
    source: Secret
    secretRef:
      namespace: crossplane-system
      name: wso2is-acme-admin
      key: credentials
```

## Example: a sub-organization

To manage the users, groups and roles inside a sub-organization, set `organizationID` in the
spec. The organization APIs reject basic authentication. Use the client credentials of an
application of the root organization that meets three conditions:

1. The application has the `client_credentials` and `organization_switch` grants.
2. The application may call the organization APIs that you need. Their identifiers start with
   `/o/`.
3. The application is shared with the organization.

```yaml
apiVersion: wso2is.crossplane.io/v1beta1
kind: ProviderConfig
metadata:
  name: partner
spec:
  organizationID: 2f2c0e6a-6a5b-4f53-9c6a-8a3f5d1e2b7c
  credentials:
    source: Secret
    secretRef:
      namespace: crossplane-system
      name: wso2is-b2b-admin
      key: credentials
```

[Organizations](organizations.md) walks you through the whole setup.

## A server with a private certificate

If a private certificate authority signed the server's certificate, put the authority's PEM
certificate into the Secret as `ca_certificate`. JSON needs `\n` for each line break, so it is
easiest to build the Secret with a small script:

```shell
jq -n --arg ca "$(cat ca.pem)" \
  '{base_url: "https://id.internal", username: "admin", password: "change-me", ca_certificate: $ca}' \
  > credentials.json
kubectl create secret generic wso2is-internal -n crossplane-system --from-file=credentials=credentials.json
rm credentials.json
```

## Namespaced managed resources

Crossplane v2 adds namespaced managed resources in the `*.wso2is.m.crossplane.io` groups. They
use two kinds of provider configuration:

* A **ProviderConfig** in the same namespace as the managed resource. It may only read a Secret
  in its own namespace. The provider ignores `secretRef.namespace` and always uses the managed
  resource's namespace. It also refuses the `Environment` and `Filesystem` credential sources,
  because those would let the users of one namespace read the provider's own environment and
  files.
* A **ClusterProviderConfig**, which a cluster administrator creates. Every namespace can use it,
  and it supports every credential source.

```yaml
apiVersion: wso2is.m.crossplane.io/v1beta1
kind: ProviderConfig
metadata:
  name: default
  namespace: team-a
spec:
  credentials:
    source: Secret
    secretRef:
      namespace: team-a
      name: wso2is-team-a
      key: credentials
```

A namespaced managed resource names its configuration with `spec.providerConfigRef`, which needs
both `kind` and `name`. Set `kind: ProviderConfig` for a ProviderConfig in the same namespace, or
`kind: ClusterProviderConfig` for a ClusterProviderConfig. If you leave `providerConfigRef` out,
the managed resource uses the ClusterProviderConfig named `default`.

## Environment variables have no effect

The Terraform provider falls back to `WSO2IS_*` environment variables when its configuration
leaves a setting empty. The Crossplane provider always passes every setting, so these variables
have no effect, even when a DeploymentRuntimeConfig sets them on the provider pod. The provider
logs a message at startup when it finds one. This rule keeps the settings of one ProviderConfig
from leaking into another.

## Rotate credentials

The provider reads the Secret again for every reconcile. To rotate a password or a client secret,
update the Secret. The next reconcile uses the new value. The provider caches access tokens per
server, tenant, organization, client and client secret, so a changed secret always gets a new
token.
