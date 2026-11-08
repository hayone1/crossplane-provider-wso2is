#!/usr/bin/env bash
# End-to-end test of the provider against a real WSO2 Identity Server.
#
# The script creates a kind cluster, installs the CRDs, and runs the provider binary outside
# the cluster. For each manifest that you name, it applies the manifest, waits until every
# managed resource in it is Ready and Synced, checks that the provider does not update them
# again, deletes the manifest, and waits until the managed resources are gone. A deleted
# managed resource deletes its object on the server or resets it. Objects that a manifest
# takes over, such as settings, flows and templates, therefore end with their defaults, not
# with the values that they had before.
#
# Usage:
#   test/e2e/run.sh examples/cluster/identity/user.yaml examples/namespaced/claim/claim.yaml
#   test/e2e/run.sh            # every example in examples/cluster and examples/namespaced
#
# The server and the credentials come from the environment, or from .env in the repository:
#   WSO2IS_BASE_URL, WSO2IS_USERNAME, WSO2IS_PASSWORD, and optionally
#   WSO2IS_INSECURE_SKIP_VERIFY=true for a server with a self-signed certificate.
#
# Options (environment variables):
#   KEEP=1       keep the cluster and the provider running afterwards
#   CLUSTER=name use or create this kind cluster (default provider-wso2is-e2e)
#   TIMEOUT=5m   how long to wait for each manifest to become ready
#   POLL=20s     the provider's poll interval
#   IDLE=40s     how long to watch a ready manifest for update loops (0 turns the check off)
#   PROVIDER=... use this provider binary instead of building one
#   E2E_LOG=file copy the provider log to this file at the end
#
# Needs docker, kind, kubectl, jq, openssl and go.
set -euo pipefail

here=$(cd "$(dirname "$0")" && pwd)
root=$(cd "$here/../.." && pwd)
work=$(mktemp -d)
cluster=${CLUSTER:-provider-wso2is-e2e}
timeout=${TIMEOUT:-5m}
export KUBECONFIG="$work/kubeconfig"

if [[ -f "$root/.env" ]]; then
  set -a
  # shellcheck disable=SC1091
  . "$root/.env"
  set +a
fi
: "${WSO2IS_BASE_URL:?set WSO2IS_BASE_URL (or put it into .env)}"
: "${WSO2IS_USERNAME:?set WSO2IS_USERNAME}"
: "${WSO2IS_PASSWORD:?set WSO2IS_PASSWORD}"

log() { printf '\n\033[1m== %s\033[0m\n' "$*"; }
fail() {
  printf '\033[31mFAIL: %s\033[0m\n' "$*"
  echo "--- provider log (last 60 lines)"
  tail -60 "$work/provider.log" || true
  exit 1
}

cleanup() {
  local rc=$?
  if [[ -n "${E2E_LOG:-}" && -f "$work/provider.log" ]]; then
    cp "$work/provider.log" "$E2E_LOG"
  fi
  if [[ "${KEEP:-}" != 1 ]]; then
    # Delete whatever is left while the provider still runs, so that a failed run does not
    # leave objects on the server.
    if [[ -n "${provider_pid:-}" ]] && kill -0 "$provider_pid" 2>/dev/null; then
      kubectl delete managed --all -A --wait=true --timeout=300s >/dev/null 2>&1 ||
        echo "WARNING: some managed resources were not deleted; check the server for leftovers"
    fi
    [[ -n "${provider_pid:-}" ]] && kill "$provider_pid" 2>/dev/null || true
    kind delete cluster --name "$cluster" >/dev/null 2>&1 || true
    rm -rf "$work"
  else
    echo "kept: KUBECONFIG=$work/kubeconfig, provider log $work/provider.log, provider PID ${provider_pid:-none}"
  fi
  exit $rc
}
trap cleanup EXIT

