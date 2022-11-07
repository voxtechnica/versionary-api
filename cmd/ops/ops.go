package main

import (
	"fmt"
	"os"
	"versionary-api/pkg/app"

	"github.com/spf13/cobra"
)

// gitHash returns the git hash of the compiled application.
// It is embedded in the binary and is automatically updated by the build process.
// go build -ldflags "-X main.gitHash=`git rev-parse HEAD`"
var gitHash string

// ops is the application object, containing global configuration settings and initialized services.
var ops = app.Application{
	Name:       "Versionary CLI",
	BaseDomain: "versionary.net",
	Description: "Versionary API demonstrates a way to manage versioned entities in a database with a serverless architecture.\n\t" +
		"Ops provides commands for performing various operational tasks, such as initializing the database tables.",
	GitHash: gitHash,
}

// main is the entry point for the application.
func main() {
	// Root Command
	rootCmd := &cobra.Command{
		Use:     "ops",
		Short:   "ops is a command line tool for managing the Versionary API",
		Long:    ops.Description,
		Version: ops.GitHash,
	}

	// About Command
	aboutCmd := &cobra.Command{
		Use:   "about",
		Short: "Print application information",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := ops.InitMock("")
			if err != nil {
				return err
			}
			fmt.Println(ops.About())
			return nil
		},
	}
	rootCmd.AddCommand(aboutCmd)

	// Initialize the application commands:
	initBucketCmd(rootCmd)
	initImageCmd(rootCmd)
	initTableCmd(rootCmd)
	initTokenCmd(rootCmd)
	initTuidCmd(rootCmd)
	initUserCmd(rootCmd)

	// Execute the specified command:
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
