package view

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
// View Table
//==============================================================================

// rowViews is a TableRow definition for Views (without versions).
var rowViews = v.TableRow[View]{
	RowName:      "views",
	PartKeyName:  "id",
	PartKeyValue: func(view View) string { return view.ID },
	SortKeyName:  "id",
	SortKeyValue: func(view View) string { return view.ID },
	JsonValue:    func(view View) []byte { return view.CompressedJSON() },
	TimeToLive:   func(view View) int64 { return view.ExpiresAt.Unix() },
}

// rowViewsDate is a TableRow definition for current View versions,
// partitioned by LastSeenOn date.
var rowViewsDate = v.TableRow[View]{
	RowName:      "views_date",
	PartKeyName:  "date",
	PartKeyValue: func(view View) string { return view.CreatedOn() },
	SortKeyName:  "id",
	SortKeyValue: func(view View) string { return view.ID },
	JsonValue:    func(view View) []byte { return view.CompressedJSON() },
	TimeToLive:   func(view View) int64 { return view.ExpiresAt.Unix() },
}

// rowViewsDevice is a TableRow definition for current View versions,
// partitioned by Device ID.
var rowViewsDevice = v.TableRow[View]{
	RowName:      "views_device",
	PartKeyName:  "device_id",
	PartKeyValue: func(view View) string { return view.Client.DeviceID },
	SortKeyName:  "id",
	SortKeyValue: func(view View) string { return view.ID },
	JsonValue:    func(view View) []byte { return view.CompressedJSON() },
	TimeToLive:   func(view View) int64 { return view.ExpiresAt.Unix() },
}

// NewTable instantiates a new DynamoDB View table.
func NewTable(dbClient *dynamodb.Client, env string) v.Table[View] {
	if env == "" {
		env = "dev"
	}
	return v.Table[View]{
		Client:     dbClient,
		EntityType: "View",
		TableName:  "views" + "_" + env,
		TTL:        true,
		EntityRow:  rowViews,
		IndexRows: map[string]v.TableRow[View]{
			rowViewsDate.RowName:   rowViewsDate,
			rowViewsDevice.RowName: rowViewsDevice,
		},
	}
}

// NewMemTable creates an in-memory View table for testing purposes.
func NewMemTable(table v.Table[View]) v.MemTable[View] {
	return v.NewMemTable(table)
}

//==============================================================================
// View Service
//==============================================================================

// Service is used to manage a View database.
type Service struct {
	EntityType string
	Table      v.TableReadWriter[View]
}

// NewService creates a new View service backed by a Versionary table for the specified environment.
func NewService(dbClient *dynamodb.Client, env string) Service {
	table := NewTable(dbClient, env)
	return Service{
		EntityType: table.EntityType,
		Table:      table,
	}
}

// NewMockService creates a new View service backed by an in-memory table for testing purposes.
func NewMockService(env string) Service {
	table := NewMemTable(NewTable(nil, env))
	return Service{
		EntityType: table.EntityType,
		Table:      table,
	}
}

//------------------------------------------------------------------------------
// View Versions
//------------------------------------------------------------------------------

// Create a View in the View table.
func (s Service) Create(ctx context.Context, view View) (View, []string, error) {
	t := tuid.NewID()
	at, _ := t.Time()
	view.ID = t.String()
	view.CreatedAt = at
	view.ExpiresAt = at.AddDate(1, 0, 0)
	problems := view.Validate()
	if len(problems) > 0 {
		return view, problems, fmt.Errorf("error creating %s %s: invalid field(s): %s", s.EntityType, view.ID, strings.Join(problems, ", "))
	}
	err := s.Table.WriteEntity(ctx, view)
	if err != nil {
		return view, problems, fmt.Errorf("error creating %s %s: %w", s.EntityType, view.ID, err)
	}
	return view, problems, nil
}

// Write a View to the View table. This method assumes that the View has all the required fields.
// It would most likely be used for "refreshing" the index rows in the View table.
func (s Service) Write(ctx context.Context, view View) (View, error) {
	return view, s.Table.WriteEntity(ctx, view)
}

// Delete a View from the View table. The deleted View is returned.
func (s Service) Delete(ctx context.Context, id string) (View, error) {
	return s.Table.DeleteEntityWithID(ctx, id)
}

// Exists checks if a View exists in the View table.
func (s Service) Exists(ctx context.Context, id string) bool {
	return s.Table.EntityExists(ctx, id)
}

// Read a specified View from the View table.
func (s Service) Read(ctx context.Context, id string) (View, error) {
	return s.Table.ReadEntity(ctx, id)
}

// ReadAsJSON gets a specified View from the View table, serialized as JSON.
func (s Service) ReadAsJSON(ctx context.Context, id string) ([]byte, error) {
	return s.Table.ReadEntityAsJSON(ctx, id)
}

// ReadViewIDs returns a paginated list of View IDs in the View table.
// Sorting is chronological (or reverse). The offset is the last ID returned in a previous request.
func (s Service) ReadViewIDs(ctx context.Context, reverse bool, limit int, offset string) ([]string, error) {
	return s.Table.ReadEntityIDs(ctx, reverse, limit, offset)
}

// ReadViews returns a paginated list of Views in the View table.
// Sorting is chronological (or reverse). The offset is the last ID returned in a previous request.
// Note that this is a best-effort attempt to return the requested Views, retrieved individually, in parallel.
// It is probably not the best way to page through a large View table.
func (s Service) ReadViews(ctx context.Context, reverse bool, limit int, offset string) []View {
	ids, err := s.Table.ReadEntityIDs(ctx, reverse, limit, offset)
	if err != nil {
		return []View{}
	}
	return s.Table.ReadEntities(ctx, ids)
}

