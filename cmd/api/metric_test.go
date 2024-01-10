package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/voxtechnica/tuid-go"
	"github.com/voxtechnica/versionary"

	"versionary-api/pkg/metric"
)

func TestMetricHappyPath(t *testing.T) {
	expect := assert.New(t)
	// Create a new Metric
	m := metric.Metric{
		Title:      "Test Metric",
		EntityID:   tuid.NewID().String(),
		EntityType: "Content",
		Tags:       []string{"test"},
		Value:      1.2,
		Units:      "seconds",
	}
	j, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/v1/metrics", bytes.NewBuffer(j))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusCreated, w.Code, "HTTP Status Code")
		if expect.NoError(json.NewDecoder(w.Body).Decode(&m), "Decode JSON Metric") {
			expect.NotEmpty(m.ID)
			expect.NotEmpty(m.CreatedAt)
			expect.Equal("Test Metric", m.Title)
			expect.Equal("Content", m.EntityType)
			expect.Equal(1.2, m.Value)
			expect.Equal("seconds", m.Units)
		}
	}

	// Read the Metric
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/metrics/"+m.ID, nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var x metric.Metric
		if expect.NoError(json.NewDecoder(w.Body).Decode(&x), "Decode JSON Metric") {
			expect.Equal(m, x, "Read Metric")
		}
	}

	// Metric Exists
	w = httptest.NewRecorder()
	req, err = http.NewRequest("HEAD", "/v1/metrics/"+m.ID, nil)
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNoContent, w.Code, "HTTP Status Code")
	}

	// Read all Metrics
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/metrics?limit=100&reverse=true", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var ms []metric.Metric
		if expect.NoError(json.NewDecoder(w.Body).Decode(&ms)) && expect.NotEmpty(ms) {
			expect.Contains(ms, m, "Read Metrics")
		}
	}

	// Read Metrics by Entity ID
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/metrics?entity="+m.EntityID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var ms []metric.Metric
		if expect.NoError(json.NewDecoder(w.Body).Decode(&ms)) && expect.NotEmpty(ms) {
			expect.Contains(ms, m, "Read Metrics")
		}
	}

	// Read Metrics by Entity Type
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/metrics?type=Content", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var ms []metric.Metric
		if expect.NoError(json.NewDecoder(w.Body).Decode(&ms)) && expect.NotEmpty(ms) {
			expect.Contains(ms, m, "Read Metrics")
		}
	}

	// Read Metrics by Tag
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/metrics?tag=test", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var ms []metric.Metric
		if expect.NoError(json.NewDecoder(w.Body).Decode(&ms)) && expect.NotEmpty(ms) {
			expect.Contains(ms, m, "Read Metrics")
		}
	}

	// Read Metric Labels
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/metric_labels", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		label := versionary.TextValue{
			Key:   m.ID,
			Value: m.String(),
		}
		var labels []versionary.TextValue
		if expect.NoError(json.NewDecoder(w.Body).Decode(&labels)) && expect.NotEmpty(labels) {
			expect.Contains(labels, label, "Metric Labels")
		}
	}

	// Read Entity IDs
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/metric_entity_ids", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var ids []string
		if expect.NoError(json.NewDecoder(w.Body).Decode(&ids)) && expect.NotEmpty(ids) {
			expect.Contains(ids, m.EntityID, "Metric Entity IDs")
		}
	}

	// Read Entity Types
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/metric_entity_types", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var types []string
		if expect.NoError(json.NewDecoder(w.Body).Decode(&types)) && expect.NotEmpty(types) {
			expect.Contains(types, m.EntityType, "Metric Entity Types")
		}
	}

	// Read Metric Tags
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/metric_tags", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var tags []string
		if expect.NoError(json.NewDecoder(w.Body).Decode(&tags)) && expect.NotEmpty(tags) {
			expect.Contains(tags, "test", "Metric Entity Tags")
		}
	}

	// Read Metric Stats by Entity ID
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/metric_stats?entity="+m.EntityID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var stats metric.MetricStat
		if expect.NoError(json.NewDecoder(w.Body).Decode(&stats)) {
			expect.Equal(m.EntityID, stats.EntityID, "MetricStat Entity ID")
			expect.Equal(int64(1), stats.Count, "MetricStat Count")
			expect.Equal(m.Value, stats.Sum, "MetricStat Sum")
		}
	}

	// Read Metric Stats by Entity Type
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/metric_stats?type="+m.EntityType, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var stats metric.MetricStat
		if expect.NoError(json.NewDecoder(w.Body).Decode(&stats)) {
			expect.Equal(m.EntityType, stats.EntityType, "MetricStat Entity Type")
			expect.Equal(int64(1), stats.Count, "MetricStat Count")
			expect.Equal(m.Value, stats.Sum, "MetricStat Sum")
		}
	}

	// Read Metric Stats by Tag
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/metric_stats?tag=test", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var stats metric.MetricStat
		if expect.NoError(json.NewDecoder(w.Body).Decode(&stats)) {
			expect.Equal("test", stats.Tag, "MetricStat Tag")
			expect.Equal(int64(1), stats.Count, "MetricStat Count")
			expect.Equal(m.Value, stats.Sum, "MetricStat Sum")
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
		var deleted metric.Metric
		if expect.NoError(json.NewDecoder(w.Body).Decode(&deleted), "Decode JSON Metric") {
			expect.Equal(m, deleted, "Deleted Metric")
		}
	}

	// Verify that the Metric no longer exists
	w = httptest.NewRecorder()
	req, err = http.NewRequest("HEAD", "/v1/metrics/"+m.ID, nil)
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNotFound, w.Code, "HTTP Status Code")
	}
}

