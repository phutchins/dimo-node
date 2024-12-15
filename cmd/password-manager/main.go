package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/dimo/dimo-node/utils"
)

func main() {
	// Define command-line flags
	updateCmd := flag.NewFlagSet("update", flag.ExitOnError)
	updateStack := updateCmd.String("stack", "", "Stack name (required)")

	getCmd := flag.NewFlagSet("get", flag.ExitOnError)
	getStack := getCmd.String("stack", "", "Stack name (required)")
	getService := getCmd.String("service", "", "Service name (required)")

	// Check if a subcommand is provided
	if len(os.Args) < 2 {
		fmt.Println("Expected 'update' or 'get' subcommands")
		os.Exit(1)
	}

	// Parse subcommands
	switch os.Args[1] {
	case "update":
		updateCmd.Parse(os.Args[2:])
		if *updateStack == "" {
			log.Fatal("Stack name is required")
		}
		if err := utils.UpdatePasswords(*updateStack); err != nil {
			log.Fatal(err)
		}
		fmt.Println("Passwords updated successfully")

	case "get":
		getCmd.Parse(os.Args[2:])
		if *getStack == "" || *getService == "" {
			log.Fatal("Stack name and service name are required")
		}
		password, err := utils.GetPassword(*getStack, *getService)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(password)

	default:
		fmt.Println("Expected 'update' or 'get' subcommands")
		os.Exit(1)
	}
}
