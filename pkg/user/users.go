package user

import (
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	v "github.com/voxtechnica/versionary"
)

const (
	rowUsers      = "users"
	rowUsersEmail = "users_email"
	rowUsersGroup = "users_group"
)

// NewUserTable creates a new DynamoDB table for users.
func NewUserTable(dbClient *dynamodb.Client, env string) v.Table[User] {
	if env == "" {
		env = "dev"
	}
	return v.Table[User]{
		Client:     dbClient,
		EntityType: "User",
		TableName:  "users" + "_" + env,
		TTL:        false,
		EntityRow: v.TableRow[User]{
			RowName:       rowUsers,
			PartKeyName:   "id",
			PartKeyValue:  func(e User) string { return e.ID },
			PartKeyValues: nil,
			SortKeyName:   "update_id",
			SortKeyValue:  func(e User) string { return e.UpdateID },
			JsonValue:     func(e User) []byte { return e.CompressedJSON() },
			TextValue:     nil,
			NumericValue:  nil,
			TimeToLive:    nil,
		},
		IndexRows: map[string]v.TableRow[User]{
			rowUsersEmail: {
				RowName:       rowUsersEmail,
				PartKeyName:   "email",
				PartKeyValue:  func(e User) string { return e.Email },
				PartKeyValues: nil,
				SortKeyName:   "id",
				SortKeyValue:  func(e User) string { return e.ID },
				JsonValue:     func(e User) []byte { return e.CompressedJSON() },
				TextValue:     nil,
				NumericValue:  nil,
				TimeToLive:    nil,
			},
			rowUsersGroup: {
				RowName:       rowUsersGroup,
				PartKeyName:   "group_id",
				PartKeyValue:  nil,
				PartKeyValues: func(e User) []string { return e.GroupIDs },
				SortKeyName:   "id",
				SortKeyValue:  func(e User) string { return e.ID },
				JsonValue:     func(e User) []byte { return e.CompressedJSON() },
				TextValue:     nil,
				NumericValue:  nil,
				TimeToLive:    nil,
			},
		},
	}
}

// NewUserMemTable creates an in-memory User table for testing purposes.
func NewUserMemTable(table v.Table[User]) v.MemTable[User] {
	return v.NewMemTable(table)
}

// UserService is used to manage Users in a DynamoDB table.
type UserService struct {
	EntityType string
	Table      v.TableReadWriter[User]
}
