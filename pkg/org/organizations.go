package org

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/voxtechnica/tuid-go"
	v "github.com/voxtechnica/versionary"
)

// rowOrganizations is a TableRow definition for Organization versions.
var rowOrganizations = v.TableRow[Organization]{
	RowName:      "organizations_version",
	PartKeyName:  "id",
	PartKeyValue: func(o Organization) string { return o.ID },
	SortKeyName:  "update_id",
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

// NewOrganizationTable instantiates a new DynamoDB table for organizations.
func NewOrganizationTable(dbClient *dynamodb.Client, env string) v.Table[Organization] {
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

// NewOrganizationMemTable creates an in-memory Organization table for testing purposes.
func NewOrganizationMemTable(table v.Table[Organization]) v.MemTable[Organization] {
	return v.NewMemTable(table)
}

// OrganizationService is used to manage Organizations in a DynamoDB table.
type OrganizationService struct {
	EntityType string
	Table      v.TableReadWriter[Organization]
}

//==============================================================================
// Organization Versions
//==============================================================================

// Create an Organization in the Organization table.
func (s OrganizationService) Create(ctx context.Context, o Organization) (Organization, []string, error) {
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
func (s OrganizationService) Update(ctx context.Context, o Organization) (Organization, []string, error) {
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
func (s OrganizationService) Write(ctx context.Context, o Organization) (Organization, error) {
	return o, s.Table.WriteEntity(ctx, o)
}

// Delete an Organization from the Organization table. The deleted Organization is returned.
func (s OrganizationService) Delete(ctx context.Context, id string) (Organization, error) {
	return s.Table.DeleteEntityWithID(ctx, id)
}

// Exists checks if an Organization exists in the Organization table.
func (s OrganizationService) Exists(ctx context.Context, id string) bool {
	return s.Table.EntityExists(ctx, id)
}

// Read a specified Organization from the Organization table.
func (s OrganizationService) Read(ctx context.Context, id string) (Organization, error) {
	return s.Table.ReadEntity(ctx, id)
}

// ReadAsJSON gets a specified Organization from the Organization table, serialized as JSON.
func (s OrganizationService) ReadAsJSON(ctx context.Context, id string) ([]byte, error) {
	return s.Table.ReadEntityAsJSON(ctx, id)
}

// VersionExists checks if a specified Organization version exists in the Organization table.
func (s OrganizationService) VersionExists(ctx context.Context, id, versionID string) bool {
	return s.Table.EntityVersionExists(ctx, id, versionID)
}

// ReadVersion gets a specified Organization version from the Organization table.
func (s OrganizationService) ReadVersion(ctx context.Context, id, versionID string) (Organization, error) {
	return s.Table.ReadEntityVersion(ctx, id, versionID)
}

// ReadVersionAsJSON gets a specified Organization version from the Organization table, serialized as JSON.
func (s OrganizationService) ReadVersionAsJSON(ctx context.Context, id, versionID string) ([]byte, error) {
	return s.Table.ReadEntityVersionAsJSON(ctx, id, versionID)
}

// ReadVersions returns paginated versions of the specified Organization.
// Sorting is chronological (or reverse). The offset is the last ID returned in a previous request.
func (s OrganizationService) ReadVersions(ctx context.Context, id string, reverse bool, limit int, offset string) ([]Organization, error) {
	return s.Table.ReadEntityVersions(ctx, id, reverse, limit, offset)
}

// ReadVersionsAsJSON returns paginated versions of the specified Organization, serialized as JSON.
// Sorting is chronological (or reverse). The offset is the last ID returned in a previous request.
func (s OrganizationService) ReadVersionsAsJSON(ctx context.Context, id string, reverse bool, limit int, offset string) ([]byte, error) {
	return s.Table.ReadEntityVersionsAsJSON(ctx, id, reverse, limit, offset)
}

// ReadAllVersions returns all versions of the specified Organization in chronological order.
// Caution: this may be a LOT of data!
func (s OrganizationService) ReadAllVersions(ctx context.Context, id string) ([]Organization, error) {
	return s.Table.ReadAllEntityVersions(ctx, id)
}

// ReadAllVersionsAsJSON returns all versions of the specified Organization, serialized as JSON.
// Caution: this may be a LOT of data!
func (s OrganizationService) ReadAllVersionsAsJSON(ctx context.Context, id string) ([]byte, error) {
	return s.Table.ReadAllEntityVersionsAsJSON(ctx, id)
}

// ReadOrganizationIDs returns a paginated list of Organization IDs in the Organization table.
// Sorting is chronological (or reverse). The offset is the last ID returned in a previous request.
func (s OrganizationService) ReadOrganizationIDs(ctx context.Context, reverse bool, limit int, offset string) ([]string, error) {
	return s.Table.ReadEntityIDs(ctx, reverse, limit, offset)
}

// ReadOrganizations returns a paginated list of Organizations in the Organization table.
// Sorting is chronological (or reverse). The offset is the last ID returned in a previous request.
// Note that this is a best-effort attempt to return the requested Organizations, retrieved individually, in parallel.
// It is probably not the best way to page through a large Organization table.
func (s OrganizationService) ReadOrganizations(ctx context.Context, reverse bool, limit int, offset string) []Organization {
	ids, err := s.Table.ReadEntityIDs(ctx, reverse, limit, offset)
	if err != nil {
		return []Organization{}
	}
	return s.Table.ReadEntities(ctx, ids)
}

//==============================================================================
// Organizations by Status
//==============================================================================

// ReadStatuses returns a paginated Status list for which there are Organizations in the Organization table.
// Sorting is alphabetical (or reverse). The offset is the last Status returned in a previous request.
func (s OrganizationService) ReadStatuses(ctx context.Context, reverse bool, limit int, offset string) ([]string, error) {
	return s.Table.ReadPartKeyValues(ctx, rowOrganizationsStatus, reverse, limit, offset)
}

// ReadAllStatuses returns a complete, alphabetical Status list for which there are Organizations in the Organization table.
func (s OrganizationService) ReadAllStatuses(ctx context.Context) ([]string, error) {
	return s.Table.ReadAllPartKeyValues(ctx, rowOrganizationsStatus)
}

// ReadOrganizationsByStatus returns paginated Organizations by Status. Sorting is chronological (or reverse).
// The offset is the ID of the last Organization returned in a previous request.
func (s OrganizationService) ReadOrganizationsByStatus(ctx context.Context, status string, reverse bool, limit int, offset string) ([]Organization, error) {
	return s.Table.ReadEntitiesFromRow(ctx, rowOrganizationsStatus, status, reverse, limit, offset)
}

// ReadOrganizationsByStatusAsJSON returns paginated JSON Organizations by Status. Sorting is chronological (or reverse).
// The offset is the ID of the last Organization returned in a previous request.
func (s OrganizationService) ReadOrganizationsByStatusAsJSON(ctx context.Context, status string, reverse bool, limit int, offset string) ([]byte, error) {
	return s.Table.ReadEntitiesFromRowAsJSON(ctx, rowOrganizationsStatus, status, reverse, limit, offset)
}

// ReadAllOrganizationsByStatus returns the complete list of Organizations, sorted chronologically by CreatedAt timestamp.
// Caution: this may be a LOT of data!
func (s OrganizationService) ReadAllOrganizationsByStatus(ctx context.Context, status string) ([]Organization, error) {
	return s.Table.ReadAllEntitiesFromRow(ctx, rowOrganizationsStatus, status)
}

// ReadAllOrganizationsByStatusAsJSON returns the complete list of Organizations, serialized as JSON.
// Caution: this may be a LOT of data!
func (s OrganizationService) ReadAllOrganizationsByStatusAsJSON(ctx context.Context, status string) ([]byte, error) {
	return s.Table.ReadAllEntitiesFromRowAsJSON(ctx, rowOrganizationsStatus, status)
}
