package app

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"versionary-api/pkg/event"
	"versionary-api/pkg/user"
)

type Application struct {
	Name         string           // Name of the application
	Env          string           // Environment name (e.g. "dev", "test", "staging", "prod")
	AWSConfig    aws.Config       // AWS Config
	DBClient     *dynamodb.Client // DynamoDB client
	EntityTypes  []string         // List of entity types (e.g. "Event", "User", etc.)
	EventService event.EventService
	OrgService   user.OrganizationService
	TokenService user.TokenService
	UserService  user.UserService
}

func (a *Application) Init(env string) error {
	// startTime := time.Now()

	// Set the operating environment, which may be an environment variable or a command-line flag
	if env == "" {
		a.Env = "dev"
	} else {
		a.Env = env
	}

	// Set up AWS Config
	ctx := context.Background()
	aws, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("error loading AWS config: %w", err)
	}
	a.AWSConfig = aws
	a.DBClient = dynamodb.NewFromConfig(aws)
	if err != nil {
		return fmt.Errorf("error creating DynamoDB client: %w", err)
	}

	// Entity Types
	a.EntityTypes = []string{
		"Event",
		"Organization",
		"User",
		"Token",
	}

	// Initialize Services
	a.EventService = event.EventService{
		EntityType: "Event",
		Table:      event.NewEventTable(a.DBClient, a.Env),
	}
	a.OrgService = user.OrganizationService{
		EntityType: "Organization",
		Table:      user.NewOrganizationTable(a.DBClient, a.Env),
	}
	a.TokenService = user.TokenService{
		EntityType: "Token",
		Table:      user.NewTokenTable(a.DBClient, a.Env),
	}
	a.UserService = user.UserService{
		EntityType: "User",
		Table:      user.NewUserTable(a.DBClient, a.Env),
	}

	// fmt.Println("Initialized Application in ", time.Since(startTime))
	return nil
}

func (a *Application) InitMock(env string) error {
	// Set the operating environment, which may be an environment variable or a command-line flag
	if env == "" {
		a.Env = "dev"
	} else {
		a.Env = env
	}

	// Entity Types
	a.EntityTypes = []string{
		"Event",
		"Organization",
		"User",
		"Token",
	}

	// Initialize Services
	a.EventService = event.EventService{
		EntityType: "Event",
		Table:      event.NewEventMemTable(event.NewEventTable(a.DBClient, a.Env)),
	}
	a.OrgService = user.OrganizationService{
		EntityType: "Organization",
		Table:      user.NewOrganizationMemTable(user.NewOrganizationTable(a.DBClient, a.Env)),
	}
	a.TokenService = user.TokenService{
		EntityType: "Token",
		Table:      user.NewTokenMemTable(user.NewTokenTable(a.DBClient, a.Env)),
	}
	a.UserService = user.UserService{
		EntityType: "User",
		Table:      user.NewUserMemTable(user.NewUserTable(a.DBClient, a.Env)),
	}
	return nil
}
