# Handoff

This file is the memory between work sessions. Read it at the start of a session, and update it
before you stop (see the `session-handoff` skill). Never put secret values here.

## Last updated

2026-11-09 (fourth session): v0.1.0 published (GHCR, Upbound Marketplace), uptest local only, lab destroyed. Earlier, 2026-11-08 (third session): the provider now uses terraform-provider-wso2is v0.1.2, which
contains the fixes found here, so the read guard is gone (only the configuration checks
remain). The repository is on GitHub with its first commit. The Upjet bug is reported upstream.

## State

* **Generated:** Upjet v2.5.1 generates all 52 resources in both scopes from
  terraform-provider-wso2is v0.1.2, with the plugin framework provider in-process.
  `make generate` passes (run it through `_debug/scripts/lowmem`, see Environment).
* **Hand-written configuration** (file map in `contributing/development.md`): groups and kinds;
  external names in three groups (25 kinds created by Crossplane, 12 kinds that take an existing
  object over, 15 singletons); references; late initialization off; identity provider secret
  workaround; Crossplane wording in CRD descriptions; credentials parsing; a wrapper
  (`internal/clients/configcheck.go`) that runs each resource's config checks before writes;
  leader election on by default.
* **Unit tests:** `go test ./config/... ./internal/...` passes (2026-11-08, with v0.1.2). They
  cover the configuration (every resource configured, examples and docs rows for every kind,
  derived IDs, defaults matching the Terraform schema), credentials parsing, ProviderConfig
  resolution (namespace and source rules), the configuration checks, that all 52 resources
  report a read without an ID as missing, and identity provider secrets through Upjet's secret
  code.
* **End to end against the lab with v0.1.2** (2026-11-08): `test/e2e/run.sh` passed all 106
  manifests (every example in both scopes, with the organization and tenant fixtures), with no
  update loops. The lab was clean afterwards (log: `_debug/e2e.log`).
* **GitHub:** CI, Security, Scorecard and End to End (WSO2 in Docker) pass on `main`.
* **End to end against the lab with v0.1.1** (`test/e2e/run.sh`: kind, CRDs, provider outside
  the cluster, update-loop check after a poll, delete) on 2026-11-07: all 52 cluster-scoped examples, all 53
  namespaced examples, the organization example in both scopes (fixture: M2M application with
  organization_switch, 47 authorized organization APIs, share, ProviderConfig with
  `organizationID`), and the tenant fixture (EmailSender) passed. The lab was clean afterwards.
* **Lint:** `golangci-lint run ./...` reports 0 issues (2026-11-08) (local golangci-lint 2.15.0, built with
  Go 1.27). `make lint` downloads v2.13.0, which cannot read Go 1.27 export data; run
  `GOTOOLCHAIN=go1.26.10 make lint` locally (CI installs Go 1.26 from go.mod).
* **uptest** (`make e2e` or `make uptest` against the server in `.env`): 7 examples with 11
  managed resources, one or more per API area (applications, identity, claims, identity
  providers, organizations, notifications, settings), passed apply, import and delete.
* **Package:** `make local-deploy` (with `PLATFORMS=linux_amd64` and a separate `KUBECONFIG`)
  builds the image and the xpkg, installs Crossplane 2.4.3 in the kind cluster `local-dev`, and
  the provider becomes Installed and Healthy. Leader election takes the lease
  `crossplane-system/crossplane-leader-election-provider-wso2is`.
* **Sign-in methods:** all three were verified against the lab: basic authentication (every
  run), client credentials with the organization switch (the organization fixture), and
  `access_token` (a token from a test M2M application created and deleted a Group).
* **Review:** a sub-agent reviewed code and docs in four rounds. All verified findings are fixed
  or documented (see "Done this session").

## Environment

