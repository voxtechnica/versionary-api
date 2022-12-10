package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
	gin "github.com/gin-gonic/gin"
	"github.com/voxtechnica/tuid-go"

	"versionary-api/pkg/app"
	"versionary-api/pkg/event"
	"versionary-api/pkg/token"
	"versionary-api/pkg/user"
)

// gitHash provides the git hash of the compiled application.
// It is embedded in the binary and is automatically updated by the build process.
// go build -ldflags "-X main.gitHash=`git rev-parse HEAD`"
var gitHash string

// gitOrigin provides the git origin of the compiled application.
// It is embedded in the binary and is automatically updated by the build process.
// go build -ldflags "-X main.gitOrigin=`git config --get remote.origin.url`"
var gitOrigin string

// api is the application object, containing global configuration settings and initialized services.
var api = app.Application{
	Name:        "Versionary API",
	BaseDomain:  "versionary.net",
	Description: "Versionary API demonstrates a way to manage versioned entities in a database with a serverless architecture.",
	GitHash:     gitHash,
}

// main is the entry point for the application. It can run as either as an AWS Lambda function with an API Gateway
// proxy, or as a command-line application, serving requests on localhost for local development, debugging, etc.
func main() {
	startTime := time.Now()

	// Flag: application version
	var version bool
	flag.BoolVar(&version, "version", false, "Print version and exit")

	// Flag: environment stage (default is either the STAGE_NAME environment variable or "dev")
	env := os.Getenv("STAGE_NAME")
	if env == "" {
		env = "dev"
	}
	flag.StringVar(&env, "env", env, "Operating Environment <dev | qa | staging | prod>")

	// Initialize the application, including required services:
	flag.Parse()
	err := api.Init(env)
	if err != nil {
		log.Fatal(err)
	}

	// Show application version
	if version {
		fmt.Println(api.About())
		os.Exit(0)
	}

	// Set up the Gin Router
	router := gin.New()
	router.Use(gin.Recovery())
	if env == "dev" {
		gin.SetMode(gin.DebugMode)
		router.Use(gin.Logger())
	} else {
		gin.SetMode(gin.ReleaseMode)
		gin.DisableConsoleColor()
	}

	// Add the API endpoints
	registerRoutes(router)

	// Identify operating environment (AWS or on localhost)
	_, ok := os.LookupEnv("LAMBDA_TASK_ROOT")
	if ok {
		// Run API as an AWS Lambda function with an API Gateway proxy
		router.TrustedPlatform = "X-Forwarded-For"
		ginLambda := ginadapter.NewV2(router)
		lambda.Start(func(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
			return ginLambda.ProxyWithContext(ctx, req)
		})
	} else {
		// Run API on localhost for local development, debugging, etc.
		_ = router.SetTrustedProxies(nil) // disable IP allow list
		log.Println("AWS Region:", api.AWSConfig.Region)
		log.Println("Environment Stage:", env)
		log.Println("Initialized in", time.Since(startTime))
		log.Fatal(router.Run(":8080"))
	}
}

// registerRoutes initializes all the API endpoints.
// Placing these registrations in a separate function allows
// them to be called from both the main and TestMain functions.
func registerRoutes(r *gin.Engine) {
	r.Use(bearerTokenHandler())
	r.NoRoute(notFound)
	registerContentRoutes(r)
	registerDeviceRoutes(r)
	registerEmailRoutes(r)
	registerEventRoutes(r)
	registerImageRoutes(r)
	registerOrganizationRoutes(r)
	registerTokenRoutes(r)
	registerTuidRoutes(r)
	registerUserRoutes(r)
	registerViewRoutes(r)
	registerDiagRoutes(r)
}

// notFound handles a request for a non-existent API endpoint.
func notFound(c *gin.Context) {
	c.JSON(http.StatusNotFound, APIEvent{
		CreatedAt: time.Now(),
		LogLevel:  "ERROR",
		Code:      http.StatusNotFound,
		Message:   "not found: API endpoint",
		URI:       c.Request.URL.String(),
	})
}

// abortWithError aborts the request with the specified error.
func abortWithError(c *gin.Context, code int, err error) {
	var e event.Event
	if errors.As(err, &e) {
		c.AbortWithStatusJSON(code, APIEvent{
			EventID:   e.ID,
			CreatedAt: e.CreatedAt,
			LogLevel:  e.LogLevel.String(),
			Code:      code,
			Message:   e.Message,
			URI:       e.URI,
		})
	} else {
		c.AbortWithStatusJSON(code, APIEvent{
			CreatedAt: time.Now(),
			LogLevel:  "ERROR",
			Code:      code,
			Message:   err.Error(),
			URI:       c.Request.URL.String(),
		})
	}
}

// bearerTokenHandler is a middleware function that reads a Bearer token, adding both the Token
// and the associated User to the request. If an error occurs, nothing is added to the request
// and processing continues. Authorization, if required, should be handled by a subsequent handler.
func bearerTokenHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if h != "" {
			b, a, f := strings.Cut(strings.TrimSpace(h), " ")
			if f && strings.ToLower(b) == "bearer" && len(a) > 0 {
				t, u, err := tokenUser(c, a)
				if err != nil {
					abortWithError(c, http.StatusUnauthorized, err)
					return
				} else {
					c.Set("token", t)
					c.Set("user", u.Scrub())
				}
			}
		}
		c.Next()
	}
}

