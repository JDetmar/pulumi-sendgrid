package main

import (
	"fmt"
	"time"

	"github.com/JDetmar/pulumi-sendgrid/sdk/go/sendgrid"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Create a unique pool name for the test to avoid conflicts
		testPoolName := fmt.Sprintf("test-pool-%d", time.Now().Unix())

		// Create a SendGrid IP Pool
		ipPool, err := sendgrid.NewIpPool(ctx, "test-ip-pool", &sendgrid.IpPoolArgs{
			Name: pulumi.String(testPoolName),
		})
		if err != nil {
			return err
		}

		// Export the outputs for verification
		ctx.Export("poolName", ipPool.PoolName)
		ctx.Export("name", ipPool.Name)
		ctx.Export("ips", ipPool.Ips)

		return nil
	})
}
