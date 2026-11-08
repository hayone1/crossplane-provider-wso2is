# Getting started

In this tutorial, you use Crossplane to set up sign-in for a web application on WSO2 Identity
Server. You create:

* an application that signs users in with OpenID Connect,
* a user with a password from a Kubernetes Secret,
* a group with the user in it, and
* a role that the group has.

The tutorial takes about 20 minutes.

## What you need

* A Kubernetes cluster and `kubectl`. A local [kind](https://kind.sigs.k8s.io) cluster works.
* [Helm](https://helm.sh) 3.
* A WSO2 Identity Server 7.3 that the cluster can reach, and the username and password of an
  administrator.

For a test server on your own computer, use the official Docker image `wso2/wso2is:7.3.0`. Its
administrator is `admin` with the password `admin`, and it uses a self-signed certificate.
Step 2 shows how to start it so that a kind cluster can reach it.

## Step 1: Install Crossplane and the provider

1. Install Crossplane.

   ```shell
   helm repo add crossplane-stable https://charts.crossplane.io/stable
   helm repo update
   helm install crossplane crossplane-stable/crossplane \
     --namespace crossplane-system --create-namespace --wait
   ```

2. Install the provider. Replace the version with the latest release.

   ```shell
   kubectl apply -f - <<'EOF'
   apiVersion: pkg.crossplane.io/v1
   kind: Provider
   metadata:
     name: provider-wso2is
   spec:
     package: ghcr.io/hayone1/provider-wso2is:v0.1.0
   EOF
   ```

3. Wait until the provider is healthy. The first start takes a minute, because Crossplane
   installs more than 100 resource definitions.

   ```shell
   kubectl wait provider.pkg/provider-wso2is --for=condition=Healthy --timeout=5m
   ```

## Step 2: Connect the provider to the server

1. Create a Secret with the server's address and the administrator's credentials. Replace the
   three values with your own.

   ```shell
   kubectl create secret generic wso2is-creds -n crossplane-system \
     --from-literal=credentials='{"base_url": "https://id.example.com", "username": "admin", "password": "change-me"}'
   ```

   For a test server with a self-signed certificate, add `"insecure_skip_verify": true` to the
   JSON object.

   If WSO2 runs in Docker on the same computer as a kind cluster, start it in kind's Docker
   network: `docker run -d --name wso2is --network kind -p 9443:9443 wso2/wso2is:7.3.0`. The
   provider then reaches it at `https://wso2is:9443`.

2. Create the ClusterProviderConfig named `default`. Namespaced managed resources use it unless
   they name another one.

   ```shell
   kubectl apply -f - <<'EOF'
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
   EOF
   ```

## Step 3: Create the application

Save this manifest as `app.yaml`. It creates the application and writes its OAuth2 client ID
and client secret into the Secret `notes-oidc` in the same namespace.

```yaml
apiVersion: application.wso2is.m.crossplane.io/v1alpha1
kind: Application
metadata:
  name: notes
  namespace: default
spec:
  forProvider:
    name: Notes
    accessUrl: https://notes.example.com
    oidc:
      grantTypes:
        - authorization_code
        - refresh_token
      callbackUrls:
        - https://notes.example.com/callback
      pkce:
        mandatory: true
  writeConnectionSecretToRef:
    name: notes-oidc
```

Apply it, and wait until it is ready:

```shell
kubectl apply -f app.yaml
kubectl wait application.application.wso2is.m.crossplane.io/notes -n default --for=condition=Ready --timeout=2m
```

Your web application can now read its client ID from the Secret:

```shell
kubectl get secret notes-oidc -n default -o jsonpath='{.data.client_id}' | base64 -d
```

## Step 4: Create a user, a group and a role

Save this manifest as `people.yaml`. The Secret comes first, so that it exists when the
provider creates the user.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: dana-password
  namespace: default
stringData:
  password: Change-Me-Passw0rd!
---
apiVersion: identity.wso2is.m.crossplane.io/v1alpha1
kind: User
metadata:
  name: dana
  namespace: default
spec:
  forProvider:
    username: dana
    givenName: Dana
    familyName: Lee
    email: dana@example.com
    passwordSecretRef:
      name: dana-password
      key: password
---
apiVersion: identity.wso2is.m.crossplane.io/v1alpha1
kind: Group
metadata:
  name: editors
  namespace: default
spec:
  forProvider:
    name: editors
---
apiVersion: identity.wso2is.m.crossplane.io/v1alpha1
kind: GroupMembers
metadata:
  name: editors
  namespace: default
spec:
  forProvider:
    groupIdRef:
      name: editors
    userIdsRefs:
      - name: dana
---
apiVersion: identity.wso2is.m.crossplane.io/v1alpha1
kind: Role
metadata:
  name: editors
  namespace: default
spec:
  forProvider:
    name: editors
    permissions:
      - internal_user_mgt_view
---
apiVersion: identity.wso2is.m.crossplane.io/v1alpha1
kind: RoleAssignment
metadata:
  name: editors
  namespace: default
spec:
  forProvider:
    roleIdRef:
      name: editors
    groupIdsRefs:
      - name: editors
```

Apply it:

```shell
kubectl apply -f people.yaml
```

The fields that end in `Ref` name other managed resources. Crossplane fills in their IDs as
soon as the provider has created those objects. The GroupMembers and the RoleAssignment therefore
become ready a little after the user, the group and the role.

## Step 5: Check the result

1. List the managed resources. All of them should show `True` under `SYNCED` and `READY`.

   ```shell
   kubectl get managed -n default
   ```

2. Open the WSO2 Console (`https://<your server>/console`) and sign in as the administrator.
   Under **Applications**, you see "Notes". Under **User Management**, you see the user
   `dana`, the group `editors` with `dana` in it, and the role `editors`.

3. Sign in to My Account (`https://<your server>/myaccount`) as `dana` with the password from
   the Secret.

## Step 6: Change something

Edit `people.yaml`, add the permission `internal_user_mgt_list` to the role, and apply the file
again. Within a minute, the role in the Console shows both permissions.

Now change the role in the Console instead, for example by removing a permission. Crossplane
notices the difference at its next check, which happens every ten minutes by default, and puts
the permission back.

## Step 7: Clean up

Delete the managed resources. The provider deletes the objects on the server too.

```shell
kubectl delete -f people.yaml -f app.yaml
```

To remove the provider, delete the ClusterProviderConfig, the Secret and the provider package:

```shell
kubectl delete clusterproviderconfig.wso2is.m.crossplane.io/default
kubectl delete secret wso2is-creds -n crossplane-system
kubectl delete provider.pkg/provider-wso2is
```

## Next steps

* Read [Connect the provider](authentication.md) to use an M2M application instead of the
  administrator's password.
* Read [Known behaviour and limitations](behaviour.md) before you manage settings that already
  exist on a server.
* Browse [examples/](../examples) for a manifest of every kind.
