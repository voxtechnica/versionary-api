package email

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
)

var svc Service

func TestMain(m *testing.M) {
	// AWS SES Client testing requires AWS credentials to be set in the environment
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal("error loading AWS config:", err)
	}

	// Set up an in-memory table for testing
	table := NewMemTable(NewTable(nil, "test"))
	if !table.IsValid() {
		log.Fatal("error initializing in-memory table")
	}

	// Set up the Email Service
	svc = Service{
		EntityType: "Email",
		Client:     ses.NewFromConfig(cfg),
		Table:      table,
		DefaultFrom: Identity{
			Name:    "Test Account",
			Address: "noreply@versionary.net",
		},
		DefaultSubject: "Test Message",
		SafeDomains:    []string{"simulator.amazonses.com"},
		LimitSending:   true,
	}

	// Run the tests
	m.Run()
}

func TestSend(t *testing.T) {
	expect := assert.New(t)
	ctx := context.Background()
	to := Identity{
		Name:    "Test Recipient",
		Address: "success@simulator.amazonses.com",
	}
	body := "This is a test message,\nsplit over two lines.\n"
	var e Email
	var problems []string
	var err error

	// Create a new email, send it to the simulator, and verify that it was sent (happy path)
	e = Email{To: []Identity{to}, BodyText: body}
	e, problems, err = svc.Create(ctx, e)
	expect.Empty(problems)
	if expect.NoError(err) {
		expect.NotEmpty(e.ID)
		expect.Equal(SENT, e.Status)
		expect.Contains(e.EventMessage, "sent message")
	}
}