* **WSO2 IS 7.3.0 lab: destroyed** on 2026-11-09 at the user's request (`task lab:destroy` in
  `../terraform-provider-wso2is`; Terraform state empty, DNS name gone). It was clean before.
  `.env` now points at the local Docker server (`task wso2:up` there, `https://localhost:9443`,
  admin/admin, `WSO2IS_INSECURE_SKIP_VERIFY=true`), for both `test/e2e/run.sh` and
  `UPTEST_CLOUD_CREDENTIALS`. `task lab:up -- -auto-approve` recreates the lab in about ten
  minutes, with a new admin password (ask the user first: it costs about EUR 42 a month).
* **Memory:** WSL has 6 GB (`.wslconfig`). Building the 104 generated packages with two
  parallel compile jobs crashed WSL twice. Run heavy Go commands through
  `_debug/scripts/lowmem` (one compile job, hard 3 GB cap through `systemd-run`; set
  `LOWMEM_MAX=3800M` for `make generate`, `make lint` or `make local-deploy`). `go build ./...`
  then takes about 3.5 minutes with a peak of 1.9 GB. Another agent may clear the Go build
  cache; a build that fails with "package ... is not in std" was hit by that and only needs a
  rerun.
* **kind:** no clusters are running (checked 2026-11-08). `test/e2e/run.sh` creates `provider-wso2is-e2e` and
  `make local-deploy` creates `local-dev`. Give `make local-deploy` its own `KUBECONFIG`, or it
  switches the user's current context.
* **Disk:** build tools live in `.cache/tools` (600 MB, kept). `_output/` and the provider image
  were removed at the end of the session; `make build` recreates them.
* Tools: Go 1.27.3 (go.mod pins `go 1.26.8`, `toolchain go1.26.10`), kind 0.33, kubectl 1.33,
  Helm 3.18, GNU Make 4.3, golangci-lint 2.15.0. `make` downloads uptest, chainsaw, the
  Crossplane CLI v2.2.1 and Terraform 1.5.7 into `.cache/tools`.

## Done this session

* **uptest runs locally only** (2026-11-09, at the user's request, because the lab will be
  destroyed). The `/test-examples` workflow (`e2e.yaml`) and the secret
  `UPTEST_CLOUD_CREDENTIALS` are gone. `make e2e` and `make uptest` load `.env`, where each
  developer keeps `UPTEST_CLOUD_CREDENTIALS` (the credentials JSON, see `example.env`), and stop
  early when it or `UPTEST_EXAMPLE_LIST` is missing. `make e2e UPTEST_EXAMPLE_LIST=examples/cluster/identity/group.yaml,examples/namespaced/identity/role.yaml`
  passed apply, import and delete against the lab (log: `_debug/uptest-local.log`). Do not test
  the cluster-scoped and namespaced example of one kind together: same server names, HTTP 409.
* **Marketplace listing is live** (2026-11-09):
  https://marketplace.upbound.io/providers/codejourney/provider-wso2is shows the icon
  (`extensions/icons/icon.svg`), the readme, release notes and the managed resources. The user
  turned on Publish for the repository in the Upbound console.
