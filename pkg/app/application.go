package app

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/voxtechnica/tuid-go"

	"versionary-api/pkg/event"
	"versionary-api/pkg/user"
)

// About provides basic information about the API.
type About struct {
	Name        string `json:"name"`
	GitHash     string `json:"gitHash,omitempty"`
	BuildTime   string `json:"buildTime"`
	Language    string `json:"language"`
	Environment string `json:"environment"`
	Description string `json:"description,omitempty"`
}

// String supports the Stringer interface.
func (a About) String() string {
	return fmt.Sprintf("Name: %s\nGitHash: %s\nBuildTime: %s\nLanguage: %s\nEnvironment: %s\nDescription: %s",
		a.Name, a.GitHash, a.BuildTime, a.Language, a.Environment, a.Description)
}

// Application is the main application object, which contains configuration settings, keys, and initialized services.
type Application struct {
	Name         string           // Name of the application
	GitHash      string           // Git hash of the application
	BuildTime    string           // Executable build time
	Language     string           // Go Compiler version (e.g. "go1.x")
	Environment  string           // Environment name (e.g. "dev", "test", "staging", "prod")
	Description  string           // Description of the application
	AWSConfig    aws.Config       // AWS Configuration
	DBClient     *dynamodb.Client // DynamoDB client
	EntityTypes  []string         // Valid entity type names (e.g. "Event", "User", etc.)
	EventService event.EventService
	OrgService   user.OrganizationService
	TokenService user.TokenService
	UserService  user.UserService
}

// About returns basic information about the initialized Application.
func (a *Application) About() About {
	return About{
		Name:        a.Name,
		GitHash:     a.GitHash,
		BuildTime:   a.BuildTime,
		Language:    a.Language,
		Environment: a.Environment,
		Description: a.Description,
	}
}

// setDefaults sets default configuration settings for the application.
func (a *Application) setDefaults() {
	if a.Name == "" {
		a.Name = "Versionary API"
	}
	if a.BuildTime == "" {
		p, err := os.Executable()
		if err == nil {
			s, err := os.Stat(p)
			if err == nil {
				a.BuildTime = s.ModTime().String()
			} else {
				a.BuildTime = err.Error()
			}
		} else {
			a.BuildTime = err.Error()
		}
	}
	if a.Language == "" {
		a.Language = runtime.Version() + " (" + runtime.GOOS + " " + runtime.GOARCH + ")"
	}
	if a.Environment == "" {
		a.Environment = "dev"
	}

	// Entity Types
	// TODO: Update this list and initialize new services below as new entity types are added
	a.EntityTypes = []string{
		"Event",
		"Organization",
		"User",
		"Token",
	}
}

// Init initializes the application, including clients and services.
func (a *Application) Init(env string) error {
	// startTime := time.Now()

	// Set default values for the application
	a.Environment = env
	a.setDefaults()

	// Initialize AWS Clients
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("error loading AWS config: %w", err)
	}
	a.AWSConfig = cfg
	a.DBClient = dynamodb.NewFromConfig(cfg)
	if err != nil {
		return fmt.Errorf("error creating DynamoDB client: %w", err)
	}

	// Initialize Services
	a.EventService = event.EventService{
		EntityType: "Event",
		Table:      event.NewEventTable(a.DBClient, a.Environment),
	}
	a.OrgService = user.OrganizationService{
		EntityType: "Organization",
		Table:      user.NewOrganizationTable(a.DBClient, a.Environment),
	}
	a.TokenService = user.TokenService{
		EntityType: "Token",
		Table:      user.NewTokenTable(a.DBClient, a.Environment),
	}
	a.UserService = user.UserService{
		EntityType: "User",
		Table:      user.NewUserTable(a.DBClient, a.Environment),
	}

	// fmt.Println("Initialized Application in ", time.Since(startTime))
	return nil
}

// InitMock initializes the application for testing, including mock clients and services.
func (a *Application) InitMock(env string) error {
	// Set default values for the application
	a.Environment = env
	a.setDefaults()

	// Initialize Services
	a.EventService = event.EventService{
		EntityType: "Event",
		Table:      event.NewEventMemTable(event.NewEventTable(a.DBClient, a.Environment)),
	}
	a.OrgService = user.OrganizationService{
		EntityType: "Organization",
		Table:      user.NewOrganizationMemTable(user.NewOrganizationTable(a.DBClient, a.Environment)),
	}
	a.TokenService = user.TokenService{
		EntityType: "Token",
		Table:      user.NewTokenMemTable(user.NewTokenTable(a.DBClient, a.Environment)),
	}
	a.UserService = user.UserService{
		EntityType: "User",
		Table:      user.NewUserMemTable(user.NewUserTable(a.DBClient, a.Environment)),
	}
	return nil
}

// TokenUser reads a specified Token and its associated User.
func (a *Application) TokenUser(ctx context.Context, tokenID string) (user.Token, user.User, error) {
	// Validate the Application
	if a.TokenService.Table == nil || a.UserService.Table == nil {
		return user.Token{}, user.User{}, fmt.Errorf("application not initialized")
	}
	// Validate the bearer token
	if tokenID == "" || !tuid.IsValid(tuid.TUID(tokenID)) {
		return user.Token{}, user.User{}, fmt.Errorf("invalid bearer token")
	}
	// Read the specified token
	t, err := a.TokenService.Read(ctx, tokenID)
	if err != nil {
		// tokens expire, so this will be a common response
		return user.Token{}, user.User{}, fmt.Errorf("error reading token: %w", err)
	}
	// Read the associated user
	u, err := a.UserService.Read(ctx, t.UserID)
	if err != nil {
		return t, user.User{}, fmt.Errorf("error reading user %s from token: %w", t.UserID, err)
	}
	// Check that the user is active (not disabled)
	if u.Status == user.DISABLED {
		return t, u, fmt.Errorf("user %s status is %s", u.ID, u.Status)
	}
	return t, u, nil
}
