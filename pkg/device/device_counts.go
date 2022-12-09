package device

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	v "github.com/voxtechnica/versionary"
)

//==============================================================================
// Device Count Table
//==============================================================================

// rowCountsDate is a TableRow definition for Device Counts, indexed by Date.
// It's an EntityRow, but the "versions" are actually the counts for each day.
var rowCountsDate = v.TableRow[Count]{
	RowName:      "counts_date",
	PartKeyName:  "type",
	PartKeyValue: func(c Count) string { return c.Type() },
	SortKeyName:  "date",
	SortKeyValue: func(c Count) string { return c.Date }, // YYYY-MM-DD
	JsonValue:    func(c Count) []byte { return c.CompressedJSON() },
}

// NewCountTable instantiates a new DynamoDB Device Count table.
func NewCountTable(dbClient *dynamodb.Client, env string) v.Table[Count] {
	if env == "" {
		env = "dev"
	}
	return v.Table[Count]{
		Client:     dbClient,
		EntityType: "DeviceCount",
		TableName:  "device_counts" + "_" + env,
		TTL:        false,
		EntityRow:  rowCountsDate,
	}
}

// NewCountMemTable creates an in-memory Device Count table for testing purposes.
func NewCountMemTable(table v.Table[Count]) v.MemTable[Count] {
	return v.NewMemTable(table)
}

//==============================================================================
// Device Count Service
//==============================================================================

// CountService is used to manage a Device Count database.
type CountService struct {
	EntityType string
	Table      v.TableReadWriter[Count]
}

// NewCountService creates a new Device Count service backed by a Versionary Table for the specified environment.
func NewCountService(dbClient *dynamodb.Client, env string) CountService {
	table := NewCountTable(dbClient, env)
	return CountService{
		EntityType: table.TableName,
		Table:      table,
	}
}

// NewMockCountService creates a new Device Count service backed by an in-memory table for testing purposes.
func NewMockCountService(env string) CountService {
	table := NewCountMemTable(NewCountTable(nil, env))
	return CountService{
		EntityType: table.TableName,
		Table:      table,
	}
}

//------------------------------------------------------------------------------
// Device Counts
//------------------------------------------------------------------------------

// Write a Device Count to the database.
func (s CountService) Write(ctx context.Context, dc Count) (Count, []string, error) {
	problems := dc.Validate()
	if len(problems) > 0 {
		return dc, problems, fmt.Errorf("error writing %s: invalid field(s): %s", s.EntityType, strings.Join(problems, ", "))
	}
	err := s.Table.WriteEntity(ctx, dc)
	if err != nil {
		return dc, problems, fmt.Errorf("error writing %s %s: %w", s.EntityType, dc.Date, err)
	}
	return dc, problems, nil
}

// Exists checks if a Device Count exists in the database.
func (s CountService) Exists(ctx context.Context, date string) bool {
	return s.Table.EntityVersionExists(ctx, s.EntityType, date)
}

// Read a Device Count from the database.
func (s CountService) Read(ctx context.Context, date string) (Count, error) {
	return s.Table.ReadEntityVersion(ctx, s.EntityType, date)
}

// ReadAsJSON gets a Device Count from the database, serialized as JSON.
func (s CountService) ReadAsJSON(ctx context.Context, date string) ([]byte, error) {
	return s.Table.ReadEntityVersionAsJSON(ctx, s.EntityType, date)
}

// ReadCounts reads paginated Device Counts from the database.
// Sorting is chronological (or reverse). The offset is the last date returned in a previous request.
func (s CountService) ReadCounts(ctx context.Context, reverse bool, limit int, offset string) ([]Count, error) {
	return s.Table.ReadEntityVersions(ctx, s.EntityType, reverse, limit, offset)
}

// ReadCountsAsJSON reads paginated Device Counts from the database, serialized as JSON.
// Sorting is chronological (or reverse). The offset is the last date returned in a previous request.
func (s CountService) ReadCountsAsJSON(ctx context.Context, reverse bool, limit int, offset string) ([]byte, error) {
	return s.Table.ReadEntityVersionsAsJSON(ctx, s.EntityType, reverse, limit, offset)
}
