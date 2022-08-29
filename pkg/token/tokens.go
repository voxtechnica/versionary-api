package token

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/voxtechnica/tuid-go"
	v "github.com/voxtechnica/versionary"
)

// rowTokens is a TableRow definition for OAuth Bearer Tokens. Tokens are not versioned.
var rowTokens = v.TableRow[Token]{
	RowName:      "tokens_version",
	PartKeyName:  "id",
	PartKeyValue: func(t Token) string { return t.ID },
	SortKeyName:  "id",
	SortKeyValue: func(t Token) string { return t.ID },
	JsonValue:    func(t Token) []byte { return t.CompressedJSON() },
	TimeToLive:   func(t Token) int64 { return t.ExpiresAt.Unix() },
}

// rowTokensUser is a TableRow definition for Tokens by User ID.
var rowTokensUser = v.TableRow[Token]{
	RowName:      "tokens_user",
	PartKeyName:  "user_id",
	PartKeyValue: func(t Token) string { return string(t.UserID) },
	SortKeyName:  "id",
	SortKeyValue: func(t Token) string { return t.ID },
	JsonValue:    func(t Token) []byte { return t.CompressedJSON() },
	TimeToLive:   func(t Token) int64 { return t.ExpiresAt.Unix() },
}

// NewTokenTable instantiates a new DynamoDB table for OAuth Bearer Tokens.
func NewTokenTable(dbClient *dynamodb.Client, env string) v.Table[Token] {
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

// NewTokenMemTable creates an in-memory Token table for testing purposes.
func NewTokenMemTable(table v.Table[Token]) v.MemTable[Token] {
	return v.NewMemTable(table)
}

// TokenService is used to manage Tokens in a DynamoDB table.
type TokenService struct {
	EntityType string
	Table      v.TableReadWriter[Token]
}

//==============================================================================
// Tokens
//==============================================================================

// Create a Token in the Token table.
func (s TokenService) Create(ctx context.Context, t Token) (Token, error) {
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
func (s TokenService) Write(ctx context.Context, t Token) (Token, error) {
	return t, s.Table.WriteEntity(ctx, t)
}

// Delete a Token from the Token table. The deleted Token is returned.
func (s TokenService) Delete(ctx context.Context, id string) (Token, error) {
	return s.Table.DeleteEntityWithID(ctx, id)
}

// Read a specified Token from the Token table.
func (s TokenService) Read(ctx context.Context, id string) (Token, error) {
	return s.Table.ReadEntity(ctx, id)
}

// ReadAsJSON gets a specified Token from the Token table, serialized as JSON.
func (s TokenService) ReadAsJSON(ctx context.Context, id string) ([]byte, error) {
	return s.Table.ReadEntityAsJSON(ctx, id)
}

// ReadTokenIDs returns a paginated list of Token IDs in the Token table.
// Sorting is chronological (or reverse). The offset is the last ID returned in a previous request.
func (s TokenService) ReadTokenIDs(ctx context.Context, reverse bool, limit int, offset string) ([]string, error) {
	return s.Table.ReadEntityIDs(ctx, reverse, limit, offset)
}

// ReadTokens returns a paginated list of Tokens in the Token table.
// Sorting is chronological (or reverse). The offset is the last ID returned in a previous request.
// Note that this is a best-effort attempt to return the requested Tokens, retrieved individually, in parallel.
// It is probably not the best way to page through a large Token table.
func (s TokenService) ReadTokens(ctx context.Context, reverse bool, limit int, offset string) []Token {
	ids, err := s.Table.ReadEntityIDs(ctx, reverse, limit, offset)
	if err != nil {
		return []Token{}
	}
	return s.Table.ReadEntities(ctx, ids)
}

//==============================================================================
// Tokens by User ID
//==============================================================================

// ReadUserIDs returns a paginated list of User IDs for which there are Tokens in the Token table.
// Sorting is alphabetical (or reverse). The offset is the last User returned in a previous request.
func (s TokenService) ReadUserIDs(ctx context.Context, reverse bool, limit int, offset string) ([]string, error) {
	return s.Table.ReadPartKeyValues(ctx, rowTokensUser, reverse, limit, offset)
}

// ReadAllUserIDs returns a complete, alphabetical list of User IDs for which there are Tokens in the Token table.
func (s TokenService) ReadAllUserIDs(ctx context.Context) ([]string, error) {
	return s.Table.ReadAllPartKeyValues(ctx, rowTokensUser)
}

// ReadAllTokenIDsByUserID returns a complete list of Token IDs for a specified User ID in the Token table.
func (s TokenService) ReadAllTokenIDsByUserID(ctx context.Context, userID string) ([]string, error) {
	return s.Table.ReadAllSortKeyValues(ctx, rowTokensUser, userID)
}

// ReadTokensByUserID returns paginated Tokens by User ID. Sorting is chronological (or reverse).
// The offset is the ID of the last Token returned in a previous request.
func (s TokenService) ReadTokensByUserID(ctx context.Context, userID string, reverse bool, limit int, offset string) ([]Token, error) {
	return s.Table.ReadEntitiesFromRow(ctx, rowTokensUser, userID, reverse, limit, offset)
}

// ReadTokensByUserIDAsJSON returns paginated JSON Tokens by User ID. Sorting is chronological (or reverse).
// The offset is the ID of the last Token returned in a previous request.
func (s TokenService) ReadTokensByUserIDAsJSON(ctx context.Context, userID string, reverse bool, limit int, offset string) ([]byte, error) {
	return s.Table.ReadEntitiesFromRowAsJSON(ctx, rowTokensUser, userID, reverse, limit, offset)
}

// ReadAllTokensByUserID returns the complete list of Tokens, sorted chronologically by CreatedAt timestamp.
func (s TokenService) ReadAllTokensByUserID(ctx context.Context, userID string) ([]Token, error) {
	return s.Table.ReadAllEntitiesFromRow(ctx, rowTokensUser, userID)
}

// ReadAllTokensByUserIDAsJSON returns the complete list of Tokens, serialized as JSON.
func (s TokenService) ReadAllTokensByUserIDAsJSON(ctx context.Context, userID string) ([]byte, error) {
	return s.Table.ReadAllEntitiesFromRowAsJSON(ctx, rowTokensUser, userID)
}

// DeleteAllTokensByUserID deletes all Tokens for a specified User ID from the Token table.
func (s TokenService) DeleteAllTokensByUserID(ctx context.Context, userID string) error {
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
