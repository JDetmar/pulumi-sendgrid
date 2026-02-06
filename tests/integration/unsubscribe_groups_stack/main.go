package main

import (
	"fmt"
	"time"

	"github.com/JDetmar/pulumi-sendgrid/sdk/go/sendgrid"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Create a unique name for the test
		testName := fmt.Sprintf("pulumi-test-%d", time.Now().Unix())

		// Create a SendGrid Unsubscribe Group with all options
		group, err := sendgrid.NewUnsubscribeGroup(ctx, "test-unsubscribe-group", &sendgrid.UnsubscribeGroupArgs{
			Name:        pulumi.String(testName),
			Description: pulumi.String("Test group created by Pulumi integration test"),
			IsDefault:   pulumi.Bool(false),
		})
		if err != nil {
			return err
		}

		// Create a second group with minimal options
		group2, err := sendgrid.NewUnsubscribeGroup(ctx, "test-unsubscribe-group-minimal", &sendgrid.UnsubscribeGroupArgs{
			Name:        pulumi.String(testName + "-minimal"),
			Description: pulumi.String("Minimal test group"),
		})
		if err != nil {
			return err
		}

		// Export the outputs for verification
		ctx.Export("groupId", group.GroupId)
		ctx.Export("groupName", group.Name)
		ctx.Export("groupDescription", group.Description)
		ctx.Export("groupIsDefault", group.IsDefault)
		ctx.Export("groupUnsubscribes", group.Unsubscribes)

		ctx.Export("group2Id", group2.GroupId)
		ctx.Export("group2Name", group2.Name)

		return nil
	})
}
