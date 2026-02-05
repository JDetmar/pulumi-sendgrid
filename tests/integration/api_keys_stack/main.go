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
		testName := fmt.Sprintf("pulumi-stack-test-%d", time.Now().Unix())

		// Create a SendGrid API Key
		apiKey, err := sendgrid.NewApiKey(ctx, "test-api-key", &sendgrid.ApiKeyArgs{
			Name:   pulumi.String(testName),
			Scopes: pulumi.StringArray{pulumi.String("mail.send")},
		})
		if err != nil {
			return err
		}

		// Export the outputs for verification
		ctx.Export("apiKeyId", apiKey.ApiKeyId)
		ctx.Export("apiKeyName", apiKey.Name)
		ctx.Export("apiKeyValue", apiKey.ApiKey)

		return nil
	})
}
