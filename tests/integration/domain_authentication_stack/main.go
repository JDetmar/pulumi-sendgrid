package main

import (
	"fmt"
	"time"

	"github.com/JDetmar/pulumi-sendgrid/sdk/go/sendgrid"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Create a unique subdomain for the test to avoid conflicts
		testSubdomain := fmt.Sprintf("test%d", time.Now().Unix())

		// Create a SendGrid Domain Authentication
		// Note: This uses a test domain - in real usage, you'd use your actual domain
		domainAuth, err := sendgrid.NewDomainAuthentication(ctx, "test-domain-auth", &sendgrid.DomainAuthenticationArgs{
			Domain:            pulumi.String("example.com"),
			Subdomain:         pulumi.String(testSubdomain),
			AutomaticSecurity: pulumi.Bool(true),
		})
		if err != nil {
			return err
		}

		// Export the outputs for verification
		ctx.Export("domainId", domainAuth.DomainId)
		ctx.Export("domain", domainAuth.Domain)
		ctx.Export("subdomain", domainAuth.Subdomain)
		ctx.Export("valid", domainAuth.Valid)
		ctx.Export("mailCname", domainAuth.MailCname)
		ctx.Export("dkim1", domainAuth.Dkim1)
		ctx.Export("dkim2", domainAuth.Dkim2)

		return nil
	})
}
