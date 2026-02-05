package main

import (
	sendgrid "github.com/JDetmar/pulumi-sendgrid/sdk/go/sendgrid"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		myRandomResource, err := sendgrid.NewRandom(ctx, "myRandomResource", &sendgrid.RandomArgs{
			Length: pulumi.Int(24),
		})
		if err != nil {
			return err
		}
		_, err = sendgrid.NewRandomComponent(ctx, "myRandomComponent", &sendgrid.RandomComponentArgs{
			Length: pulumi.Int(24),
		})
		if err != nil {
			return err
		}
		ctx.Export("output", pulumi.StringMap{
			"value": myRandomResource.Result,
		})
		return nil
	})
}
