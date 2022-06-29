package user

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/voxtechnica/tuid-go"
	v "github.com/voxtechnica/versionary"
)

// rowUsers is a TableRow definition for User versions.
var rowUsers = v.TableRow[User]{
	RowName:      "users",
	PartKeyName:  "id",
	PartKeyValue: func(u User) string { return u.ID },
	SortKeyName:  "update_id",
	SortKeyValue: func(u User) string { return u.VersionID },
	JsonValue:    func(u User) []byte { return u.CompressedJSON() },
}

// rowUsersEmail is a TableRow definition for Users by email address.
// There should be only one User per email address. It's effectively a unique ID for the User.
var rowUsersEmail = v.TableRow[User]{
	RowName:      "users_email",
	PartKeyName:  "email",
	PartKeyValue: func(u User) string { return standardizeEmail(u.Email) },
	SortKeyName:  "id",
	SortKeyValue: func(u User) string { return u.ID },
	JsonValue:    func(u User) []byte { return u.CompressedJSON() },
}

// rowUsersOrg is a TableRow definition for Users by Organization ID.
var rowUsersOrg = v.TableRow[User]{
	RowName:      "users_org",
	PartKeyName:  "org_id",
	PartKeyValue: func(u User) string { return u.OrgID },
	SortKeyName:  "id",
	SortKeyValue: func(u User) string { return u.ID },
	JsonValue:    func(u User) []byte { return u.CompressedJSON() },
}

// rowUsersRole is a TableRow definition for Users by Role.
var rowUsersRole = v.TableRow[User]{
	RowName:       "users_role",
	PartKeyName:   "role",
	PartKeyValues: func(u User) []string { return u.Roles },
	SortKeyName:   "id",
	SortKeyValue:  func(u User) string { return u.ID },
	JsonValue:     func(u User) []byte { return u.CompressedJSON() },
}

