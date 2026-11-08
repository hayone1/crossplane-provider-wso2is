// Package config holds the Upjet configuration of the WSO2 Identity Server provider.
package config

import (
	// The provider schema and metadata are embedded into the binary.
	_ "embed"

	ujconfig "github.com/crossplane/upjet/v2/pkg/config"
	wso2is "github.com/hayone1/terraform-provider-wso2is/provider"

	"github.com/hayone1/crossplane-provider-wso2is/internal/version"
)

const (
	resourcePrefix = "wso2is"
	modulePath     = "github.com/hayone1/crossplane-provider-wso2is"

	rootGroupCluster    = "wso2is.crossplane.io"
	rootGroupNamespaced = "wso2is.m.crossplane.io"
)

//go:embed schema.json
var providerSchema string

//go:embed provider-metadata.yaml
var providerMetadata string

// GetProvider returns the configuration of the cluster-scoped managed resources
// (*.wso2is.crossplane.io).
func GetProvider() *ujconfig.Provider {
	return buildProvider(rootGroupCluster)
}

// GetProviderNamespaced returns the configuration of the namespaced managed resources
// (*.wso2is.m.crossplane.io) that Crossplane v2 introduced.
func GetProviderNamespaced() *ujconfig.Provider {
	return buildProvider(rootGroupNamespaced,
		ujconfig.WithExampleManifestConfiguration(ujconfig.ExampleManifestConfiguration{
			ManagedResourceNamespace: "default",
		}))
}

func buildProvider(rootGroup string, opts ...ujconfig.ProviderOption) *ujconfig.Provider {
	opts = append([]ujconfig.ProviderOption{
		ujconfig.WithShortName("wso2is"),
		ujconfig.WithRootGroup(rootGroup),
		ujconfig.WithFeaturesPackage("internal/features"),
		// Every resource runs through the in-process ("no-fork") Terraform plugin
		// framework provider, so the CLI and plugin SDK include lists stay empty.
		ujconfig.WithIncludeList([]string{}),
		ujconfig.WithTerraformPluginSDKIncludeList([]string{}),
		ujconfig.WithTerraformPluginFrameworkIncludeList(ExternalNameConfigured()),
		ujconfig.WithTerraformPluginFrameworkProvider(wso2is.New(version.Version)()),
		ujconfig.WithDefaultResourceOptions(ExternalNameConfigurations()),
	}, opts...)

	pc := ujconfig.NewProvider([]byte(providerSchema), resourcePrefix, modulePath, []byte(providerMetadata), opts...)
	for _, configure := range configurators {
		configure(pc)
	}
	pc.ConfigureResources()
	return pc
}
