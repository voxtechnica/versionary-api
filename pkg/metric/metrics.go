package metric

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
// Metric Table
//==============================================================================

// rowMetrics is a TableRow definition for Metrics, indexed by ID.
var rowMetrics = v.TableRow[Metric]{
	RowName:      "metrics",
	PartKeyName:  "id",
	PartKeyValue: func(m Metric) string { return m.ID },
	PartKeyLabel: func(m Metric) string { return m.String() },
	SortKeyName:  "id",
	SortKeyValue: func(m Metric) string { return m.ID },
	JsonValue:    func(m Metric) []byte { return m.CompressedJSON() },
	TimeToLive:   func(m Metric) int64 { return m.ExpiresAt.Unix() },
}

// rowMetricsEntity is a TableRow definition for Metrics, indexed by Entity ID.
var rowMetricsEntity = v.TableRow[Metric]{
	RowName:      "metrics_entity",
	PartKeyName:  "entity_id",
	PartKeyValue: func(m Metric) string { return m.EntityID },
	SortKeyName:  "id",
	SortKeyValue: func(m Metric) string { return m.ID },
	JsonValue:    func(m Metric) []byte { return m.CompressedJSON() },
	NumericValue: func(m Metric) float64 { return m.Value },
	TimeToLive:   func(m Metric) int64 { return m.ExpiresAt.Unix() },
}

// rowMetricsEntityType is a TableRow definition for Metrics, indexed by EntityType.
var rowMetricsEntityType = v.TableRow[Metric]{
	RowName:      "metrics_entity_type",
	PartKeyName:  "entity_type",
	PartKeyValue: func(m Metric) string { return m.EntityType },
	SortKeyName:  "id",
	SortKeyValue: func(m Metric) string { return m.ID },
	JsonValue:    func(m Metric) []byte { return m.CompressedJSON() },
	NumericValue: func(m Metric) float64 { return m.Value },
	TimeToLive:   func(m Metric) int64 { return m.ExpiresAt.Unix() },
}

