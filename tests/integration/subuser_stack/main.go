package main

import (
	"fmt"
	"time"

	"github.com/JDetmar/pulumi-sendgrid/sdk/go/sendgrid"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		testUsername := fmt.Sprintf("pulumitest%d", time.Now().Unix())
		testEmail := fmt.Sprintf("%s@example.com", testUsername)

		subuser, err := sendgrid.NewSubuser(ctx, "test-subuser", &sendgrid.SubuserArgs{
			Username: pulumi.String(testUsername),
			Email:    pulumi.String(testEmail),
			Password: pulumi.String("PulumiT3st!Pass"),
		})
		if err != nil {
			return err
		}

		ctx.Export("username", subuser.Username)
		ctx.Export("email", subuser.Email)
		ctx.Export("userId", subuser.UserId)
		ctx.Export("disabled", subuser.Disabled)

		return nil
	})
}
