package event

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/voxtechnica/tuid-go"
	v "github.com/voxtechnica/versionary"
)

var (
	// Event Service
	ctx      = context.Background()
	table    = NewTable(nil, "test")
	memTable = NewMemTable(table)
	service  = Service{EntityType: "Event", Table: memTable}

	// Known timestamps
	t1 = time.Date(2022, time.April, 1, 12, 0, 0, 0, time.UTC)
	t2 = time.Date(2022, time.April, 1, 13, 0, 0, 0, time.UTC)
	t3 = time.Date(2022, time.April, 1, 14, 0, 0, 0, time.UTC)
	t4 = time.Date(2022, time.April, 2, 12, 0, 0, 0, time.UTC)
	t5 = time.Date(2022, time.April, 3, 12, 0, 0, 0, time.UTC)

	// Known entity IDs
	id1 = tuid.NewID().String()
	id2 = tuid.NewID().String()
	id3 = tuid.NewID().String()

	// Known events
	e1 = Event{
		ID:         tuid.NewIDWithTime(t1).String(),
		UserID:     tuid.NewID().String(),
		EntityID:   id1,
		EntityType: "Thing",
		OtherIDs:   []string{id2, id3},
		LogLevel:   INFO,
		Message:    "Test event 1",
		CreatedAt:  t1,
		ExpiresAt:  t1.AddDate(1, 0, 0),
	}
	e2 = Event{
		ID:         tuid.NewIDWithTime(t2).String(),
		EntityID:   id2,
		EntityType: "Thing",
		OtherIDs:   []string{id1},
		LogLevel:   INFO,
		Message:    "Test event 2",
		CreatedAt:  t2,
		ExpiresAt:  t2.AddDate(1, 0, 0),
	}
	e3 = Event{
		ID:         tuid.NewIDWithTime(t3).String(),
		EntityID:   id1,
		EntityType: "Thing",
		LogLevel:   INFO,
		Message:    "Test event 3",
		CreatedAt:  t3,
		ExpiresAt:  t3.AddDate(1, 0, 0),
	}
	e4 = Event{
		ID:         tuid.NewIDWithTime(t4).String(),
		EntityID:   id2,
		EntityType: "Thing",
		LogLevel:   WARN,
		Message:    "Test event 4",
		CreatedAt:  t4,
		ExpiresAt:  t4.AddDate(1, 0, 0),
	}
	e5 = Event{
		ID:         tuid.NewIDWithTime(t5).String(),
		EntityID:   id3,
		EntityType: "Thing",
		LogLevel:   ERROR,
		Message:    "Test event 5",
		CreatedAt:  t5,
		ExpiresAt:  t5.AddDate(1, 0, 0),
	}
	knownEvents   = []Event{e1, e2, e3, e4, e5}
	knownEventIDs = []string{e1.ID, e2.ID, e3.ID, e4.ID, e5.ID}
)

func TestMain(m *testing.M) {
	// Check the table/row definitions
	if !memTable.IsValid() {
		log.Fatal("invalid table configuration")
	}
	// Write known events
	for _, e := range knownEvents {
		if _, err := service.Write(ctx, e); err != nil {
			log.Fatal(err)
		}
	}
	// Run the tests
	m.Run()
}

func TestCreateReadDelete(t *testing.T) {
	expect := assert.New(t)
	// Create an event
	e, _, err := service.Create(ctx, Event{
		EntityID:   id3,
		EntityType: "Thing",
		LogLevel:   DEBUG,
		Message:    "CRUD Test",
	})
	if expect.NoError(err) {
		// Read the event
		eCheck, err := service.Read(ctx, e.ID)
		if expect.NoError(err) {
			// Check the event
			expect.Equal(e, eCheck)
		}
		// Read the event as JSON
		eCheckJSON, err := service.ReadAsJSON(ctx, e.ID)
		if expect.NoError(err) {
			expect.Contains(string(eCheckJSON), e.ID)
		}
		// Delete the event
		eDelete, err := service.Delete(ctx, e.ID)
		if expect.NoError(err) {
			// Check the event
			expect.Equal(e, eDelete)
		}
		// Read the event
		_, err = service.Read(ctx, e.ID)
		expect.ErrorIs(err, v.ErrNotFound, "expected ErrNotFound")
	}
}

func TestReadEventIDs(t *testing.T) {
	expect := assert.New(t)
	ids, err := service.ReadEventIDs(ctx, false, 10, tuid.MinID)
	if expect.NoError(err) {
		// Check the IDs (note that the CRUD test event may also be present)
		expect.GreaterOrEqual(len(ids), 5)
		expect.Subset(ids, knownEventIDs)
	}
}

