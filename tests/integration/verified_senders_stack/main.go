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
		testName := fmt.Sprintf("pulumi-test-sender-%d", time.Now().Unix())

		// Create a SendGrid Verified Sender
		sender, err := sendgrid.NewVerifiedSender(ctx, "test-verified-sender", &sendgrid.VerifiedSenderArgs{
			Nickname:  pulumi.String(testName),
			FromEmail: pulumi.String("pulumi-test@example.com"),
			FromName:  pulumi.String("Pulumi Test"),
			ReplyTo:   pulumi.String("pulumi-reply@example.com"),
			Address:   pulumi.String("123 Test Street"),
			City:      pulumi.String("San Francisco"),
			State:     pulumi.String("CA"),
			Zip:       pulumi.String("94105"),
			Country:   pulumi.String("USA"),
		})
		if err != nil {
			return err
		}

		// Export the outputs for verification
		ctx.Export("senderId", sender.SenderId)
		ctx.Export("nickname", sender.Nickname)
		ctx.Export("fromEmail", sender.FromEmail)
		ctx.Export("verified", sender.Verified)

		return nil
	})
}
