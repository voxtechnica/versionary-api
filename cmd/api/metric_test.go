package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"versionary-api/pkg/metric"
)

func TestCreateAndDeleteMetric(t *testing.T) {
	expect := assert.New(t)
	// Create a new Metric
	var m metric.Metric
	w := httptest.NewRecorder()
	body := `{"title":"Test Metric","value":1.0}`
	req, err := http.NewRequest("POST", "/v1/metrics", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		fmt.Println("THIS IS M: ", w.Body)
		expect.Equal(http.StatusCreated, w.Code, "HTTP Status Code")
		if expect.NoError(json.NewDecoder(w.Body).Decode(&m), "Decode JSON Metric") {
			expect.NotEmpty(m.ID)
			expect.NotEmpty(m.CreatedAt)
			expect.Equal("Test Metric", m.Title)
			expect.Equal(1.0, m.Value)
		}
	}

	// Delete the Metric
	w = httptest.NewRecorder()
	req, err = http.NewRequest("DELETE", "/v1/metrics/"+m.ID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var m2 metric.Metric
		if expect.NoError(json.NewDecoder(w.Body).Decode(&m2), "Decode JSON Metric") {
			expect.Equal(m.ID, m2.ID, "Metric ID")
			expect.Equal(m.Title, m2.Title, "Metric Title")
			expect.Equal(m.Value, m2.Value, "Metric Value")
		}
	}
}

func TestReadMetric(t *testing.T) {
	expect := assert.New(t)
	// Read known Metric
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/metrics/"+metricOne.ID, nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code)
		var m metric.Metric
		if expect.NoError(json.NewDecoder(w.Body).Decode(&m), "Decode JSON Metric") {
			expect.Equal(metricOne.ID, m.ID, "Metric ID")
			expect.Equal("API Test Metric One", m.Title, "Metric Title")
			expect.Equal(100.0, m.Value, "Metric Value")
		}
	}
}