func TestReadEvents(t *testing.T) {
	expect := assert.New(t)
	events := service.ReadEvents(ctx, false, 10, tuid.MinID)
	expect.GreaterOrEqual(len(events), 5)
	ids := v.Map(events, func(e Event) string { return e.ID })
	expect.Subset(ids, knownEventIDs)
}

func TestReadRecentEvents(t *testing.T) {
	expect := assert.New(t)
	events, err := service.ReadRecentEvents(ctx, 5)
	if expect.NoError(err) {
		// Expect 5 recent events, in reverse chronological order
		expect.Equal(5, len(events))
		for i := 1; i < len(events); i++ {
			expect.Less(events[i].CreatedAt, events[i-1].CreatedAt)
		}
	}
}

func TestReadDates(t *testing.T) {
	expect := assert.New(t)
	dates, err := service.ReadDates(ctx, false, 3, "0")
	if expect.NoError(err) {
		// Expect 3 dates, in chronological order
		expect.Equal(3, len(dates))
		for i := 1; i < len(dates); i++ {
			expect.Greater(dates[i], dates[i-1])
		}
	}
	dates, err = service.ReadDates(ctx, true, 3, "9999-99-99")
	if expect.NoError(err) {
		// Expect 3 dates, in reverse chronological order
		expect.Equal(3, len(dates))
		for i := 1; i < len(dates); i++ {
			expect.Less(dates[i], dates[i-1])
		}
	}
}

func TestReadAllDates(t *testing.T) {
	expect := assert.New(t)
	dates, err := service.ReadAllDates(ctx)
	if assert.NoError(t, err) {
		// Expect at least 3 dates, in chronological order
		expect.GreaterOrEqual(len(dates), 3)
		for i := 1; i < len(dates); i++ {
			expect.Greater(dates[i], dates[i-1])
		}
	}
}

func TestReadEventsByDate(t *testing.T) {
	expect := assert.New(t)
	// Read all events
	d := t1.Format("2006-01-02")
	all := []Event{e1, e2, e3}
	events, err := service.ReadEventsByDate(ctx, d, false, 10, tuid.MinID)
	if expect.NoError(err) {
		expect.Equal(len(all), len(events))
		expect.Equal(all, events)
	}
	// Read only the first event
	events, err = service.ReadEventsByDate(ctx, d, false, 1, tuid.MinID)
	if expect.NoError(err) {
		expect.Equal(1, len(events))
		expect.Equal(e1, events[0])
	}
	// Read only the last event
	events, err = service.ReadEventsByDate(ctx, d, true, 1, tuid.MaxID)
	if expect.NoError(err) {
		expect.Equal(1, len(events))
		expect.Equal(e3, events[0])
	}
	// Read zero events for an invalid date
	events, err = service.ReadEventsByDate(ctx, "0", false, 1, tuid.MinID)
	if expect.NoError(err) {
		expect.Equal(0, len(events))
	}
	// Read zero events for a missing date
	events, err = service.ReadEventsByDate(ctx, "", false, 1, tuid.MinID)
	if expect.NoError(err) {
		expect.Equal(0, len(events))
	}
}

func TestReadEntityIDs(t *testing.T) {
	expect := assert.New(t)
	// Read all entity IDs
	all := []string{id1, id2, id3, e1.UserID}
	ids, err := service.ReadEntityIDs(ctx, false, 10, tuid.MinID)
	if expect.NoError(err) {
		expect.Equal(len(all), len(ids))
		expect.Equal(all, ids)
	}
	// Read only the first entity ID
	ids, err = service.ReadEntityIDs(ctx, false, 1, tuid.MinID)
	if expect.NoError(err) {
		expect.Equal(1, len(ids))
		expect.Equal(id1, ids[0])
	}
	// Read only the last entity ID
	ids, err = service.ReadEntityIDs(ctx, true, 1, tuid.MaxID)
	if expect.NoError(err) {
		expect.Equal(1, len(ids))
		expect.Equal(e1.UserID, ids[0])
	}
}

