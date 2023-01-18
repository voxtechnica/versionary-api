package org

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
// Organization Table
//==============================================================================

// rowOrganizations is a TableRow definition for Organization versions.
var rowOrganizations = v.TableRow[Organization]{
	RowName:      "organizations_version",
	PartKeyName:  "id",
	PartKeyValue: func(o Organization) string { return o.ID },
	PartKeyLabel: func(o Organization) string { return o.Name },
	SortKeyName:  "version_id",
	SortKeyValue: func(o Organization) string { return o.VersionID },
	JsonValue:    func(o Organization) []byte { return o.CompressedJSON() },
}

// rowOrganizationsStatus is a TableRow definition for Organizations by Status.
var rowOrganizationsStatus = v.TableRow[Organization]{
	RowName:      "organizations_status",
	PartKeyName:  "status",
	PartKeyValue: func(o Organization) string { return string(o.Status) },
	SortKeyName:  "id",
	SortKeyValue: func(o Organization) string { return o.ID },
	JsonValue:    func(o Organization) []byte { return o.CompressedJSON() },
}

// NewTable instantiates a new DynamoDB table for organizations.
func NewTable(dbClient *dynamodb.Client, env string) v.Table[Organization] {
	if env == "" {
		env = "dev"
	}
	return v.Table[Organization]{
		Client:     dbClient,
		EntityType: "Organization",
		TableName:  "organizations" + "_" + env,
		TTL:        false,
		EntityRow:  rowOrganizations,
		IndexRows: map[string]v.TableRow[Organization]{
			rowOrganizationsStatus.RowName: rowOrganizationsStatus,
		},
	}
}

// NewMemTable creates an in-memory Organization table for testing purposes.
func NewMemTable(table v.Table[Organization]) v.MemTable[Organization] {
	return v.NewMemTable(table)
}

//==============================================================================
// Organization Service
//==============================================================================

// Service is used to manage Organizations in a DynamoDB table.
type Service struct {
	EntityType string
	Table      v.TableReadWriter[Organization]
}

// NewService creates a new Organization service backed by a Versionary Table for the specified environment.
func NewService(dbClient *dynamodb.Client, env string) Service {
	table := NewTable(dbClient, env)
	return Service{
		EntityType: table.EntityType,
		Table:      table,
	}
}

// NewMockService creates a new Organization service backed by an in-memory table for testing purposes.
func NewMockService(env string) Service {
	table := NewMemTable(NewTable(nil, env))
	return Service{
		EntityType: table.EntityType,
		Table:      table,
	}
}

//------------------------------------------------------------------------------
// Organization Versions
//------------------------------------------------------------------------------

// Create an Organization in the Organization table.
func (s Service) Create(ctx context.Context, o Organization) (Organization, []string, error) {
	t := tuid.NewID()
	at, _ := t.Time()
	o.ID = t.String()
	o.CreatedAt = at
	o.VersionID = t.String()
	o.UpdatedAt = at
	if o.Status == "" {
		o.Status = PENDING
	}
	problems := o.Validate()
	if len(problems) > 0 {
		return o, problems, fmt.Errorf("error creating %s %s: invalid field(s): %s", s.EntityType, o.ID, strings.Join(problems, ", "))
	}
	err := s.Table.WriteEntity(ctx, o)
	if err != nil {
		return o, problems, fmt.Errorf("error creating %s %s %s: %w", s.EntityType, o.ID, o.Name, err)
	}
	return o, problems, nil
}

// Update an Organization in the Organization table. If a previous version does not exist, the Organization is created.
func (s Service) Update(ctx context.Context, o Organization) (Organization, []string, error) {
	t := tuid.NewID()
	at, _ := t.Time()
	o.VersionID = t.String()
	o.UpdatedAt = at
	problems := o.Validate()
	if len(problems) > 0 {
		return o, problems, fmt.Errorf("error updating %s %s: invalid field(s): %s", s.EntityType, o.ID, strings.Join(problems, ", "))
	}
	return o, problems, s.Table.UpdateEntity(ctx, o)
}

// Write an Organization to the Organization table. This method assumes that the Organization has all the required fields.
// It would most likely be used for "refreshing" the index rows in the Organization table.
func (s Service) Write(ctx context.Context, o Organization) (Organization, error) {
	return o, s.Table.WriteEntity(ctx, o)
}

