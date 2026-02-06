package examples

import (
	"github.com/pulumi/providertest/providers"
	goprovider "github.com/pulumi/pulumi-go-provider"
	pulumirpc "github.com/pulumi/pulumi/sdk/v3/proto/go"

	"github.com/JDetmar/pulumi-sendgrid/provider"
)

// providerFactory creates a provider server for integration tests.
var providerFactory = func(_ providers.PulumiTest) (pulumirpc.ResourceProviderServer, error) {
	return goprovider.RawServer("sendgrid", "1.0.0", provider.Provider())(nil)
}
