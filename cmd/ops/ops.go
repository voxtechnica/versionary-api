package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:     "ops",
		Version: "0.1.0",
		Short:   "ops is a command line tool for managing the Versionary API",
		Long: `Versionary API demonstrates a way to manage versioned entities in a database with a serverless architecture.
Ops provides commands for peforming various operational tasks, such as initializing the database tables.`,
	}
	initTableCmd(rootCmd)
	initTokenCmd(rootCmd)
	initTuidCmd(rootCmd)
	initUserCmd(rootCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
