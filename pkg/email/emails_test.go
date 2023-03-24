package email

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/stretchr/testify/assert"
	"github.com/voxtechnica/tuid-go"
)

var svc Service
var (
	ctx = context.Background()

	// Known timestamps
	t1 = time.Date(2022, time.April, 1, 12, 0, 0, 0, time.UTC)
	t2 = time.Date(2022, time.April, 1, 13, 0, 0, 0, time.UTC)
	t3 = time.Date(2022, time.April, 1, 14, 0, 0, 0, time.UTC)

	// User IDs
	id1 = tuid.NewIDWithTime(t1).String()
	id2 = tuid.NewIDWithTime(t2).String()
	id3 = tuid.NewIDWithTime(t3).String()

	// Knowns emails
	e1 = Email{
		ID:        id1,
		CreatedAt: t1,
		VersionID: id1,
		UpdatedAt: t1,
		From: Identity{
			Name:    "QA Team",
			Address: "qa_team@test.net",
		},
		To: []Identity{{
			Name:    "Dev Team",
			Address: "dev_team@test.net",
		}},
		Subject:  "Test Message 1",
		BodyText: "This is a test message from QA team to Dev team",
		Status:   SENT,
	}

	e2 = Email{
		ID:        id2,
		CreatedAt: t2,
		VersionID: id2,
		UpdatedAt: t2,
		From: Identity{
			Name:    "Sales Team",
			Address: "sales_team@test.net",
		},
		To: []Identity{{
			Name:    "Support Team",
			Address: "support_team@test.net",
		}},
		Subject:  "Test Message 2",
		BodyText: "This is a test message from Sales team to Support team",
		Status:   UNSENT,
	}

	e3 = Email{
		ID:        id3,
		CreatedAt: t3,
		VersionID: id3,
		UpdatedAt: t3,
		From: Identity{
			Name:    "HR Team",
			Address: "hr_team@test.net",
		},
		To: []Identity{{
			Name:    "Finance Team",
			Address: "finance_team@test.net",
		}},
		Subject:  "Test Message 3",
		BodyText: "This is a test message from HR team to Finance team",
		Status:   ERROR,
	}

	knownEmails       = []Email{e1, e2, e3}
	knownEmailIDs     = []string{id1, id2, id3}
	knownTestStatuses = []string{"ERROR", "SENT", "UNSENT"}
)

func TestMain(m *testing.M) {
	// AWS SES Client testing requires AWS credentials to be set in the environment
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

	// Write the known emails to the table
	for _, e := range []Email{e1, e2, e3} {
		if _, err := svc.Write(ctx, e); err != nil {
			log.Fatal("error writing email to table:", err)
		}
	}

	// Run the tests
	m.Run()
}

