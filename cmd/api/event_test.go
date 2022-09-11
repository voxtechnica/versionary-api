package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/voxtechnica/tuid-go"

	"versionary-api/pkg/event"
)

func TestCreateEvent(t *testing.T) {
	expect := assert.New(t)
	// Create an event: happy path
	j, err := json.Marshal(event.Event{
		UserID:     regularUser.ID,
		EntityID:   userOrg.ID,
		EntityType: "Organization",
		LogLevel:   event.ERROR,
		Message:    "read organization " + userOrg.ID + ": " + userOrg.Name,
		URI:        "/v1/organizations/" + userOrg.ID,
		Err:        errors.New("read organization " + userOrg.ID + ": " + userOrg.Name),
	})
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/v1/events", bytes.NewBuffer(j))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusCreated, w.Code, "HTTP Status Code")
		var e event.Event
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.True(tuid.IsValid(tuid.TUID(e.ID)), "Valid Event ID")
		}
	}
	// Create an event: invalid JSON body
	w = httptest.NewRecorder()
	req, err = http.NewRequest("POST", "/v1/events", strings.NewReader(""))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusBadRequest, e.Code, "Event Code")
			expect.Contains(e.Message, "invalid JSON", "Event Message")
		}
	}
	// Create an event: validation errors
	w = httptest.NewRecorder()
	req, err = http.NewRequest("POST", "/v1/events", strings.NewReader("{}"))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusUnprocessableEntity, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusUnprocessableEntity, e.Code, "Event Code")
			expect.Contains(e.Message, "unprocessable entity", "Event Message")
		}
	}
	// Create an event: missing authorization token
	w = httptest.NewRecorder()
	req, err = http.NewRequest("POST", "/v1/events", bytes.NewBuffer(j))
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusUnauthorized, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusUnauthorized, e.Code, "Event Code")
			expect.Contains(e.Message, "unauthenticated", "Event Message")
		}
	}
	// Create an event: missing admin role
	w = httptest.NewRecorder()
	req, err = http.NewRequest("POST", "/v1/events", bytes.NewBuffer(j))
	req.Header.Set("Authorization", "Bearer "+regularToken)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusForbidden, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusForbidden, e.Code, "Event Code")
			expect.Contains(e.Message, "unauthorized", "Event Message")
		}
	}
}

func TestReadEvents(t *testing.T) {
	// Initialize the database
	expect := assert.New(t)
	ctx := context.Background()
	e1, problems, err := api.EventService.Create(ctx, event.Event{
		UserID:     regularUser.ID,
		EntityID:   userOrg.ID,
		EntityType: "Organization",
		LogLevel:   event.TRACE,
		Message:    "read organization " + userOrg.ID + ": " + userOrg.Name,
		URI:        "/v1/organizations/" + userOrg.ID,
	})
	expect.Empty(problems, "No Problems")
	expect.NoError(err, "Create Event")
	expect.True(tuid.IsValid(tuid.TUID(e1.ID)), "Valid Event ID")
	// Read events: no authorization token
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/events", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusUnauthorized, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusUnauthorized, e.Code, "Event Code")
			expect.Contains(e.Message, "unauthenticated", "Event Message")
		}
	}
	// Read events: invalid authorization token
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/events", nil)
	req.Header.Set("Authorization", "Bearer "+tuid.NewID().String())
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusUnauthorized, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusUnauthorized, e.Code, "Event Code")
			expect.Contains(e.Message, "not found", "Event Message")
		}
	}
	// Read events: missing admin role
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/events", nil)
	req.Header.Set("Authorization", "Bearer "+regularToken)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusForbidden, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusForbidden, e.Code, "Event Code")
			expect.Contains(e.Message, "unauthorized", "Event Message")
		}
	}
	// Read events: invalid pagination param (reverse)
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/events?reverse=forward", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusBadRequest, e.Code, "Event Code")
			expect.Contains(e.Message, "invalid", "Event Message")
		}
	}
	// Read events: invalid pagination param (limit)
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/events?limit=true", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusBadRequest, e.Code, "Event Code")
			expect.Contains(e.Message, "invalid", "Event Message")
		}
	}
	// Read events: invalid entity ID
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/events?entity=bad_id", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusBadRequest, e.Code, "Event Code")
			expect.Contains(e.Message, "invalid TUID", "Event Message")
		}
	}
	// Read events: invalid LogLevel
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/events?log_level=invalid", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusBadRequest, e.Code, "Event Code")
			expect.Contains(e.Message, "invalid log level", "Event Message")
		}
	}
	// Read events: invalid date
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/events?date=invalid", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusBadRequest, e.Code, "Event Code")
			expect.Contains(e.Message, "invalid date", "Event Message")
		}
	}
	// Read recent events (happy path)
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/events?limit=1", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var es []event.Event
		if expect.NoError(json.NewDecoder(w.Body).Decode(&es), "Decode JSON Events") {
			expect.NotEmpty(es, "Events")
		}
	}
}

