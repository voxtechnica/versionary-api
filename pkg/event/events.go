package event

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/voxtechnica/tuid-go"
	v "github.com/voxtechnica/versionary"
)

// rowEvents is a TableRow definition for Events by Event ID. Events are not versioned.
var rowEvents = v.TableRow[Event]{
	RowName:       "events",
	PartKeyName:   "id",
	PartKeyValue:  func(e Event) string { return e.ID },
	PartKeyValues: nil,
	SortKeyName:   "id",
	SortKeyValue:  func(e Event) string { return e.ID },
	JsonValue:     func(e Event) []byte { return e.CompressedJSON() },
	TextValue:     nil,
	NumericValue:  nil,
	TimeToLive:    func(e Event) int64 { return e.ExpiresAt.Unix() },
}

// rowEventsDate is a TableRow definition for Events by Event Date.
var rowEventsDate = v.TableRow[Event]{
	RowName:       "events_date",
	PartKeyName:   "date",
	PartKeyValue:  func(e Event) string { return e.CreatedOn() },
	PartKeyValues: nil,
	SortKeyName:   "id",
	SortKeyValue:  func(e Event) string { return e.ID },
	JsonValue:     func(e Event) []byte { return e.CompressedJSON() },
	TextValue:     nil,
	NumericValue:  nil,
	TimeToLive:    func(e Event) int64 { return e.ExpiresAt.Unix() },
}

// rowEventsEntity is a TableRow definition for Events by Entity ID.
var rowEventsEntity = v.TableRow[Event]{
	RowName:       "events_entity",
	PartKeyName:   "entity_id",
	PartKeyValue:  nil,
	PartKeyValues: func(e Event) []string { return e.IDs() },
	SortKeyName:   "id",
	SortKeyValue:  func(e Event) string { return e.ID },
	JsonValue:     func(e Event) []byte { return e.CompressedJSON() },
	TextValue:     nil,
	NumericValue:  nil,
	TimeToLive:    func(e Event) int64 { return e.ExpiresAt.Unix() },
}

// rowEventsLogLevel is a TableRow definition for Events by LogLevel.
var rowEventsLogLevel = v.TableRow[Event]{
	RowName:       "events_log_level",
	PartKeyName:   "log_level",
	PartKeyValue:  func(e Event) string { return string(e.LogLevel) },
	PartKeyValues: nil,
	SortKeyName:   "id",
	SortKeyValue:  func(e Event) string { return e.ID },
	JsonValue:     func(e Event) []byte { return e.CompressedJSON() },
	TextValue:     nil,
	NumericValue:  nil,
	TimeToLive:    func(e Event) int64 { return e.ExpiresAt.Unix() },
}

// NewEventTable instantiates a new DynamoDB table definition for events.
func NewEventTable(dbClient *dynamodb.Client, env string) v.Table[Event] {
	if env == "" {
		env = "dev"
	}
	return v.Table[Event]{
		Client:     dbClient,
		EntityType: "Event",
		TableName:  "events" + "_" + env,
		TTL:        true,
		EntityRow:  rowEvents,
		IndexRows: map[string]v.TableRow[Event]{
			rowEventsDate.RowName:     rowEventsDate,
			rowEventsEntity.RowName:   rowEventsEntity,
			rowEventsLogLevel.RowName: rowEventsLogLevel,
		},
	}
}

// NewEventMemTable creates an in-memory Event table for testing purposes.
func NewEventMemTable(table v.Table[Event]) v.MemTable[Event] {
	return v.NewMemTable(table)
}

// EventService is used to manage the Event log in a DynamoDB table.
type EventService struct {
	EntityType string
	Table      v.TableReadWriter[Event]
}

// Create an Event in the Event log.
func (s EventService) Create(ctx context.Context, e Event) (Event, error) {
	t := tuid.NewID()
	at, _ := t.Time()
	e.ID = t.String()
	e.CreatedAt = at
	e.ExpiresAt = at.AddDate(1, 0, 0)
	if e.LogLevel == "" {
		e.LogLevel = INFO
	}
	if v := e.Validate(); len(v) > 0 {
		return e, fmt.Errorf("error creating %s %s: invalid field(s): %s", s.EntityType, e.ID, strings.Join(v, ", "))
	}
	return e, s.Table.WriteEntity(ctx, e)
}

// Write an Event to the Event log. This method assumes that the Event has all the required fields.
// It would most likely be used for "refreshing" the index rows in the Event table.
func (s EventService) Write(ctx context.Context, e Event) (Event, error) {
	return e, s.Table.WriteEntity(ctx, e)
}

// Delete an Event from the Event log. The deleted Event is returned.
func (s EventService) Delete(ctx context.Context, id string) (Event, error) {
	return s.Table.DeleteEntityWithID(ctx, id)
}

// Read a specified Event from the Event log.
func (s EventService) Read(ctx context.Context, id string) (Event, error) {
	return s.Table.ReadEntity(ctx, id)
}

// ReadAsJSON gets a specified Event from the Event log, serialized as JSON.
func (s EventService) ReadAsJSON(ctx context.Context, id string) ([]byte, error) {
	return s.Table.ReadEntityAsJSON(ctx, id)
}

// ReadEventIDs returns a paginated list of Event IDs in the Event log.
// Sorting is chronological (or reverse). The offset is the last ID returned in a previous request.
func (s EventService) ReadEventIDs(ctx context.Context, reverse bool, limit int, offset string) ([]string, error) {
	return s.Table.ReadEntityIDs(ctx, reverse, limit, offset)
}

