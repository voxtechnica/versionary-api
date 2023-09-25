package metric

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/voxtechnica/tuid-go"
	v "github.com/voxtechnica/versionary"
)

//==============================================================================
// Metric Table
//==============================================================================

// rowMetrics is a TableRow definition for Metrics, indexed by ID.
var rowMetrics = v.TableRow[Metric]{
	RowName:       "metrics",
	PartKeyName:   "id",
	PartKeyValue:  func(m Metric) string { return m.ID },
	PartKeyValues: func(m Metric) []string { return m.Tags },
	PartKeyLabel:  func(m Metric) string { return m.Title },
	SortKeyName:   "id",
	SortKeyValue:  func(m Metric) string { return m.ID },
	JsonValue:     func(m Metric) []byte { return m.CompressedJSON() },
	TextValue:     nil,
	NumericValue:  func(m Metric) float64 { return m.Value },
	TimeToLive:    nil,
}

// rowMetricsEntity is a TableRow definition for Metrics, indexed by Entity ID.
var rowMetricsEntity = v.TableRow[Metric]{
	RowName:       "metrics_entity",
	PartKeyName:   "entity_id",
	PartKeyValue:  nil,
	PartKeyValues: func(m Metric) []string { return m.IDs() },
	SortKeyName:   "id",
	SortKeyValue:  func(m Metric) string { return m.ID },
	JsonValue:     func(m Metric) []byte { return m.CompressedJSON() },
	TextValue:     nil,
	NumericValue:  func(m Metric) float64 { return m.Value },
	TimeToLive:    nil,
}

// rowMetricsEntityType is a TableRow definition for Metrics, indexed by EntityType.
var rowMetricsEntityType = v.TableRow[Metric]{
	RowName:       "metrics_entity_type",
	PartKeyName:   "entity_type",
	PartKeyValue:  func(m Metric) string { return m.EntityType },
	PartKeyValues: nil,
	SortKeyName:   "id",
	SortKeyValue:  func(m Metric) string { return m.ID },
	JsonValue:     func(m Metric) []byte { return m.CompressedJSON() },
	TextValue:     nil,
	NumericValue:  func(m Metric) float64 { return m.Value },
	TimeToLive:    nil,
}

// rowMetricsByTag is a TableRow definition for Metrics, indexed by Tag.
var rowMetricsByTag = v.TableRow[Metric]{
	RowName:       "metricsByTag",
	PartKeyName:   "tag",
	PartKeyValues: func(m Metric) []string { return m.Tags },
	SortKeyName:   "id",
	SortKeyValue:  func(m Metric) string { return m.ID },
	JsonValue:     func(m Metric) []byte { return m.CompressedJSON() },
}

// NewTable instantiates a new DynamoDB table definition for metrics.
func NewTable(dbClient *dynamodb.Client, env string) v.Table[Metric] {
	if env == "" {
		env = "dev"
	}
	return v.Table[Metric]{
		Client:     dbClient,
		EntityType: "Metric",
		TableName:  "metrics" + "_" + env,
		TTL:        false,
		EntityRow:  rowMetrics,
		IndexRows: map[string]v.TableRow[Metric]{
			rowMetricsEntity.RowName:     rowMetricsEntity,
			rowMetricsEntityType.RowName: rowMetricsEntityType,
			rowMetricsByTag.RowName:      rowMetricsByTag,
		},
	}
}

// NewMemTable creates an in-memory Metric table for testing purposes.
func NewMemTable(table v.Table[Metric]) v.MemTable[Metric] {
	return v.NewMemTable(table)
}

//==============================================================================
// Metric Service
//==============================================================================

// Service is used to manage Mertics in a DynamoDB table.
type Service struct {
	EntityType string
	Table      v.TableReadWriter[Metric]
}

// NewService creates a new Metric service backed by a Versionary Table for specified environment.
func NewService(dbClient *dynamodb.Client, env string) Service {
	table := NewTable(dbClient, env)
	return Service{
		EntityType: table.EntityType,
		Table:      table,
	}
}

// NewMockService creates a new Metric service backed by an in-memory table for testing purposes.
func NewMockService(env string) Service {
	table := NewMemTable(NewTable(nil, env))
	return Service{
		EntityType: table.EntityType,
		Table:      table,
	}
}

//------------------------------------------------------------------------------
// Metric Versions
//------------------------------------------------------------------------------

// Create a Metric in the table.
func (s Service) Create(ctx context.Context, m Metric) (Metric, []string, error) {
	t := tuid.NewID()
	at, _ := t.Time()
	m.ID = t.String()
	m.CreatedAt = at
	problems := m.Validate()
	if len(problems) > 0 {
		return m, problems, fmt.Errorf("error creating %s %s: invalid field(s): %s", s.EntityType, m.ID, strings.Join(problems, ", "))
	}
	return m, problems, s.Table.WriteEntity(ctx, m)

}

// Write a Metric to the Metric table. This method assumes that the Metric has all the required fields.
// It would most likely be used for "refreshing" the index rows in the Metric table.
func (s Service) Write(ctx context.Context, m Metric) (Metric, error) {
	return m, s.Table.WriteEntity(ctx, m)
}

// Read a Metric from the Metric table.
func (s Service) Read(ctx context.Context, id string) (Metric, error) {
	return s.Table.ReadEntity(ctx, id)
}

// Exists checks if a Metric exists in the Metric table.
func (s Service) Exists(ctx context.Context, id string) bool {
	return s.Table.EntityExists(ctx, id)
}

// ReadAsJSON gets a specified Metric from the Metric table, serialized as JSON.
func (s Service) ReadAsJSON(ctx context.Context, id string) ([]byte, error) {
	return s.Table.ReadEntityAsJSON(ctx, id)
}

// Delete a Metric from the Metric table. Deleted Metrics is returned
func (s Service) Delete(ctx context.Context, id string) (Metric, error) {
	return s.Table.DeleteEntityWithID(ctx, id)
}
