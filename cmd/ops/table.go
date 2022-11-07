package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"versionary-api/pkg/device"
	"versionary-api/pkg/email"
	"versionary-api/pkg/event"
	"versionary-api/pkg/image"
	"versionary-api/pkg/org"
	"versionary-api/pkg/token"
	"versionary-api/pkg/user"
	"versionary-api/pkg/view"

	"github.com/spf13/cobra"
	"github.com/voxtechnica/versionary"
)

// initTableCmd initializes the table commands.
func initTableCmd(root *cobra.Command) {
	tableCmd := &cobra.Command{
		Use:   "table",
		Short: "Manage DynamoDB tables",
	}
	tableCmd.Flags().StringP("env", "e", "", "Operating environment: dev | test | staging | prod")
	root.AddCommand(tableCmd)

	checkCmd := &cobra.Command{
		Use:   "check [entityType...]",
		Short: "Ensure that DynamoDB table(s) exist",
		Long: `Check each specified DynamoDB table, creating them if they do not exist.
If no entity types are specified, all tables in the specified environment will be checked.`,
		RunE: checkTables,
	}
	checkCmd.Flags().StringP("env", "e", "", "Operating environment: dev | test | staging | prod")
	_ = checkCmd.MarkFlagRequired("env")
	tableCmd.AddCommand(checkCmd)

	deleteCmd := &cobra.Command{
		Use:   "delete [entityType...]",
		Short: "Delete DynamoDB table(s)",
		Long: `Delete each specified DynamoDB table in a non-production environment.
If no entity types are specified, all tables in the specified environment will be deleted.`,
		RunE: deleteTables,
	}
	deleteCmd.Flags().StringP("env", "e", "", "Operating environment: dev | test | staging")
	_ = deleteCmd.MarkFlagRequired("env")
	tableCmd.AddCommand(deleteCmd)
}

// checkTables checks each table in the specified environment.
func checkTables(cmd *cobra.Command, args []string) error {
	err := ops.Init(cmd.Flag("env").Value.String())
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
		tables = ops.EntityTypes
	}
	fmt.Printf("Checking %d Table(s) in %s: %s\n", len(tables), ops.Environment, strings.Join(tables, ", "))
	for _, entity := range tables {
		// TODO: add new DynamoDB tables here
		switch entity {
		case "Device":
			checkTable(ctx, device.NewTable(ops.DBClient, ops.Environment))
		case "DeviceCount":
			checkTable(ctx, device.NewCountTable(ops.DBClient, ops.Environment))
		case "Email":
			checkTable(ctx, email.NewTable(ops.DBClient, ops.Environment))
		case "Event":
			checkTable(ctx, event.NewTable(ops.DBClient, ops.Environment))
		case "Image":
			checkTable(ctx, image.NewTable(ops.DBClient, ops.Environment))
		case "Organization":
			checkTable(ctx, org.NewTable(ops.DBClient, ops.Environment))
		case "Token":
			checkTable(ctx, token.NewTable(ops.DBClient, ops.Environment))
		case "User":
			checkTable(ctx, user.NewTable(ops.DBClient, ops.Environment))
		case "View":
			checkTable(ctx, view.NewTable(ops.DBClient, ops.Environment))
		case "ViewCount":
			checkTable(ctx, view.NewCountTable(ops.DBClient, ops.Environment))
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

// deleteTables deletes each table in the specified environment.
func deleteTables(cmd *cobra.Command, args []string) error {
	err := ops.Init(cmd.Flag("env").Value.String())
	if err != nil {
		return fmt.Errorf("error initializing application: %w", err)
	}
	ctx := context.Background()

	// Not for use in production!
	if ops.Environment == "prod" {
		return errors.New("delete tables in production? Really? Use the AWS console instead")
	}

	// Delete each table
	var tables []string
	if len(args) > 0 {
		// If table names were specified, only delete those.
		tables = args
	} else {
		// Otherwise, delete all tables.
		tables = ops.EntityTypes
	}
	fmt.Printf("Deleting %d Table(s) in %s: %s\n", len(tables), ops.Environment, strings.Join(tables, ", "))
	for _, entity := range tables {
		// TODO: add new DynamoDB tables here
		switch entity {
		case "Device":
			deleteTable(ctx, device.NewTable(ops.DBClient, ops.Environment))
		case "DeviceCount":
			deleteTable(ctx, device.NewCountTable(ops.DBClient, ops.Environment))
		case "Email":
			deleteTable(ctx, email.NewTable(ops.DBClient, ops.Environment))
		case "Event":
			deleteTable(ctx, event.NewTable(ops.DBClient, ops.Environment))
		case "Image":
			deleteTable(ctx, image.NewTable(ops.DBClient, ops.Environment))
		case "Organization":
			deleteTable(ctx, org.NewTable(ops.DBClient, ops.Environment))
		case "Token":
			deleteTable(ctx, token.NewTable(ops.DBClient, ops.Environment))
		case "User":
			deleteTable(ctx, user.NewTable(ops.DBClient, ops.Environment))
		case "View":
			deleteTable(ctx, view.NewTable(ops.DBClient, ops.Environment))
		case "ViewCount":
			deleteTable(ctx, view.NewCountTable(ops.DBClient, ops.Environment))
		default:
			fmt.Println("Skipping unknown entity type:", entity)
		}
	}
	return nil
}

// deleteTable deletes a DynamoDB table.
// Note that Versionary already logs its activity to the console.
func deleteTable[T any](ctx context.Context, table versionary.Table[T]) {
	if !table.TableExists(ctx) {
		log.Println("table", table.TableName, "MISSING - skipping")
		return
	}
	if err := table.DeleteTable(ctx); err != nil {
		log.Println("table", table.TableName, "ERROR deleting table:", err)
	}
}
