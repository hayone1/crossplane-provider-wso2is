# Crossplane provider for WSO2 Identity Server

> This provider is an independent community project. It is not affiliated with, sponsored or
> endorsed by WSO2. WSO2 is a trademark of its owner, and this page uses it only to name the
> product that the provider works with.

`provider-wso2is` manages [WSO2 Identity Server](https://wso2.com/identity-server/) 7.3 with
Crossplane. It covers applications and API authorization, users, groups and roles, claims and
OpenID Connect scopes, identity providers, organizations (B2B), tenants, user stores,
notifications, branding, approval workflows, webhooks, and the tenant-wide settings.

[Upjet](https://github.com/crossplane/upjet) generated the provider from the
[WSO2 Identity Server Terraform provider](https://registry.terraform.io/providers/hayone1/wso2is/latest).
The provider runs the Terraform provider inside its own process, so it needs no Terraform
binary.

Every managed resource comes in two API variants. The cluster-scoped variant uses the groups
`*.wso2is.crossplane.io`. The namespaced variant, which Crossplane v2 introduced, uses the
groups `*.wso2is.m.crossplane.io`.

To get started, read the
[README](https://github.com/hayone1/crossplane-provider-wso2is#readme) and the
[examples](https://github.com/hayone1/crossplane-provider-wso2is/tree/main/examples).
