package main

import (
	"fmt"
	"time"

	"github.com/JDetmar/pulumi-sendgrid/sdk/go/sendgrid"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		testName := fmt.Sprintf("pulumi-test-%d", time.Now().Unix())

		webhook, err := sendgrid.NewEventWebhook(ctx, "test-webhook", &sendgrid.EventWebhookArgs{
			Url:          pulumi.String("https://example.com/webhook"),
			FriendlyName: pulumi.String(testName),
			Enabled:      pulumi.Bool(false),
			Bounce:       pulumi.Bool(true),
			Click:        pulumi.Bool(true),
			Delivered:    pulumi.Bool(true),
			Dropped:      pulumi.Bool(true),
			Open:         pulumi.Bool(false),
			Processed:    pulumi.Bool(false),
			SpamReport:   pulumi.Bool(true),
			Deferred:     pulumi.Bool(false),
			Unsubscribe:  pulumi.Bool(false),
		})
		if err != nil {
			return err
		}

		ctx.Export("webhookId", webhook.WebhookId)
		ctx.Export("url", webhook.Url)
		ctx.Export("friendlyName", webhook.FriendlyName)
		ctx.Export("enabled", webhook.Enabled)

		return nil
	})
}
