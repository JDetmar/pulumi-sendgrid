package main

import (
	sendgrid "github.com/JDetmar/pulumi-sendgrid/sdk/go/sendgrid"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		myApiKey, err := sendgrid.NewApiKey(ctx, "myApiKey", &sendgrid.ApiKeyArgs{
			Name:   pulumi.String("my-app-api-key"),
			Scopes: pulumi.StringArray{pulumi.String("mail.send"), pulumi.String("alerts.read")},
		})
		if err != nil {
			return err
		}

		myTemplate, err := sendgrid.NewTemplate(ctx, "myTemplate", &sendgrid.TemplateArgs{
			Name:       pulumi.String("welcome-email"),
			Generation: pulumi.String("dynamic"),
		})
		if err != nil {
			return err
		}

		myEventWebhook, err := sendgrid.NewEventWebhook(ctx, "myEventWebhook", &sendgrid.EventWebhookArgs{
			Url:          pulumi.String("https://example.com/webhooks/sendgrid"),
			FriendlyName: pulumi.String("My App Webhook"),
			Enabled:      pulumi.Bool(false),
			Delivered:    pulumi.Bool(true),
			Open:         pulumi.Bool(true),
			Click:        pulumi.Bool(true),
			Bounce:       pulumi.Bool(true),
			Dropped:      pulumi.Bool(true),
			SpamReport:   pulumi.Bool(true),
		})
		if err != nil {
			return err
		}

		myDomainAuth, err := sendgrid.NewDomainAuthentication(ctx, "myDomainAuth", &sendgrid.DomainAuthenticationArgs{
			Domain:            pulumi.String("example.com"),
			AutomaticSecurity: pulumi.Bool(true),
		})
		if err != nil {
			return err
		}

		ctx.Export("apiKeyId", myApiKey.ApiKeyId)
		ctx.Export("templateId", myTemplate.TemplateId)
		ctx.Export("webhookId", myEventWebhook.WebhookId)
		ctx.Export("domainId", myDomainAuth.DomainId)
		return nil
	})
}