func TestReadEvent(t *testing.T) {
	// Initialize the database
	expect := assert.New(t)
	ctx := context.Background()
	e1, problems, err := api.EventService.Create(ctx, event.Event{
		UserID:     regularUser.ID,
		EntityID:   userOrg.ID,
		EntityType: "Organization",
		LogLevel:   event.TRACE,
		Message:    "read organization " + userOrg.ID + ": " + userOrg.Name,
		URI:        "/v1/organizations/" + userOrg.ID,
	})
	expect.Empty(problems, "No Problems")
	expect.NoError(err, "Create Event")
	expect.True(tuid.IsValid(tuid.TUID(e1.ID)), "Valid Event ID")
	// Read event: invalid ID
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/events/bad_id", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusBadRequest, e.Code, "Event Code")
			expect.Contains(e.Message, "invalid path parameter", "Event Message")
		}
	}
	// Read event: not found
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/events/"+tuid.NewID().String(), nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNotFound, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusNotFound, e.Code, "Event Code")
			expect.Contains(e.Message, "not found", "Event Message")
		}
	}
	// Read event: happy path
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/events/"+e1.ID, nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var e event.Event
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal(e1, e, "Event")
		}
	}
}

func TestExistsEvent(t *testing.T) {
	// Initialize the database
	expect := assert.New(t)
	ctx := context.Background()
	e1, problems, err := api.EventService.Create(ctx, event.Event{
		UserID:     regularUser.ID,
		EntityID:   userOrg.ID,
		EntityType: "Organization",
		LogLevel:   event.TRACE,
		Message:    "read organization " + userOrg.ID + ": " + userOrg.Name,
		URI:        "/v1/organizations/" + userOrg.ID,
	})
	expect.Empty(problems, "No Problems")
	expect.NoError(err, "Create Event")
	expect.True(tuid.IsValid(tuid.TUID(e1.ID)), "Valid Event ID")
	// Exists event: invalid ID
	w := httptest.NewRecorder()
	req, err := http.NewRequest("HEAD", "/v1/events/bad_id", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
	}
	// Exists event: not found
	w = httptest.NewRecorder()
	req, err = http.NewRequest("HEAD", "/v1/events/"+tuid.NewID().String(), nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNotFound, w.Code, "HTTP Status Code")
	}
	// Exists event: happy path
	w = httptest.NewRecorder()
	req, err = http.NewRequest("HEAD", "/v1/events/"+e1.ID, nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNoContent, w.Code, "HTTP Status Code")
	}
}