// Delete an Organization from the Organization table. The deleted Organization is returned.
func (s Service) Delete(ctx context.Context, id string) (Organization, error) {
	return s.Table.DeleteEntityWithID(ctx, id)
}

// DeleteVersion deletes a specified Organization version from the Organization table.
// The deleted Organization version is returned.
func (s Service) DeleteVersion(ctx context.Context, id, versionID string) (Organization, error) {
	return s.Table.DeleteEntityVersionWithID(ctx, id, versionID)
}

// Exists checks if an Organization exists in the Organization table.
func (s Service) Exists(ctx context.Context, id string) bool {
	return s.Table.EntityExists(ctx, id)
}

// Read a specified Organization from the Organization table.
func (s Service) Read(ctx context.Context, id string) (Organization, error) {
	return s.Table.ReadEntity(ctx, id)
}

// ReadAsJSON gets a specified Organization from the Organization table, serialized as JSON.
func (s Service) ReadAsJSON(ctx context.Context, id string) ([]byte, error) {
	return s.Table.ReadEntityAsJSON(ctx, id)
}

// VersionExists checks if a specified Organization version exists in the Organization table.
func (s Service) VersionExists(ctx context.Context, id, versionID string) bool {
	return s.Table.EntityVersionExists(ctx, id, versionID)
}

// ReadVersion gets a specified Organization version from the Organization table.
func (s Service) ReadVersion(ctx context.Context, id, versionID string) (Organization, error) {
	return s.Table.ReadEntityVersion(ctx, id, versionID)
}

// ReadVersionAsJSON gets a specified Organization version from the Organization table, serialized as JSON.
func (s Service) ReadVersionAsJSON(ctx context.Context, id, versionID string) ([]byte, error) {
	return s.Table.ReadEntityVersionAsJSON(ctx, id, versionID)
}

// ReadVersions returns paginated versions of the specified Organization.
// Sorting is chronological (or reverse). The offset is the last ID returned in a previous request.
func (s Service) ReadVersions(ctx context.Context, id string, reverse bool, limit int, offset string) ([]Organization, error) {
	return s.Table.ReadEntityVersions(ctx, id, reverse, limit, offset)
}

// ReadVersionsAsJSON returns paginated versions of the specified Organization, serialized as JSON.
// Sorting is chronological (or reverse). The offset is the last ID returned in a previous request.
func (s Service) ReadVersionsAsJSON(ctx context.Context, id string, reverse bool, limit int, offset string) ([]byte, error) {
	return s.Table.ReadEntityVersionsAsJSON(ctx, id, reverse, limit, offset)
}

// ReadAllVersions returns all versions of the specified Organization in chronological order.
// Caution: this may be a LOT of data!
func (s Service) ReadAllVersions(ctx context.Context, id string) ([]Organization, error) {
	return s.Table.ReadAllEntityVersions(ctx, id)
}

// ReadAllVersionsAsJSON returns all versions of the specified Organization, serialized as JSON.
// Caution: this may be a LOT of data!
func (s Service) ReadAllVersionsAsJSON(ctx context.Context, id string) ([]byte, error) {
	return s.Table.ReadAllEntityVersionsAsJSON(ctx, id)
}

// ReadIDs returns a paginated list of Organization IDs in the Organization table.
// Sorting is chronological (or reverse). The offset is the last ID returned in a previous request.
func (s Service) ReadIDs(ctx context.Context, reverse bool, limit int, offset string) ([]string, error) {
	return s.Table.ReadEntityIDs(ctx, reverse, limit, offset)
}

// ReadAllIDs returns all Organization IDs in the Organization table.
// Caution: this may be a LOT of data!
func (s Service) ReadAllIDs(ctx context.Context) ([]string, error) {
	return s.Table.ReadAllEntityIDs(ctx)
}

// ReadNames returns a paginated list of Organization IDs and Names in the Organization table.
// Sorting is alphabetical (or reverse). The offset is the last ID returned in a previous request.
func (s Service) ReadNames(ctx context.Context, reverse bool, limit int, offset string) ([]v.TextValue, error) {
	return s.Table.ReadEntityLabels(ctx, reverse, limit, offset)
}

