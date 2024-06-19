package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"versionary-api/pkg/event"
	"versionary-api/pkg/metric"

	"github.com/gin-gonic/gin"
	"github.com/voxtechnica/tuid-go"
	v "github.com/voxtechnica/versionary"
)

// registerMetricRoutes initializes the Metric routes with the Gin router.
func registerMetricRoutes(r *gin.Engine) {
	r.POST("/v1/metrics", roleAuthorizer("admin"), createMetric)
	r.GET("/v1/metrics", roleAuthorizer("admin"), readMetrics)
	r.GET("/v1/metrics/:id", readMetric)
	r.HEAD("/v1/metrics/:id", existsMetric)
	r.DELETE("/v1/metrics/:id", roleAuthorizer("admin"), deleteMetric)
	r.GET("/v1/metric_labels", roleAuthorizer("admin"), readMetricLabels)
	r.GET("/v1/metric_entity_ids", roleAuthorizer("admin"), readMetricEntityIDs)
	r.GET("/v1/metric_entity_types", roleAuthorizer("admin"), readMetricEntityTypes)
	r.GET("/v1/metric_tags", roleAuthorizer("admin"), readMetricTags)
	r.GET("/v1/metric_stats", roleAuthorizer("admin"), readMetricStats)
}

// createMetric handles the HTTP request to create a new Metric.
//
// createMetric creates a new Metric.
//
// @Summary Create Metric
// @Description Create a new Metric
// @Description Create a new Metric.
// @Tags Metric
// @Accept json
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Success 201 {object} metric.Metric "Newly-created Metric"
// @Failure 400 {object} APIEvent "Bad Request (invalid JSON body)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 422 {object} APIEvent "Metric validation errors"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Header 201 {string} Location "URL of the newly created Metric"
// @Router /v1/metrics [post]
func createMetric(c *gin.Context) {
	// Parse the request body as a Metric.
	var m metric.Metric
	if err := c.ShouldBindJSON(&m); err != nil {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid JSON body: %w", err))
		return
	}
	// Create a new Metric
	m, problems, err := api.MetricService.Create(c, m)
	if len(problems) > 0 && err != nil {
		abortWithError(c, http.StatusUnprocessableEntity, fmt.Errorf("unprocessable entity: %w", err))
		return
	}
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}
	// Return the new Metric
	c.Header("Location", c.Request.URL.String()+"/"+m.ID)
	c.JSON(http.StatusCreated, m)
}

// readMetrics handles the HTTP request to read all Metrics.
//
// @Summary Read Metrics
// @Description Get Metrics
// @Description Get Metrics, paging with reverse, limit and offset or date range.
// @Description Optionally, filter by entity ID, entity type, or tag.
// @Tags Metric
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param entity query string false "Entity ID (a TUID)"
// @Param type query string false "Entity Type (e.g. User, Organization, etc.)"
// @Param tag query string false "Tag"
// @Param from query string false "Inclusive Begining of Date Range (YYYY-MM-DD)"
// @Param to query string false "Exclusive End of Date Range (YYYY-MM-DD)"
// @Param reverse query bool false "Reverse Order (default: false)"
// @Param limit query int false "Limit (default: 100)"
// @Param offset query string false "Offset (default: forward/reverse alphanumeric)"
// @Success 200 {array} metric.Metric "Metrics"
// @Failure 400 {object} APIEvent "Bad Request (invalid parameter)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/metric [get]
func readMetrics(c *gin.Context) {
	// Parse pagination query parameters, with defaults
	reverse, limit, offset, err := paginationParams(c, false, 100)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}

	// Validate other parameters
	entityID := c.Query("entity")
	if entityID != "" && !tuid.IsValid(tuid.TUID(entityID)) {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid TUID parameter, entity: %s", entityID))
		return
	}
	entityType := c.Query("type")
	tag := c.Query("tag")
	from := strings.TrimSpace(c.Query("from"))
	if from != "" {
		_, err = time.Parse("2006-01-02", from)
		if err != nil {
			abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid date %s: %w", from, err))
			return
		}
	}
	to := strings.TrimSpace(c.Query("to"))
	if to != "" {
		_, err = time.Parse("2006-01-02", to)
		if err != nil {
			abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid date %s: %w", to, err))
			return
		}
	}

	// Gather the specified Metrics
	var ms []metric.Metric
	var messageBase string
	if entityID != "" {
		if from != "" && to != "" {
			ms, err = api.MetricService.ReadMetricRangeByEntityID(c, entityID, from, to, reverse)
		} else {
			ms, err = api.MetricService.ReadMetricsByEntityID(c, entityID, reverse, limit, offset)
		}
		if err != nil {
			messageBase = fmt.Sprintf("read metrics for entity %s", entityID)
		}
	} else if entityType != "" {
		if from != "" && to != "" {
			ms, err = api.MetricService.ReadMetricRangeByEntityType(c, entityType, from, to, reverse)
		} else {
			ms, err = api.MetricService.ReadMetricsByEntityType(c, entityType, reverse, limit, offset)
		}
		if err != nil {
			messageBase = fmt.Sprintf("read metrics for entity type %s", entityType)
		}
	} else if tag != "" {
		if from != "" && to != "" {
			ms, err = api.MetricService.ReadMetricRangeByTag(c, tag, from, to, reverse)
		} else {
			ms, err = api.MetricService.ReadMetricsByTag(c, tag, reverse, limit, offset)
		}
		if err != nil {
			messageBase = fmt.Sprintf("read metrics for tag %s", tag)
		}
	} else {
		ms = api.MetricService.ReadMetrics(c, reverse, limit, offset)
	}

	// Check for errors and return the Metrics
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: "Metric",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("%s: %w", messageBase, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.JSON(http.StatusOK, ms)
}

