package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/voxtechnica/tuid-go"
	"github.com/voxtechnica/versionary"
	"golang.org/x/exp/slices"
	"versionary-api/pkg/event"
)

func main() {
	// Parse the command line flags:
	allActions := []string{"create-tables", "create-tuid"}
	var action string
	actionUsage := "Action (required): " + strings.Join(allActions, " | ")
	flag.StringVar(&action, "action", "", actionUsage)
	flag.StringVar(&action, "a", "", actionUsage+" (short form)")

	allEntityTypes := []string{"Event", "Token", "User"}
	var entityType string
	entityTypeUsage := "Entity Type: " + strings.Join(allEntityTypes, " | ") + " | blank for all"
	flag.StringVar(&entityType, "entity-type", "", entityTypeUsage)
	flag.StringVar(&entityType, "e", "", entityTypeUsage+" (short form)")

	allEnvironments := []string{"dev", "test", "staging", "prod"}
	var env string
	environmentUsage := "Environment: " + strings.Join(allEnvironments, " | ")
	flag.StringVar(&env, "env", "dev", environmentUsage)
	flag.Parse()

	// Validate the action:
	if action == "" {
		fmt.Println("Missing required flag: -action")
		flag.Usage()
		os.Exit(0)
	}
	if !slices.Contains(allActions, action) {
		fmt.Printf("Invalid action: %s\n", action)
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Validate entity type(s):
	var entityTypes []string
	if entityType == "" {
		entityTypes = allEntityTypes
	} else if !slices.Contains(allEntityTypes, entityType) {
		fmt.Printf("Invalid entity-type: %s\n", entityType)
		flag.PrintDefaults()
		os.Exit(1)
	} else {
		entityTypes = []string{entityType}
	}

	// Initialize services:
	ctx := context.Background()
	aws, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		fmt.Printf("Error loading AWS config: %v\n", err)
		os.Exit(1)
	}
	dbClient := dynamodb.NewFromConfig(aws)
	if err != nil {
		fmt.Printf("Error creating DynamoDB client: %v\n", err)
		return
	}

	// Execute the command:
	switch action {
	case "create-tables":
		createTables(ctx, dbClient, env, entityTypes)
	case "create-tuid":
		fmt.Println(tuid.NewID())
	default:
		fmt.Printf("Skipping unknown action: %s\n", action)
	}
}

func createTables(ctx context.Context, dbClient *dynamodb.Client, env string, entities []string) {
	fmt.Println("Checking/Creating Tables:", strings.Join(entities, ", "))
	for _, e := range entities {
		switch e {
		case "Event":
			createTable(ctx, event.NewTable(dbClient, env))
		//case "Token":
		//	createTable(ctx, auth.GetTokenTable(dbClient, env))
		//case "User":
		//	createTable(ctx, auth.GetUserTable(dbClient, env))
		default:
			fmt.Printf("Skipping unknown entity type: %s\n", e)
		}
	}
}

// createTable creates a DynamoDB table if it does not already exist.
// Note that Versionary already logs its activity to the console.
func createTable[T any](ctx context.Context, table versionary.Table[T]) {
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