//------------------------------------------------------------------------------
// Views by Date (YYYY-MM-DD)
//------------------------------------------------------------------------------

// ReadDates returns a paginated Date list for which there are Views in the View table.
// Sorting is chronological (or reverse). The offset is the last Date returned in a previous request.
func (s Service) ReadDates(ctx context.Context, reverse bool, limit int, offset string) ([]string, error) {
	return s.Table.ReadPartKeyValues(ctx, rowViewsDate, reverse, limit, offset)
}

// ReadAllDates returns a complete, chronological list of Dates for which there are Views in the View table.
func (s Service) ReadAllDates(ctx context.Context) ([]string, error) {
	return s.Table.ReadAllPartKeyValues(ctx, rowViewsDate)
}

// ReadViewsByDate returns paginated Views by Date. Sorting is chronological (or reverse).
// The offset is the ID of the last View returned in a previous request.
func (s Service) ReadViewsByDate(ctx context.Context, date string, reverse bool, limit int, offset string) ([]View, error) {
	return s.Table.ReadEntitiesFromRow(ctx, rowViewsDate, date, reverse, limit, offset)
}

// ReadViewsByDateAsJSON returns paginated JSON Views by Date. Sorting is chronological (or reverse).
// The offset is the ID of the last View returned in a previous request.
func (s Service) ReadViewsByDateAsJSON(ctx context.Context, date string, reverse bool, limit int, offset string) ([]byte, error) {
	return s.Table.ReadEntitiesFromRowAsJSON(ctx, rowViewsDate, date, reverse, limit, offset)
}

// ReadAllViewsByDate returns the complete list of Views, sorted chronologically by CreatedAt timestamp.
func (s Service) ReadAllViewsByDate(ctx context.Context, date string) ([]View, error) {
	return s.Table.ReadAllEntitiesFromRow(ctx, rowViewsDate, date)
}

// ReadAllViewsByDateAsJSON returns the complete list of Views, serialized as JSON.
func (s Service) ReadAllViewsByDateAsJSON(ctx context.Context, date string) ([]byte, error) {
	return s.Table.ReadAllEntitiesFromRowAsJSON(ctx, rowViewsDate, date)
}

// CountViewsByDate returns a ViewCount for Views in the View table on the specified Date.
func (s Service) CountViewsByDate(ctx context.Context, date string) (Count, error) {
	c := Count{}
	if !util.IsValidDate(date) {
		return c, fmt.Errorf("count views by date: invalid date: %s", date)

	}
	c.Date = date
	limit := 10000
	offset := "-" // before numbers
	views, err := s.ReadViewsByDate(ctx, date, false, limit, offset)
	if err != nil {
		return c, fmt.Errorf("count views by date %s: %w", date, err)
	}
	for len(views) > 0 {
		for _, view := range views {
			c = c.Increment(view)
		}
		offset = views[len(views)-1].ID
		views, err = s.ReadViewsByDate(ctx, date, false, limit, offset)
		if err != nil {
			return c, fmt.Errorf("count views by date %s: %w", date, err)
		}
	}
	return c, nil
}

//------------------------------------------------------------------------------
// Views by Device ID
//------------------------------------------------------------------------------

// ReadDeviceIDs returns a paginated Device ID list for which there are Views in the View table.
// Sorting is chronological (or reverse). The offset is the last Device ID returned in a previous request.
func (s Service) ReadDeviceIDs(ctx context.Context, reverse bool, limit int, offset string) ([]string, error) {
	return s.Table.ReadPartKeyValues(ctx, rowViewsDevice, reverse, limit, offset)
}

// ReadAllDeviceIDs returns a complete, chronological list of Device IDs for which there are Views in the View table.
func (s Service) ReadAllDeviceIDs(ctx context.Context) ([]string, error) {
	return s.Table.ReadAllPartKeyValues(ctx, rowViewsDevice)
}

// ReadViewsByDeviceID returns paginated Views by Device ID. Sorting is chronological (or reverse).
// The offset is the ID of the last View returned in a previous request.
func (s Service) ReadViewsByDeviceID(ctx context.Context, deviceID string, reverse bool, limit int, offset string) ([]View, error) {
	return s.Table.ReadEntitiesFromRow(ctx, rowViewsDevice, deviceID, reverse, limit, offset)
}

// ReadViewsByDeviceIDAsJSON returns paginated JSON Views by Device ID. Sorting is chronological (or reverse).
// The offset is the ID of the last View returned in a previous request.
func (s Service) ReadViewsByDeviceIDAsJSON(ctx context.Context, deviceID string, reverse bool, limit int, offset string) ([]byte, error) {
	return s.Table.ReadEntitiesFromRowAsJSON(ctx, rowViewsDevice, deviceID, reverse, limit, offset)
}

// ReadAllViewsByDeviceID returns the complete list of Views, sorted chronologically by CreatedAt timestamp.
func (s Service) ReadAllViewsByDeviceID(ctx context.Context, deviceID string) ([]View, error) {
	return s.Table.ReadAllEntitiesFromRow(ctx, rowViewsDevice, deviceID)
}

// ReadAllViewsByDeviceIDAsJSON returns the complete list of Views, serialized as JSON.
func (s Service) ReadAllViewsByDeviceIDAsJSON(ctx context.Context, deviceID string) ([]byte, error) {
	return s.Table.ReadAllEntitiesFromRowAsJSON(ctx, rowViewsDevice, deviceID)
}
