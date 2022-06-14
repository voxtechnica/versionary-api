package main

import (
	"context"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/awslabs/aws-lambda-go-api-proxy/gin"
	"github.com/gin-gonic/gin"
	"github.com/voxtechnica/tuid-go"
	user_agent "github.com/voxtechnica/user-agent"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
	"versionary-api/pkg/event"
)

func main() {
	startTime := time.Now()
	// Identify operating environment (dev, test, staging, prod)
	env := os.Getenv("STAGE_NAME")
	if env == "" {
		env = "dev"
	}
	println("STAGE_NAME: " + env)

	// Initialize required services:
	ctx := context.Background()
	awsCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal("Failed to load AWS config: " + err.Error())
	}
	db := dynamodb.NewFromConfig(awsCfg)
	eventTable := event.NewEventTable(db, env)
	eventService := event.EventService{
		EntityType: "Event",
		Table:      eventTable,
	}
	log.Println("Initialized", eventService.EntityType, "service")

	// Setup Gin Routes
	g := gin.Default()
	if env == "dev" {
		g.Use(gin.Logger())
		g.Use(gin.Recovery())
	} else {
		gin.SetMode(gin.ReleaseMode)
		gin.DisableConsoleColor()
	}
	g.GET("/", GetAbout)
	g.GET("/user_agent", GetUserAgent)
	g.POST("/v1/tuids", PostTUID)
	g.GET("/v1/tuids", GetTUIDs)
	g.GET("/v1/tuids/:id", GetTUID)
	g.NoRoute(func(c *gin.Context) { c.JSON(404, gin.H{"message": "not found"}) })

	// Identify operating environment (AWS or on localhost)
	_, ok := os.LookupEnv("LAMBDA_TASK_ROOT")
	if ok {
		// Run API as an AWS Lambda function with an API Gateway proxy
		ginLambda := ginadapter.NewV2(g)
		lambda.Start(func(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
			return ginLambda.ProxyWithContext(ctx, req)
		})
	} else {
		// Run API on localhost for local development, debugging, etc.
		log.Println("Initialized API in ", time.Since(startTime))
		log.Fatal(g.Run(":8080"))
	}
}

func GetAbout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Versionary API"})
}

func GetUserAgent(c *gin.Context) {
	header := c.Request.Header.Get("User-Agent")
	ua := user_agent.Parse(header)
	c.JSON(http.StatusOK, ua)
}

func GetTUIDs(c *gin.Context) {
	limit := c.DefaultQuery("limit", "5")
	intLimit, err := strconv.Atoi(limit)
	if err != nil {
		intLimit = 5
	}
	ids := make([]tuid.TUIDInfo, intLimit)
	for i := 0; i < intLimit; i++ {
		ids[i], err = tuid.NewID().Info()
		if err != nil {
			c.JSON(http.StatusInternalServerError, err)
			return
		}
	}
	c.JSON(http.StatusOK, ids)
}

func GetTUID(c *gin.Context) {
	id := tuid.TUID(c.Param("id"))
	info, err := id.Info()
	if err != nil {
		c.JSON(http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, info)
}

func PostTUID(c *gin.Context) {
	t := tuid.NewID()
	info, err := t.Info()
	if err != nil {
		c.JSON(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, info)
}
