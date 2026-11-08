# Troubleshooting

This page shows how to find out why a managed resource is not ready, and how to fix the most
common problems.

## Find the error message

1. Look at the managed resource's conditions and events.

   ```shell
   kubectl describe <kind>.<group> <name>
   ```

   The conditions tell you what state the resource is in:

   * `Synced=False` means that the last reconcile failed. The message says why.
   * `Ready=False` with the reason `Creating` means that the object does not exist on the
     server yet.
   * `LastAsyncOperation=False` means that the last create, update or delete failed. The
     provider runs these operations in the background, so their errors appear here.

2. Read the provider's log. Turn on debug logging first if the log says too little (see
   [More detail in the log](#more-detail-in-the-log)).

   ```shell
   kubectl logs -n crossplane-system deployment/$(kubectl get deployment -n crossplane-system -o name | grep provider-wso2is | cut -d/ -f2)
   ```

Error messages from the server contain an HTTP status and a WSO2 error code, for example
`HTTP 400 ORG-60080`. The code tells you which rule the server applied.

## Common problems

### "cannot resolve references"

A field that ends in `Ref` or `Selector` names a managed resource that does not have the value
yet, or that does not exist. For an ID, the value appears after the provider has created the
other object. Check the name and the namespace. Crossplane retries until the value is there.

### "invalid credentials of ProviderConfig ..."

The provider could not read the credentials. The rest of the message says what is wrong, for
example "the credentials contain unknown keys clientId" or "more than one sign-in method is set".
[Connect the provider](authentication.md) lists the keys and the rules.

### "a namespaced ProviderConfig supports only the Secret credentials source"

A namespaced ProviderConfig may only read a Secret in its own namespace. Use a
ClusterProviderConfig for the `Environment` and `Filesystem` sources.

### HTTP 401 or 403 from the server

The server rejected the sign-in or the request:

* Check the username and password, or the client ID and client secret, in the Secret.
* An M2M application needs authorization for every management API that the managed resources
  use. Add the API on the application's **API Authorization** tab in the Console.
* The organization APIs reject basic authentication. See [Organizations](organizations.md).

### "diff contains fields that require resource replacement"

You changed a setting that the server accepts only when it creates the object, or a setting that
is part of the object's ID. Crossplane never replaces objects on its own. Change the setting
back, or create a new managed resource and delete the old one. See
[Changes that need a new object](behaviour.md#changes-that-need-a-new-object).

### A password or secret is empty on the first try

The provider reads each `...SecretRef` when it reconciles. If the Secret does not exist yet, the
provider sends no value, and the server may reject the request, for example with "The minimum
length of password should be 8". Crossplane retries, and the next attempt uses the Secret.
Create Secrets before the managed resources that use them, or put them first in the same file.

### The provider or the server rejects a value

The provider checks the rules that connect several fields before it creates or updates an
object. Its messages use the Terraform names of the fields, for example `provider_name` for
`providerName`. The server checks single values and reports them with an HTTP status, for
example `HTTP 400 FM-10001: Invalid flow type.` Compare the value with the field's description
in `kubectl explain <kind>.spec.forProvider.<field>`. See
[Which values the provider checks](behaviour.md#which-values-the-provider-checks).

### The managed resource stays in "Creating" for minutes

Some objects take time on the server. A user store needs 10 to 20 seconds, and user sharing can
take up to two minutes. The condition `LastAsyncOperation` shows the result of the last
operation that finished. If an approval workflow
(WorkflowAssociation) applies to the operation, the operation waits for approval and never
finishes on its own.

### Two managed resources keep changing the same object

Two managed resources manage the same object, for example two ServerConfiguration resources in
one tenant, or two GroupMembers for one group. Delete one of them. See
[Two managed resources for the same object](behaviour.md#two-managed-resources-for-the-same-object).

### The object exists on the server, but the managed resource tries to create it again

For the kinds that Crossplane creates, the provider does not look for objects with the same name.
It creates a new object, and the server may answer HTTP 409 because the name is taken. This
happens when someone created the object outside Crossplane, or when the provider restarted
while it created the object (see [A restart during a create](behaviour.md#a-restart-during-a-create)).
To take over the existing object, set the `crossplane.io/external-name` annotation to the
object's ID. [Managed resources](resources.md) shows the ID format of each kind.

### Deleting a managed resource hangs

The provider could not delete the object on the server. `kubectl describe` shows the error.
Common causes are an organization that still has sub-organizations, an identity provider that
applications still use (set `forceDelete: true`), and an OrganizationDiscovery whose
OrganizationDiscoveryConfiguration was deleted first. Fix the cause on the server, or remove the
object by hand and then remove the managed resource's finalizer:

```shell
kubectl patch <kind>.<group> <name> --type merge -p '{"metadata":{"finalizers":[]}}'
```

Removing the finalizer leaves the object on the server if it still exists.

## More detail in the log

Create a DeploymentRuntimeConfig that turns debug logging on, and point the provider at it:

```yaml
apiVersion: pkg.crossplane.io/v1beta1
kind: DeploymentRuntimeConfig
metadata:
  name: debug
spec:
  deploymentTemplate:
    spec:
      selector: {}
      template:
        spec:
          containers:
            - name: package-runtime
              args:
                - --debug
```

```shell
kubectl patch provider.pkg/provider-wso2is --type merge -p '{"spec":{"runtimeConfigRef":{"name":"debug"}}}'
```

The debug log shows every reconcile and the full error of each failed request.