// ReadAllNames returns all Organization IDs and Names in the Organization table.
// Caution: this may be a LOT of data!
func (s Service) ReadAllNames(ctx context.Context, sortByValue bool) ([]v.TextValue, error) {
	return s.Table.ReadAllEntityLabels(ctx, sortByValue)
}

// FilterNames returns a filtered list of Organization IDs and Names in the Organization table.
// The case-insensitive contains query is split into words, and the words are compared with the value in the TextValue.
// If anyMatch is true, then a TextValue is included in the results if any of the words are found (OR filter).
// If anyMatch is false, then the TextValue must contain all the words in the query string (AND filter).
// The filtered results are sorted alphabetically by value, not by ID.
func (s Service) FilterNames(ctx context.Context, contains string, anyMatch bool) ([]v.TextValue, error) {
	filter, err := util.ContainsFilter(contains, anyMatch)
	if err != nil {
		return []v.TextValue{}, err
	}
	return s.Table.FilterEntityLabels(ctx, filter)
}

// ReadOrganizations returns a paginated list of Organizations in the Organization table.
// Sorting is chronological (or reverse). The offset is the last ID returned in a previous request.
// Note that this is a best-effort attempt to return the requested Organizations, retrieved individually, in parallel.
// It is probably not the best way to page through a large Organization table.
func (s Service) ReadOrganizations(ctx context.Context, reverse bool, limit int, offset string) []Organization {
	ids, err := s.Table.ReadEntityIDs(ctx, reverse, limit, offset)
	if err != nil {
		return []Organization{}
	}
	return s.Table.ReadEntities(ctx, ids)
}

//------------------------------------------------------------------------------
// Organizations by Status
//------------------------------------------------------------------------------

// ReadStatuses returns a paginated Status list for which there are Organizations in the Organization table.
// Sorting is alphabetical (or reverse). The offset is the last Status returned in a previous request.
func (s Service) ReadStatuses(ctx context.Context, reverse bool, limit int, offset string) ([]string, error) {
	return s.Table.ReadPartKeyValues(ctx, rowOrganizationsStatus, reverse, limit, offset)
}

// ReadAllStatuses returns a complete, alphabetical Status list for which there are Organizations in the Organization table.
func (s Service) ReadAllStatuses(ctx context.Context) ([]string, error) {
	return s.Table.ReadAllPartKeyValues(ctx, rowOrganizationsStatus)
}

// ReadOrganizationsByStatus returns paginated Organizations by Status. Sorting is chronological (or reverse).
// The offset is the ID of the last Organization returned in a previous request.
func (s Service) ReadOrganizationsByStatus(ctx context.Context, status string, reverse bool, limit int, offset string) ([]Organization, error) {
	return s.Table.ReadEntitiesFromRow(ctx, rowOrganizationsStatus, status, reverse, limit, offset)
}

// ReadOrganizationsByStatusAsJSON returns paginated JSON Organizations by Status. Sorting is chronological (or reverse).
// The offset is the ID of the last Organization returned in a previous request.
func (s Service) ReadOrganizationsByStatusAsJSON(ctx context.Context, status string, reverse bool, limit int, offset string) ([]byte, error) {
	return s.Table.ReadEntitiesFromRowAsJSON(ctx, rowOrganizationsStatus, status, reverse, limit, offset)
}

// ReadAllOrganizationsByStatus returns the complete list of Organizations, sorted chronologically by CreatedAt timestamp.
// Caution: this may be a LOT of data!
func (s Service) ReadAllOrganizationsByStatus(ctx context.Context, status string) ([]Organization, error) {
	return s.Table.ReadAllEntitiesFromRow(ctx, rowOrganizationsStatus, status)
}

// ReadAllOrganizationsByStatusAsJSON returns the complete list of Organizations, serialized as JSON.
// Caution: this may be a LOT of data!
func (s Service) ReadAllOrganizationsByStatusAsJSON(ctx context.Context, status string) ([]byte, error) {
	return s.Table.ReadAllEntitiesFromRowAsJSON(ctx, rowOrganizationsStatus, status)
}