// readMetric handles the HTTP request to read a Metric.
//
// @Summary Read Metric
// @Description Get Metric
// @Description Get Metric by ID.
// @Tags Metric
// @Produce json
// @Param id path string true "Metric ID"
// @Success 200 {object} metric.Metric "Metric"
// @Failure 400 {object} APIEvent "Bad Request (invalid path parameter ID)"
// @Failure 404 {object} APIEvent "Not Found"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/metric/{id} [get]
func readMetric(c *gin.Context) {
	// Validate the path parameter ID
	id := c.Param("id")
	if !tuid.IsValid(tuid.TUID(id)) {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid path parameter ID: %s", id))
		return
	}
	// Read and return the specified Metric
	m, err := api.MetricService.ReadAsJSON(c, id)
	if err != nil && errors.Is(err, v.ErrNotFound) {
		abortWithError(c, http.StatusNotFound, fmt.Errorf("not found: metric %s", id))
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   id,
			EntityType: "Metric",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read metric %s: %w", id, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.Data(http.StatusOK, "application/json;charset=UTF-8", m)
}

// existsMetric checks if the specified Metric exists.
//
// @Summary Metric Exists
// @Description Metric Exists
// @Description Check if the specified Metric exists.
// @Tags Metric
// @Param id path string true "Metric ID"
// @Success 204 "Metric Exists"
// @Failure 400 "Bad Request (invalid path parameter ID)"
// @Failure 404 "Not Found"
// @Router /v1/metrics/{id} [head]
func existsMetric(c *gin.Context) {
	id := c.Param("id")
	if !tuid.IsValid(tuid.TUID(id)) {
		c.Status(http.StatusBadRequest)
	} else if !api.MetricService.Exists(c, id) {
		c.Status(http.StatusNotFound)
	} else {
		c.Status(http.StatusNoContent)
	}
}

// deleteMetric handles the HTTP request to delete a Metric.
//
// @Summary Delete Metric
// @Description Delete Metric
// @Description Delete and return the specified Metric.
// @Tags Metric
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param id path string true "Metric ID"
// @Success 200 {object} metric.Metric "Deleted Metric"
// @Failure 400 {object} APIEvent "Bad Request (invalid path parameter ID)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 404 {object} APIEvent "Not Found"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/metrics/{id} [delete]
func deleteMetric(c *gin.Context) {
	// Validate the path parameter ID.
	id := c.Param("id")
	if !tuid.IsValid(tuid.TUID(id)) {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid path parameter ID: %s", id))
		return
	}
	// Delete the specified Metric.
	m, err := api.MetricService.Delete(c, id)
	if err != nil && errors.Is(err, v.ErrNotFound) {
		abortWithError(c, http.StatusNotFound, fmt.Errorf("not found: metric %s", id))
		return
	}
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}
	// Return the deleted metric.
	c.JSON(http.StatusOK, m)
}

// readMetricLabels returns a paginated list of Metric IDs and labels.
// This is the preferred, more performant method for 'browsing' metrics.
//
// @Summary Read Metric Labels
// @Description Read Metric Labels
// @Description Read a paginated list of Metric IDs and labels.
// @Description This is the preferred, more performant method for 'browsing' metrics.
// @Tags Metric
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param reverse query bool false "Reverse Order (default: false)"
// @Param limit query int false "Limit (default: 100)"
// @Param offset query string false "Offset (default: forward/reverse alphanumeric)"
// @Success 200 {array} versionary.TextValue "Metric ID/Label pairs"
// @Failure 400 {object} APIEvent "Bad Request (invalid parameter)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/metric_labels [get]
func readMetricLabels(c *gin.Context) {
	// Parse pagination query parameters, with defaults
	reverse, limit, offset, err := paginationParams(c, false, 100)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}
	// Read and return the specified Metric labels
	labels, err := api.MetricService.ReadMetricLabels(c, reverse, limit, offset)
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: "Metric",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read metric labels: %w", err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.JSON(http.StatusOK, labels)
}

