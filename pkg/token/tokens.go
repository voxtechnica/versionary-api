package token

import (
	"context"
	"fmt"
	"strings"
	"versionary-api/pkg/util"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/voxtechnica/tuid-go"
	v "github.com/voxtechnica/versionary"
)

//==============================================================================
// Token Table
//==============================================================================

// rowTokens is a TableRow definition for OAuth Bearer Tokens. Tokens are not versioned.
var rowTokens = v.TableRow[Token]{
	RowName:      "tokens",
	PartKeyName:  "id",
	PartKeyValue: func(t Token) string { return t.ID },
	PartKeyLabel: func(t Token) string { return t.UserID },
	SortKeyName:  "id",
	SortKeyValue: func(t Token) string { return t.ID },
	JsonValue:    func(t Token) []byte { return t.CompressedJSON() },
	TimeToLive:   func(t Token) int64 { return t.ExpiresAt.Unix() },
}

// rowTokensUser is a TableRow definition for Tokens by User ID.
var rowTokensUser = v.TableRow[Token]{
	RowName:      "tokens_user",
	PartKeyName:  "user_id",
	PartKeyValue: func(t Token) string { return t.UserID },
	PartKeyLabel: func(t Token) string { return t.Email },
	SortKeyName:  "id",
	SortKeyValue: func(t Token) string { return t.ID },
	JsonValue:    func(t Token) []byte { return t.CompressedJSON() },
	TimeToLive:   func(t Token) int64 { return t.ExpiresAt.Unix() },
}

// NewTable instantiates a new DynamoDB table for OAuth Bearer Tokens.
func NewTable(dbClient *dynamodb.Client, env string) v.Table[Token] {
	if env == "" {
		env = "dev"
	}
	return v.Table[Token]{
		Client:     dbClient,
		EntityType: "Token",
		TableName:  "tokens" + "_" + env,
		TTL:        true,
		EntityRow:  rowTokens,
		IndexRows: map[string]v.TableRow[Token]{
			rowTokensUser.RowName: rowTokensUser,
		},
	}
}

// NewMemTable creates an in-memory Token table for testing purposes.
func NewMemTable(table v.Table[Token]) v.MemTable[Token] {
	return v.NewMemTable(table)
}

//==============================================================================
// Token Service
//==============================================================================

// Service is used to manage Tokens in a DynamoDB table.
type Service struct {
	EntityType string
	Table      v.TableReadWriter[Token]
}

// NewService creates a new Token service backed by a Versionary Table for the specified environment.
func NewService(dbClient *dynamodb.Client, env string) Service {
	table := NewTable(dbClient, env)
	return Service{
		EntityType: table.EntityType,
		Table:      table,
	}
}

// NewMockService creates a new Token service backed by an in-memory table for testing purposes.
func NewMockService(env string) Service {
	table := NewMemTable(NewTable(nil, env))
	return Service{
		EntityType: table.TableName,
		Table:      table,
	}
}

//------------------------------------------------------------------------------
// Tokens
//------------------------------------------------------------------------------

// Create a Token in the Token table.
func (s Service) Create(ctx context.Context, t Token) (Token, error) {
	id := tuid.NewID()
	at, _ := id.Time()
	t.ID = id.String()
	t.CreatedAt = at
	t.ExpiresAt = at.AddDate(0, 0, 30)
	if problems := t.Validate(); len(problems) > 0 {
		return t, fmt.Errorf("error creating %s %s: invalid field(s): %s", s.EntityType, t.ID, strings.Join(problems, ", "))
	}
	err := s.Table.WriteEntity(ctx, t)
	if err != nil {
		return t, fmt.Errorf("error creating Token-%s for User-%s: %w", t.ID, t.UserID, err)
	}
	return t, nil
}

// Write a Token to the Token table. This method assumes that the Token has all the required fields.
// It would most likely be used for "refreshing" the index rows in the Token table.
func (s Service) Write(ctx context.Context, t Token) (Token, error) {
	return t, s.Table.WriteEntity(ctx, t)
}

// Delete a Token from the Token table. The deleted Token is returned.
func (s Service) Delete(ctx context.Context, id string) (Token, error) {
	return s.Table.DeleteEntityWithID(ctx, id)
}

// Exists checks if a Token exists in the Token table.
func (s Service) Exists(ctx context.Context, id string) bool {
	return s.Table.EntityExists(ctx, id)
}

// Read a specified Token from the Token table.
func (s Service) Read(ctx context.Context, id string) (Token, error) {
	return s.Table.ReadEntity(ctx, id)
}

// ReadAsJSON gets a specified Token from the Token table, serialized as JSON.
func (s Service) ReadAsJSON(ctx context.Context, id string) ([]byte, error) {
	return s.Table.ReadEntityAsJSON(ctx, id)
}

// ReadIDs returns a paginated list of Token IDs and associated User IDs in the Token table.
// Sorting is chronological (or reverse). The offset is the last ID returned in a previous request.
func (s Service) ReadIDs(ctx context.Context, reverse bool, limit int, offset string) ([]v.TextValue, error) {
	return s.Table.ReadEntityLabels(ctx, reverse, limit, offset)
}

// ReadAllIDs returns a list of all Token IDs and associated User IDs in the Token table.
// Sorting is chronological (or reverse).
func (s Service) ReadAllIDs(ctx context.Context, sortByValue bool) ([]v.TextValue, error) {
	return s.Table.ReadAllEntityLabels(ctx, sortByValue)
}