// tokenUser reads a specified Token and its associated User.
func tokenUser(ctx context.Context, tokenID string) (token.Token, user.User, error) {
	// Validate the Application
	if api.TokenService.Table == nil || api.UserService.Table == nil {
		return token.Token{}, user.User{}, fmt.Errorf("application not initialized")
	}
	// Validate the bearer token
	if tokenID == "" || !tuid.IsValid(tuid.TUID(tokenID)) {
		return token.Token{}, user.User{}, fmt.Errorf("invalid bearer token")
	}
	// Read the specified token
	t, err := api.TokenService.Read(ctx, tokenID)
	if err != nil {
		// tokens expire, so this will be a common response
		return token.Token{}, user.User{}, fmt.Errorf("error reading token: %w", err)
	}
	// Read the associated user
	u, err := api.UserService.Read(ctx, t.UserID)
	if err != nil {
		return t, user.User{}, fmt.Errorf("error reading user %s from token: %w", t.UserID, err)
	}
	// Check that the user is active (not disabled)
	if u.Status == user.DISABLED {
		return t, u, fmt.Errorf("user %s status is %s", u.ID, u.Status)
	}
	return t, u, nil
}

// userAuthenticator is a middleware function that ensures that the request is authenticated.
// If the user is not present (no valid bearer token), the request is aborted with a 401 Unauthorized status.
// If an authenticated user is present, the processing continues. Anonymous requests are rejected.
// In many cases, the user will be authorized for their content in a subsequent handler function.
func userAuthenticator() gin.HandlerFunc {
	return func(c *gin.Context) {
		_, ok := c.Get("user")
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, APIEvent{
				CreatedAt: time.Now(),
				LogLevel:  "ERROR",
				Code:      http.StatusUnauthorized,
				Message:   "unauthenticated",
				URI:       c.Request.URL.String(),
			})
			return
		}
		c.Next()
	}
}

// roleAuthorizer is a middleware function that checks the request for a user with the specified role.
// If the user is not present (no valid bearer token), the request is aborted with a 401 Unauthorized status.
// If the user does not have the specified role, the request is aborted with a 403 Forbidden status.
// If the user is an administrator or has the specified role, then processing continues.
func roleAuthorizer(r string) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, ok := c.Get("user")
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, APIEvent{
				CreatedAt: time.Now(),
				LogLevel:  "ERROR",
				Code:      http.StatusUnauthorized,
				Message:   "unauthenticated",
				URI:       c.Request.URL.String(),
			})
			return
		}
		if u.(user.User).HasRole(r) {
			c.Next()
		} else {
			c.AbortWithStatusJSON(http.StatusForbidden, APIEvent{
				CreatedAt: time.Now(),
				LogLevel:  "ERROR",
				Code:      http.StatusForbidden,
				Message:   "unauthorized",
				URI:       c.Request.URL.String(),
			})
		}
	}
}

// contextToken returns the typed Token associated with the request.
func contextToken(c *gin.Context) (token.Token, bool) {
	t, ok := c.Get("token")
	if !ok {
		return token.Token{}, false
	}
	return t.(token.Token), true
}

// contextUser returns the typed User associated with the request.
func contextUser(c *gin.Context) (user.User, bool) {
	u, ok := c.Get("user")
	if !ok {
		return user.User{}, false
	}
	return u.(user.User), true
}

// contextUserID returns the UserID associated with the request.
// An empty string indicates that the request is anonymous.
func contextUserID(c *gin.Context) string {
	u, ok := c.Get("user")
	if !ok {
		return ""
	}
	return u.(user.User).ID
}

// paginationParams parses pagination query parameters (reverse, limit, offset), with supplied defaults.
func paginationParams(c *gin.Context, reverse bool, limit int) (bool, int, string, error) {
	var err error
	// Reverse
	r := c.Query("reverse")
	if r != "" {
		reverse, err = strconv.ParseBool(r)
		if err != nil {
			return reverse, limit, "", fmt.Errorf("bad request: invalid parameter, reverse: %w", err)
		}
	}
	// Limit
	l := c.Query("limit")
	if l != "" {
		limit, err = strconv.Atoi(l)
		if err != nil || limit < 1 {
			return reverse, limit, "", fmt.Errorf("bad request: invalid parameter, limit: %s", l)
		}
	}
	// Offset
	offset := c.Query("offset")
	if offset == "" {
		if reverse {
			offset = "|" // after letters
		} else {
			offset = "-" // before numbers
		}
	}
	return reverse, limit, offset, err
}

// gitCommitURL returns the URL for the commit of the compiled application.
// Example: git@github.com:voxtechnica/versionary-api.git is converted to something like:
// https://github.com/voxtechnica/versionary-api/commit/23ff1ad8e3c6beb5332ed320f6605132a993e13b
// Note: if you use a service other than GitHub, you may need to modify this function.
func gitCommitURL() string {
	if gitOrigin == "" || gitHash == "" {
		return ""
	}
	baseURL := strings.TrimSuffix(gitOrigin, ".git")
	baseURL = strings.ReplaceAll(baseURL, ":", "/")
	baseURL = strings.ReplaceAll(baseURL, "///", "//")
	baseURL = strings.Replace(baseURL, "git@", "https://", 1)
	return baseURL + "/commit/" + gitHash
}

// APIEvent is a summary of an event.Event, used for API error responses.
type APIEvent struct {
	EventID   string    `json:"eventID,omitempty"`
	CreatedAt time.Time `json:"createdAt,omitempty"`
	LogLevel  string    `json:"logLevel"`
	Code      int       `json:"code"`
	Message   string    `json:"message"`
	URI       string    `json:"uri,omitempty"`
}

// String returns a string representation of the APIEvent.
func (e APIEvent) String() string {
	if e.Code != 0 {
		return fmt.Sprintf("%s %d %s", e.LogLevel, e.Code, e.Message)
	}
	return e.Message
}

// Error returns a string representation of the APIEvent, supporting the error interface.
func (e APIEvent) Error() string {
	return e.String()
}
