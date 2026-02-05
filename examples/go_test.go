//go:build go || all
// +build go all

package examples

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pulumi/providertest/pulumitest"
	"github.com/pulumi/providertest/pulumitest/opttest"
	"github.com/stretchr/testify/require"
)

func TestGoExampleLifecycle(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)

	module := filepath.Join(cwd, "../sdk/go/pulumi-sendgrid")
	pt := pulumitest.NewPulumiTest(t, "go",
		opttest.GoModReplacement("github.com/JDetmar/pulumi-sendgrid/sdk/go/pulumi-sendgrid", module),
		opttest.AttachProviderServer("sendgrid", providerFactory),
		opttest.SkipInstall(),
	)

	pt.Preview(t)
	pt.Up(t)
	pt.Destroy(t)
}