// ReadTokens returns a paginated list of Tokens in the Token table.
// Sorting is chronological (or reverse). The offset is the last ID returned in a previous request.
// Note that this is a best-effort attempt to return the requested Tokens, retrieved individually, in parallel.
// It is probably not the best way to page through a large Token table.
func (s Service) ReadTokens(ctx context.Context, reverse bool, limit int, offset string) []Token {
	ids, err := s.Table.ReadEntityIDs(ctx, reverse, limit, offset)
	if err != nil {
		return []Token{}
	}
	return s.Table.ReadEntities(ctx, ids)
}

//------------------------------------------------------------------------------
// Tokens by User ID
//------------------------------------------------------------------------------

// ReadUsers returns a paginated list of User IDs and Emails for which there are Tokens in the Token table.
// Sorting is alphabetical (or reverse). The offset is the last User returned in a previous request.
func (s Service) ReadUsers(ctx context.Context, reverse bool, limit int, offset string) ([]v.TextValue, error) {
	return s.Table.ReadPartKeyLabels(ctx, rowTokensUser, reverse, limit, offset)
}

// ReadAllUsers returns a complete, alphabetical list of User IDs and Emails for which there are Tokens in the Token table.
func (s Service) ReadAllUsers(ctx context.Context, sortByValue bool) ([]v.TextValue, error) {
	return s.Table.ReadAllPartKeyLabels(ctx, rowTokensUser, sortByValue)
}

// FilterUsers returns a filtered list of User IDs and Emails for which there are Tokens in the Token table.
// The filter is a case-insensitive substring match on the Email address.
// If anyMatch is true, then a TextValue is included in the results if any of the words are found (OR filter).
// If anyMatch is false, then the TextValue must contain all the words in the query string (AND filter).
// The filtered results are sorted alphabetically by Email address, not by ID.
func (s Service) FilterUsers(ctx context.Context, contains string, anyMatch bool) ([]v.TextValue, error) {
	filter, err := util.ContainsFilter(contains, anyMatch)
	if err != nil {
		return []v.TextValue{}, err
	}
	return s.Table.FilterPartKeyLabels(ctx, rowTokensUser, filter)
}

// ReadUserIDs returns a paginated list of User IDs for which there are Tokens in the Token table.
// Sorting is alphabetical (or reverse). The offset is the last User returned in a previous request.
func (s Service) ReadUserIDs(ctx context.Context, reverse bool, limit int, offset string) ([]string, error) {
	return s.Table.ReadPartKeyValues(ctx, rowTokensUser, reverse, limit, offset)
}

// ReadAllUserIDs returns a complete, alphabetical list of User IDs for which there are Tokens in the Token table.
func (s Service) ReadAllUserIDs(ctx context.Context) ([]string, error) {
	return s.Table.ReadAllPartKeyValues(ctx, rowTokensUser)
}

// ReadAllTokenIDsByUserID returns a complete list of Token IDs for a specified User ID in the Token table.
func (s Service) ReadAllTokenIDsByUserID(ctx context.Context, userID string) ([]string, error) {
	return s.Table.ReadAllSortKeyValues(ctx, rowTokensUser, userID)
}

// ReadTokensByUserID returns paginated Tokens by User ID. Sorting is chronological (or reverse).
// The offset is the ID of the last Token returned in a previous request.
func (s Service) ReadTokensByUserID(ctx context.Context, userID string, reverse bool, limit int, offset string) ([]Token, error) {
	return s.Table.ReadEntitiesFromRow(ctx, rowTokensUser, userID, reverse, limit, offset)
}

// ReadTokensByUserIDAsJSON returns paginated JSON Tokens by User ID. Sorting is chronological (or reverse).
// The offset is the ID of the last Token returned in a previous request.
func (s Service) ReadTokensByUserIDAsJSON(ctx context.Context, userID string, reverse bool, limit int, offset string) ([]byte, error) {
	return s.Table.ReadEntitiesFromRowAsJSON(ctx, rowTokensUser, userID, reverse, limit, offset)
}

// ReadAllTokensByUserID returns the complete list of Tokens, sorted chronologically by CreatedAt timestamp.
func (s Service) ReadAllTokensByUserID(ctx context.Context, userID string) ([]Token, error) {
	return s.Table.ReadAllEntitiesFromRow(ctx, rowTokensUser, userID)
}

// ReadAllTokensByUserIDAsJSON returns the complete list of Tokens, serialized as JSON.
func (s Service) ReadAllTokensByUserIDAsJSON(ctx context.Context, userID string) ([]byte, error) {
	return s.Table.ReadAllEntitiesFromRowAsJSON(ctx, rowTokensUser, userID)
}

// DeleteAllTokensByUserID deletes all Tokens for a specified User ID from the Token table.
func (s Service) DeleteAllTokensByUserID(ctx context.Context, userID string) error {
	ids, err := s.ReadAllTokenIDsByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("error deleting all Tokens for User-%s: %w", userID, err)
	}
	for _, id := range ids {
		_, err := s.Delete(ctx, id)
		if err != nil {
			return fmt.Errorf("error deleting Token-%s for User-%s: %w", id, userID, err)
		}
	}
	return nil
}
