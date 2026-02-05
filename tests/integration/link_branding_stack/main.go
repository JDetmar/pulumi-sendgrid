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
		testSubdomain := fmt.Sprintf("lnk%d", time.Now().Unix())

		// Create a SendGrid Link Branding
		// Note: This uses a test domain - in real usage, you'd use your actual domain
		linkBrand, err := sendgrid.NewLinkBranding(ctx, "test-link-branding", &sendgrid.LinkBrandingArgs{
			Domain:    pulumi.String("example.com"),
			Subdomain: pulumi.String(testSubdomain),
			Default:   pulumi.Bool(false),
		})
		if err != nil {
			return err
		}

		// Export the outputs for verification
		ctx.Export("linkId", linkBrand.LinkId)
		ctx.Export("domain", linkBrand.Domain)
		ctx.Export("subdomain", linkBrand.Subdomain)
		ctx.Export("valid", linkBrand.Valid)
		ctx.Export("ownerCname", linkBrand.OwnerCname)
		ctx.Export("brandCname", linkBrand.BrandCname)

		return nil
	})
}
