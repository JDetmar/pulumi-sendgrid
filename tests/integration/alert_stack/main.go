package main

import (
	"github.com/JDetmar/pulumi-sendgrid/sdk/go/sendgrid"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Create a usage_limit alert
		alert, err := sendgrid.NewAlert(ctx, "test-usage-alert", &sendgrid.AlertArgs{
			Type:       pulumi.String("usage_limit"),
			EmailTo:    pulumi.String("pulumi-test@example.com"),
			Percentage: pulumi.Int(90),
		})
		if err != nil {
			return err
		}

		ctx.Export("alertId", alert.AlertId)
		ctx.Export("alertType", alert.Type)
		ctx.Export("emailTo", alert.EmailTo)
		ctx.Export("percentage", alert.Percentage)

		return nil
	})
}