func TestReadEventsByEntityID(t *testing.T) {
	expect := assert.New(t)
	// Read all events
	all := []Event{e1, e2, e3}
	events, err := service.ReadEventsByEntityID(ctx, id1, false, 10, tuid.MinID)
	if expect.NoError(err) {
		expect.Equal(len(all), len(events))
		expect.Equal(all, events)
	}
	// Read only the first event
	events, err = service.ReadEventsByEntityID(ctx, id1, false, 1, tuid.MinID)
	if expect.NoError(err) {
		expect.Equal(1, len(events))
		expect.Equal(e1, events[0])
	}
	// Read only the last event
	events, err = service.ReadEventsByEntityID(ctx, id1, true, 1, tuid.MaxID)
	if expect.NoError(err) {
		expect.Equal(1, len(events))
		expect.Equal(e3, events[0])
	}
	// Read zero events for an invalid entity ID
	events, err = service.ReadEventsByEntityID(ctx, "0", false, 1, tuid.MinID)
	if expect.NoError(err) {
		expect.Equal(0, len(events))
	}
}

func TestReadAllEntityTypes(t *testing.T) {
	expect := assert.New(t)
	// Read all entity types
	all := []string{"Thing"}
	types, err := service.ReadAllEntityTypes(ctx)
	if expect.NoError(err) {
		expect.Equal(len(all), len(types))
		expect.Equal(all, types)
	}
}

func TestReadEventsByEntityType(t *testing.T) {
	expect := assert.New(t)
	// Read all events
	events, err := service.ReadEventsByEntityType(ctx, "Thing", false, 10, tuid.MinID)
	if expect.NoError(err) {
		expect.GreaterOrEqual(len(events), len(knownEvents))
		expect.Subset(events, knownEvents)
	}
	// Read only the first event
	events, err = service.ReadEventsByEntityType(ctx, "Thing", false, 1, tuid.MinID)
	if expect.NoError(err) {
		expect.Equal(1, len(events))
		expect.Equal(e1, events[0])
	}
	// Read only the last event
	events, err = service.ReadEventsByEntityType(ctx, "Thing", true, 1, tuid.MaxID)
	if expect.NoError(err) {
		expect.Equal(1, len(events))
		expect.GreaterOrEqual(events[0].ID, e5.ID)
	}
	// Read zero events for an invalid entity type
	events, err = service.ReadEventsByEntityType(ctx, "", false, 1, tuid.MinID)
	if expect.NoError(err) {
		expect.Equal(0, len(events))
	}
}

func TestReadLogLevels(t *testing.T) {
	expect := assert.New(t)
	// Read log levels
	all := []string{"ERROR", "INFO", "WARN"}
	levels, err := service.ReadLogLevels(ctx, false, 10, "-")
	if expect.NoError(err) {
		expect.GreaterOrEqual(len(levels), len(all))
		expect.Subset(levels, all)
	}
	// Read only the first log level, avoiding the transient CRUD event (DEBUG)
	levels, err = service.ReadLogLevels(ctx, false, 1, "DEBUG")
	if expect.NoError(err) {
		expect.Equal(1, len(levels))
		expect.Equal(all[0], levels[0])
	}
	// Read only the last log level
	levels, err = service.ReadLogLevels(ctx, true, 1, "|")
	if expect.NoError(err) {
		expect.Equal(1, len(levels))
		expect.Equal(all[len(all)-1], levels[0])
	}
}

func TestReadAllLogLevels(t *testing.T) {
	expect := assert.New(t)
	// Read all log levels, including the potential transient CRUD event (DEBUG)
	all := []string{"ERROR", "INFO", "WARN"}
	levels, err := service.ReadAllLogLevels(ctx)
	if expect.NoError(err) {
		expect.GreaterOrEqual(len(levels), len(all))
		expect.Subset(levels, all)
	}
}

func TestReadEventsByLogLevel(t *testing.T) {
	expect := assert.New(t)
	// Read all events
	all := []Event{e1, e2, e3}
	events, err := service.ReadEventsByLogLevel(ctx, "INFO", false, 10, tuid.MinID)
	if expect.NoError(err) {
		expect.Equal(len(all), len(events))
		expect.Equal(all, events)
	}
	// Read only the first event
	events, err = service.ReadEventsByLogLevel(ctx, "INFO", false, 1, tuid.MinID)
	if expect.NoError(err) {
		expect.Equal(1, len(events))
		expect.Equal(all[0], events[0])
	}
	// Read only the last event
	events, err = service.ReadEventsByLogLevel(ctx, "INFO", true, 1, tuid.MaxID)
	if expect.NoError(err) {
		expect.Equal(1, len(events))
		expect.Equal(all[len(all)-1], events[0])
	}
	// Read zero events for an invalid log level
	events, err = service.ReadEventsByLogLevel(ctx, "ZZZZZ", false, 1, tuid.MinID)
	if expect.NoError(err) {
		expect.Equal(0, len(events))
	}
}
