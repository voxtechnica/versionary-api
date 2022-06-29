package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"versionary-api/pkg/app"
	"versionary-api/pkg/event"
	"versionary-api/pkg/user"

	"github.com/spf13/cobra"
	"github.com/voxtechnica/versionary"
)

func initTableCmd(root *cobra.Command) {
	tableCmd := &cobra.Command{
		Use:   "table [entityType]...",
		Short: "Ensure that DynamoDB table(s) exist",
		Long: `Check each specified DynamoDB table, creating them if they do not exist.
If no entity types are specified, all tables will be checked.`,
		RunE: checkTables,
	}
	tableCmd.Flags().StringP("env", "e", "", "Operating environment: dev | test | staging | prod")
	tableCmd.MarkFlagRequired("env")
	root.AddCommand(tableCmd)
}

// checkTables checks each table in the specified environment.
func checkTables(cmd *cobra.Command, args []string) error {
	var a app.Application
	err := a.Init(cmd.Flag("env").Value.String())
	if err != nil {
		return fmt.Errorf("error initializing application: %w", err)
	}
	ctx := context.Background()

	// Check each table
	var tables []string
	if len(args) > 0 {
		// If table names were specified, only check those.
		tables = args
	} else {
		// Otherwise, check all tables.
		tables = a.EntityTypes
	}
	fmt.Printf("Checking %d Table(s) in %s: %s\n", len(tables), a.Env, strings.Join(tables, ", "))
	for _, entity := range tables {
		// TODO: add new DynamoDB tables here
		switch entity {
		case "Event":
			checkTable(ctx, event.NewEventTable(a.DBClient, a.Env))
		case "Organization":
			checkTable(ctx, user.NewOrganizationTable(a.DBClient, a.Env))
		case "Token":
			checkTable(ctx, user.NewTokenTable(a.DBClient, a.Env))
		case "User":
			checkTable(ctx, user.NewUserTable(a.DBClient, a.Env))
		default:
			fmt.Println("Skipping unknown entity type:", entity)
		}
	}
	return nil
}

// checkTable creates a DynamoDB table if it does not already exist.
// Note that Versionary already logs its activity to the console.
func checkTable[T any](ctx context.Context, table versionary.Table[T]) {
	if !table.IsValid() {
		log.Println("table", table.TableName, "INVALID table definition - skipping")
		return
	}
	if !table.TableExists(ctx) {
		if err := table.CreateTable(ctx); err != nil {
			log.Println("table", table.TableName, "ERROR creating table:", err)
		}
	}
}