func TestSend(t *testing.T) {
	expect := assert.New(t)
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

func TestEmailCRUD(t *testing.T) {
	expect := assert.New(t)
	to := Identity{
		Name:    "CRUD Test Recipient",
		Address: "hr_team@test.net",
	}
	body := "This is CRUD test message,\nsplit over two lines.\n"
	var e Email
	var problems []string
	var err error

	// Create a new email
	e = Email{To: []Identity{to}, BodyText: body}
	e, problems, err = svc.Create(ctx, e)
	expect.Empty(problems)
	if expect.NoError(err) {
		expect.NotEmpty(e.ID)
	}

	// Read the email back
	e, err = svc.Read(ctx, e.ID)
	if expect.NoError(err) {
		expect.Equal(to, e.To[0])
		expect.Equal(body, e.BodyText)
	}

	// Update the email
	e.BodyText = "This is a new body"
	eUpdated, _, err := svc.Update(ctx, e)
	if expect.NoError(err) {
		expect.Equal("This is a new body", eUpdated.BodyText)
	}

	// Delete the email
	eDeleted, err := svc.Delete(ctx, e.ID)
	if expect.NoError(err) {
		expect.Equal(eUpdated.ID, eDeleted.ID)
	}

	// User does not exist
	eExist := svc.Exists(ctx, eDeleted.ID)
	expect.False(eExist)
}


func TestReadAsJSON(t *testing.T) {
	expect := assert.New(t)
	eCheckJSON, err := svc.ReadAsJSON(context.Background(), id1)
	if expect.NoError(err) {
		expect.Contains(string(eCheckJSON), id1)
	}
}

func TestVersionExists(t *testing.T) {
	expect := assert.New(t)
	vExists := svc.VersionExists(ctx, id3, id3)
	expect.True(vExists)
}

func TestReadVersion(t *testing.T) {
	expect := assert.New(t)
	vExist, err := svc.ReadVersion(ctx, id1, id1)
	if expect.NoError(err) {
		expect.Equal(e1, vExist)
	}
}

func TestReadVersionAsJSON(t *testing.T) {
	expect := assert.New(t)
	vJSON, err := svc.ReadVersionAsJSON(ctx, id3, id3)
	if expect.NoError(err) {
		expect.Contains(string(vJSON), id3)
	}
}

func TestReadVersions(t *testing.T) {
	expect := assert.New(t)
	vExist, err := svc.ReadVersions(ctx, id1, false, 1, "")
	if expect.NoError(err) && expect.NotEmpty(vExist) {
		expect.Equal(e1, vExist[0])
	}
}

func TestReadVersionsAsJSON(t *testing.T) {
	expect := assert.New(t)
	vJSON, err := svc.ReadVersionAsJSON(ctx, id2, id2)
	if expect.NoError(err) {
		expect.Contains(string(vJSON), id2)
	}
}

func TestReadAllVersions(t *testing.T) {
	expect := assert.New(t)
	allVersions, err := svc.ReadAllVersions(ctx, id2)
	if expect.NoError(err) && expect.NotEmpty(allVersions) {
		expect.Equal(e2, allVersions[0])
	}
}

func TestReadAllVersionsAsJSON(t *testing.T) {
	expect := assert.New(t)
	versionsCheckJSON, err := svc.ReadAllVersionsAsJSON(ctx, id3)
	if expect.NoError(err) {
		expect.Contains(string(versionsCheckJSON), id3)
	}
}

func TestReadIDs(t *testing.T) {
	expect := assert.New(t)
	ids, err := svc.ReadEmailIDs(ctx, false, 10, tuid.MinID)
	if expect.NoError(err) {
		expect.GreaterOrEqual(len(ids), 3)
		expect.Subset(ids, knownEmailIDs)
	}
}
func TestReadEmailSubjects(t *testing.T) {
	expect := assert.New(t)
	idsAndSubjects, err := svc.ReadEmailSubjects(ctx, false, 10, "")
	if expect.NoError(err) {
		expect.GreaterOrEqual(len(idsAndSubjects), 3)
	}
}

func TestReadAllEmailSubjects(t *testing.T) {
	expect := assert.New(t)
	idsAndSubjects, err := svc.ReadAllEmailSubjects(ctx, true)
	if expect.NoError(err) {
		expect.GreaterOrEqual(len(idsAndSubjects), 3)
	}
}

func TestFilterEmailSubjects(t *testing.T) {
	expect := assert.New(t)
	idsAndSubjects, err := svc.FilterEmailSubjects(ctx, "3", true)
	if expect.NoError(err) && expect.NotEmpty(idsAndSubjects) {
		expect.Equal(1, len(idsAndSubjects))
	}
}

func TestFilterEmailSubjectsNegative(t *testing.T) {
	expect := assert.New(t)
	idsAndSubjects, err := svc.FilterEmailSubjects(ctx, "no such subject", false)
	if expect.NoError(err) {
		expect.Empty(idsAndSubjects)
	}
}

func TestReadEmails(t *testing.T) {
	expect := assert.New(t)
	emails := svc.ReadEmails(ctx, false, 5, "")
	if expect.NotEmpty(emails) {
		expect.GreaterOrEqual(len(emails), 4)
		expect.Subset(emails, knownEmails)
	}
}

//------------------------------------------------------------------------------
// Emails by Address
//------------------------------------------------------------------------------

func TestReadAddresses(t *testing.T) {
	expect := assert.New(t)
	addresses, err := svc.ReadAddresses(ctx, false, 10, "")
	if expect.NoError(err) {
		expect.GreaterOrEqual(len(addresses), 8)
	}
}

func TestReadAllAddresses(t *testing.T) {
	expect := assert.New(t)
	addresses, err := svc.ReadAllAddresses(ctx)
	if expect.NoError(err) && expect.NotEmpty(addresses) {
		expect.GreaterOrEqual(len(addresses), 8)
		expect.Equal("dev_team@test.net", addresses[0])
	}
}

func TestReadEmailsByAddress(t *testing.T) {
	expect := assert.New(t)
	emailByAddress, err := svc.ReadEmailsByAddress(ctx, "sales_team@test.net", false, 1, "")
	if expect.NoError(err) && expect.NotEmpty(emailByAddress) {
		expect.Equal(emailByAddress[0], e2)
	}
}

func TestReadEmailsByAddressNegative(t *testing.T) {
	expect := assert.New(t)
	emailByAddress, err := svc.ReadEmailsByAddress(ctx, "fake_team@test.net", false, 1, "")
	if expect.NoError(err) {
		expect.Empty(emailByAddress)
	}
}

func TestReadEmailsByAddressAsJSON(t *testing.T) {
	expect := assert.New(t)
	jEmails, err := svc.ReadEmailsByAddressAsJSON(ctx, "qa_team@test.net", false, 1, "")
	if expect.NoError(err) && expect.NotEmpty(jEmails) {
		expect.Contains(string(jEmails), e1.From.Address)
	}
}

func TestReadAllEmailsByAddress(t *testing.T) {
	expect := assert.New(t)
	emails, err := svc.ReadAllEmailsByAddress(ctx, "qa_team@test.net")
	if expect.NoError(err) && expect.NotEmpty(emails) {
		expect.Equal(1, len(emails))
	}
}

func TestReadAllEmailsByAddressAsJSON(t *testing.T) {
	expect := assert.New(t)
	jEmails, err := svc.ReadAllEmailsByAddressAsJSON(ctx, "dev_team@test.net")
	if expect.NoError(err) && expect.NotEmpty(jEmails) {
		expect.Contains(string(jEmails), e1.To[0].Address)
	}
}

//------------------------------------------------------------------------------
// Emails by Status
//------------------------------------------------------------------------------

func TestReadStatuses(t *testing.T) {
	expect := assert.New(t)
	// Read statuses
	statuses, err := svc.ReadStatuses(ctx, false, 10, "-")
	if expect.NoError(err) {
		expect.Equal(statuses, knownTestStatuses)
	}
}

func TestReadAllStatuses(t *testing.T) {
	expect := assert.New(t)
	// Read all statuses
	statuses, err := svc.ReadAllStatuses(ctx)
	if expect.NoError(err) {
		expect.Equal(statuses, knownTestStatuses)
	}
}

func TestReadEmailsByStatus(t *testing.T) {
	expect := assert.New(t)
	checkEmails, err := svc.ReadEmailsByStatus(ctx, "ERROR", false, 1, "")
	if expect.NoError(err) && expect.NotEmpty(checkEmails) {
		expect.Equal(e3, checkEmails[0])
	}
}

func TestReadEmailsByStatusAsJSON(t *testing.T) {
	expect := assert.New(t)
	checkEmails, err := svc.ReadEmailsByStatusAsJSON(ctx, "UNSENT", false, 1, "")
	if expect.NoError(err) && expect.NotEmpty(checkEmails) {
		expect.Contains(string(checkEmails), e2.Status)
	}
}

func TestReadAllEmailsByStatus(t *testing.T) {
	expect := assert.New(t)
	checkEmails, err := svc.ReadAllEmailsByStatus(ctx, "ERROR")
	if expect.NoError(err) && expect.NotEmpty(checkEmails) {
		expect.Equal(1, len(checkEmails))
	}
}

func TestReadAllEmailsByStatusAsJSON(t *testing.T) {
	expect := assert.New(t)
	checkEmails, err := svc.ReadAllEmailsByStatusAsJSON(ctx, "ERROR")
	if expect.NoError(err) && expect.NotEmpty(checkEmails) {
		expect.Contains(string(checkEmails), e3.Status)
	}
}
