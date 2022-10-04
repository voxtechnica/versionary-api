package app

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"
	"versionary-api/pkg/client"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/voxtechnica/tuid-go"

	"versionary-api/pkg/device"
	"versionary-api/pkg/event"
	"versionary-api/pkg/org"
	"versionary-api/pkg/token"
	"versionary-api/pkg/user"
	"versionary-api/pkg/view"
)

// About provides basic information about the API.
type About struct {
	Name        string    `json:"name"`
	BaseDomain  string    `json:"baseDomain"`
	GitHash     string    `json:"gitHash,omitempty"`
	BuildTime   time.Time `json:"buildTime"`
	Language    string    `json:"language"`
	Environment string    `json:"environment"`
	Description string    `json:"description,omitempty"`
}

// String supports the Stringer interface.
func (a About) String() string {
	return fmt.Sprintf("Name: %s\nGitHash: %s\nBuildTime: %s\nLanguage: %s\nEnvironment: %s\nDescription: %s",
		a.Name, a.GitHash, a.BuildTime, a.Language, a.Environment, a.Description)
}

// Application is the main application object, which contains configuration settings, keys, and initialized services.
type Application struct {
	Name               string                // Name of the application
	GitHash            string                // Git hash of the application
	BuildTime          time.Time             // Executable build time
	Language           string                // Go Compiler version (e.g. "go1.x")
	Environment        string                // Environment name (e.g. "dev", "test", "staging", "prod")
	BaseDomain         string                // Base domain for the application (e.g. "versionary.net")
	AdminURL           string                // Admin App URL (e.g. "https://admin.versionary.net")
	APIURL             string                // API URL (e.g. "https://api.versionary.net")
	WebURL             string                // Web URL (e.g. "https://www.versionary.net")
	Description        string                // Description of the application
	EntityTypes        []string              // Valid entity type names (e.g. "Event", "User", etc.)
	AWSConfig          aws.Config            // AWS Configuration
	DBClient           *dynamodb.Client      // DynamoDB client
	ParameterStore     client.ParameterStore // AWS SSM Parameter Store client
	DeviceService      device.DeviceService
	DeviceCountService device.CountService
	EventService       event.EventService
	OrgService         org.OrganizationService
	TokenService       token.TokenService
	UserService        user.UserService
	ViewService        view.ViewService
	ViewCountService   view.CountService
}

