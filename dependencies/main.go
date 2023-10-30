package dependencies

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Run(ctx *pulumi.Context) error {
	//func Run(infraCtx *pulumi.Context) *pulumi.Context {
	//var depsCtx *pulumi.Context

	//depsCtx = ctx
	//cts.Ref
	//ctx.StackReference.GetOutput("publicIp")
	publicIp := accessConfigs.Index(pulumi.Int(0)).NatIp().Elem()
	fmt.Printf("Public IP: %s", publicIp)

	return nil
}