if [[ $# -eq 0 ]]; then
  mapfile -t manifests < <(find "$root/examples/cluster" "$root/examples/namespaced" -name '*.yaml' \
    ! -path '*/providerconfig/*' ! -name 'install.yaml' | sort)
else
  manifests=("$@")
fi

log "building the provider"
provider=${PROVIDER:-}
if [[ -z "$provider" ]]; then
  provider="$work/provider"
  (cd "$root" && go build -o "$provider" ./cmd/provider)
fi

log "creating the kind cluster $cluster"
if ! kind get clusters 2>/dev/null | grep -qx "$cluster"; then
  kind create cluster --name "$cluster" --wait 180s >/dev/null
fi
kind get kubeconfig --name "$cluster" > "$KUBECONFIG"
kubectl apply --server-side -f "$root/package/crds" >/dev/null
kubectl wait crd --all --for=condition=Established --timeout=120s >/dev/null

log "starting the provider"
"$provider" --debug --no-leader-election --poll="${POLL:-20s}" > "$work/provider.log" 2>&1 &
provider_pid=$!

log "creating the provider configurations"
creds=$(jq -n --arg url "$WSO2IS_BASE_URL" --arg user "$WSO2IS_USERNAME" --arg pass "$WSO2IS_PASSWORD" \
  --arg insecure "${WSO2IS_INSECURE_SKIP_VERIFY:-false}" \
  '{base_url: $url, username: $user, password: $pass, insecure_skip_verify: ($insecure == "true")}')
kubectl create namespace crossplane-system --dry-run=client -o yaml | kubectl apply -f - >/dev/null
kubectl -n crossplane-system create secret generic wso2is-creds --from-literal=credentials="$creds" \
  --dry-run=client -o yaml | kubectl apply -f - >/dev/null
kubectl apply -f - >/dev/null <<EOF
apiVersion: wso2is.crossplane.io/v1beta1
kind: ProviderConfig
metadata:
  name: default
spec:
  credentials:
    source: Secret
    secretRef: {namespace: crossplane-system, name: wso2is-creds, key: credentials}
---
apiVersion: wso2is.m.crossplane.io/v1beta1
kind: ClusterProviderConfig
metadata:
  name: default
spec:
  credentials:
    source: Secret
    secretRef: {namespace: crossplane-system, name: wso2is-creds, key: credentials}
EOF

# managed_in prints "<resource>/<name> <namespace>" for each managed resource in a manifest.
managed_in() {
  kubectl apply --dry-run=client -f "$1" -o json |
    jq -r '(if .kind == "List" then .items[] else . end)
      | select(.apiVersion | test("wso2is(\\.m)?\\.crossplane\\.io/v1alpha1$"))
      | "\(.kind | ascii_downcase).\(.apiVersion | split("/")[0])/\(.metadata.name) \(.metadata.namespace // "")"'
}

# wait_ready waits until a managed resource is Ready and Synced.
wait_ready() {
  local res=$1 ns=$2 nsflag=()
  [[ -n "$ns" ]] && nsflag=(-n "$ns")
  if ! kubectl wait "$res" "${nsflag[@]}" --for=condition=Ready --timeout="$timeout" >/dev/null; then
    kubectl get "$res" "${nsflag[@]}" -o jsonpath='{.status.conditions}' | jq . || true
    fail "$res did not become Ready"
  fi
  kubectl wait "$res" "${nsflag[@]}" --for=condition=Synced --timeout=120s >/dev/null ||
    fail "$res did not become Synced"
}

# Manifests with the line "# e2e: tenant" manage objects inside a tenant. They use the
# ProviderConfig (or ClusterProviderConfig) named "tenant", which the script points at a
# test tenant with a random domain.
if grep -lq '^# e2e: tenant' "${manifests[@]}"; then
  log "creating the test tenant"
  domain="e2e$(date +%s).example"
  owner_password="E2e-Owner-$(openssl rand -hex 8)!"
  kubectl -n crossplane-system create secret generic e2e-tenant-owner --from-literal=password="$owner_password" \
    --dry-run=client -o yaml | kubectl apply -f - >/dev/null
  tenant_creds=$(jq -n --arg url "$WSO2IS_BASE_URL" --arg user "owner@$domain" --arg pass "$owner_password" \
    --arg domain "$domain" --arg insecure "${WSO2IS_INSECURE_SKIP_VERIFY:-false}" \
    '{base_url: $url, tenant_domain: $domain, username: $user, password: $pass, insecure_skip_verify: ($insecure == "true")}')
  kubectl -n crossplane-system create secret generic e2e-tenant-creds --from-literal=credentials="$tenant_creds" \
    --dry-run=client -o yaml | kubectl apply -f - >/dev/null
  kubectl apply -f - >/dev/null <<TENANT
apiVersion: tenant.wso2is.crossplane.io/v1alpha1
kind: Tenant
metadata:
  name: e2e-tenant
spec:
  forProvider:
    domain: $domain
    ownerUsername: owner
    ownerEmail: owner@$domain
    ownerFirstName: Test
    ownerLastName: Owner
    ownerPasswordSecretRef: {namespace: crossplane-system, name: e2e-tenant-owner, key: password}
---
apiVersion: wso2is.crossplane.io/v1beta1
kind: ProviderConfig
metadata:
  name: tenant
spec:
  credentials:
    source: Secret
    secretRef: {namespace: crossplane-system, name: e2e-tenant-creds, key: credentials}
---
apiVersion: wso2is.m.crossplane.io/v1beta1
kind: ClusterProviderConfig
metadata:
  name: tenant
spec:
  credentials:
    source: Secret
    secretRef: {namespace: crossplane-system, name: e2e-tenant-creds, key: credentials}
TENANT
  wait_ready tenant.tenant.wso2is.crossplane.io/e2e-tenant ""
  echo "ready: tenant $domain"
fi

# Manifests with the line "# e2e: organization" manage objects inside a sub-organization.
# They use the ProviderConfig (or ClusterProviderConfig) named "organization". The script
# builds the setup of docs/organizations.md with managed resources: an M2M application with
# the organization_switch grant, authorized for every organization API, an organization, and
# a share of the application with the organization. The ProviderConfig signs in with the
# application's connection secret key "credentials" and sets organizationID.
org_fixture=""
if grep -lq '^# e2e: organization' "${manifests[@]}"; then
  log "creating the test organization"
  org_fixture="$work/organization.yaml"
  # The credentials go to curl through a config file on a pipe, so that they never appear in
  # the process list.
  api() {
    local insecure=()
    [[ "${WSO2IS_INSECURE_SKIP_VERIFY:-false}" == true ]] && insecure=(-k)
    # curl reads backslash escapes in quoted config values, so quotes and backslashes in the
    # credentials are escaped first.
    local user pass
    user=$(printf '%s' "$WSO2IS_USERNAME" | sed 's/[\\"]/\\&/g')
    pass=$(printf '%s' "$WSO2IS_PASSWORD" | sed 's/[\\"]/\\&/g')
    curl -sf "${insecure[@]}" -K <(printf 'user = "%s:%s"\n' "$user" "$pass") "$WSO2IS_BASE_URL$1"
  }
  cat > "$org_fixture" <<ORG
apiVersion: application.wso2is.crossplane.io/v1alpha1
kind: Application
metadata:
  name: e2e-org-admin
spec:
  forProvider:
    name: e2e organization administration
    templateId: m2m-application
    oidc:
      grantTypes: [client_credentials, organization_switch]
  writeConnectionSecretToRef: {namespace: crossplane-system, name: e2e-org-client}
---
apiVersion: organization.wso2is.crossplane.io/v1alpha1
kind: Organization
metadata:
  name: e2e-org
spec:
  forProvider:
    name: e2e organization $(date +%s)
ORG
  n=0
  apis=$(api "/api/server/v1/api-resources?filter=type+eq+ORGANIZATION&limit=1000") ||
    fail "cannot list the organization APIs of the server"
  [[ $(jq '[.links[]? | select(.rel == "next")] | length' <<<"$apis") == 0 ]] ||
    fail "the server lists the organization APIs on more than one page"
  while read -r api_id; do
    scopes=$(api "/api/server/v1/api-resources/$api_id" | jq -c '[.scopes[].name]') ||
      fail "cannot read the scopes of the API resource $api_id"
    [[ "$scopes" == "[]" ]] && continue
    n=$((n + 1))
    cat >> "$org_fixture" <<ORG
---
apiVersion: application.wso2is.crossplane.io/v1alpha1
kind: AuthorizedAPI
metadata:
  name: e2e-org-admin-api-$n
  labels: {e2e.wso2is.crossplane.io/fixture: organization}
spec:
  forProvider:
    applicationIdRef: {name: e2e-org-admin}
    apiResourceId: $api_id
    scopes: $scopes
ORG
  # Authorizing the User Sharing API (/o/api/server/v1/users) also authorizes its version 2
  # (/o/api/server/v2/users), so a separate AuthorizedAPI for version 2 fails with HTTP 409.
  done < <(jq -r '.apiResources[] | select(.type == "ORGANIZATION" and .identifier != "/o/api/server/v2/users") | .id' <<<"$apis")
  [[ $n -gt 0 ]] || fail "the server lists no organization APIs"
  kubectl apply -f "$org_fixture" >/dev/null
  while read -r res ns; do
    [[ -n "$res" ]] && wait_ready "$res" "$ns"
  done < <(managed_in "$org_fixture")
  # Share the application only after it may call the organization APIs.
  kubectl apply -f - >/dev/null <<ORG
apiVersion: application.wso2is.crossplane.io/v1alpha1
kind: ApplicationShare
metadata:
  name: e2e-org-admin
spec:
  forProvider:
    applicationIdRef: {name: e2e-org-admin}
    organizationIdsRefs: [{name: e2e-org}]
ORG
  wait_ready applicationshare.application.wso2is.crossplane.io/e2e-org-admin ""
  org_id=$(kubectl get organization.organization.wso2is.crossplane.io/e2e-org -o jsonpath='{.status.atProvider.id}')
  # The application's connection secret holds only the client credentials. A server with a
  # self-signed certificate also needs insecure_skip_verify, so the script then builds a
  # Secret with both.
  org_secret=e2e-org-client
  if [[ "${WSO2IS_INSECURE_SKIP_VERIFY:-false}" == true ]]; then
    org_secret=e2e-org-creds
    kubectl -n crossplane-system get secret e2e-org-client -o jsonpath='{.data.credentials}' | base64 -d |
      jq -c '. + {insecure_skip_verify: true}' > "$work/org-creds.json"
    kubectl -n crossplane-system create secret generic e2e-org-creds --from-file=credentials="$work/org-creds.json" \
      --dry-run=client -o yaml | kubectl apply -f - >/dev/null
    rm -f "$work/org-creds.json"
  fi
  kubectl apply -f - >/dev/null <<ORG
apiVersion: wso2is.crossplane.io/v1beta1
kind: ProviderConfig
metadata:
  name: organization
spec:
  baseURL: $WSO2IS_BASE_URL
  organizationID: $org_id
  credentials:
    source: Secret
    secretRef: {namespace: crossplane-system, name: $org_secret, key: credentials}
---
apiVersion: wso2is.m.crossplane.io/v1beta1
kind: ClusterProviderConfig
metadata:
  name: organization
spec:
  baseURL: $WSO2IS_BASE_URL
  organizationID: $org_id
  credentials:
    source: Secret
    secretRef: {namespace: crossplane-system, name: $org_secret, key: credentials}
ORG
  echo "ready: organization $org_id with $n authorized APIs"
fi

passed=0
for m in "${manifests[@]}"; do
  log "testing $m"
  kubectl apply -f "$m" >/dev/null || fail "kubectl apply -f $m"
  while read -r res ns; do
    [[ -z "$res" ]] && continue
    wait_ready "$res" "$ns"
    echo "ready: $res"
  done < <(managed_in "$m")

  # A ready managed resource must stay as it is. After the next poll, the provider must not
  # have updated any of the manifest's resources again; otherwise an update loop exists.
  if [[ "${IDLE:-40s}" != 0 ]]; then
    lines=$(wc -l < "$work/provider.log")
    sleep "${IDLE:-40s}"
    while read -r res _; do
      name=${res##*/}
      if tail -n +"$lines" "$work/provider.log" | grep "Async update starting" | grep -q "\"resourceName\": \"$name\""; then
        fail "the provider updated $res again after it was ready (update loop)"
      fi
    done < <(managed_in "$m")
    echo "no update loop"
  fi

  kubectl delete -f "$m" --wait=true --timeout="$timeout" >/dev/null || fail "$m was not deleted in time"
  passed=$((passed + 1))
done

if [[ -n "$org_fixture" ]]; then
  log "deleting the test organization"
  kubectl delete providerconfig.wso2is.crossplane.io/organization clusterproviderconfig.wso2is.m.crossplane.io/organization \
    --wait=true --timeout=120s >/dev/null || true
  kubectl delete applicationshare.application.wso2is.crossplane.io/e2e-org-admin --wait=true --timeout="$timeout" >/dev/null ||
    fail "the test organization's share was not deleted in time"
  # The authorizations go before the application: the server answers requests for a
  # deleted application by ID for about 15 minutes, and deleting its authorizations fails
  # until then (fixed in terraform-provider-wso2is after v0.1.1).
  kubectl delete authorizedapi.application.wso2is.crossplane.io -l e2e.wso2is.crossplane.io/fixture=organization \
    --wait=true --timeout="$timeout" >/dev/null || fail "the test organization's API authorizations were not deleted in time"
  kubectl delete -f "$org_fixture" --ignore-not-found --wait=true --timeout="$timeout" >/dev/null ||
    fail "the test organization was not deleted in time"
fi

# The update-loop check reads the provider log. Make sure that the log has the format that
# the check expects, so that the check cannot pass by accident.
if [[ "${IDLE:-40s}" != 0 ]] && ! grep -q '"resourceName": "' "$work/provider.log"; then
  fail "the provider log has no \"resourceName\" fields, so the update-loop check did not work"
fi

if kubectl get tenant.tenant.wso2is.crossplane.io/e2e-tenant >/dev/null 2>&1; then
  log "deleting the test tenant"
  kubectl delete providerconfig.wso2is.crossplane.io/tenant clusterproviderconfig.wso2is.m.crossplane.io/tenant \
    --wait=true --timeout=120s >/dev/null || true
  kubectl delete tenant.tenant.wso2is.crossplane.io/e2e-tenant --wait=true --timeout="$timeout" >/dev/null ||
    fail "the test tenant was not deleted in time"
fi

log "passed: $passed manifest(s)"
