# Brief: build a Crossplane provider for WSO2 Identity Server

This brief gives you everything you need to build `crossplane-provider-wso2is`. It is a
Crossplane provider for WSO2 Identity Server, generated with Upjet from the Terraform provider
`hayone1/wso2is`. All facts were checked on 2026-11-07.

Read [HANDOFF.md](HANDOFF.md) too. It has the current state of the work, and it overrides this
brief when the two disagree.

## 1. Goal and definition of done

Build the provider in this folder (`/home/hayone/github/crossplane-provider-wso2is`). The
planned GitHub repository is `github.com/hayone1/crossplane-provider-wso2is`, and the Go module
path is the same. The provider exposes the Terraform provider's 52 resources as Crossplane
managed resources.

The work is done when all of these are true:

* Upjet generates cluster-scoped and namespaced APIs for all 52 resources. If you skip a
  resource, the README lists it with the reason.
* A `ProviderConfig` can sign in to WSO2 with each method that the Terraform provider supports.
* Every resource has an example manifest. `uptest` passes against a real WSO2 IS 7.3.0 for at
  least one resource in each API area: applications, identity, claims, identity providers,
  organizations, notifications and settings.
* The package builds as an OCI image (`xpkg`). The repository has CI, a README and an up-to-date
  `HANDOFF.md`.
* Nothing is published without the user's approval. That includes the GitHub repository, images,
  the Upbound Marketplace and releases.

## 2. How the user wants you to work

* **Ask before outward-facing actions.** These include creating repositories, pushing, publishing
  images, creating cloud servers (they cost money) and changing settings in the user's accounts.
* **Follow the user's other Crossplane provider.** `/home/hayone/github/crossplane-provider-hostinger`
  (https://github.com/hayone1/crossplane-provider-hostinger) is a finished Upjet v2.5.1 provider.
  Copy its structure, CI, release pipeline, security workflows, README layout and `HANDOFF.md`
  style. Read its `HANDOFF.md` first. The main difference is explained in section 4: the
  Hostinger Terraform provider uses the plugin SDK v2, and the WSO2 provider uses the plugin
  framework.
* **Build system.** The Hostinger project keeps the template's `make`-based build (the `build/`
  submodule), and `make` is installed in `~/.local/bin`. Do the same here. The user's Terraform
  projects use Task (taskfile.dev) instead of `make`. Ask the user whether they also want a
  `Taskfile.yml` with wrappers for the common commands.
* **Scratch files** go in a git-ignored `_debug/` folder.
* **Docs** must be clear and tutorial-like. Use ninth-grade English and complete sentences, and
  avoid hard-to-follow phrases where a pronoun hides the subject. The rules are in
  [contributing/docs-style.md](contributing/docs-style.md), and the `write-docs` skill applies
  them.
* **Keep `HANDOFF.md` up to date.** Read it at the start of each session, and update it at the
  end. Never write secret values into it. The `session-handoff` skill describes how.
* **Test against a real server.** Do not call something done until it works against WSO2.
* **Document every bug that you cannot fix.** Say what the server or the Terraform provider does,
  and what users should do about it.
* **Fix bugs in the right place.** If the cause is in the Terraform provider, fix it in
  `/home/hayone/github/terraform-provider-wso2is` with a test, and release a new version (ask
  first). Only add an override in this repository when the fix cannot go upstream. Follow that
  repository's `CLAUDE.md` when you work there.

## 3. The source: terraform-provider-wso2is

| Item | Value |
|------|-------|
| Local copy | `/home/hayone/github/terraform-provider-wso2is` |
| Repository | https://github.com/hayone1/terraform-provider-wso2is (public) |
| Go module | `github.com/hayone1/terraform-provider-wso2is` |
| Framework | terraform-plugin-framework v1.19.0, protocol 6, no SDK v2 code |
| Version to use | v0.1.1 or later (released 2026-11-07, on the Go proxy and the Terraform Registry). v0.1.1 adds the public `provider` package. |
| Registry address | `hayone1/wso2is` on registry.terraform.io (the OpenTofu registry listing was pending on 2026-11-07) |
| Server support | WSO2 IS 7.3.x. 7.2 works partly, and 7.1 is not supported. |
| Resources | 52, listed in `internal/provider/registry.go`. There are also 12 data sources, which Upjet ignores. |

Read these files in the Terraform repository before you design anything:

* `HANDOFF.md`: the project's current state.
* `templates/guides/server-behaviour.md.tmpl`: server behaviour that affects users.
* `contributing/wso2-api-behaviour.md`: every server quirk and how the provider handles it.
* `docs/resources/*.md`: the generated reference pages. Upjet's scraper reads them.
* `internal/provider/provider.go`: the provider configuration schema.

