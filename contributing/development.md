# Development

This page explains how the provider is built, how to change it, and how to test a change
against a real server.

## How the provider works

[Upjet](https://github.com/crossplane/upjet) generates the managed resources from the schema of
the Terraform provider `hayone1/wso2is`. At run time, the provider runs that Terraform provider
inside its own process ("no-fork" mode) through the Terraform plugin framework protocol. It does
not start Terraform or a plugin process.

These parts are written by hand. Everything else is generated, and its files start with `zz_`.

| Path | What it does |
|------|--------------|
| `config/provider.go` | Builds the Upjet configuration for both API variants. |
| `config/resources.go` | Sets the group and kind of every resource, the references, and the connection details of applications. |
| `config/external_name.go` | Says how each resource finds its object: a server-assigned ID, an ID computed from the spec, or a fixed ID. It also turns late initialization off. |
| `config/idp.go` | Works around an Upjet bug with secret maps inside the identity provider's maps of objects. |
| `config/docs.go` | Rewrites the descriptions from the Terraform docs into Crossplane terms. |
| `internal/clients/wso2is.go` | Turns a ProviderConfig and its Secret into the Terraform provider configuration. |
| `internal/clients/configcheck.go` | Runs each resource's configuration checks before a create or an update, which Upjet does not do on its own. |
| `internal/apis/scheme.go` | Lets the generated reference resolvers find their target types without import cycles. |
| `apis/*/v1beta1/types.go` | The ProviderConfig, ClusterProviderConfig and ProviderConfigUsage types. |
| `cmd/provider/main.go` | Starts the controllers. |
| `examples/cluster/` | One example for every kind. `hack/namespaced-examples.py` writes `examples/namespaced/` from it. |
| `test/e2e/run.sh` | Tests examples against a real server. |

## Tools

You need Go (the version in `go.mod`), GNU Make, Docker, kind, kubectl, jq and Python 3. Fetch
the build scripts once:

```shell
make submodules
```

## Regenerate the provider

Run `make generate` after you change anything in `config/`, or after you update the Terraform
provider. The command does these steps:

1. It downloads Terraform 1.5.7 and writes the provider schema to `config/schema.json`.
2. It clones the Terraform provider's docs and scrapes them into
   `config/provider-metadata.yaml`.
3. It runs the Upjet generator, controller-gen and angryjet, and rewrites the reference
   resolvers with Upjet's resolver tool.

```shell
make generate
```

Generating and compiling 104 resources needs a lot of memory. On a machine with less than
8 GB, limit Go to one compile job: `GOFLAGS=-p=1 make generate`.

## Update the Terraform provider

1. Change the version of `github.com/hayone1/terraform-provider-wso2is` in `go.mod`, and
   `TERRAFORM_PROVIDER_VERSION` in the `Makefile`, to the same release.
2. Run `go mod tidy` and `make generate`.
3. Check the release notes for new resources. Add each new resource to the tables in
   `config/resources.go` and `config/external_name.go`, add an example in `examples/cluster/`,
   run `python3 hack/namespaced-examples.py`, and add a row to `docs/resources.md`. The unit
   tests fail until every resource has all of them.
4. Run the tests below.

## Run the tests

* **Unit tests** check the configuration, the credentials parser and the configuration checks.

  ```shell
  go test ./config/... ./internal/...
  ```

* **Linting** uses golangci-lint with `.golangci.yml`.

  ```shell
  make lint
  ```

  `make lint` downloads the golangci-lint version that CI uses. That build cannot read packages
  that a newer Go compiled, and it then fails with "export data version 5 is greater than
  maximum supported version". Run it with the Go version of `go.mod` instead:
  `GOTOOLCHAIN=go1.26.10 make lint`.

* **End-to-end tests** apply examples to a real server, wait until every managed resource is
  ready, watch for update loops, and delete the examples again. The script creates a kind
  cluster and runs the provider outside the cluster. It reads the server and the credentials
  from `.env` or from the environment.

  ```shell
  cp example.env .env     # then fill in your server
  test/e2e/run.sh examples/cluster/identity/user.yaml examples/namespaced/claim/claim.yaml
  test/e2e/run.sh         # every example in both API variants
  ```

  For a server in Docker on your computer, run `docker run -p 9443:9443 wso2/wso2is:7.3.0`,
  and set `WSO2IS_BASE_URL=https://localhost:9443`, `WSO2IS_USERNAME=admin`,
  `WSO2IS_PASSWORD=admin` and `WSO2IS_INSECURE_SKIP_VERIFY=true`.

  An example with the line `# e2e: tenant` manages objects inside a tenant. The script then
  creates a test tenant first and points the ProviderConfig `tenant` at it.

* **The package** builds with `make build`. `make local-deploy` installs Crossplane and the
  locally built package into a kind cluster and waits until the provider is healthy.

* **uptest** tests the packaged provider inside the cluster. For each example, it checks
  apply, import and delete against your own server. CI does not run it, because it needs a
  real server. Put `UPTEST_CLOUD_CREDENTIALS` into `.env` first. It holds the credentials JSON
  object for your server (see `example.env`). Then run:

  ```shell
  make e2e UPTEST_EXAMPLE_LIST=examples/cluster/identity/group.yaml,examples/namespaced/identity/role.yaml
  ```

  `make e2e` builds the package and deploys it to kind first. If the provider is already
  deployed, run `make uptest` with the same variable. Do not test the cluster-scoped and the
  namespaced example of the same kind in one run. They create objects with the same names on
  the server, so the second one fails with HTTP 409.

* **Running the provider outside a cluster.** Leader election is on by default, and it needs a
  namespace that only exists inside a cluster. `make run` and `test/e2e/run.sh` therefore pass
  `--no-leader-election`. Do the same when you start the binary yourself.

## Change an example

Edit the file in `examples/cluster/`, and then write the namespaced copy:

```shell
python3 hack/namespaced-examples.py
```

Test both copies with `test/e2e/run.sh`.

## Continuous integration

* `ci.yml` lints, runs the unit tests, checks that generated files are current, deploys the
  package to kind, and builds the artifacts.
* `e2e-docker.yml` runs every example against WSO2 IS 7.3.0 in Docker, in both API variants.
* `security.yml` runs govulncheck, CodeQL, gitleaks and the dependency review. `scorecard.yml`
  runs the OpenSSF Scorecard.
* `release.yml` publishes a release after CI passes on `main`. The version follows from the
  Conventional Commit subjects since the last tag (`hack/next-version.sh`).

## Release

Releases are automatic. After CI passes on `main`, the workflow creates a new version unless
every commit since the last tag is of type `docs`, `ci`, `chore`, `test` or `style`. A `feat`
raises the minor version, a breaking change (`type!:`) raises the major version (the minor
version while the major version is 0), and anything else raises the patch version. The workflow scans the image, pushes the package to
`ghcr.io/hayone1` and mirrors it to `xpkg.upbound.io/codejourney`, signs it with cosign, attests
the build provenance and the SBOM, and creates the GitHub release.

The mirror to Upbound needs the repository secrets `XPKG_MIRROR_ACCESS_ID` and
`XPKG_MIRROR_TOKEN`.
