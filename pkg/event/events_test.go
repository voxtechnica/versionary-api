package event

import (
	"context"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/voxtechnica/tuid-go"
	v "github.com/voxtechnica/versionary"
)

var (
	// Event EventService
	ctx      = context.Background()
	table    = NewEventTable(nil, "test")
	memTable = NewEventMemTable(table)
	service  = EventService{EntityType: "Event", Table: memTable}

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
)

func TestMain(m *testing.M) {
	// Check the table/row definitions
	if !memTable.IsValid() {
		log.Fatal("invalid table configuration")
	}
	// Write known events
	for _, e := range []Event{e1, e2, e3, e4, e5} {
		if _, err := service.Write(ctx, e); err != nil {
			log.Fatal(err)
		}
	}
	// Run the tests
	m.Run()
}

func TestCreateReadDelete(t *testing.T) {
	// Create an event
	e, err := service.Create(ctx, Event{
		EntityID:   id3,
		EntityType: "Thing",
		LogLevel:   DEBUG,
		Message:    "CRUD Test",
	})
	if err != nil {
		t.Fatal(err)
	}
	// Read the event
	eCheck, err := service.Read(ctx, e.ID)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(e, eCheck); diff != "" {
		t.Error(diff)
	}
	// Read the event as JSON
	eCheckJSON, err := service.ReadAsJSON(ctx, e.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(eCheckJSON), e.ID) {
		t.Error("event JSON does not contain ID")
	}
	// Delete the event
	if _, err = service.Delete(ctx, e.ID); err != nil {
		t.Fatal(err)
	}
	// Read the event
	if _, err = service.Read(ctx, e.ID); err != v.ErrNotFound {
		t.Fatal("expected ErrNotFound")
	}
}

func TestReadEventIDs(t *testing.T) {
	ids, err := service.ReadEventIDs(ctx, false, 10, tuid.MinID)
	if err != nil {
		t.Fatal(err)
	}
	// Check the IDs (note that the CRUD test event may also be present)
	if len(ids) < 5 {
		t.Errorf("expected 5 events, got %d", len(ids))
	}
	expected := []string{e1.ID, e2.ID, e3.ID, e4.ID, e5.ID}
	for _, id := range expected {
		if !v.Contains(ids, id) {
			t.Errorf("expected ID %s to be in %v", id, ids)
		}
	}
}

func TestReadEvents(t *testing.T) {
	events := service.ReadEvents(ctx, false, 10, tuid.MinID)
	if len(events) < 5 {
		t.Errorf("expected 5 events, got %d", len(events))
	}
	ids := v.Map(events, func(e Event) string { return e.ID })
	expected := []string{e1.ID, e2.ID, e3.ID, e4.ID, e5.ID}
	for _, id := range expected {
		if !v.Contains(ids, id) {
			t.Errorf("expected event %s to be in %v", id, events)
		}
	}
}

func TestReadRecentEvents(t *testing.T) {
	events, err := service.ReadRecentEvents(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) < 5 {
		t.Fatalf("expected 5 events, got %d", len(events))
	}
	oldest := events[len(events)-1].ID
	if e1.ID != oldest {
		t.Errorf("expected oldest event to be %s, got %s", e1.ID, oldest)
	}
}

func TestReadDates(t *testing.T) {
	dates, err := service.ReadDates(ctx, false, 3, "0")
	if err != nil {
		t.Fatal(err)
	}
	if len(dates) != 3 {
		t.Fatalf("expected 3 dates, got %d", len(dates))
	}
	if dates[0] > dates[1] {
		t.Error("expected dates to be sorted chronologically")
	}
	if dates[1] > dates[2] {
		t.Error("expected dates to be sorted chronologically")
	}
	dates, err = service.ReadDates(ctx, true, 3, "9999-99-99")
	if err != nil {
		t.Fatal(err)
	}
	if len(dates) != 3 {
		t.Fatalf("expected 3 dates, got %d", len(dates))
	}
	if dates[0] < dates[1] {
		t.Error("expected dates to be sorted reverse-chronologically")
	}
	if dates[1] < dates[2] {
		t.Error("expected dates to be sorted reverse-chronologically")
	}
}