### What v0.1.1 changed for Upjet

* **Public package.** `github.com/hayone1/terraform-provider-wso2is/provider` has
  `New(version string) func() provider.Provider` and the constant `Address`.
* **Read works from the ID and the arguments.** Upjet does not call `ImportState`. It builds the
  prior state from the external name (as `id`) and `spec.forProvider`, and then calls `Read`.
  Every acceptance test in the Terraform repository now checks this (`acctest.Test` in
  `internal/acctest/upjet.go`). The check clears all computed attributes and calls
  `ReadResource` through the plugin protocol, as Upjet does. Five resources failed the check and
  were fixed: branding preference, branding text, notification template, secret and governance
  connector. Some optional arguments are not restored by `Read` when they are empty, and that is
  expected. Section 6.5 lists them.
* **Short-lived provider instances are cheap.** Upjet configures a new provider server for every
  reconcile. The provider now shares OAuth tokens (per server, tenant, organization, client and
  scopes) and HTTP connections (per TLS setting) across all instances in the process. A
  reconcile no longer requests a new token.
* **The provider instance never changes.** One `provider.New(version)()` value can serve every
  `ProviderConfig` at the same time, because `Configure` returns the API client in the response.
* **Upjet accepts the provider.** A smoke test on 2026-11-07 fetched v0.1.1 through the Go proxy,
  loaded it with `ujconfig.NewProvider(..., WithTerraformPluginFrameworkProvider(...))`, and ran
  Upjet's CRD type builder. All 52 resources were configured and matched to their framework
  resources, and all 104 type builds (both scopes) passed, before any custom configuration. The
  program and its inputs are in `_debug/upjet-smoke/` (git-ignored).

### Provider configuration (becomes the ProviderConfig credentials)

| Terraform attribute | Env var fallback | Notes |
|---------------------|------------------|-------|
| `base_url` | `WSO2IS_BASE_URL` | The server origin without a path, for example `https://id.example.com`. |
| `tenant_domain` | `WSO2IS_TENANT_DOMAIN` | The default is `carbon.super`. |
| `organization_id` | `WSO2IS_ORGANIZATION_ID` | A sub-organization. It needs client credentials with the `organization_switch` grant, or an access token for the organization. The server rejects basic authentication for organizations. |
| `username`, `password` | `WSO2IS_USERNAME`, `WSO2IS_PASSWORD` | Basic authentication. Creating tenants needs it. |
| `client_id`, `client_secret`, `scopes` | `WSO2IS_CLIENT_ID`, `WSO2IS_CLIENT_SECRET`, `WSO2IS_SCOPES` | Client credentials. The default scopes are `["SYSTEM"]`. |
| `access_token` | `WSO2IS_ACCESS_TOKEN` | The provider uses the token as it is and never refreshes it. |
| `insecure_skip_verify`, `ca_certificate` | `WSO2IS_INSECURE_SKIP_VERIFY`, `WSO2IS_CA_CERTIFICATE` | TLS settings. `ca_certificate` is PEM text or a file path. Pass PEM text from Crossplane. |
| `request_timeout`, `max_retries` | `WSO2IS_REQUEST_TIMEOUT`, `WSO2IS_MAX_RETRIES` | The defaults are 60 seconds and 4 retries. |

Only one group of credentials may be set. When the configuration sets any credential, the
provider ignores all credential environment variables. The provider pod must not have `WSO2IS_*`
environment variables, because they would apply to every `ProviderConfig` that sets no value.
Use one `ProviderConfig` for each tenant and each organization.

## 4. Approach: Upjet v2, in-process ("no-fork")

* Use **Upjet v2.5.1** (released 2026-09-30) with **Crossplane v2.4.x**. Upjet v2.5.1 needs
  **Go 1.26.8** or later. The Hostinger project pins `go 1.26.8` with `toolchain go1.26.10`.
* Start from https://github.com/crossplane/upjet-provider-template (branch `main`), the same
  template that the Hostinger project used. This folder already holds files, so do not clone into
  it. Ask the user whether to create the GitHub repository from the template now
  (`gh repo create hayone1/crossplane-provider-wso2is --template crossplane/upjet-provider-template`),
  or to copy the template's files into this folder and create the repository later. Keep
  `BRIEF.md`, `HANDOFF.md`, `CLAUDE.md`, `.claude/` and `contributing/` either way.
