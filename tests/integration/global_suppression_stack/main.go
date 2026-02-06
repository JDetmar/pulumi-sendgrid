package main

import (
	"fmt"
	"time"

	"github.com/JDetmar/pulumi-sendgrid/sdk/go/sendgrid"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		testEmail := fmt.Sprintf("pulumi-test-%d@example.com", time.Now().Unix())

		suppression, err := sendgrid.NewGlobalSuppression(ctx, "test-suppression", &sendgrid.GlobalSuppressionArgs{
			Email: pulumi.String(testEmail),
		})
		if err != nil {
			return err
		}

		ctx.Export("email", suppression.Email)
		ctx.Export("createdAt", suppression.CreatedAt)

		return nil
	})
}