func TestDeleteEvent(t *testing.T) {
	// Initialize the database
	expect := assert.New(t)
	ctx := context.Background()
	e1, problems, err := api.EventService.Create(ctx, event.Event{
		UserID:     regularUser.ID,
		EntityID:   userOrg.ID,
		EntityType: "Organization",
		LogLevel:   event.TRACE,
		Message:    "read organization " + userOrg.ID + ": " + userOrg.Name,
		URI:        "/v1/organizations/" + userOrg.ID,
	})
	expect.Empty(problems, "No Problems")
	expect.NoError(err, "Create Event")
	expect.True(tuid.IsValid(tuid.TUID(e1.ID)), "Valid Event ID")
	// Delete event: missing authorization token
	w := httptest.NewRecorder()
	req, err := http.NewRequest("DELETE", "/v1/events/"+e1.ID, nil)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusUnauthorized, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusUnauthorized, e.Code, "Event Code")
			expect.Contains(e.Message, "unauthenticated", "Event Message")
		}
	}
	// Delete event: missing admin role
	w = httptest.NewRecorder()
	req, err = http.NewRequest("DELETE", "/v1/events/"+e1.ID, nil)
	req.Header.Set("Authorization", "Bearer "+regularToken)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusForbidden, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusForbidden, e.Code, "Event Code")
			expect.Contains(e.Message, "unauthorized", "Event Message")
		}
	}
	// Delete event: invalid ID
	w = httptest.NewRecorder()
	req, err = http.NewRequest("DELETE", "/v1/events/bad_id", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusBadRequest, e.Code, "Event Code")
			expect.Contains(e.Message, "invalid path parameter", "Event Message")
		}
	}
	// Delete event: happy path
	w = httptest.NewRecorder()
	req, err = http.NewRequest("DELETE", "/v1/events/"+e1.ID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var e event.Event
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal(e1, e, "Event")
		}
	}
	// Delete event: not found
	w = httptest.NewRecorder()
	req, err = http.NewRequest("DELETE", "/v1/events/"+e1.ID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNotFound, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusNotFound, e.Code, "Event Code")
			expect.Contains(e.Message, "not found", "Event Message")
		}
	}
}

func TestEventEntityIDs(t *testing.T) {
	// Initialize the database
	expect := assert.New(t)
	ctx := context.Background()
	e1, problems, err := api.EventService.Create(ctx, event.Event{
		UserID:     regularUser.ID,
		EntityID:   userOrg.ID,
		EntityType: "Organization",
		LogLevel:   event.TRACE,
		Message:    "read organization " + userOrg.ID + ": " + userOrg.Name,
		URI:        "/v1/organizations/" + userOrg.ID,
	})
	expect.Empty(problems, "No Problems")
	expect.NoError(err, "Create Event")
	expect.True(tuid.IsValid(tuid.TUID(e1.ID)), "Valid Event ID")
	// Get event entity IDs: missing authorization token
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/event_entity_ids", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusUnauthorized, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusUnauthorized, e.Code, "Event Code")
			expect.Contains(e.Message, "unauthenticated", "Event Message")
		}
	}
	// Get event entity IDs: missing admin role
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/event_entity_ids", nil)
	req.Header.Set("Authorization", "Bearer "+regularToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusForbidden, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusForbidden, e.Code, "Event Code")
			expect.Contains(e.Message, "unauthorized", "Event Message")
		}
	}
	// Get event entity IDs: invalid pagination param (limit)
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/event_entity_ids?limit=0", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusBadRequest, e.Code, "Event Code")
			expect.Contains(e.Message, "invalid", "Event Message")
		}
	}
	// Get event entity IDs: happy path
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/event_entity_ids?reverse=true&limit=1", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var ids []string
		if expect.NoError(json.NewDecoder(w.Body).Decode(&ids), "Decode JSON Event") {
			expect.Equal(1, len(ids), "Event Count")
		}
	}
}