// ReadEvents returns a paginated list of Events in the Event log.
// Sorting is chronological (or reverse). The offset is the last ID returned in a previous request.
// Note that this is a best-effort attempt to return the requested Events, retrieved individually, in parallel.
// It is probably not the best way to page through a large Event log.
func (s EventService) ReadEvents(ctx context.Context, reverse bool, limit int, offset string) []Event {
	ids, err := s.Table.ReadEntityIDs(ctx, reverse, limit, offset)
	if err != nil {
		return []Event{}
	}
	return s.Table.ReadEntities(ctx, ids)
}

// ReadRecentEvents returns a list of recent Events in the Event log, retrieved as Events by Date.
// Starting with the most recent day, we gather events in reverse-chronological order until the limit is reached.
func (s EventService) ReadRecentEvents(ctx context.Context, limit int) ([]Event, error) {
	es := make([]Event, 0, limit)
	dates, err := s.ReadDates(ctx, true, 365, "9999-99-99")
	if err != nil {
		return es, fmt.Errorf("read recent events: unable to read dates: %w", err)
	}
	for _, d := range dates {
		dayEvents, err := s.ReadEventsByDate(ctx, d, true, limit-len(es), tuid.MaxID)
		if err != nil {
			return es, fmt.Errorf("read recent events: unable to read events for date %s: %w", d, err)
		}
		es = append(es, dayEvents...)
		if len(es) >= limit {
			break
		}
	}
	return es, nil
}

// ReadDates returns a paginated list of dates for which there are Events in the Event log.
// Sorting is chronological (or reverse). The offset is the last date returned in a previous request.
func (s EventService) ReadDates(ctx context.Context, reverse bool, limit int, offset string) ([]string, error) {
	return s.Table.ReadPartKeyValues(ctx, rowEventsDate, reverse, limit, offset)
}

// ReadAllDates returns a complete, chronological list of dates for which there are Events in the Event log.
func (s EventService) ReadAllDates(ctx context.Context) ([]string, error) {
	return s.Table.ReadAllPartKeyValues(ctx, rowEventsDate)
}

// ReadEventsByDate returns paginated Events by date, expressed as an ISO-8601 formatted yyyy-mm-dd string.
// Sorting is chronological (or reverse). The offset is the ID of the last Event returned in a previous request.
func (s EventService) ReadEventsByDate(ctx context.Context, date string, reverse bool, limit int, offset string) ([]Event, error) {
	return s.Table.ReadEntitiesFromRow(ctx, rowEventsDate, date, reverse, limit, offset)
}

// ReadEventsByDateAsJSON returns paginated JSON Events by date, expressed as an ISO-8601 formatted yyyy-mm-dd string.
// Sorting is chronological (or reverse). The offset is the ID of the last Event returned in a previous request.
func (s EventService) ReadEventsByDateAsJSON(ctx context.Context, date string, reverse bool, limit int, offset string) ([]byte, error) {
	return s.Table.ReadEntitiesFromRowAsJSON(ctx, rowEventsDate, date, reverse, limit, offset)
}

// ReadEntityIDs returns a paginated list of entity IDs for which there are Events in the Event log.
// Sorting is chronological (or reverse). The offset is the last entity ID returned in a previous request.
func (s EventService) ReadEntityIDs(ctx context.Context, reverse bool, limit int, offset string) ([]string, error) {
	return s.Table.ReadPartKeyValues(ctx, rowEventsEntity, reverse, limit, offset)
}

// ReadEventsByEntityID returns paginated Events by entity ID.
// Sorting is chronological (or reverse). The offset is the ID of the last Event returned in a previous request.
func (s EventService) ReadEventsByEntityID(ctx context.Context, entityID string, reverse bool, limit int, offset string) ([]Event, error) {
	return s.Table.ReadEntitiesFromRow(ctx, rowEventsEntity, entityID, reverse, limit, offset)
}

// ReadEventsByEntityIDAsJSON returns paginated Events by entity ID.
// Sorting is chronological (or reverse). The offset is the ID of the last Event returned in a previous request.
func (s EventService) ReadEventsByEntityIDAsJSON(ctx context.Context, entityID string, reverse bool, limit int, offset string) ([]byte, error) {
	return s.Table.ReadEntitiesFromRowAsJSON(ctx, rowEventsEntity, entityID, reverse, limit, offset)
}

// ReadLogLevels returns a paginated list of log levels for which there are Events in the Event log.
// Sorting is alphabetical (or reverse). The offset is the last LogLevel returned in a previous request.
func (s EventService) ReadLogLevels(ctx context.Context, reverse bool, limit int, offset string) ([]string, error) {
	return s.Table.ReadPartKeyValues(ctx, rowEventsLogLevel, reverse, limit, offset)
}

// ReadAllLogLevels returns a complete, alphabetical list of log levels for which there are Events in the Event log.
func (s EventService) ReadAllLogLevels(ctx context.Context) ([]string, error) {
	return s.Table.ReadAllPartKeyValues(ctx, rowEventsLogLevel)
}

// ReadEventsByLogLevel returns paginated Events by log level (TRACE, DEBUG, INFO, WARN, ERROR, FATAL).
func (s EventService) ReadEventsByLogLevel(ctx context.Context, logLevel string, reverse bool, limit int, offset string) ([]Event, error) {
	return s.Table.ReadEntitiesFromRow(ctx, rowEventsLogLevel, strings.ToUpper(logLevel), reverse, limit, offset)
}

// ReadEventsByLogLevelAsJSON returns paginated JSON Events by log level (TRACE, DEBUG, INFO, WARN, ERROR, FATAL).
func (s EventService) ReadEventsByLogLevelAsJSON(ctx context.Context, logLevel string, reverse bool, limit int, offset string) ([]byte, error) {
	return s.Table.ReadEntitiesFromRowAsJSON(ctx, rowEventsLogLevel, strings.ToUpper(logLevel), reverse, limit, offset)
}
