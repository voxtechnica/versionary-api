package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/voxtechnica/tuid-go"

	"versionary-api/pkg/metric"
)

func TestCreateAndDeleteMetric(t *testing.T) {
	expect := assert.New(t)
	// Create a new Metric
	var m metric.Metric
	w := httptest.NewRecorder()
	body := `{"title":"Test Metric","value":1.2,"units":"seconds"}`
	req, err := http.NewRequest("POST", "/v1/metrics", strings.NewReader(body))
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
			expect.Equal(1.2, m.Value)
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

func TestCreateMetric(t *testing.T) {
	// Create a new Metric: missing required fields
	expect := assert.New(t)
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
			expect.Contains(e.Message, "bad request: invalid JSON body", "Event Message")
		}
	}

	// Create a new Metric: missing authorization token
	w = httptest.NewRecorder()
	body := `{"title":"Test Metric","value":1.2,"units":"seconds"}`
	req, err = http.NewRequest("POST", "/v1/metrics", strings.NewReader(body))
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
	body = `{"title":"Test Metric","value":1.2,"units":"seconds"}`
	req, err = http.NewRequest("POST", "/v1/metrics", strings.NewReader(body))
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

func TestDeleteMetric(t *testing.T) {
	expect := assert.New(t)
	// Delete Metric: invalid metric ID
	w := httptest.NewRecorder()
	req, err := http.NewRequest("DELETE", "/v1/metrics/invalidId", nil)
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
	req, err = http.NewRequest("DELETE", "/v1/metrics/"+metricOne.ID, nil)
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
	req, err = http.NewRequest("DELETE", "/v1/metrics/"+metricOne.ID, nil)
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
	metricID := tuid.NewID().String()
	req, err = http.NewRequest("DELETE", "/v1/metrics/"+metricID, nil)
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

	// Read Metric: invalid metric ID
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/metrics/invalidId", nil)
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
	metricID := tuid.NewID().String()
	req, err = http.NewRequest("GET", "/v1/metrics/"+metricID, nil)
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

func TestReadMetrics(t *testing.T) {
	expect := assert.New(t)
	// Read all Metrics
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/metrics/", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code)
		var m []metric.Metric
		if expect.NoError(json.NewDecoder(w.Body).Decode(&m), "Decode JSON Metric") {
			expect.Len(m, 2, "Number of Metrics")
		}
	}

	// Read all Metrics: invalid parameter
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/metrics/?limit=invalid", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code)
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON APIEvent") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusBadRequest, e.Code, "Event Code")
			expect.Contains(e.Message, "bad request: invalid parameter, limit: invalid", "Event Message")
		}
	}

	// Read all Metrics: missing authorization token
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/metrics/", nil)
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
	req, err = http.NewRequest("GET", "/v1/metrics/", nil)
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

func TestMetricTags(t *testing.T) {
	expect := assert.New(t)
	// Read known metric tags
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/metric_tags", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code)
		var tags []string
		if expect.NoError(json.NewDecoder(w.Body).Decode(&tags), "Decode JSON Metric Tags") {
			expect.GreaterOrEqual(len(tags), 2, "Number of Metric Tags")
			expect.Contains(tags, metricOne.Tags[0], "Metric Tag")
		}
	}

	// Read known metric: missing authorization token
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/metric_tags", nil)
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

	// Read known metric: missing admin role
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/metric_tags", nil)
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

func TestReadMetricEntityTypes(t *testing.T) {
	expect := assert.New(t)
	// Read known metric entity types
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/metric_entity_types", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code)
		var entityTypes []string
		if expect.NoError(json.NewDecoder(w.Body).Decode(&entityTypes), "Decode JSON Metric Entity Types") {
			expect.GreaterOrEqual(len(entityTypes), 2, "Number of Metric Entity Types")
			expect.Contains(entityTypes, metricOne.EntityType, "Metric Entity Type")
		}
	}

	// Read known metric: missing authorization token
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/metric_entity_types", nil)
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

	// Read known metric: missing admin role
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/metric_entity_types", nil)
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