// rowUsersStatus is a TableRow definition for Users by Status.
var rowUsersStatus = v.TableRow[User]{
	RowName:      "users_status",
	PartKeyName:  "status",
	PartKeyValue: func(u User) string { return string(u.Status) },
	SortKeyName:  "id",
	SortKeyValue: func(u User) string { return u.ID },
	JsonValue:    func(u User) []byte { return u.CompressedJSON() },
}

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
		EntityRow:  rowUsers,
		IndexRows: map[string]v.TableRow[User]{
			rowUsersEmail.RowName:  rowUsersEmail,
			rowUsersOrg.RowName:    rowUsersOrg,
			rowUsersRole.RowName:   rowUsersRole,
			rowUsersStatus.RowName: rowUsersStatus,
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

// duplicateEmail returns a non-empty list of User IDs if the specified email address is already in
// use by another User.
func (s UserService) duplicateEmail(ctx context.Context, email, id string) ([]string, error) {
	ids, err := s.ReadUserIDsByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	var duplicates []string
	for _, i := range ids {
		if i != id {
			duplicates = append(duplicates, id)
		}
	}
	return duplicates, nil
}

//==============================================================================
// User Versions
//==============================================================================

// Create a User in the User table.
func (s UserService) Create(ctx context.Context, u User) (User, error) {
	// Validate User fields
	t := tuid.NewID()
	at, _ := t.Time()
	u.ID = t.String()
	u.CreatedAt = at
	u.VersionID = t.String()
	u.UpdatedAt = at
	if u.Status == "" {
		u.Status = PENDING
	}
	if v := u.Validate(); len(v) > 0 {
		return u, fmt.Errorf("error creating %s %s: invalid field(s): %s", s.EntityType, u.ID, strings.Join(v, ", "))
	}
	// Check for duplicate email address
	duplicates, err := s.duplicateEmail(ctx, u.Email, u.ID)
	if err != nil {
		return u, fmt.Errorf("error checking email duplicates for %s: %w", u.Email, err)
	}
	if len(duplicates) > 0 {
		return u, fmt.Errorf("error creating %s %s: email address %s is already in use by %s", s.EntityType, u.ID, u.Email, strings.Join(duplicates, ", "))
	}
	// Hash password
	if u.Password != "" {
		u.PasswordHash = hashPassword(u.ID, u.Password)
		u.Password = ""
	}
	// Create User
	err = s.Table.WriteEntity(ctx, u)
	if err != nil {
		return u, fmt.Errorf("error creating %s %s %s: %w", s.EntityType, u.ID, u.Email, err)
	}
	return u, nil
}

// Update a User in the User table. If a previous version does not exist, the User is created.
func (s UserService) Update(ctx context.Context, u User) (User, error) {
	// Validate User fields
	t := tuid.NewID()
	at, _ := t.Time()
	u.VersionID = t.String()
	u.UpdatedAt = at
	if v := u.Validate(); len(v) > 0 {
		return u, fmt.Errorf("error updating %s %s: invalid field(s): %s", s.EntityType, u.ID, strings.Join(v, ", "))
	}
	// Check for duplicate email address
	duplicates, err := s.duplicateEmail(ctx, u.Email, u.ID)
	if err != nil {
		return u, fmt.Errorf("error checking email duplicates for %s: %w", u.Email, err)
	}
	if len(duplicates) > 0 {
		return u, fmt.Errorf("error creating %s %s: email address %s is already in use by %s", s.EntityType, u.ID, u.Email, strings.Join(duplicates, ", "))
	}
	// Hash password
	if u.Password != "" {
		u.PasswordHash = hashPassword(u.ID, u.Password)
		u.Password = ""
	}
	// Update User
	return u, s.Table.UpdateEntity(ctx, u)
}

// Write a User to the User table. This method assumes that the User has all the required fields.
// It would most likely be used for "refreshing" the index rows in the User table.
func (s UserService) Write(ctx context.Context, u User) (User, error) {
	return u, s.Table.WriteEntity(ctx, u)
}

// Delete a User from the User table. The deleted User is returned.
func (s UserService) Delete(ctx context.Context, id string) (User, error) {
	return s.Table.DeleteEntityWithID(ctx, id)
}

// Read a specified User from the User table.
func (s UserService) Read(ctx context.Context, id string) (User, error) {
	if strings.Contains(id, "@") {
		return s.ReadUserByEmail(ctx, id)
	}
	return s.Table.ReadEntity(ctx, id)
}

// ReadAsJSON gets a specified User from the User table, serialized as JSON.
func (s UserService) ReadAsJSON(ctx context.Context, id string) ([]byte, error) {
	return s.Table.ReadEntityAsJSON(ctx, id)
}

// ReadVersion gets a specified User version from the User table.
func (s UserService) ReadVersion(ctx context.Context, id, versionID string) (User, error) {
	return s.Table.ReadEntityVersion(ctx, id, versionID)
}

// ReadVersionAsJSON gets a specified User version from the User table, serialized as JSON.
func (s UserService) ReadVersionAsJSON(ctx context.Context, id, versionID string) ([]byte, error) {
	return s.Table.ReadEntityVersionAsJSON(ctx, id, versionID)
}

// ReadVersions returns paginated versions of the specified User.
// Sorting is chronological (or reverse). The offset is the last ID returned in a previous request.
func (s UserService) ReadVersions(ctx context.Context, id string, reverse bool, limit int, offset string) ([]User, error) {
	return s.Table.ReadEntityVersions(ctx, id, reverse, limit, offset)
}

// ReadVersionsAsJSON returns paginated versions of the specified User, serialized as JSON.
// Sorting is chronological (or reverse). The offset is the last ID returned in a previous request.
func (s UserService) ReadVersionsAsJSON(ctx context.Context, id string, reverse bool, limit int, offset string) ([]byte, error) {
	return s.Table.ReadEntityVersionsAsJSON(ctx, id, reverse, limit, offset)
}

// ReadAllVersions returns all versions of the specified User in chronological order.
// Caution: this may be a LOT of data!
func (s UserService) ReadAllVersions(ctx context.Context, id string) ([]User, error) {
	return s.Table.ReadAllEntityVersions(ctx, id)
}

// ReadAllVersionsAsJSON returns all versions of the specified User, serialized as JSON.
// Caution: this may be a LOT of data!
func (s UserService) ReadAllVersionsAsJSON(ctx context.Context, id string) ([]byte, error) {
	return s.Table.ReadAllEntityVersionsAsJSON(ctx, id)
}

// ReadUserIDs returns a paginated list of User IDs in the User table.
// Sorting is chronological (or reverse). The offset is the last ID returned in a previous request.
func (s UserService) ReadUserIDs(ctx context.Context, reverse bool, limit int, offset string) ([]string, error) {
	return s.Table.ReadEntityIDs(ctx, reverse, limit, offset)
}

// ReadUsers returns a paginated list of Users in the User table.
// Sorting is chronological (or reverse). The offset is the last ID returned in a previous request.
// Note that this is a best-effort attempt to return the requested Users, retrieved individually, in parallel.
// It is probably not the best way to page through a large User table.
func (s UserService) ReadUsers(ctx context.Context, reverse bool, limit int, offset string) []User {
	ids, err := s.Table.ReadEntityIDs(ctx, reverse, limit, offset)
	if err != nil {
		return []User{}
	}
	return s.Table.ReadEntities(ctx, ids)
}

//==============================================================================
// Users by Email Address
//==============================================================================

// ReadEmailAddresses returns a paginated list of standardized email addresses from the User table.
// Sorting is alphabetical (or reverse). The offset is the last email address returned in a previous request.
func (s UserService) ReadEmailAddresses(ctx context.Context, reverse bool, limit int, offset string) ([]string, error) {
	return s.Table.ReadPartKeyValues(ctx, rowUsersEmail, reverse, limit, offset)
}

// ReadAllEmailAddresses returns a complete, alphabetical, standardized email address list
// for which there are Users in the User table. Caution: this may be a LOT of data!
func (s UserService) ReadAllEmailAddresses(ctx context.Context) ([]string, error) {
	return s.Table.ReadAllPartKeyValues(ctx, rowUsersEmail)
}

// ReadUserByEmail returns the first (chronological) User with the provided email address.
// There should only be one User in the User table with that address.
func (s UserService) ReadUserByEmail(ctx context.Context, email string) (User, error) {
	users, err := s.Table.ReadAllEntitiesFromRow(ctx, rowUsersEmail, standardizeEmail(email))
	if err != nil {
		return User{}, err
	}
	if len(users) == 0 {
		return User{}, v.ErrNotFound
	}
	return users[0], nil
}

// ReadAllUsersByEmail returns a complete, chronological, list of Users with the provided email address.
// There should only be one User in the User table with that address, but strange things can happen.
// This method can be used to identify any duplicates for the provided email address.
func (s UserService) ReadAllUsersByEmail(ctx context.Context, email string) ([]User, error) {
	return s.Table.ReadAllEntitiesFromRow(ctx, rowUsersEmail, standardizeEmail(email))
}

// ReadUserIDsByEmail returns a complete list of User IDs corresponding to the provided email address.
// The primary use of this method is to check to see if a given email address is already in use.
func (s UserService) ReadUserIDsByEmail(ctx context.Context, email string) ([]string, error) {
	return s.Table.ReadAllSortKeyValues(ctx, rowUsersEmail, standardizeEmail(email))
}

//==============================================================================
// Users by Organization
//==============================================================================

// ReadOrgIDs returns a paginated list of Organization IDs for which there are Users in the User table.
// Sorting is chronological (or reverse). The offset is the last ID returned in a previous request.
func (s UserService) ReadOrgIDs(ctx context.Context, reverse bool, limit int, offset string) ([]string, error) {
	return s.Table.ReadPartKeyValues(ctx, rowUsersOrg, reverse, limit, offset)
}

// ReadAllOrgIDs returns a complete, chronological list of Organization IDs for which there are Users in the User table.
func (s UserService) ReadAllOrgIDs(ctx context.Context) ([]string, error) {
	return s.Table.ReadAllPartKeyValues(ctx, rowUsersOrg)
}

// ReadUsersByOrgID returns paginated Users by Organization ID. Sorting is chronological (or reverse).
// The offset is the ID of the last User returned in a previous request.
func (s UserService) ReadUsersByOrgID(ctx context.Context, orgID string, reverse bool, limit int, offset string) ([]User, error) {
	return s.Table.ReadEntitiesFromRow(ctx, rowUsersOrg, orgID, reverse, limit, offset)
}

// ReadUsersByOrgIDAsJSON returns paginated JSON Users by Organization ID. Sorting is chronological (or reverse).
// The offset is the ID of the last User returned in a previous request.
func (s UserService) ReadUsersByOrgIDAsJSON(ctx context.Context, orgID string, reverse bool, limit int, offset string) ([]byte, error) {
	return s.Table.ReadEntitiesFromRowAsJSON(ctx, rowUsersOrg, orgID, reverse, limit, offset)
}

// ReadAllUsersByOrgID returns the complete list of Users for the specified Organization ID,
// sorted chronologically by CreatedAt timestamp. Caution: this may be a LOT of data!
func (s UserService) ReadAllUsersByOrgID(ctx context.Context, orgID string) ([]User, error) {
	return s.Table.ReadAllEntitiesFromRow(ctx, rowUsersOrg, orgID)
}

// ReadAllUsersByOrgIDAsJSON returns the complete list of Users by Organization ID, serialized as JSON.
// Caution: this may be a LOT of data!
func (s UserService) ReadAllUsersByOrgIDAsJSON(ctx context.Context, orgID string) ([]byte, error) {
	return s.Table.ReadAllEntitiesFromRowAsJSON(ctx, rowUsersOrg, orgID)
}

//==============================================================================
// Users by Role
//==============================================================================

// ReadRoles returns a paginated list of Roles for which there are Users in the User table.
// Sorting is alphabetical (or reverse). The offset is the last Role returned in a previous request.
func (s UserService) ReadRoles(ctx context.Context, reverse bool, limit int, offset string) ([]string, error) {
	return s.Table.ReadPartKeyValues(ctx, rowUsersRole, reverse, limit, offset)
}

// ReadAllRoles returns a complete, alphabetical list of Roles for which there are Users in the User table.
func (s UserService) ReadAllRoles(ctx context.Context) ([]string, error) {
	return s.Table.ReadAllPartKeyValues(ctx, rowUsersRole)
}

// ReadUsersByRole returns paginated Users by Role. Sorting is chronological (or reverse).
// The offset is the ID of the last User returned in a previous request.
func (s UserService) ReadUsersByRole(ctx context.Context, role string, reverse bool, limit int, offset string) ([]User, error) {
	return s.Table.ReadEntitiesFromRow(ctx, rowUsersRole, role, reverse, limit, offset)
}

// ReadUsersByRoleAsJSON returns paginated JSON Users by Role. Sorting is chronological (or reverse).
// The offset is the ID of the last User returned in a previous request.
func (s UserService) ReadUsersByRoleAsJSON(ctx context.Context, role string, reverse bool, limit int, offset string) ([]byte, error) {
	return s.Table.ReadEntitiesFromRowAsJSON(ctx, rowUsersRole, role, reverse, limit, offset)
}

// ReadAllUsersByRole returns the complete list of Users, sorted chronologically by CreatedAt timestamp.
// Caution: this may be a LOT of data!
func (s UserService) ReadAllUsersByRole(ctx context.Context, role string) ([]User, error) {
	return s.Table.ReadAllEntitiesFromRow(ctx, rowUsersRole, role)
}

// ReadAllUsersByRoleAsJSON returns the complete list of Users, serialized as JSON.
// Caution: this may be a LOT of data!
func (s UserService) ReadAllUsersByRoleAsJSON(ctx context.Context, role string) ([]byte, error) {
	return s.Table.ReadAllEntitiesFromRowAsJSON(ctx, rowUsersRole, role)
}

//==============================================================================
// Users by Status
//==============================================================================

// ReadStatuses returns a paginated Status list for which there are Users in the User table.
// Sorting is alphabetical (or reverse). The offset is the last Status returned in a previous request.
func (s UserService) ReadStatuses(ctx context.Context, reverse bool, limit int, offset string) ([]string, error) {
	return s.Table.ReadPartKeyValues(ctx, rowUsersStatus, reverse, limit, offset)
}

// ReadAllStatuses returns a complete, alphabetical Status list for which there are Users in the User table.
func (s UserService) ReadAllStatuses(ctx context.Context) ([]string, error) {
	return s.Table.ReadAllPartKeyValues(ctx, rowUsersStatus)
}

// ReadUsersByStatus returns paginated Users by Status. Sorting is chronological (or reverse).
// The offset is the ID of the last User returned in a previous request.
func (s UserService) ReadUsersByStatus(ctx context.Context, status string, reverse bool, limit int, offset string) ([]User, error) {
	return s.Table.ReadEntitiesFromRow(ctx, rowUsersStatus, status, reverse, limit, offset)
}

// ReadUsersByStatusAsJSON returns paginated JSON Users by Status. Sorting is chronological (or reverse).
// The offset is the ID of the last User returned in a previous request.
func (s UserService) ReadUsersByStatusAsJSON(ctx context.Context, status string, reverse bool, limit int, offset string) ([]byte, error) {
	return s.Table.ReadEntitiesFromRowAsJSON(ctx, rowUsersStatus, status, reverse, limit, offset)
}

// ReadAllUsersByStatus returns the complete list of Users, sorted chronologically by CreatedAt timestamp.
// Caution: this may be a LOT of data!
func (s UserService) ReadAllUsersByStatus(ctx context.Context, status string) ([]User, error) {
	return s.Table.ReadAllEntitiesFromRow(ctx, rowUsersStatus, status)
}

// ReadAllUsersByStatusAsJSON returns the complete list of Users, serialized as JSON.
// Caution: this may be a LOT of data!
func (s UserService) ReadAllUsersByStatusAsJSON(ctx context.Context, status string) ([]byte, error) {
	return s.Table.ReadAllEntitiesFromRowAsJSON(ctx, rowUsersStatus, status)
}