* Use the names that match the Hostinger project:

  | Setting | Value |
  |---------|-------|
  | `PROJECT_NAME` (Makefile, package name) | `provider-wso2is` |
  | Resource prefix and short name | `wso2is` |
  | Root groups | `wso2is.crossplane.io` (cluster) and `wso2is.m.crossplane.io` (namespaced) |
  | `TERRAFORM_PROVIDER_SOURCE` | `hayone1/wso2is` |
  | `TERRAFORM_PROVIDER_REPO` | `https://github.com/hayone1/terraform-provider-wso2is` |
  | `TERRAFORM_PROVIDER_VERSION` | `0.1.1` (keep it equal to the version in `go.mod`) |
  | `TERRAFORM_PROVIDER_DOWNLOAD_NAME` | `terraform-provider-wso2is` |
  | `TERRAFORM_DOCS_PATH` | `docs/resources` |
  | Image and package registry | `ghcr.io/hayone1`, mirrored to `xpkg.upbound.io/codejourney` (as for Hostinger) |

* Short groups by API area, for example `application`, `identity`, `claim`, `idp`,
  `organization`, `notification`, `branding`, `config`, `flow`, `workflow`, `tenant`,
  `governance`, `userstore` and `action`. Confirm the list with the user.

### How the framework provider is wired in

The Hostinger provider calls `ujconfig.WithTerraformProvider(<SDK provider>)` and
`WithTerraformPluginSDKIncludeList`. For the WSO2 provider, use the framework options instead:

```go
import wso2is "github.com/hayone1/terraform-provider-wso2is/provider"

ujconfig.WithIncludeList([]string{}),                      // no Terraform CLI resources
ujconfig.WithTerraformPluginFrameworkIncludeList(names),   // all 52, e.g. "wso2is_application$"
ujconfig.WithTerraformPluginFrameworkProvider(wso2is.New(version)()),
```

In `internal/clients` (the `TerraformSetupBuilder`), set `ps.FrameworkProvider` to the same
provider instance, and fill `ps.Configuration` with the provider attributes from the
`ProviderConfig` credentials secret (keys from the table above). Upjet calls
`ConfigureProvider` with that map for every reconcile.

Upjet derives the CRD types from `config/schema.json`. Check how the template handles code
generation for framework providers (the Hostinger project's `generationProvider` switch is
specific to SDK v2).

Upjet's documentation is in `docs/` of https://github.com/crossplane/upjet. Start with
`generating-a-provider.md`, `configuring-a-resource.md`, `upjet-v2-upgrade.md` and
`testing-with-uptest.md`.

## 5. Configuration work

### 5.1 Schema and metadata