* Removed `install: true` from `docker/setup-buildx-action` in both repositories (this repo
  PR #5, Hostinger PR #12; `ci:` commits, so no release).
* `.env` now also holds `XPKG_MIRROR_ACCESS_ID` and `XPKG_MIRROR_TOKEN` (the Hostinger robot).

* **v0.1.0 is published** (2026-11-09): `ghcr.io/hayone1/provider-wso2is:v0.1.0` and
  `xpkg.upbound.io/codejourney/provider-wso2is:v0.1.0`, the same digest, signed in both
  registries, release https://github.com/hayone1/crossplane-provider-wso2is/releases/tag/v0.1.0.
  Both registries allow anonymous pulls. `RELEASE_ENABLED=true` is set, so every `feat` or
  `fix` commit on `main` now releases after CI passes. Checked: the package from Upbound
  installed in a kind cluster with Crossplane 2.4.3, became Healthy, and created and deleted a
  Group on the lab. Verify signatures with cosign v3, or add `--new-bundle-format` to cosign v2.
  The Upbound robot (secrets `XPKG_MIRROR_*`) is the Hostinger one; the user created the
  repository `codejourney/provider-wso2is` in the console and gave the robot's team access.

* **terraform-provider-wso2is v0.1.2 is released** (2026-11-08) with the four fixes from the last
  session (reads without an ID, User Sharing API v2 note, deleted applications that the server
  still caches, member lists of a group or role that just disappeared). On the Terraform
  Registry and the Go proxy.
* Bumped `go.mod` and `TERRAFORM_PROVIDER_VERSION` to 0.1.2 and ran `make generate`. Before you
  run `make generate` after a version bump, delete `.work/terraform` and `.work/hayone1`: the old
  Terraform lock file pins the previous version, and `pull-docs` reuses the old docs checkout.
* Removed the read guard's ID check. `internal/clients/readguard.go` is now
  `configcheck.go` (`WithConfigChecks`) and only runs the configuration checks. The test that
  reads all 52 resources without an ID stays and now tests the Terraform provider's own fix.
* `docs/behaviour.md` ("Delete objects in the right order") and `docs/organizations.md` no
  longer ask users to delete authorizations, shares and member lists first. The Usage example
  now uses OrganizationDiscovery.
* Reported the Upjet bug: https://github.com/crossplane/upjet/issues/810 (with a failing test
  and a tested fix; the reproduction is in `_debug/upjet-src`). Linked from `config/idp.go` and
  `docs/behaviour.md`.
* Created the public repository https://github.com/hayone1/crossplane-provider-wso2is and
  pushed the first commit. `RELEASE_ENABLED` is not set, so nothing is published. Settings
  copied from the Hostinger repository: delete branches after merge, auto-merge, no merge
  commits, Dependabot alerts and security updates (needed for the dependency-review job).
* gitleaks flagged two placeholders in `docs/organizations.md` (`admin:change-me` in a curl
  command and an example API ID). The docs now use `$WSO2IS_USERNAME:$WSO2IS_PASSWORD` and
  `ID-OF-THE-USERS-API`, and the first commit was amended and force-pushed. The force push
  closed Dependabot's first two pull requests; it opens them again on its next weekly run.
* crossplane-provider-hostinger: fixed `.github/renovate.json5` (a rename had turned
  `datasourceTemplate` and similar keys into `datasourceHostinger`, so Renovate's custom managers
  did nothing). Merged as PR #11. Renovate has never run on that repository, so the Renovate
  GitHub App is probably not installed there.

## Open items

1. Nothing open for publishing.
2. When Upjet fixes issue 810, consider replacing the identity provider workaround in
   `config/idp.go`. That changes the CRD fields, so it needs a new API version or a breaking
   change note.
3. Install the Renovate GitHub App for the repositories if the user wants Renovate (account
   setting, ask first). Both repositories also have Dependabot.

## Known problems

* Upjet sends an empty value when a referenced Secret does not exist yet. The server usually
  rejects it, and Crossplane retries. Examples put Secrets first. `docs/troubleshooting.md`.
* After a provider restart, Upjet rebuilds the prior state from the current Secrets, so a Secret
  that changed while the provider was stopped does not reach the server (identity providers are
  the exception). `docs/behaviour.md`.
* Upjet runs creates in the background. A provider restart during a create can cause HTTP 409
  or a duplicate object. `docs/behaviour.md`, "A restart during a create".
* Upjet never calls `ValidateResourceConfig`, so attribute validators (allowed values, lengths)
  do not run. The configuration check wrapper runs each resource's `ValidateConfig` and
  `ConfigValidators` before writes; the server checks the rest.
* Server behaviour that the Terraform provider cannot fix is in
  `../terraform-provider-wso2is/contributing/wso2-api-behaviour.md` and, for users, in
  `docs/behaviour.md`.
