package main

import (
	"context"
	"log"
	"testing"
	"versionary-api/pkg/user"

	"github.com/gin-gonic/gin"
)

var r = gin.New()
var regularUser user.User
var regularToken string
var adminUser user.User
var adminToken string

func TestMain(m *testing.M) {
	// Initialize the application for testing
	err := api.InitMock("dev")
	if err != nil {
		log.Fatal(err)
	}

	// TODO: Populate the database with test data
	generateUsers()

	// Initialize the gin router
	r.Use(gin.Recovery())
	gin.SetMode(gin.TestMode)
	gin.DisableConsoleColor()
	registerRoutes(r)

	// Run the tests
	m.Run()
}

func generateUsers() {
	ctx := context.Background()
	// Regular API user (no special roles) and associated bearer token
	rUser, problems, err := api.UserService.Create(ctx, user.User{
		FirstName: "Regular",
		LastName:  "User",
		Email:     "info@versionary.net",
	})
	if err != nil || len(problems) > 0 {
		log.Fatal(err)
	}
	regularUser = rUser
	rToken, err := api.TokenService.Create(ctx, user.Token{
		UserID: rUser.ID,
		Email:  rUser.Email,
	})
	if err != nil {
		log.Fatal(err)
	}
	regularToken = rToken.ID

	// Admin API user (has aUser role) and associated bearer token
	aUser, problems, err := api.UserService.Create(ctx, user.User{
		FirstName: "Admin",
		LastName:  "User",
		Email:     "admin@versionary.net",
		Roles:     []string{"admin"},
		Status:    user.ENABLED,
	})
	if err != nil || len(problems) > 0 {
		log.Fatal(err)
	}
	adminUser = aUser
	aToken, err := api.TokenService.Create(ctx, user.Token{
		UserID: aUser.ID,
		Email:  aUser.Email,
	})
	if err != nil {
		log.Fatal(err)
	}
	adminToken = aToken.ID
}