func TestCreateMetric(t *testing.T) {
	expect := assert.New(t)
	validBody := `{"title":"Test Metric","entityId":"` + tuid.NewID().String() +
		`","entityType":"Content","tags":["test"],"value":1.2,"units":"seconds"}`
	invalidBody := `{"title":"","value":1.2,"units":""}`

	// Create a new Metric: invalid JSON body
	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/v1/metrics", strings.NewReader(""))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON APIEvent") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusBadRequest, e.Code, "Event Code")
			expect.Contains(e.Message, "invalid JSON", "Event Message")
		}
	}

	// Create a new Metric: missing required fields
	w = httptest.NewRecorder()
	req, err = http.NewRequest("POST", "/v1/metrics", strings.NewReader(invalidBody))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusUnprocessableEntity, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON APIEvent") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusUnprocessableEntity, e.Code, "Event Code")
			expect.Contains(e.Message, "missing", "Event Message")
		}
	}

	// Create a new Metric: missing authorization token
	w = httptest.NewRecorder()
	req, err = http.NewRequest("POST", "/v1/metrics", strings.NewReader(validBody))
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusUnauthorized, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON APIEvent") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusUnauthorized, e.Code, "Event Code")
			expect.Contains(e.Message, "unauthenticated", "Event Message")
		}
	}

	// Create a new Metric: missing admin role
	w = httptest.NewRecorder()
	req, err = http.NewRequest("POST", "/v1/metrics", strings.NewReader(validBody))
	req.Header.Set("Authorization", "Bearer "+regularToken)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusForbidden, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON APIEvent") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusForbidden, e.Code, "Event Code")
			expect.Contains(e.Message, "unauthorized", "Event Message")
		}
	}
}

func TestReadMetrics(t *testing.T) {
	expect := assert.New(t)

	// Read all Metrics: invalid parameters
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/metrics?limit=invalid", nil) // expect int
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code)
		expect.Contains(w.Body.String(), "invalid parameter, limit")
	}
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/metrics?entity=123456", nil) // expect a TUID
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code)
		expect.Contains(w.Body.String(), "invalid TUID parameter, entity")
	}
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/metrics?from=12-31-2023", nil) // expect 2006-01-02
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code)
		expect.Contains(w.Body.String(), "invalid date")
	}
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/metrics?to=12-31-2023", nil) // expect 2006-01-02
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code)
		expect.Contains(w.Body.String(), "invalid date")
	}

	// Read all Metrics: missing authorization token
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/metrics", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusUnauthorized, w.Code)
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON APIEvent") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusUnauthorized, e.Code, "Event Code")
			expect.Contains(e.Message, "unauthenticated", "Event Message")
		}
	}

	// Read all Metrics: missing admin role
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/metrics", nil)
	req.Header.Set("Authorization", "Bearer "+regularToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusForbidden, w.Code)
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON APIEvent") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusForbidden, e.Code, "Event Code")
			expect.Contains(e.Message, "unauthorized", "Event Message")
		}
	}

}

func TestReadMetric(t *testing.T) {
	expect := assert.New(t)

	// Read Metric: invalid metric ID
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/metrics/invalid", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON APIEvent") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusBadRequest, e.Code, "Event Code")
			expect.Contains(e.Message, "bad request: invalid path parameter ID", "Event Message")
		}
	}

	// Read Metric: unknown metric ID
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/metrics/"+tuid.NewID().String(), nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNotFound, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON APIEvent") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusNotFound, e.Code, "Event Code")
			expect.Contains(e.Message, "not found: metric", "Event Message")
		}
	}
}