* The template's `make generate` builds `config/schema.json` with `terraform providers schema
  -json` from the registry. It needs the version in the Makefile to be published.
* `config/provider-metadata.yaml` comes from Upjet's scraper. The scraper already works with the
  Terraform provider's docs, and it needs no prelude script (unlike Hostinger). It was tested on
  2026-11-07, and it read all 52 resources with descriptions, argument docs, examples (including
  references between resources) and import commands:

  ```shell
  go run github.com/crossplane/upjet/v2/cmd/scraper -n hayone1/wso2is --resource-prefix wso2is \
    -r <terraform-provider-wso2is>/docs/resources -o config/provider-metadata.yaml
  ```

### 5.2 External names

The external name is the Terraform `id`, which is not always the import ID that the Terraform
docs show. The table below gives the `id` format.

| Resource | Terraform `id` | Suggested external name config |
|----------|----------------|-------------------------------|
| `wso2is_application`, `wso2is_api_resource`, `wso2is_group`, `wso2is_role`, `wso2is_user`, `wso2is_identity_provider`, `wso2is_identity_verification_provider`, `wso2is_organization`, `wso2is_tenant`, `wso2is_vc_template`, `wso2is_webhook`, `wso2is_workflow`, `wso2is_workflow_association` | Server-generated ID (UUID) | `IdentifierFromProvider` |
| `wso2is_action` | The server-generated action ID. The import ID is `<type>/<id>`, but the `id` attribute holds only the action ID, and `type` is a required argument. | `IdentifierFromProvider` |
| `wso2is_governance_connector` | The connector ID (base64url of the connector name) | `IdentifierFromProvider`. `Read` looks up the category itself. |
| `wso2is_claim` | base64url of `claim_uri` | `IdentifierFromProvider`, or a `GetIDFn` that derives it from `claim_uri` |
| `wso2is_claim_dialect` | base64url of `dialect_uri` | as above, from `dialect_uri` |
| `wso2is_custom_authenticator` | base64url of `name` | as above, from `name` |
| `wso2is_userstore` | base64url of `name` (the domain) | as above, from `name` |
| `wso2is_external_claim` | `<dialect id>/<claim id>` (base64url parts) | `IdentifierFromProvider` |
| `wso2is_application_authorized_api` | `<application_id>/<api_resource_id>` | `TemplatedStringAsIdentifier` from both parameters |
| `wso2is_application_share`, `wso2is_group_members`, `wso2is_role_assignment`, `wso2is_user_share`, `wso2is_organization_discovery` | The same as the parent ID argument | `TemplatedStringAsIdentifier` with the parent parameter |
| `wso2is_branding_preference` | `<type>/<name>/<locale>` | `IdentifierFromProvider`. `Read` fills empty `type`, `application_id` and `locale` from the ID. |
| `wso2is_branding_text` | `<type>/<name>/<locale>/<screen>` | as above |
| `wso2is_notification_template` | `<channel>/<type>/<locale>[/<application_id>]` | `IdentifierFromProvider`. `Read` fills empty parts from the ID. |
| `wso2is_notification_template_type` | `<channel>/<name>` | `IdentifierFromProvider` |
| `wso2is_secret` | `<type>/<name>` | `IdentifierFromProvider`. `Read` fills an empty `type` from the ID. |
| `wso2is_oidc_scope`, `wso2is_script_library` | `name` | `ParameterAsIdentifier("name")` |
| `wso2is_flow`, `wso2is_remote_logging`, `wso2is_validation_rules` | The type argument (`flow_type`, `log_type`, `field`) | `ParameterAsIdentifier` of that argument |
| `wso2is_push_sender` | `<provider_name>_PushPublisher` | `IdentifierFromProvider` |
| Singletons: `wso2is_email_sender` (`EmailPublisher`), `wso2is_sms_sender` (`SMSPublisher`), every `*_configuration`, `wso2is_admin_advisory_banner` and `wso2is_oauth2_issuer_usage_scope` (the `id` is the type name without the `wso2is_` prefix, for example `server_configuration`) | A fixed string | A fixed external name: a custom `ExternalName` that always returns the constant |

### 5.3 Cross-resource references

Configure `r.References` so that users can write `...Ref` and `...Selector` fields:

| Resource | Field | Refers to |
|----------|-------|-----------|
| `application_authorized_api` | `application_id`, `api_resource_id` | Application, APIResource |
| `application_share` | `application_id`, `organization_ids` | Application, Organization |
| `branding_preference`, `branding_text`, `notification_template` | `application_id` | Application |
| `group_members` | `group_id`, `user_ids` | Group, User |
| `role_assignment` | `role_id`, `user_ids`, `group_ids` | Role, User, Group |
| `role` | `audience_id` (when `audience_type = "application"`) | Application |
| `user_share` | `user_id`, `organizations[].organization_id` | User, Organization |
| `organization_discovery` | `organization_id` | Organization |
| `workflow` | `steps[].role_ids`, `steps[].user_ids` | Role, User |
| `workflow_association` | `workflow_id` | Workflow |
| `external_claim` | `dialect_uri`, `mapped_local_claim_uri` | ClaimDialect (`dialect_uri`), Claim (`claim_uri`) |
| `oidc_scope` | `claims` | ExternalClaim (`claim_uri`) |
| `application` | `associated_roles.role_ids` | Role |

### 5.4 Secrets and write-only attributes

* Sensitive arguments become `...SecretRef` fields automatically:
  `action.authentication_properties`, `custom_authenticator.endpoint_authentication_properties`,
  `email_sender.password` and `secret_properties`, `identity_verification_provider.secret_properties`,
  `remote_logging.*_password`, `secret.value`, `sms_sender.authentication_properties` and
  `secret`, `tenant.owner_password`, `user.password`, `userstore.secret_properties` and
  `webhook.secret`.
* **Remove the write-only arguments** `user.password_wo` and `password_wo_version`, and
  `secret.value_wo` and `value_wo_version`, from the generated schema
  (`delete(r.TerraformResource.Schema, ...)` in the resource configuration). Upjet does not
  support write-only arguments, and the secret refs already keep values out of the spec.
* Publish `application.oidc.client_id` and `client_secret` as connection details, so that other
  workloads can read them from a Secret.

### 5.5 Behaviour to handle or document

* **Singletons.** The settings resources and `email_sender` and `sms_sender` manage settings
  that always exist. Deleting some of them only stops management, and deleting others restores
  the defaults. Each resource description says which. Document this for each kind, and warn that
  two managed resources for the same singleton overwrite each other.
* **Slow creates.** User stores take 10 to 20 seconds to deploy. User sharing is asynchronous
  and can take up to two minutes. New tenants need time before their APIs answer. The Terraform
  resources already wait, so make sure the Upjet operation timeouts allow for this.
* **One active replica.** The Terraform provider serializes writes to tenant-wide settings with
  an in-process lock (`client.LockTenantConfig`). Run the provider with leader election and one
  active replica, or concurrent reconciles of settings resources can overwrite each other.
* **Tenants** must be created with a basic-auth `ProviderConfig` of the super tenant.
* **Organizations.** Managing objects inside an organization needs a separate `ProviderConfig`
  with `organization_id` and the client credentials of an application that is shared with the
  organization. Write an example and a guide for this flow, based on
  `templates/guides/organizations.md.tmpl` in the Terraform repository.
* **JSON-string arguments.** `flow.steps` and `branding_preference.preference` are JSON
  documents that the provider compares by meaning. Check that Upjet does not report drift for
  formatting differences.
* **Secrets are not read back** by most APIs, so the provider cannot detect drift in secrets.
* **Approval workflows** that are attached to operations (for example `ADD_USER`) block creates.
  Document this.
* **Optional arguments that `Read` does not restore when they are empty.** After an import
  (management policy `Observe`), these stay empty until the user sets them or late
  initialization fills them: `identity_provider.force_delete` (a provider-only switch, read as
  `false`), and `application.associated_roles`, `authentication_sequence` and
  `claim_configuration`. The provider tracks those three only when the configuration sets them,
  so an import returns the server's current values. Check that this does not cause an update loop.

## 6. Test environment

The Hetzner lab of the Terraform repository runs **WSO2 IS 7.3.0 with PostgreSQL** at
`https://wso2is.lab.software-cloud.com`, with a trusted Let's Encrypt certificate. HANDOFF.md
says whether it is still running. It costs about EUR 42 a month while it runs.