// About returns basic information about the initialized Application.
func (a *Application) About() About {
	return About{
		Name:        a.Name,
		BaseDomain:  a.BaseDomain,
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
	if a.BuildTime.IsZero() {
		p, err := os.Executable()
		if err == nil {
			s, err := os.Stat(p)
			if err == nil {
				a.BuildTime = s.ModTime()
			}
		}
	}
	if a.Language == "" {
		a.Language = runtime.Version() + " (" + runtime.GOOS + " " + runtime.GOARCH + ")"
	}
	if a.Environment == "" {
		a.Environment = "dev"
	}
	if a.BaseDomain == "" {
		a.BaseDomain = "versionary.net"
	}
	if a.AdminURL == "" {
		if a.Environment == "prod" {
			a.AdminURL = "https://admin." + a.BaseDomain
		} else {
			a.AdminURL = "https://admin-" + a.Environment + "." + a.BaseDomain
		}
	}
	if a.APIURL == "" {
		if a.Environment == "prod" {
			a.APIURL = "https://api." + a.BaseDomain
		} else {
			a.APIURL = "https://api-" + a.Environment + "." + a.BaseDomain
		}
	}
	if a.WebURL == "" {
		if a.Environment == "prod" {
			a.WebURL = "https://www." + a.BaseDomain
		} else {
			a.WebURL = "https://www-" + a.Environment + "." + a.BaseDomain
		}
	}

	// Entity Types
	// TODO: Update this list and initialize new services below as new entity types are added
	a.EntityTypes = []string{
		"Device",
		"DeviceCount",
		"Event",
		"Organization",
		"Token",
		"User",
		"View",
		"ViewCount",
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
	a.ParameterStore = client.NewParameterStore(cfg)

	// Initialize Services
	a.DeviceService = device.DeviceService{
		EntityType: "Device",
		Table:      device.NewDeviceTable(a.DBClient, a.Environment),
	}
	a.DeviceCountService = device.CountService{
		EntityType: "DeviceCount",
		Table:      device.NewDeviceCountTable(a.DBClient, a.Environment),
	}
	a.EventService = event.EventService{
		EntityType: "Event",
		Table:      event.NewEventTable(a.DBClient, a.Environment),
	}
	a.OrgService = org.OrganizationService{
		EntityType: "Organization",
		Table:      org.NewOrganizationTable(a.DBClient, a.Environment),
	}
	a.TokenService = token.TokenService{
		EntityType: "Token",
		Table:      token.NewTokenTable(a.DBClient, a.Environment),
	}
	a.UserService = user.UserService{
		EntityType: "User",
		Table:      user.NewUserTable(a.DBClient, a.Environment),
	}
	a.ViewService = view.ViewService{
		EntityType: "View",
		Table:      view.NewViewTable(a.DBClient, a.Environment),
	}
	a.ViewCountService = view.CountService{
		EntityType: "ViewCount",
		Table:      view.NewViewCountTable(a.DBClient, a.Environment),
	}

	// fmt.Println("Initialized Application in ", time.Since(startTime))
	return nil
}

// InitMock initializes the application for testing, including mock clients and services.
func (a *Application) InitMock(env string) error {
	// Set default values for the application
	a.Environment = env
	a.setDefaults()

	// Initialize Mock Clients
	a.ParameterStore = client.NewParameterStoreMock()

	// Initialize Services
	a.DeviceService = device.DeviceService{
		EntityType: "Device",
		Table:      device.NewDeviceMemTable(device.NewDeviceTable(a.DBClient, a.Environment)),
	}
	a.DeviceCountService = device.CountService{
		EntityType: "DeviceCount",
		Table:      device.NewDeviceCountMemTable(device.NewDeviceCountTable(a.DBClient, a.Environment)),
	}
	a.EventService = event.EventService{
		EntityType: "Event",
		Table:      event.NewEventMemTable(event.NewEventTable(a.DBClient, a.Environment)),
	}
	a.OrgService = org.OrganizationService{
		EntityType: "Organization",
		Table:      org.NewOrganizationMemTable(org.NewOrganizationTable(a.DBClient, a.Environment)),
	}
	a.TokenService = token.TokenService{
		EntityType: "Token",
		Table:      token.NewTokenMemTable(token.NewTokenTable(a.DBClient, a.Environment)),
	}
	a.UserService = user.UserService{
		EntityType: "User",
		Table:      user.NewUserMemTable(user.NewUserTable(a.DBClient, a.Environment)),
	}
	a.ViewService = view.ViewService{
		EntityType: "View",
		Table:      view.NewViewMemTable(view.NewViewTable(a.DBClient, a.Environment)),
	}
	a.ViewCountService = view.CountService{
		EntityType: "ViewCount",
		Table:      view.NewViewCountMemTable(view.NewViewCountTable(a.DBClient, a.Environment)),
	}

	return nil
}

// TokenUser reads a specified Token and its associated User.
func (a *Application) TokenUser(ctx context.Context, tokenID string) (token.Token, user.User, error) {
	// Validate the Application
	if a.TokenService.Table == nil || a.UserService.Table == nil {
		return token.Token{}, user.User{}, fmt.Errorf("application not initialized")
	}
	// Validate the bearer token
	if tokenID == "" || !tuid.IsValid(tuid.TUID(tokenID)) {
		return token.Token{}, user.User{}, fmt.Errorf("invalid bearer token")
	}
	// Read the specified token
	t, err := a.TokenService.Read(ctx, tokenID)
	if err != nil {
		// tokens expire, so this will be a common response
		return token.Token{}, user.User{}, fmt.Errorf("error reading token: %w", err)
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
