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
	// r.GET("/v1/metrics/", roleAuthorizer("admin"), readMetrics)
	// r.GET("/v1/metric_tags", roleAuthorizer("admin"), readMetricTags)
	// r.GET("/v1/metric_entity_types", roleAuthorizer("admin"), readMetricEntityTypes)
	// r.GET("/v1/metric_stats", roleAuthorizer("admin"), readMetricStats)
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

// readMetricStats handles the HTTP request to read Metric statistics.
