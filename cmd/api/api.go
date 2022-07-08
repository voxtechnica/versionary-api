package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	"versionary-api/pkg/app"
	"versionary-api/pkg/user"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
	"github.com/gin-gonic/gin"
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
	Description: "Versionary API demonstrates a way to manage versioned entities in a database with a serverless architecture.",
	GitHash:     gitHash,
}

// main is the entry point for the application.
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

	// Setup the Gin Router
	r := gin.New()
	r.Use(gin.Recovery())
	if env == "dev" {
		gin.SetMode(gin.DebugMode)
		r.Use(gin.Logger())
	} else {
		gin.SetMode(gin.ReleaseMode)
		gin.DisableConsoleColor()
	}

	// Add the API endpoints
	initRoutes(r)

	// Identify operating environment (AWS or on localhost)
	_, ok := os.LookupEnv("LAMBDA_TASK_ROOT")
	if ok {
		// Run API as an AWS Lambda function with an API Gateway proxy
		r.TrustedPlatform = "X-Forwarded-For"
		ginLambda := ginadapter.NewV2(r)
		lambda.Start(func(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
			return ginLambda.ProxyWithContext(ctx, req)
		})
	} else {
		// Run API on localhost for local development, debugging, etc.
		_ = r.SetTrustedProxies(nil) // disable IP allow list
		log.Println("Environment Stage:", env)
		log.Println("Initialized in ", time.Since(startTime))
		log.Fatal(r.Run(":8080"))
	}
}

// initRoutes initializes all of the API endpoints.
func initRoutes(r *gin.Engine) {
	r.Use(bearerTokenHandler())
	initTokenRoutes(r)
	initTuidRoutes(r)
	initDiagRoutes(r)
	r.NoRoute(notFound)
}

// notFound handles a request for a non-existent API endpoint.
func notFound(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
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
				t, u, err := api.TokenUser(c, a)
				if err == nil {
					c.Set("token", t)
					c.Set("user", u.Scrub())
				}
			}
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
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":  http.StatusUnauthorized,
				"error": "unauthenticated",
			})
			return
		}
		if u.(user.User).HasRole(r) {
			c.Next()
		} else {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":  http.StatusForbidden,
				"error": "unauthorized",
			})
		}
	}
}

// isAnonymous returns true if the context does not have a logged-in user.
func isAnonymous(c *gin.Context) bool {
	_, ok := c.Get("user")
	return !ok
}

// contextToken returns the typed Token associated with the request.
func contextToken(c *gin.Context) (user.Token, bool) {
	t, ok := c.Get("token")
	if !ok {
		return user.Token{}, false
	}
	return t.(user.Token), true
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

// contextUserHasRole returns true if the context user exists and has the specified role.
func contextUserHasRole(c *gin.Context, r string) bool {
	u, ok := c.Get("user")
	return ok && u.(user.User).HasRole(r)
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