// readMetricEntityIDs returns a list of all Metric entity IDs.
// It's useful for paging through metrics by entity ID.

// @Summary List Metric Entity IDs
// @Description List Metric Entity IDs
// @Description List all entity IDs for which metrics exist.
// @Tags Metric
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Success 200 {array} string "Entity IDs"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/metric_entity_ids [get]
func readMetricEntityIDs(c *gin.Context) {
	ids, err := api.MetricService.ReadAllEntityIDs(c)
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: "Metric",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read metric entity IDs: %w", err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.JSON(http.StatusOK, ids)
}

// readMetricEntityTypes returns a list of all entity types for which metrics exist.
// It's useful for paging through metrics by entity type.
//
// @Summary List Metric Entity Types
// @Description List Metric Entity Types
// @Description List all entity types for which metrics exist.
// @Tags Metric
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Success 200 {array} string "Entity Types"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/metric_entity_types [get]
func readMetricEntityTypes(c *gin.Context) {
	types, err := api.MetricService.ReadAllEntityTypes(c)
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: "Metric",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read metric entity types: %w", err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.JSON(http.StatusOK, types)
}

// readMetricTags returns a list of all Metric tags.
// It's useful for paging through metrics by tag.
//
// @Summary List Metric Tags
// @Description List Metric Tags
// @Description List all metric tags, for which metrics exist.
// @Tags Metric
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Success 200 {array} string "Metric Tags"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/metric_tags [get]
func readMetricTags(c *gin.Context) {
	tags, err := api.MetricService.ReadAllTags(c)
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: "Metric",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read metric tags: %w", err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.JSON(http.StatusOK, tags)
}

// readMetricStats generates Metric statistics by entity ID, entity type, or tag.
// Optionally, the metrics can be filtered by date range.
//
// @Summary Read Metric Stats
// @Description Read Metric Stats
// @Description Read Metric Stats by entity ID, entity type, or tag.
// @Description Optionally, filter by date range.
// @Tags Metric
// @Produce json
// @Param entity query string false "Entity ID"
// @Param type query string false "Entity Type"
// @Param tag query string false "Tag"
// @Param from query string false "Inclusive Begining of Date Range (YYYY-MM-DD)"
// @Param to query string false "Exclusive End of Date Range (YYYY-MM-DD)"
// @Success 200 {object} metric.MetricStat "Metric Stats"
// @Failure 400 {object} APIEvent "Bad Request (query parameter entity or type or tag must be provided)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/metric_stats [get]
func readMetricStats(c *gin.Context) {
	// Validate query parameters
	entityID := c.Query("entity")
	if entityID != "" && !tuid.IsValid(tuid.TUID(entityID)) {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid TUID parameter, entity: %s", entityID))
		return
	}
	entityType := c.Query("type")
	tag := c.Query("tag")
	from := strings.TrimSpace(c.Query("from"))
	if from != "" {
		_, err := time.Parse("2006-01-02", from)
		if err != nil {
			abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid date %s: %w", from, err))
			return
		}
	}
	to := strings.TrimSpace(c.Query("to"))
	if to != "" {
		_, err := time.Parse("2006-01-02", to)
		if err != nil {
			abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid date %s: %w", to, err))
			return
		}
	}

	// Generate MetricStats for the specified parameters
	var stats metric.MetricStat
	var err error
	if entityID != "" {
		if from != "" && to != "" {
			stats, err = api.MetricService.ReadMetricStatRangeByEntityID(c, entityID, from, to)
		} else {
			stats, err = api.MetricService.ReadMetricStatByEntityID(c, entityID)
		}
	} else if entityType != "" {
		if from != "" && to != "" {
			stats, err = api.MetricService.ReadMetricStatRangeByEntityType(c, entityType, from, to)
		} else {
			stats, err = api.MetricService.ReadMetricStatByEntityType(c, entityType)
		}
	} else if tag != "" {
		if from != "" && to != "" {
			stats, err = api.MetricService.ReadMetricStatRangeByTag(c, tag, from, to)
		} else {
			stats, err = api.MetricService.ReadMetricStatByTag(c, tag)
		}
	} else {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: required query parameter: entity, type or tag"))
		return
	}
	if err != nil && errors.Is(err, v.ErrNotFound) {
		abortWithError(c, http.StatusNotFound, err)
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: "Metric",
			LogLevel:   event.ERROR,
			Message:    err.Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.JSON(http.StatusOK, stats)
}
