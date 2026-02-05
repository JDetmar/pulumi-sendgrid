package main

import (
	"github.com/JDetmar/pulumi-sendgrid/sdk/go/sendgrid"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Create a dynamic template
		dynamicTemplate, err := sendgrid.NewTemplate(ctx, "test-dynamic-template", &sendgrid.TemplateArgs{
			Name:       pulumi.String("Pulumi Test Dynamic Template"),
			Generation: pulumi.String("dynamic"),
		})
		if err != nil {
			return err
		}

		// Create a legacy template
		legacyTemplate, err := sendgrid.NewTemplate(ctx, "test-legacy-template", &sendgrid.TemplateArgs{
			Name:       pulumi.String("Pulumi Test Legacy Template"),
			Generation: pulumi.String("legacy"),
		})
		if err != nil {
			return err
		}

		// Export the template IDs
		ctx.Export("dynamicTemplateId", dynamicTemplate.TemplateId)
		ctx.Export("dynamicTemplateName", dynamicTemplate.Name)
		ctx.Export("legacyTemplateId", legacyTemplate.TemplateId)
		ctx.Export("legacyTemplateName", legacyTemplate.Name)

		return nil
	})
}
