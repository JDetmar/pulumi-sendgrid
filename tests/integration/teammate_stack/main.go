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

		teammate, err := sendgrid.NewTeammate(ctx, "test-teammate", &sendgrid.TeammateArgs{
			Email:  pulumi.String(testEmail),
			Scopes: pulumi.StringArray{pulumi.String("mail.send"), pulumi.String("alerts.read")},
		})
		if err != nil {
			return err
		}

		ctx.Export("email", teammate.Email)
		ctx.Export("isAdmin", teammate.IsAdmin)
		ctx.Export("token", teammate.Token)
		ctx.Export("scopes", teammate.Scopes)

		return nil
	})
}
