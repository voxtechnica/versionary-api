package main

import (
	"context"
	"log"
	"testing"

	"github.com/gin-gonic/gin"

	"versionary-api/pkg/org"
	"versionary-api/pkg/token"
	"versionary-api/pkg/user"
)

var r = gin.New()
var userOrg org.Organization
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
	// Test Organization
	org, problems, err := api.OrgService.Create(ctx, org.Organization{
		Name:   "Versionary",
		Status: org.ENABLED,
	})
	if err != nil || len(problems) > 0 {
		log.Fatal(err)
	}
	userOrg = org

	// Regular API user (no special roles) and associated bearer token
	rUser, problems, err := api.UserService.Create(ctx, user.User{
		GivenName:  "Regular",
		FamilyName: "User",
		Email:      "info@versionary.net",
		OrgID:      userOrg.ID,
		OrgName:    userOrg.Name,
		Roles:      []string{"creator"},
		Status:     user.PENDING,
	})
	if err != nil || len(problems) > 0 {
		log.Fatal(err)
	}
	regularUser = rUser
	rToken, err := api.TokenService.Create(ctx, token.Token{
		UserID: rUser.ID,
		Email:  rUser.Email,
	})
	if err != nil {
		log.Fatal(err)
	}
	regularToken = rToken.ID

	// Admin API user (has admin role) and associated bearer token
	aUser, problems, err := api.UserService.Create(ctx, user.User{
		GivenName:  "Admin",
		FamilyName: "User",
		Email:      "admin@versionary.net",
		Roles:      []string{"admin"},
		OrgID:      userOrg.ID,
		OrgName:    userOrg.Name,
		Status:     user.ENABLED,
	})
	if err != nil || len(problems) > 0 {
		log.Fatal(err)
	}
	adminUser = aUser
	aToken, err := api.TokenService.Create(ctx, token.Token{
		UserID: aUser.ID,
		Email:  aUser.Email,
	})
	if err != nil {
		log.Fatal(err)
	}
	adminToken = aToken.ID
}
