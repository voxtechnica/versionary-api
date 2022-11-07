package email

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
	"github.com/voxtechnica/tuid-go"
	v "github.com/voxtechnica/versionary"
	"strings"
)

//==============================================================================
// Email Table
//==============================================================================

// rowEmails is a TableRow definition for Email versions.
var rowEmails = v.TableRow[Email]{
	RowName:      "emails_version",
	PartKeyName:  "id",
	PartKeyValue: func(e Email) string { return e.ID },
	SortKeyName:  "update_id",
	SortKeyValue: func(e Email) string { return e.VersionID },
	JsonValue:    func(e Email) []byte { return e.CompressedJSON() },
}

// rowEmailsAddress is a TableRow definition for current Email versions, partitioned by Email address.
var rowEmailsAddress = v.TableRow[Email]{
	RowName:       "emails_address",
	PartKeyName:   "address",
	PartKeyValues: func(e Email) []string { return e.AllAddresses() },
	SortKeyName:   "id",
	SortKeyValue:  func(e Email) string { return e.ID },
	JsonValue:     func(e Email) []byte { return e.CompressedJSON() },
}

// rowEmailsStatus is a TableRow definition for current Email versions, partitioned by Email Status.
var rowEmailsStatus = v.TableRow[Email]{
	RowName:      "emails_status",
	PartKeyName:  "status",
	PartKeyValue: func(e Email) string { return e.Status.String() },
	SortKeyName:  "id",
	SortKeyValue: func(e Email) string { return e.ID },
	JsonValue:    func(e Email) []byte { return e.CompressedJSON() },
}

// NewTable instantiates a new DynamoDB Email table.
func NewTable(dbClient *dynamodb.Client, env string) v.Table[Email] {
	if env == "" {
		env = "dev"
	}
	return v.Table[Email]{
		Client:     dbClient,
		EntityType: "Email",
		TableName:  "emails" + "_" + env,
		TTL:        false,
		EntityRow:  rowEmails,
		IndexRows: map[string]v.TableRow[Email]{
			rowEmailsAddress.RowName: rowEmailsAddress,
			rowEmailsStatus.RowName:  rowEmailsStatus,
		},
	}
}

// NewMemTable creates an in-memory Email table for testing purposes.
func NewMemTable(table v.Table[Email]) v.MemTable[Email] {
	return v.NewMemTable(table)
}

//==============================================================================
// Email Service
//==============================================================================

// Service is a service for managing and sending Email messages.
// DynamoDB is used to store Email messages and their versions.
// SES is used to send Email messages.
type Service struct {
	EntityType     string
	Client         *ses.Client
	Table          v.TableReadWriter[Email]
	DefaultFrom    Identity // The default "from" address for outgoing emails.
	DefaultSubject string   // The default subject line for outgoing emails.
	SafeDomains    []string // Domains that are safe to send to in non-production environments.
	LimitSending   bool     // If true, limit sending emails to safe domains.
}

// Send sends an email message. Note that the email address domain
// simulator.amazonses.com can be used for testing purposes.
func (s Service) Send(ctx context.Context, e Email) (Email, error) {
	// Validate the Email message (again?)
	problems := e.Validate()
	if len(problems) > 0 {
		err := fmt.Errorf("error sending %s %s: invalid field(s): %s", s.EntityType, e.ID, strings.Join(problems, ", "))
		e.EventMessage = err.Error()
		e.Status = ERROR
		return e, err
	}

	// Only PENDING messages will be sent.
	if e.Status != PENDING {
		err := fmt.Errorf("error sending %s %s: invalid status %s (expect PENDING)", s.EntityType, e.ID, e.Status)
		e.EventMessage = err.Error()
		e.Status = ERROR
		return e, err
	}

	// Verify available recipients.
	to := s.filterRecipients(e.To)
	cc := s.filterRecipients(e.CC)
	bcc := s.filterRecipients(e.BCC)
	if len(to) == 0 && len(cc) == 0 && len(bcc) == 0 {
		// No recipients can happen in non-production environments.
		e.EventMessage = "no allowed recipients"
		e.Status = UNSENT
		return e, nil
	}

	// Verify that the email service is configured (it may not be in test environments).
	if s.Client == nil {
		e.EventMessage = "email service not configured"
		e.Status = UNSENT
		return e, nil
	}

	// Build the email message.
	var destination types.Destination
	if len(to) > 0 {
		destination.ToAddresses = v.Map(to, func(i Identity) string { return i.String() })
	}
	if len(cc) > 0 {
		destination.CcAddresses = v.Map(cc, func(i Identity) string { return i.String() })
	}
	if len(bcc) > 0 {
		destination.BccAddresses = v.Map(bcc, func(i Identity) string { return i.String() })
	}
	subject := types.Content{
		Charset: aws.String("UTF-8"),
		Data:    aws.String(e.Subject),
	}
	var body types.Body
	if e.BodyText != "" {
		body.Text = &types.Content{
			Charset: aws.String("UTF-8"),
			Data:    aws.String(e.BodyText),
		}
	}
	if e.BodyHTML != "" {
		body.Html = &types.Content{
			Charset: aws.String("UTF-8"),
			Data:    aws.String(e.BodyHTML),
		}
	}
	input := &ses.SendEmailInput{
		Source:      aws.String(e.From.String()),
		Destination: &destination,
		Message: &types.Message{
			Subject: &subject,
			Body:    &body,
		},
	}

	// Send the email message.
	output, err := s.Client.SendEmail(ctx, input)
	if err != nil {
		err = fmt.Errorf("error sending %s %s: %w", s.EntityType, e.ID, err)
		e.EventMessage = err.Error()
		e.Status = ERROR
		return e, err
	}
	e.EventMessage = "sent message " + *output.MessageId
	e.Status = SENT
	return e, nil
}

