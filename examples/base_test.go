package examples

import (
	"github.com/JDetmar/pulumi-sendgrid/provider"
	"github.com/pulumi/providertest/providers"
	goprovider "github.com/pulumi/pulumi-go-provider"
	pulumirpc "github.com/pulumi/pulumi/sdk/v3/proto/go"
)

// providerFactory creates a provider server for integration tests.
// Currently unused but kept for future test implementations.
var _ = func(_ providers.PulumiTest) (pulumirpc.ResourceProviderServer, error) {
	return goprovider.RawServer("sendgrid", "1.0.0", provider.Provider())(nil)
}