func TestReadMetricStats(t *testing.T) {
	expect := assert.New(t)
	// Read metric stats with known metric entity ID
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/metric_stats/"+metricTwo.EntityID, nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code)
		var m metric.MetricStat
		if expect.NoError(json.NewDecoder(w.Body).Decode(&m), "Decode JSON Metric") {
			expect.Equal(metricTwo.EntityID, m.EntityID, "Metric Stat Entity ID")
			expect.Equal("Test Entity", m.EntityType, "Metric Stat Entity Type")
		}
	}

	// Read metric stats with invalid metric entity ID
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/metric_stats/invalidId", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code)
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON APIEvent") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusBadRequest, e.Code, "Event Code")
			expect.Contains(e.Message, "bad request: invalid path parameter entity ID: invalidId", "Event Message")
		}
	}

	// Read metric stats with unknown metric entity ID
	w = httptest.NewRecorder()
	entityID := tuid.NewID().String()
	req, err = http.NewRequest("GET", "/v1/metric_stats/"+entityID, nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNotFound, w.Code)
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Metric") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusNotFound, e.Code, "Event Code")
			expect.Contains(e.Message, entityID, "Event Message")
		}
	}

}

func TestFilterMetricStats(t *testing.T) {
	expect := assert.New(t)
	// Filter metric stats with all params
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/metric_stats?entity="+metricTwo.EntityID+"&type="+metricTwo.EntityType+"&from=2023-01-01&to=2023-12-31", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code)
		var m metric.MetricStat
		if expect.NoError(json.NewDecoder(w.Body).Decode(&m), "Decode JSON Metric") {
			expect.Equal(metricTwo.EntityID, m.EntityID, "Metric Stat Entity ID")
			expect.Equal("Test Entity", m.EntityType, "Metric Stat Entity Type")
		}
	}

	// Filter metric stats with entity ID
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/metric_stats?entity="+metricTwo.EntityID, nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code)
		var m metric.MetricStat
		if expect.NoError(json.NewDecoder(w.Body).Decode(&m), "Decode JSON Metric") {
			expect.Equal(metricTwo.EntityID, m.EntityID, "Metric Stat Entity ID")
			expect.Equal("Test Entity", m.EntityType, "Metric Stat Entity Type")
		}
	}

	// Filter metric stats with entity type
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/metric_stats?type="+metricTwo.EntityType, nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code)
		var m metric.MetricStat
		if expect.NoError(json.NewDecoder(w.Body).Decode(&m), "Decode JSON Metric") {
			expect.Equal(metricTwo.EntityID, m.EntityID, "Metric Stat Entity ID")
			expect.Equal("Test Entity", m.EntityType, "Metric Stat Entity Type")
		}
	}

	// Filter metric stats with tag
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/metric_stats?tag=v1", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code)
		var m metric.MetricStat
		if expect.NoError(json.NewDecoder(w.Body).Decode(&m), "Decode JSON Metric") {
			expect.Contains(m.Tags, "v1", "Metric Stat Tag")
		}
	}

	// Filter metric stats with tag and dates
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/metric_stats?tag=v2&from=2023-01-01&to=2023-12-31", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code)
		var m metric.MetricStat
		if expect.NoError(json.NewDecoder(w.Body).Decode(&m), "Decode JSON Metric") {
			expect.Contains(m.Tags, "v2", "Metric Stat Tag")
		}
	}

	// Filter metric stats with invalid time param
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/metric_stats?from=2023-01-01T00:00:00Z", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code)
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON APIEvent") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusBadRequest, e.Code, "Event Code")
			expect.Contains(e.Message, "bad request: valid entity, type or tag must be provided", "Event Message")
		}
	}

	// Filter metric stats with invalid entity ID param
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/metric_stats?entity=invalidId", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code)
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON APIEvent") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusBadRequest, e.Code, "Event Code")
			expect.Contains(e.Message, "bad request: valid entity, type or tag must be provided", "Event Message")
		}
	}

	// Filter metric stats with unknown entity ID param
	entityID := tuid.NewID().String()
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/metric_stats?entity="+entityID, nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNotFound, w.Code)
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON APIEvent") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusNotFound, e.Code, "Event Code")
		}
	}
}