// rowMetricsTag is a TableRow definition for Metrics, indexed by Tag.
var rowMetricsTag = v.TableRow[Metric]{
	RowName:       "metrics_tag",
	PartKeyName:   "tag",
	PartKeyValues: func(m Metric) []string { return m.Tags },
	SortKeyName:   "id",
	SortKeyValue:  func(m Metric) string { return m.ID },
	JsonValue:     func(m Metric) []byte { return m.CompressedJSON() },
	NumericValue:  func(m Metric) float64 { return m.Value },
	TimeToLive:    func(m Metric) int64 { return m.ExpiresAt.Unix() },
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
		TTL:        true,
		EntityRow:  rowMetrics,
		IndexRows: map[string]v.TableRow[Metric]{
			rowMetricsEntity.RowName:     rowMetricsEntity,
			rowMetricsEntityType.RowName: rowMetricsEntityType,
			rowMetricsTag.RowName:        rowMetricsTag,
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

// Service is used to manage Metrics in a DynamoDB table.
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
// Metric Service Methods
//------------------------------------------------------------------------------

// Create a Metric in the table.
func (s Service) Create(ctx context.Context, m Metric) (Metric, []string, error) {
	t := tuid.NewID()
	at, _ := t.Time()
	m.ID = t.String()
	m.CreatedAt = at
	m.ExpiresAt = at.AddDate(1, 0, 0)
	if m.Title == "" {
		m.Title = "Untitled"
	}
	problems := m.Validate()
	if len(problems) > 0 {
		return m, problems, fmt.Errorf("error creating %s %s: invalid field(s): %s", s.EntityType, m.ID, strings.Join(problems, ", "))
	}
	err := s.Table.WriteEntity(ctx, m)
	if err != nil {
		return m, problems, fmt.Errorf("error creating %s %s %s: %w", s.EntityType, m.ID, m.Title, err)
	}
	return m, problems, nil
}

// Write a Metric to the Metric table. This method assumes that the Metric has all the required fields.
// It would most likely be used for "refreshing" the index rows in the Metric table.
func (s Service) Write(ctx context.Context, m Metric) (Metric, error) {
	return m, s.Table.WriteEntity(ctx, m)
}

// Read a specified Metric from the Metric table.
func (s Service) Read(ctx context.Context, id string) (Metric, error) {
	return s.Table.ReadEntity(ctx, id)
}

// Exists checks if a Metric exists in the Metric table.
func (s Service) Exists(ctx context.Context, id string) bool {
	return s.Table.EntityExists(ctx, id)
}

// ReadAsJSON reads a specified Metric from the Metric table, serialized as JSON.
func (s Service) ReadAsJSON(ctx context.Context, id string) ([]byte, error) {
	return s.Table.ReadEntityAsJSON(ctx, id)
}

// Delete a Metric from the Metric table. The deleted Metric is returned.
func (s Service) Delete(ctx context.Context, id string) (Metric, error) {
	return s.Table.DeleteEntityWithID(ctx, id)
}

// ReadMetricIDs returns a paginated list of Metric IDs from the Metric table.
func (s Service) ReadMetricIDs(ctx context.Context, reverse bool, limit int, offset string) ([]string, error) {
	return s.Table.ReadEntityIDs(ctx, reverse, limit, offset)
}

// ReadMetricLabels returns a paginated list of Metric IDs and Labels from the Metric table.
// This is the preferred, more performant method for 'browsing' metrics.
func (s Service) ReadMetricLabels(ctx context.Context, reverse bool, limit int, offset string) ([]v.TextValue, error) {
	return s.Table.ReadEntityLabels(ctx, reverse, limit, offset)
}

// ReadMetrics returns a paginated list of Metrics in the Metric table.
// This 'expensive' method returns the full Metric objects, retrieved with parallel reads.
func (s Service) ReadMetrics(ctx context.Context, reverse bool, limit int, offset string) []Metric {
	ids, err := s.Table.ReadEntityIDs(ctx, reverse, limit, offset)
	if err != nil {
		return []Metric{}
	}
	return s.Table.ReadEntities(ctx, ids)
}

//------------------------------------------------------------------------------
// Metrics by Entity ID
//------------------------------------------------------------------------------

// ReadAllEntityIDs returns a list of all Entity IDs in the table.
func (s Service) ReadAllEntityIDs(ctx context.Context) ([]string, error) {
	return s.Table.ReadAllPartKeyValues(ctx, rowMetricsEntity)
}

// ReadMetricsByEntityID returns a paginated list of Metrics for a specified Entity ID.
func (s Service) ReadMetricsByEntityID(ctx context.Context, entityID string, reverse bool, limit int, offset string) ([]Metric, error) {
	return s.Table.ReadEntitiesFromRow(ctx, rowMetricsEntity, entityID, reverse, limit, offset)
}

// ReadMetricsByEntityIDAsJSON returns a paginated list of Metrics for a specified Entity ID, serialized as JSON.
func (s Service) ReadMetricsByEntityIDAsJSON(ctx context.Context, entityID string, reverse bool, limit int, offset string) ([]byte, error) {
	return s.Table.ReadEntitiesFromRowAsJSON(ctx, rowMetricsEntity, entityID, reverse, limit, offset)
}

// ReadAllMetricsByEntityID returns the complete list of Metrics for a specified Entity ID.
func (s Service) ReadAllMetricsByEntityID(ctx context.Context, entityID string) ([]Metric, error) {
	return s.Table.ReadAllEntitiesFromRow(ctx, rowMetricsEntity, entityID)
}

// ReadAllMetricsByEntityIDAsJSON returns the complete list of Metrics for a specified Entity ID, serialized as JSON.
func (s Service) ReadAllMetricsByEntityIDAsJSON(ctx context.Context, entityID string) ([]byte, error) {
	return s.Table.ReadAllEntitiesFromRowAsJSON(ctx, rowMetricsEntity, entityID)
}

// ReadMetricRangeByEntityID returns a list of Metrics for a specified Entity ID and date range.
// The start date is inclusive, and the end date is exclusive.
// The dates are expected to be in the format "yyyy-mm-dd".
func (s Service) ReadMetricRangeByEntityID(ctx context.Context, entityID string, startDate, endDate string, reverse bool) ([]Metric, error) {
	if entityID == "" {
		return []Metric{}, nil
	}
	start, end, err := util.DateRangeIDs(startDate, endDate)
	if err != nil {
		return []Metric{}, fmt.Errorf("read metric range for entity %s: %w", entityID, err)
	}
	return s.Table.ReadEntityRangeFromRow(ctx, rowMetricsEntity, entityID, start, end, reverse)
}

// ReadMetricStatByEntityID returns a MetricStats object for a specified Entity ID.
func (s Service) ReadMetricStatByEntityID(ctx context.Context, entityID string) (MetricStat, error) {
	if entityID == "" {
		return MetricStat{}, fmt.Errorf("read MetricStat for unspecified entity: %w", v.ErrNotFound)
	}
	values, err := s.Table.ReadAllNumericValues(ctx, rowMetricsEntity, entityID, false)
	if err != nil {
		return MetricStat{}, fmt.Errorf("read MetricStat for entity %s: %w", entityID, err)
	}
	if len(values) == 0 {
		return MetricStat{}, fmt.Errorf("read MetricStat for entity %s: %w", entityID, v.ErrNotFound)
	}
	return NewMetricStat(entityID, "", "", values), nil
}

// ReadMetricStatRangeByEntityID generates a MetricStat for a specified Entity ID and date range.
// The start date is inclusive, and the end date is exclusive.
// The dates are expected to be in the format "yyyy-mm-dd".
func (s Service) ReadMetricStatRangeByEntityID(ctx context.Context, entityID, startDate, endDate string) (MetricStat, error) {
	if entityID == "" {
		return MetricStat{}, fmt.Errorf("read MetricStat range for unspecified entity: %w", v.ErrNotFound)
	}
	start, end, err := util.DateRangeIDs(startDate, endDate)
	if err != nil {
		return MetricStat{}, fmt.Errorf("read MetricStat range for entity %s: %w", entityID, err)
	}
	values, err := s.Table.ReadNumericValueRange(ctx, rowMetricsEntity, entityID, start, end, false)
	if err != nil {
		return MetricStat{}, fmt.Errorf("read MetricStat range for entity %s: %w", entityID, err)
	}
	if len(values) == 0 {
		return MetricStat{}, fmt.Errorf("read MetricStat range for entity %s: %w", entityID, v.ErrNotFound)
	}
	return NewMetricStat(entityID, "", "", values), nil
}

//------------------------------------------------------------------------------
// Metrics by Entity Type
//------------------------------------------------------------------------------

// ReadAllEntityTypes returns a list of all Entity Types in the table.
func (s Service) ReadAllEntityTypes(ctx context.Context) ([]string, error) {
	return s.Table.ReadAllPartKeyValues(ctx, rowMetricsEntityType)
}

// ReadMetricsByEntityType returns a paginated list of Metrics for a specified Entity Type.
func (s Service) ReadMetricsByEntityType(ctx context.Context, entityType string, reverse bool, limit int, offset string) ([]Metric, error) {
	return s.Table.ReadEntitiesFromRow(ctx, rowMetricsEntityType, entityType, reverse, limit, offset)
}

// ReadMetricsByEntityTypeAsJSON returns a paginated list of Metrics for a specified Entity Type, serialized as JSON.
func (s Service) ReadMetricsByEntityTypeAsJSON(ctx context.Context, entityType string, reverse bool, limit int, offset string) ([]byte, error) {
	return s.Table.ReadEntitiesFromRowAsJSON(ctx, rowMetricsEntityType, entityType, reverse, limit, offset)
}

// ReadAllMetricsByEntityType returns the complete list of Metrics for a specified Entity Type.
func (s Service) ReadAllMetricsByEntityType(ctx context.Context, entityType string) ([]Metric, error) {
	return s.Table.ReadAllEntitiesFromRow(ctx, rowMetricsEntityType, entityType)
}

// ReadAllMetricsByEntityTypeAsJSON returns the complete list of Metrics for a specified Entity Type, serialized as JSON.
func (s Service) ReadAllMetricsByEntityTypeAsJSON(ctx context.Context, entityType string) ([]byte, error) {
	return s.Table.ReadAllEntitiesFromRowAsJSON(ctx, rowMetricsEntityType, entityType)
}

// ReadMetricRangeByEntityType returns a list of Metrics for a specified Entity Type and date range.
// The start date is inclusive, and the end date is exclusive.
// The dates are expected to be in the format "yyyy-mm-dd".
func (s Service) ReadMetricRangeByEntityType(ctx context.Context, entityType, startDate, endDate string, reverse bool) ([]Metric, error) {
	if entityType == "" {
		return []Metric{}, nil
	}
	start, end, err := util.DateRangeIDs(startDate, endDate)
	if err != nil {
		return []Metric{}, fmt.Errorf("read metric range for entity type %s: %w", entityType, err)
	}
	return s.Table.ReadEntityRangeFromRow(ctx, rowMetricsEntityType, entityType, start, end, reverse)
}

// ReadMetricStatByEntityType returns a MetricStats object for a specified Entity Type.
func (s Service) ReadMetricStatByEntityType(ctx context.Context, entityType string) (MetricStat, error) {
	if entityType == "" {
		return MetricStat{}, fmt.Errorf("read MetricStat for unspecified entity type: %w", v.ErrNotFound)
	}
	values, err := s.Table.ReadAllNumericValues(ctx, rowMetricsEntityType, entityType, false)
	if err != nil {
		return MetricStat{}, fmt.Errorf("read MetricStat for entity type %s: %w", entityType, err)
	}
	if len(values) == 0 {
		return MetricStat{}, fmt.Errorf("read MetricStat for entity type %s: %w", entityType, v.ErrNotFound)
	}
	return NewMetricStat("", entityType, "", values), nil
}

// ReadMetricStatRangeByEntityType generates a MetricStat for a specified Entity Type and date range.
// The start date is inclusive, and the end date is exclusive.
// The dates are expected to be in the format "yyyy-mm-dd".
func (s Service) ReadMetricStatRangeByEntityType(ctx context.Context, entityType, startDate, endDate string) (MetricStat, error) {
	if entityType == "" {
		return MetricStat{}, fmt.Errorf("read MetricStat range for unspecified entity type: %w", v.ErrNotFound)
	}
	start, end, err := util.DateRangeIDs(startDate, endDate)
	if err != nil {
		return MetricStat{}, fmt.Errorf("read MetricStat range for entity type %s: %w", entityType, err)
	}
	values, err := s.Table.ReadNumericValueRange(ctx, rowMetricsEntityType, entityType, start, end, false)
	if err != nil {
		return MetricStat{}, fmt.Errorf("read MetricStat range for entity type %s: %w", entityType, err)
	}
	if len(values) == 0 {
		return MetricStat{}, fmt.Errorf("read MetricStat range for entity type %s: %w", entityType, v.ErrNotFound)
	}
	return NewMetricStat("", entityType, "", values), nil
}

//------------------------------------------------------------------------------
// Metrics by Tag
//------------------------------------------------------------------------------

// ReadAllTags returns a list of all Tags in the Metrics table.
func (s Service) ReadAllTags(ctx context.Context) ([]string, error) {
	return s.Table.ReadAllPartKeyValues(ctx, rowMetricsTag)
}

// ReadMetricsByTag returns a paginated list of Metrics for a specified Tag.
func (s Service) ReadMetricsByTag(ctx context.Context, tag string, reverse bool, limit int, offset string) ([]Metric, error) {
	return s.Table.ReadEntitiesFromRow(ctx, rowMetricsTag, tag, reverse, limit, offset)
}

// ReadMetricsByTagAsJSON returns a paginated list of Metrics for a specified Tag, serialized as JSON.
func (s Service) ReadMetricsByTagAsJSON(ctx context.Context, tag string, reverse bool, limit int, offset string) ([]byte, error) {
	return s.Table.ReadEntitiesFromRowAsJSON(ctx, rowMetricsTag, tag, reverse, limit, offset)
}

// ReadAllMetricsByTag returns the complete list of Metrics, sorted chronologically.
func (s Service) ReadAllMetricsByTag(ctx context.Context, tag string) ([]Metric, error) {
	return s.Table.ReadAllEntitiesFromRow(ctx, rowMetricsTag, tag)
}

// ReadAllMetricsByTagsAsJSON returns the complete list of Metrics, serialized as JSON.
func (s Service) ReadAllMetricsByTagAsJSON(ctx context.Context, tag string) ([]byte, error) {
	return s.Table.ReadAllEntitiesFromRowAsJSON(ctx, rowMetricsTag, tag)
}

// ReadMetricRangeByTag returns a list of Metrics for a specified Tag and date range.
// The start date is inclusive, and the end date is exclusive.
// The dates are expected to be in the format "yyyy-mm-dd".
func (s Service) ReadMetricRangeByTag(ctx context.Context, tag, startDate, endDate string, reverse bool) ([]Metric, error) {
	if tag == "" {
		return []Metric{}, nil
	}
	start, end, err := util.DateRangeIDs(startDate, endDate)
	if err != nil {
		return []Metric{}, fmt.Errorf("read metric range for tag %s: %w", tag, err)
	}
	return s.Table.ReadEntityRangeFromRow(ctx, rowMetricsTag, tag, start, end, reverse)
}

// ReadMetricStatByTag returns a MetricStats object for a specified Tag.
func (s Service) ReadMetricStatByTag(ctx context.Context, tag string) (MetricStat, error) {
	if tag == "" {
		return MetricStat{}, fmt.Errorf("read MetricStat for unspecified tag: %w", v.ErrNotFound)
	}
	values, err := s.Table.ReadAllNumericValues(ctx, rowMetricsTag, tag, false)
	if err != nil {
		return MetricStat{}, fmt.Errorf("read MetricStat for tag %s: %w", tag, err)
	}
	if len(values) == 0 {
		return MetricStat{}, fmt.Errorf("read MetricStat for tag %s: %w", tag, v.ErrNotFound)
	}
	return NewMetricStat("", "", tag, values), nil
}

// ReadMetricStatRangeByTag generates a MetricStat for a specified Tag and date range.
// The start date is inclusive, and the end date is exclusive.
// The dates are expected to be in the format "yyyy-mm-dd".
func (s Service) ReadMetricStatRangeByTag(ctx context.Context, tag, startDate, endDate string) (MetricStat, error) {
	if tag == "" {
		return MetricStat{}, fmt.Errorf("read MetricStat range for unspecified tag: %w", v.ErrNotFound)
	}
	start, end, err := util.DateRangeIDs(startDate, endDate)
	if err != nil {
		return MetricStat{}, fmt.Errorf("read MetricStat range for tag %s: %w", tag, err)
	}
	values, err := s.Table.ReadNumericValueRange(ctx, rowMetricsTag, tag, start, end, false)
	if err != nil {
		return MetricStat{}, fmt.Errorf("read MetricStat range for tag %s: %w", tag, err)
	}
	if len(values) == 0 {
		return MetricStat{}, fmt.Errorf("read MetricStat range for tag %s: %w", tag, v.ErrNotFound)
	}
	return NewMetricStat("", "", tag, values), nil
}
