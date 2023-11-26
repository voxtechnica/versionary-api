package main

import (
	"errors"
	"fmt"
	"net/http"
	"versionary-api/pkg/event"
	"versionary-api/pkg/metric"

	"github.com/gin-gonic/gin"
	"github.com/voxtechnica/tuid-go"
	v "github.com/voxtechnica/versionary"
)

// registerMetricRoutes initializes the Metric routes with the Gin router.
func registerMetricRoutes(r *gin.Engine) {
	r.POST("/v1/metrics", roleAuthorizer("admin"), createMetric)
	r.DELETE("/v1/metrics/:id", roleAuthorizer("admin"), deleteMetric)
	r.GET("/v1/metrics/:id", readMetric)
	r.GET("/v1/metrics/", roleAuthorizer("admin"), readMetrics)
	r.GET("/v1/metric_tags", roleAuthorizer("admin"), readMetricTags)
	r.GET("/v1/metric_entity_types", roleAuthorizer("admin"), readMetricEntityTypes)
	r.GET("/v1/metric_stats", filterMetricStats)
	r.GET("/v1/metric_stats/:entityId", readMetricStats)
	// r.GET("/v1/metric_hist", roleAuthorizer("admin"), readMetricHistograms)
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
// @Header 201 {string} Location "URL of the newly created Event"
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

// readMetrics handles the HTTP request to read all Metrics.
//
// @Summary Read Metric
// @Description Get Metrics
// @Description Get Metrics, paging with reverse, limit and offset.
// @Tags Metric
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param reverse query bool false "Reverse Order (default: false)"
// @Param limit query int false "Limit (default: 10)"
// @Param offset query string false "Offset (default: forward/reverse alphanumeric)"
// @Success 200 {object} metric.Metric "Metric"
// @Failure 400 {object} APIEvent "Bad Request (invalid path parameter ID)"
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
	// Read and return all Metrics
	m := api.MetricService.ReadMetrics(c, reverse, limit, offset)
	c.JSON(http.StatusOK, m)
}

// readMetricTags returns a list of Metrics tags.
// It's useful for paging through metrics by tag.
//
// @Summary List Metric Tags
// @Description List Metric Tags
// @Description List metric tags, for which metric exist.
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

// readMetricEntityTypes returns a list of all metric entity types
//
// @Summary List Metric Entity Types
// @Description List Metric Entity Types
// @Description List metric entity types.
// @Tags Metric
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Success 200 {array} string "Content Types"
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
			Message:    fmt.Errorf("read metric types: %w", err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.JSON(http.StatusOK, types)
}

// filterMetricStats handles the HTTP request to filter Metric statistics by entity ID, entity type, from date, to date.
//
// @Summary Filter Metric Stats
// @Description Filter Metric Stats
// @Description Filter metric stats by entity ID, entity type, from date, to date.
// @Tags Metric
// @Produce json
// @Param entity query string true "Entity ID"
// @Param type query string true "Entity Type"
// @Param tag query string true "Tag"
// @Param from query string false "From ISO Date"
// @Param to query string false "To ISO Date"
// @Success 200 {object} metric "Metric Stats"
// @Failure 400 {object} APIEvent "Bad Request (query parameter entity or type or tag must be provided)"
// @Failure 404 {object} APIEvent "Not Found"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/metric_stats [get]
func filterMetricStats(c *gin.Context) {
	// Parse query parameters
	entityId := c.Query("entity")
	entityType := c.Query("type")
	fromTimeStr := c.Query("from")
	toTimeStr := c.Query("to")
	tag := c.Query("tag")

	// Logic to call API service to retrieve MetricStats based on the provided parameters.
	var metricStats metric.MetricStat
	var errMessage string
	var err error

	if entityId != "" && tuid.IsValid(tuid.TUID(entityId)) {
		metricStats, err = api.MetricService.GenerateStatsForEntityIDByDate(c, entityId, fromTimeStr, toTimeStr)
		errMessage = fmt.Sprintf("search MetricStats by entityId (%s) within date range", entityId)
	} else if entityType != "" {
		metricStats, err = api.MetricService.GenerateStatsForEntityTypeByDate(c, entityType, fromTimeStr, toTimeStr)
		errMessage = fmt.Sprintf("search MetricStats by entityType (%s) within date range", entityType)
	} else if tag != "" {
		metricStats, err = api.MetricService.GenerateStatsForTag(c, tag)
		errMessage = fmt.Sprintf("search MetricStats by tag (%s) within date range", tag)
	} else {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: valid entity, type or tag must be provided"))
		return
	}
	if err != nil && errors.Is(err, v.ErrNotFound) {
		abortWithError(c, http.StatusNotFound, err)
		return
	}

	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: api.MetricService.EntityType,
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("%s: %w", errMessage, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}

	c.JSON(http.StatusOK, metricStats)
}

// readMetricStats handles the HTTP request to read Metric statistics.
//
// @Summary Read Metric Stats
// @Description Read Metric Stats
// @Description Read metric stats by entity ID.
// @Tags Metric
// @Produce json
// @Param entityId path string true "Entity ID"
// @Param reverse query bool false "Reverse Order (default: false)"
// @Param limit query int false "Limit (default: 10)"
// @Param offset query string false "Offset (default: forward/reverse alphanumeric)"
// @Success 200 {object} metric "Metric Stats"
// @Failure 400 {object} APIEvent "Bad Request (invalid path parameter entityId)"
// @Failure 404 {object} APIEvent "Not Found"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/metric_stats/{entityId} [get]
func readMetricStats(c *gin.Context) {
	// Validate the path parameter entity ID
	entityId := c.Param("entityId")
	if !tuid.IsValid(tuid.TUID(entityId)) {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid path parameter entity ID: %s", entityId))
		return
	}

	// Parse pagination query parameters, with defaults
	reverse, limit, offset, err := paginationParams(c, false, 100)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}
	// Read and return the specified Metric
	m, err := api.MetricService.GenerateStatsForEntityID(c, entityId, reverse, limit, offset)
	if err != nil && errors.Is(err, v.ErrNotFound) {
		abortWithError(c, http.StatusNotFound, err)
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   entityId,
			EntityType: "Metric",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read metric %s: %w", entityId, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.JSON(http.StatusOK, m)
}

