package main

import (
	"dependencies"
	"infrastructure"
)

func main() {
	err := infrastructure.Run("infra/dev")
	if err != nil {
		return
	}

	err = dependencies.Run("deps/dev")
	if err != nil {
		return
	}

	//test := infraCtx.GetOutput("publicIp").(string)

	//fmt.Printf("Public IP: %s", test)

	return

}