func TestReadAllDates(t *testing.T) {
	dates, err := service.ReadAllDates(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(dates) < 3 {
		t.Fatalf("expected 3 dates, got %d", len(dates))
	}
	if dates[0] > dates[1] {
		t.Error("expected dates to be sorted chronologically")
	}
	if dates[1] > dates[2] {
		t.Error("expected dates to be sorted chronologically")
	}
}

func TestReadEventsByDate(t *testing.T) {
	// Read all events
	d := t1.Format("2006-01-02")
	events, err := service.ReadEventsByDate(ctx, d, false, 10, tuid.MinID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
	if events[0].ID != e1.ID {
		t.Errorf("expected event %s to be first", e1.ID)
	}
	if events[1].ID != e2.ID {
		t.Errorf("expected event %s to be second", e2.ID)
	}
	if events[2].ID != e3.ID {
		t.Errorf("expected event %s to be third", e3.ID)
	}
	// Read only the first event
	events, err = service.ReadEventsByDate(ctx, d, false, 1, tuid.MinID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].ID != e1.ID {
		t.Errorf("expected event %s to be first", e1.ID)
	}
	// Read only the last event
	events, err = service.ReadEventsByDate(ctx, d, true, 1, tuid.MaxID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].ID != e3.ID {
		t.Errorf("expected event %s to be first", e3.ID)
	}
	// Read zero events for an invalid date
	events, err = service.ReadEventsByDate(ctx, "0", false, 1, tuid.MinID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
	// Read zero events for a missing date
	events, err = service.ReadEventsByDate(ctx, "", false, 1, tuid.MinID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestReadEntityIDs(t *testing.T) {
	// Read all entity IDs
	expected := []string{id1, id2, id3, e1.UserID}
	ids, err := service.ReadEntityIDs(ctx, false, 10, tuid.MinID)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != len(expected) {
		t.Fatalf("expected %d IDs, got %d", len(expected), len(ids))
	}
	for i, id := range ids {
		if id != expected[i] {
			t.Errorf("expected ID %s to be %s", id, expected[i])
		}
	}
	// Read only the first entity ID
	ids, err = service.ReadEntityIDs(ctx, false, 1, tuid.MinID)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 {
		t.Fatalf("expected 1 ID, got %d", len(ids))
	}
	if ids[0] != id1 {
		t.Errorf("expected ID %s to be %s", ids[0], id1)
	}
	// Read only the last entity ID
	ids, err = service.ReadEntityIDs(ctx, true, 1, tuid.MaxID)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 {
		t.Fatalf("expected 1 ID, got %d", len(ids))
	}
	if ids[0] != e1.UserID {
		t.Errorf("expected ID %s to be %s", ids[0], e1.UserID)
	}
}

func TestReadEventsByEntityID(t *testing.T) {
	// Read all events
	events, err := service.ReadEventsByEntityID(ctx, id1, false, 10, tuid.MinID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
	if events[0].ID != e1.ID {
		t.Errorf("expected event %s to be first", e1.ID)
	}
	if events[1].ID != e2.ID {
		t.Errorf("expected event %s to be second", e2.ID)
	}
	if events[2].ID != e3.ID {
		t.Errorf("expected event %s to be third", e3.ID)
	}
	// Read only the first event
	events, err = service.ReadEventsByEntityID(ctx, id1, false, 1, tuid.MinID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].ID != e1.ID {
		t.Errorf("expected event %s to be first", e1.ID)
	}
	// Read only the last event
	events, err = service.ReadEventsByEntityID(ctx, id1, true, 1, tuid.MaxID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].ID != e3.ID {
		t.Errorf("expected event %s to be first", e3.ID)
	}
	// Read zero events for an invalid entity ID
	events, err = service.ReadEventsByEntityID(ctx, "0", false, 1, tuid.MinID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatalf("expected 0 events, got %d", len(events))
	}
}

func TestReadLogLevels(t *testing.T) {
	// Read log levels
	expected := []string{"ERROR", "INFO", "WARN"}
	levels, err := service.ReadLogLevels(ctx, false, 10, "DEBUG")
	if err != nil {
		t.Fatal(err)
	}
	if len(levels) != len(expected) {
		t.Fatalf("expected %d levels, got %d", len(expected), len(levels))
	}
	for i, level := range levels {
		if level != expected[i] {
			t.Errorf("expected level %s to be %s", level, expected[i])
		}
	}
	// Read only the first log level
	levels, err = service.ReadLogLevels(ctx, false, 1, "DEBUG")
	if err != nil {
		t.Fatal(err)
	}
	if len(levels) != 1 {
		t.Fatalf("expected 1 level, got %d", len(levels))
	}
	if levels[0] != "ERROR" {
		t.Errorf("expected level %s to be %s", levels[0], "ERROR")
	}
	// Read only the last log level
	levels, err = service.ReadLogLevels(ctx, true, 1, "ZZZZZZZZZZ")
	if err != nil {
		t.Fatal(err)
	}
	if len(levels) != 1 {
		t.Fatalf("expected 1 level, got %d", len(levels))
	}
	if levels[0] != "WARN" {
		t.Errorf("expected level %s to be %s", levels[0], "WARN")
	}
}

func TestReadAllLogLevels(t *testing.T) {
	// Read all log levels
	levels, err := service.ReadAllLogLevels(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(levels) < 3 {
		t.Fatalf("expected 3 levels, got %d", len(levels))
	}
	if levels[0] > levels[1] {
		t.Errorf("expected log level to be sorted alphabetically: %s should be less than %s", levels[0], levels[1])
	}
	if levels[1] > levels[2] {
		t.Errorf("expected log level to be sorted alphabetically: %s should be less than %s", levels[1], levels[2])
	}
}

func TestReadEventsByLogLevel(t *testing.T) {
	// Read all events
	events, err := service.ReadEventsByLogLevel(ctx, "INFO", false, 10, tuid.MinID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
	if events[0].ID != e1.ID {
		t.Errorf("expected event %s to be first", e1.ID)
	}
	if events[1].ID != e2.ID {
		t.Errorf("expected event %s to be second", e2.ID)
	}
	if events[2].ID != e3.ID {
		t.Errorf("expected event %s to be third", e3.ID)
	}
	// Read only the first event
	events, err = service.ReadEventsByLogLevel(ctx, "INFO", false, 1, tuid.MinID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].ID != e1.ID {
		t.Errorf("expected event %s to be first", e1.ID)
	}
	// Read only the last event
	events, err = service.ReadEventsByLogLevel(ctx, "INFO", true, 1, tuid.MaxID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].ID != e3.ID {
		t.Errorf("expected event %s to be first", e3.ID)
	}
	// Read zero events for an invalid log level
	events, err = service.ReadEventsByLogLevel(ctx, "ZZZZZ", false, 1, tuid.MinID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatalf("expected 0 events, got %d", len(events))
	}
}