func TestExistsMetric(t *testing.T) {
	expect := assert.New(t)

	// Metric Exists: invalid metric ID
	w := httptest.NewRecorder()
	req, err := http.NewRequest("HEAD", "/v1/metrics/invalid", nil)
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
	}

	// Metric Exists: unknown metric ID
	w = httptest.NewRecorder()
	req, err = http.NewRequest("HEAD", "/v1/metrics/"+tuid.NewID().String(), nil)
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNotFound, w.Code, "HTTP Status Code")
	}
}

func TestDeleteMetric(t *testing.T) {
	expect := assert.New(t)
	id := tuid.NewID().String()

	// Delete Metric: invalid metric ID
	w := httptest.NewRecorder()
	req, err := http.NewRequest("DELETE", "/v1/metrics/invalid", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON APIEvent") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusBadRequest, e.Code, "Event Code")
			expect.Contains(e.Message, "bad request: invalid path parameter ID", "Event Message")
		}
	}

	// Delete Metric: missing authorization token
	w = httptest.NewRecorder()
	req, err = http.NewRequest("DELETE", "/v1/metrics/"+id, nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusUnauthorized, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON APIEvent") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusUnauthorized, e.Code, "Event Code")
			expect.Contains(e.Message, "unauthenticated", "Event Message")
		}
	}

	// Delete Metric: missing admin role
	w = httptest.NewRecorder()
	req, err = http.NewRequest("DELETE", "/v1/metrics/"+id, nil)
	req.Header.Set("Authorization", "Bearer "+regularToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusForbidden, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON APIEvent") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusForbidden, e.Code, "Event Code")
			expect.Contains(e.Message, "unauthorized", "Event Message")
		}
	}

	// Delete Metric: unknown metric ID
	w = httptest.NewRecorder()
	req, err = http.NewRequest("DELETE", "/v1/metrics/"+id, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNotFound, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON APIEvent") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusNotFound, e.Code, "Event Code")
			expect.Contains(e.Message, "not found: metric", "Event Message")
		}
	}
}

func TestReadMetricLabels(t *testing.T) {
	expect := assert.New(t)

	// Invalid pagination parameters
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/metric_labels?limit=invalid", nil) // expect int
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code)
		expect.Contains(w.Body.String(), "invalid parameter, limit")
	}

	// Missing authorization token
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/metric_labels", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusUnauthorized, w.Code)
	}

	// Missing admin role
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/metric_labels", nil)
	req.Header.Set("Authorization", "Bearer "+regularToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusForbidden, w.Code)
	}
}

func TestReadMetricEntityIDs(t *testing.T) {
	expect := assert.New(t)

	// Missing authorization token
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/metric_entity_ids", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusUnauthorized, w.Code)
	}

	// Missing admin role
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/metric_entity_ids", nil)
	req.Header.Set("Authorization", "Bearer "+regularToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusForbidden, w.Code)
	}
}

func TestReadMetricEntityTypes(t *testing.T) {
	expect := assert.New(t)

	// Missing authorization token
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/metric_entity_types", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusUnauthorized, w.Code)
	}

	// Missing admin role
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/metric_entity_types", nil)
	req.Header.Set("Authorization", "Bearer "+regularToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusForbidden, w.Code)
	}
}

func TestMetricTags(t *testing.T) {
	expect := assert.New(t)

	// Missing authorization token
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/metric_tags", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusUnauthorized, w.Code)
	}

	// Missing admin role
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/metric_tags", nil)
	req.Header.Set("Authorization", "Bearer "+regularToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusForbidden, w.Code)
	}
}

func TestReadMetricStats(t *testing.T) {
	expect := assert.New(t)

	// Missing authorization token
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/metric_stats", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusUnauthorized, w.Code)
	}

	// Missing admin role
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/metric_stats", nil)
	req.Header.Set("Authorization", "Bearer "+regularToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusForbidden, w.Code)
	}

	// Invalid entity ID
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/metric_stats?entity=123456", nil) // expect a TUID
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code)
		expect.Contains(w.Body.String(), "invalid TUID parameter, entity")
	}

	// Invalid range dates
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/metric_stats?from=12-31-2023", nil) // expect 2006-01-02
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code)
		expect.Contains(w.Body.String(), "invalid date")
	}
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/metric_stats?to=12-31-2023", nil) // expect 2006-01-02
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code)
		expect.Contains(w.Body.String(), "invalid date")
	}

	// Missing query parameter for grouping results
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/metric_stats", nil) // expect entity, type, or tag parameter
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code)
		expect.Contains(w.Body.String(), "required query parameter")
	}
}