func TestEventEntityTypes(t *testing.T) {
	// Initialize the database
	expect := assert.New(t)
	ctx := context.Background()
	e1, problems, err := api.EventService.Create(ctx, event.Event{
		UserID:     regularUser.ID,
		EntityID:   userOrg.ID,
		EntityType: "Organization",
		LogLevel:   event.TRACE,
		Message:    "read organization " + userOrg.ID + ": " + userOrg.Name,
		URI:        "/v1/organizations/" + userOrg.ID,
	})
	expect.Empty(problems, "No Problems")
	expect.NoError(err, "Create Event")
	expect.True(tuid.IsValid(tuid.TUID(e1.ID)), "Valid Event ID")
	// Get event entity types: missing authorization token
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/event_entity_types", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusUnauthorized, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusUnauthorized, e.Code, "Event Code")
			expect.Contains(e.Message, "unauthenticated", "Event Message")
		}
	}
	// Get event entity types: missing admin role
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/event_entity_types", nil)
	req.Header.Set("Authorization", "Bearer "+regularToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusForbidden, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusForbidden, e.Code, "Event Code")
			expect.Contains(e.Message, "unauthorized", "Event Message")
		}
	}
	// Get event entity types: happy path
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/event_entity_types", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var types []string
		if expect.NoError(json.NewDecoder(w.Body).Decode(&types), "Decode JSON Event") {
			expect.Contains(types, "Organization", "Event Entity Type")
		}
	}
}

func TestEventLogLevels(t *testing.T) {
	// Initialize the database
	expect := assert.New(t)
	ctx := context.Background()
	e1, problems, err := api.EventService.Create(ctx, event.Event{
		UserID:     regularUser.ID,
		EntityID:   userOrg.ID,
		EntityType: "Organization",
		LogLevel:   event.TRACE,
		Message:    "read organization " + userOrg.ID + ": " + userOrg.Name,
		URI:        "/v1/organizations/" + userOrg.ID,
	})
	expect.Empty(problems, "No Problems")
	expect.NoError(err, "Create Event")
	expect.True(tuid.IsValid(tuid.TUID(e1.ID)), "Valid Event ID")
	// Get event log levels: missing authorization token
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/event_log_levels", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusUnauthorized, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusUnauthorized, e.Code, "Event Code")
			expect.Contains(e.Message, "unauthenticated", "Event Message")
		}
	}
	// Get event log levels: missing admin role
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/event_log_levels", nil)
	req.Header.Set("Authorization", "Bearer "+regularToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusForbidden, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusForbidden, e.Code, "Event Code")
			expect.Contains(e.Message, "unauthorized", "Event Message")
		}
	}
	// Get event log levels: happy path
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/event_log_levels", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var ll []string
		if expect.NoError(json.NewDecoder(w.Body).Decode(&ll), "Decode JSON Event") {
			expect.Contains(ll, "TRACE", "Event LogLevel")
		}
	}
}

func TestEventDates(t *testing.T) {
	// Initialize the database
	expect := assert.New(t)
	ctx := context.Background()
	e1, problems, err := api.EventService.Create(ctx, event.Event{
		UserID:     regularUser.ID,
		EntityID:   userOrg.ID,
		EntityType: "Organization",
		LogLevel:   event.TRACE,
		Message:    "read organization " + userOrg.ID + ": " + userOrg.Name,
		URI:        "/v1/organizations/" + userOrg.ID,
	})
	expect.Empty(problems, "No Problems")
	expect.NoError(err, "Create Event")
	expect.True(tuid.IsValid(tuid.TUID(e1.ID)), "Valid Event ID")
	// Get event dates: missing authorization token
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/event_dates", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusUnauthorized, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusUnauthorized, e.Code, "Event Code")
			expect.Contains(e.Message, "unauthenticated", "Event Message")
		}
	}
	// Get event dates: missing admin role
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/event_dates", nil)
	req.Header.Set("Authorization", "Bearer "+regularToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusForbidden, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusForbidden, e.Code, "Event Code")
			expect.Contains(e.Message, "unauthorized", "Event Message")
		}
	}
	// Get event dates: invalid pagination param (limit)
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/event_dates?limit=-1", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusBadRequest, e.Code, "Event Code")
			expect.Contains(e.Message, "invalid", "Event Message")
		}
	}
	// Get event dates: happy path
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/event_dates?reverse=true&limit=10", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var dates []string
		if expect.NoError(json.NewDecoder(w.Body).Decode(&dates), "Decode JSON Event") {
			expect.GreaterOrEqual(len(dates), 1, "Event Date Count")
			expect.Contains(dates, e1.CreatedOn(), "Event Date")
		}
	}
}
