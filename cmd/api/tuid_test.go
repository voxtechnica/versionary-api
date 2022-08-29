package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/voxtechnica/tuid-go"
)

func TestCreateTUID(t *testing.T) {
	expect := assert.New(t)
	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/v1/tuids", nil)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusCreated, w.Code, "HTTP Status Code")
		var info tuid.TUIDInfo
		if expect.NoError(json.NewDecoder(w.Body).Decode(&info), "Decode JSON TUID") {
			expect.True(tuid.IsValid(info.ID), "Valid TUID")
		}
	}
}

func TestReadTUIDs(t *testing.T) {
	expect := assert.New(t)
	// Test with a valid limit
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/tuids?limit=10", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var tuids []tuid.TUIDInfo
		if expect.NoError(json.NewDecoder(w.Body).Decode(&tuids), "Decode JSON TUIDs") {
			expect.NotEmpty(tuids, "TUIDs")
			expect.Equal(10, len(tuids), "TUIDs Length")
			for _, t := range tuids {
				expect.True(tuid.IsValid(t.ID), "Valid TUID")
			}
		}
	}
	// Test with an invalid limit
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/tuids?limit=0", nil)
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON APIEvent") {
			expect.Equal("ERROR", e.LogLevel, "Event Log Level")
			expect.Equal(http.StatusBadRequest, e.Code, "Event Code")
			expect.Contains(e.Message, "invalid limit", "Event Message")
		}
	}
}

func TestReadTUID(t *testing.T) {
	expect := assert.New(t)
	// Test with a valid TUID
	expected, _ := tuid.TUID("9GEG9f25zjGI3ath").Info()
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/tuids/"+expected.ID.String(), nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var info tuid.TUIDInfo
		if expect.NoError(json.NewDecoder(w.Body).Decode(&info), "Decode JSON TUID") {
			expect.True(tuid.IsValid(info.ID), "Valid TUID")
			expect.Equal(expected, info, "TUID")
		}
	}
	// Test invalid TUID (invalid TUID)
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/tuids/bad_tuid", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON APIEvent") {
			expect.Equal("ERROR", e.LogLevel, "Event Log Level")
			expect.Equal(http.StatusBadRequest, e.Code, "Event Code")
			expect.Contains(e.Message, "invalid digit", "Event Message")
		}
	}
	// Test invalid TUID (bad timestamp)
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/tuids/0123456789", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON APIEvent") {
			expect.Equal("ERROR", e.LogLevel, "Event Log Level")
			expect.Equal(http.StatusBadRequest, e.Code, "Event Code")
			expect.Contains(e.Message, "invalid TUID", "Event Message")
		}
	}
}
