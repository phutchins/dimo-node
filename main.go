package main

import (
	"dependencies"
	"infrastructure"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		err := infrastructure.Run(ctx)
		if err != nil {
			return err
		}

		err = dependencies.Run(ctx)
		if err != nil {
			return err
		}

		//test := infraCtx.GetOutput("publicIp").(string)

		//fmt.Printf("Public IP: %s", test)

		return nil
	})

}