* **Credentials:** `.env` in this folder (git-ignored) has `WSO2IS_BASE_URL`, `WSO2IS_USERNAME`
  and `WSO2IS_PASSWORD` for the lab's admin user. Never print or commit the values.
* **Recommended setup:** run Crossplane in a local kind cluster, and point the `ProviderConfig`
  at the lab's public URL. No TLS settings are needed, because the certificate is trusted.
* **Lab commands** live in the Terraform repository. Run them from there:
  `task lab:status`, `task lab:logs`, `task lab:up` and `task lab:destroy`. Ask the user before
  you create or destroy the lab.
* **Without the lab:** run `task wso2:up` in the Terraform repository. It starts WSO2 in Docker
  at `https://localhost:9443` (admin/admin, self-signed certificate, tenant and flow deletion
  enabled). Reach it from kind through the Docker network, with `insecure_skip_verify = true`.
* **Clean up:** name test objects with a `tfacc` part. Then `task sweep` in the Terraform
  repository can remove leftovers from the server.

Use `uptest` v2.2.0 for end-to-end tests of the generated examples, as the Hostinger project
does.

## 7. Packaging and release (needs the user's decisions)

* Copy the Hostinger release pipeline: automatic versions from Conventional Commit subjects,
  the package on `ghcr.io/hayone1` with a mirror on `xpkg.upbound.io/codejourney`, a Grype scan,
  cosign signatures, SLSA provenance, an SBOM and a GitHub release. Ask before you enable it.
* License: Apache 2.0, the same as the Terraform provider (which also has a NOTICE file).
* In `package/crossplane.yaml`, describe the provider as an independent community project. Name
  WSO2 only to say which product it works with, as the Hostinger package does for Hostinger.
* Pin the Terraform provider version in `go.mod` and the Makefile together.

## 8. Questions to ask the user at the start

1. Create the GitHub repository from the template now, or scaffold locally first?
2. Are the root groups `wso2is.crossplane.io` and `wso2is.m.crossplane.io` and the short group
   list right?
3. Should the repository get a `Taskfile.yml` with wrappers for the `make` targets?
4. When, and where, should the package be published?

## 9. References

* Terraform provider: https://github.com/hayone1/terraform-provider-wso2is
* The user's Hostinger Crossplane provider: https://github.com/hayone1/crossplane-provider-hostinger
* Upjet: https://github.com/crossplane/upjet (docs in `docs/`)
* Upjet provider template: https://github.com/crossplane/upjet-provider-template
* Crossplane: https://github.com/crossplane/crossplane and https://docs.crossplane.io/
* uptest: https://github.com/crossplane/uptest
* WSO2 IS docs: https://is.docs.wso2.com/en/latest/ (source: https://github.com/wso2/docs-is)
* WSO2 IS source and releases: https://github.com/wso2/product-is
* WSO2 REST API specs: https://github.com/wso2/identity-api-server