// filterRecipients returns a list of recipients that are allowed to receive emails.
func (s Service) filterRecipients(recipients []Identity) []Identity {
	if !s.LimitSending {
		return recipients
	}
	var filtered []Identity
	for _, r := range recipients {
		if s.isSafeDomain(r) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// isSafeDomain returns true if the email address domain is in the list of safe domains.
func (s Service) isSafeDomain(i Identity) bool {
	// Unrestricted sending in production.
	if !s.LimitSending {
		return true
	}
	// Completely restricted sending.
	if s.SafeDomains == nil {
		return false
	}
	// Restricted sending to safe domains.
	_, domain, found := strings.Cut(i.Address, "@")
	if !found {
		return false
	}
	for _, d := range s.SafeDomains {
		if d == domain {
			return true
		}
	}
	return false
}

//------------------------------------------------------------------------------
// Email Versions
//------------------------------------------------------------------------------

// Create an Email message in the Email table.
// If the message status is PENDING, then it will also be sent.
func (s Service) Create(ctx context.Context, e Email) (Email, []string, error) {
	t := tuid.NewID()
	at, _ := t.Time()
	e.ID = t.String()
	e.VersionID = t.String()
	e.CreatedAt = at
	e.UpdatedAt = at
	if !e.From.IsValid() {
		e.From = s.DefaultFrom
	}
	if e.Subject == "" {
		e.Subject = s.DefaultSubject
	}
	if !e.Status.IsValid() {
		e.Status = PENDING
	}
	problems := e.Validate()
	if len(problems) > 0 {
		return e, problems, fmt.Errorf("error creating %s %s: invalid field(s): %s", s.EntityType, e.ID, strings.Join(problems, ", "))
	}
	if e.Status == PENDING {
		// Send the email message, updating the status.
		e, _ = s.Send(ctx, e)
	}
	err := s.Table.WriteEntity(ctx, e)
	if err != nil {
		return e, problems, fmt.Errorf("error creating %s %s: %w", s.EntityType, e.ID, err)
	}
	return e, problems, nil
}

// Update an Email message in the Email table. If message status is PENDING, then it will also be sent.
// This can be used to retry sending a message (e.g. if it previously failed with a transient ERROR).
func (s Service) Update(ctx context.Context, e Email) (Email, []string, error) {
	t := tuid.NewID()
	at, _ := t.Time()
	e.VersionID = t.String()
	e.UpdatedAt = at
	problems := e.Validate()
	if len(problems) > 0 {
		return e, problems, fmt.Errorf("error updating %s %s: invalid field(s): %s", s.EntityType, e.ID, strings.Join(problems, ", "))
	}
	if e.Status == PENDING {
		// retry sending the email; results are reflected in the returned Email
		e, _ = s.Send(ctx, e)
	}
	err := s.Table.UpdateEntity(ctx, e)
	if err != nil {
		return e, problems, fmt.Errorf("error updating %s %s: %w", s.EntityType, e.ID, err)
	}
	return e, problems, nil
}

// Write an Email to the Email table. This method assumes that the Email has all the required fields.
// It would most likely be used for "refreshing" the index rows in the Email table.
func (s Service) Write(ctx context.Context, o Email) (Email, error) {
	return o, s.Table.WriteEntity(ctx, o)
}

// Delete an Email from the Email table. The deleted Email is returned.
func (s Service) Delete(ctx context.Context, id string) (Email, error) {
	return s.Table.DeleteEntityWithID(ctx, id)
}

// Exists checks if an Email exists in the Email table.
func (s Service) Exists(ctx context.Context, id string) bool {
	return s.Table.EntityExists(ctx, id)
}

// Read a specified Email from the Email table.
func (s Service) Read(ctx context.Context, id string) (Email, error) {
	return s.Table.ReadEntity(ctx, id)
}

// ReadAsJSON gets a specified Email from the Email table, serialized as JSON.
func (s Service) ReadAsJSON(ctx context.Context, id string) ([]byte, error) {
	return s.Table.ReadEntityAsJSON(ctx, id)
}

// VersionExists checks if a specified Email version exists in the Email table.
func (s Service) VersionExists(ctx context.Context, id, versionID string) bool {
	return s.Table.EntityVersionExists(ctx, id, versionID)
}

// ReadVersion gets a specified Email version from the Email table.
func (s Service) ReadVersion(ctx context.Context, id, versionID string) (Email, error) {
	return s.Table.ReadEntityVersion(ctx, id, versionID)
}

// ReadVersionAsJSON gets a specified Email version from the Email table, serialized as JSON.
func (s Service) ReadVersionAsJSON(ctx context.Context, id, versionID string) ([]byte, error) {
	return s.Table.ReadEntityVersionAsJSON(ctx, id, versionID)
}

// ReadVersions returns paginated versions of the specified Email.
// Sorting is chronological (or reverse). The offset is the last ID returned in a previous request.
func (s Service) ReadVersions(ctx context.Context, id string, reverse bool, limit int, offset string) ([]Email, error) {
	return s.Table.ReadEntityVersions(ctx, id, reverse, limit, offset)
}

// ReadVersionsAsJSON returns paginated versions of the specified Email, serialized as JSON.
// Sorting is chronological (or reverse). The offset is the last ID returned in a previous request.
func (s Service) ReadVersionsAsJSON(ctx context.Context, id string, reverse bool, limit int, offset string) ([]byte, error) {
	return s.Table.ReadEntityVersionsAsJSON(ctx, id, reverse, limit, offset)
}

// ReadAllVersions returns all versions of the specified Email in chronological order.
// Caution: this may be a LOT of data!
func (s Service) ReadAllVersions(ctx context.Context, id string) ([]Email, error) {
	return s.Table.ReadAllEntityVersions(ctx, id)
}

// ReadAllVersionsAsJSON returns all versions of the specified Email, serialized as JSON.
// Caution: this may be a LOT of data!
func (s Service) ReadAllVersionsAsJSON(ctx context.Context, id string) ([]byte, error) {
	return s.Table.ReadAllEntityVersionsAsJSON(ctx, id)
}

// ReadEmailIDs returns a paginated list of Email IDs in the Email table.
// Sorting is chronological (or reverse). The offset is the last ID returned in a previous request.
func (s Service) ReadEmailIDs(ctx context.Context, reverse bool, limit int, offset string) ([]string, error) {
	return s.Table.ReadEntityIDs(ctx, reverse, limit, offset)
}

// ReadEmails returns a paginated list of Emails in the Email table.
// Sorting is chronological (or reverse). The offset is the last ID returned in a previous request.
// Note that this is a best-effort attempt to return the requested Emails, retrieved individually, in parallel.
// It is probably not the best way to page through a large Email table.
func (s Service) ReadEmails(ctx context.Context, reverse bool, limit int, offset string) []Email {
	ids, err := s.Table.ReadEntityIDs(ctx, reverse, limit, offset)
	if err != nil {
		return []Email{}
	}
	return s.Table.ReadEntities(ctx, ids)
}

//------------------------------------------------------------------------------
// Emails by Address
//------------------------------------------------------------------------------

// ReadAddresses returns a paginated Address list for which there are Emails in the Email table.
// Sorting is alphabetical (or reverse). The offset is the last Address returned in a previous request.
func (s Service) ReadAddresses(ctx context.Context, reverse bool, limit int, offset string) ([]string, error) {
	return s.Table.ReadPartKeyValues(ctx, rowEmailsAddress, reverse, limit, offset)
}

// ReadAllAddresses returns a complete, alphabetical Address list for which there are Emails in the Email table.
func (s Service) ReadAllAddresses(ctx context.Context) ([]string, error) {
	return s.Table.ReadAllPartKeyValues(ctx, rowEmailsAddress)
}

// ReadEmailsByAddress returns paginated Emails by Address. Sorting is chronological (or reverse).
// The offset is the ID of the last Email returned in a previous request.
func (s Service) ReadEmailsByAddress(ctx context.Context, address string, reverse bool, limit int, offset string) ([]Email, error) {
	return s.Table.ReadEntitiesFromRow(ctx, rowEmailsAddress, address, reverse, limit, offset)
}

// ReadEmailsByAddressAsJSON returns paginated JSON Emails by Address. Sorting is chronological (or reverse).
// The offset is the ID of the last Email returned in a previous request.
func (s Service) ReadEmailsByAddressAsJSON(ctx context.Context, address string, reverse bool, limit int, offset string) ([]byte, error) {
	return s.Table.ReadEntitiesFromRowAsJSON(ctx, rowEmailsAddress, address, reverse, limit, offset)
}

// ReadAllEmailsByAddress returns the complete list of Emails, sorted chronologically by CreatedAt timestamp.
// Caution: this may be a LOT of data!
func (s Service) ReadAllEmailsByAddress(ctx context.Context, address string) ([]Email, error) {
	return s.Table.ReadAllEntitiesFromRow(ctx, rowEmailsAddress, address)
}

// ReadAllEmailsByAddressAsJSON returns the complete list of Emails, serialized as JSON.
// Caution: this may be a LOT of data!
func (s Service) ReadAllEmailsByAddressAsJSON(ctx context.Context, address string) ([]byte, error) {
	return s.Table.ReadAllEntitiesFromRowAsJSON(ctx, rowEmailsAddress, address)
}

//------------------------------------------------------------------------------
// Emails by Status
//------------------------------------------------------------------------------

// ReadStatuses returns a paginated Status list for which there are Emails in the Email table.
// Sorting is alphabetical (or reverse). The offset is the last Status returned in a previous request.
func (s Service) ReadStatuses(ctx context.Context, reverse bool, limit int, offset string) ([]string, error) {
	return s.Table.ReadPartKeyValues(ctx, rowEmailsStatus, reverse, limit, offset)
}

// ReadAllStatuses returns a complete, alphabetical Status list for which there are Emails in the Email table.
func (s Service) ReadAllStatuses(ctx context.Context) ([]string, error) {
	return s.Table.ReadAllPartKeyValues(ctx, rowEmailsStatus)
}

// ReadEmailsByStatus returns paginated Emails by Status. Sorting is chronological (or reverse).
// The offset is the ID of the last Email returned in a previous request.
func (s Service) ReadEmailsByStatus(ctx context.Context, status string, reverse bool, limit int, offset string) ([]Email, error) {
	return s.Table.ReadEntitiesFromRow(ctx, rowEmailsStatus, status, reverse, limit, offset)
}

// ReadEmailsByStatusAsJSON returns paginated JSON Emails by Status. Sorting is chronological (or reverse).
// The offset is the ID of the last Email returned in a previous request.
func (s Service) ReadEmailsByStatusAsJSON(ctx context.Context, status string, reverse bool, limit int, offset string) ([]byte, error) {
	return s.Table.ReadEntitiesFromRowAsJSON(ctx, rowEmailsStatus, status, reverse, limit, offset)
}

// ReadAllEmailsByStatus returns the complete list of Emails, sorted chronologically by CreatedAt timestamp.
// Caution: this may be a LOT of data!
func (s Service) ReadAllEmailsByStatus(ctx context.Context, status string) ([]Email, error) {
	return s.Table.ReadAllEntitiesFromRow(ctx, rowEmailsStatus, status)
}

// ReadAllEmailsByStatusAsJSON returns the complete list of Emails, serialized as JSON.
// Caution: this may be a LOT of data!
func (s Service) ReadAllEmailsByStatusAsJSON(ctx context.Context, status string) ([]byte, error) {
	return s.Table.ReadAllEntitiesFromRowAsJSON(ctx, rowEmailsStatus, status)
}
