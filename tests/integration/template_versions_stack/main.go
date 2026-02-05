package main

import (
	"github.com/JDetmar/pulumi-sendgrid/sdk/go/sendgrid"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// First, create a dynamic template (parent resource)
		template, err := sendgrid.NewTemplate(ctx, "test-template", &sendgrid.TemplateArgs{
			Name:       pulumi.String("Pulumi Test Template for Versions"),
			Generation: pulumi.String("dynamic"),
		})
		if err != nil {
			return err
		}

		// Create an active template version with HTML content
		activeVersion, err := sendgrid.NewTemplateVersion(ctx, "test-active-version", &sendgrid.TemplateVersionArgs{
			TemplateId:           template.TemplateId,
			Name:                 pulumi.String("Active Version"),
			Subject:              pulumi.String("Hello {{name}}!"),
			HtmlContent:          pulumi.String("<h1>Welcome, {{name}}!</h1><p>This is a test email from Pulumi.</p>"),
			Active:               pulumi.Int(1),
			GeneratePlainContent: pulumi.Bool(true),
		})
		if err != nil {
			return err
		}

		// Create an inactive draft version
		draftVersion, err := sendgrid.NewTemplateVersion(ctx, "test-draft-version", &sendgrid.TemplateVersionArgs{
			TemplateId:   template.TemplateId,
			Name:         pulumi.String("Draft Version"),
			Subject:      pulumi.String("Draft: Hello {{name}}"),
			HtmlContent:  pulumi.String("<h1>Draft Content</h1><p>This version is not active.</p>"),
			PlainContent: pulumi.String("Draft Content\n\nThis version is not active."),
			Active:       pulumi.Int(0),
		})
		if err != nil {
			return err
		}

		// Export the IDs and values for verification
		ctx.Export("templateId", template.TemplateId)
		ctx.Export("templateName", template.Name)
		ctx.Export("activeVersionId", activeVersion.VersionId)
		ctx.Export("activeVersionName", activeVersion.Name)
		ctx.Export("activeVersionSubject", activeVersion.Subject)
		ctx.Export("activeVersionActive", activeVersion.Active)
		ctx.Export("draftVersionId", draftVersion.VersionId)
		ctx.Export("draftVersionName", draftVersion.Name)
		ctx.Export("draftVersionActive", draftVersion.Active)

		return nil
	})
}
