package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/voxtechnica/tuid-go"
	v "github.com/voxtechnica/versionary"

	"versionary-api/pkg/event"
)

// registerEventRoutes initializes the Event routes with the Gin router.
func registerEventRoutes(r *gin.Engine) {
	r.POST("/v1/events", roleAuthorizer("admin"), createEvent)
	r.GET("/v1/events", roleAuthorizer("admin"), readEvents)
	r.GET("/v1/events/:id", readEvent)
	r.HEAD("/v1/events/:id", existsEvent)
	r.DELETE("/v1/events/:id", roleAuthorizer("admin"), deleteEvent)
	r.GET("/v1/event_entity_ids", roleAuthorizer("admin"), readEventEntityIDs)
	r.GET("/v1/event_entity_types", roleAuthorizer("admin"), readEventEntityTypes)
	r.GET("/v1/event_log_levels", roleAuthorizer("admin"), readEventLogLevels)
	r.GET("/v1/event_dates", roleAuthorizer("admin"), readEventDates)
}

// createEvent creates a new Event.
//
// @Summary Create a new Event
// @Description Create a new Event.
// @Tags Event
// @Accept json
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Success 201 {object} event.Event "Newly-created Event"
// @Failure 400 {object} APIEvent "Bad Request (invalid JSON body)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 422 {object} APIEvent "Event validation errors"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Header 201 {string} Location "URL of the newly created Event"
// @Router /v1/events [post]
func createEvent(c *gin.Context) {
	// Parse the request body as an Event
	var e event.Event
	if err := c.ShouldBindJSON(&e); err != nil {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid JSON body: %w", err))
		return
	}
	// Create a new Event
	e, problems, err := api.EventService.Create(c, e)
	if len(problems) > 0 && err != nil {
		abortWithError(c, http.StatusUnprocessableEntity, fmt.Errorf("unprocessable entity: %w", err))
		return
	}
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}
	// Return the new Event
	c.Header("Location", c.Request.URL.String()+"/"+e.ID)
	c.JSON(http.StatusCreated, e)
}

// readEvents returns a paginated list of Events.
//
// @Summary List Events
// @Description List Events, paging with reverse, limit, and offset.
// @Description Optionally, filter by date, entity ID, or log level.
// @Description If no filter is specified, the default is to return up to limit recent Events in reverse order.
// @Tags Event
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param entity query string false "Entity ID (a TUID)"
// @Param type query string false "Entity Type (e.g. User, Organization, etc.)"
// @Param log_level query string false "Log Level"  Enums(TRACE, DEBUG, INFO, WARN, ERROR, FATAL)
// @Param date query string false "Date (YYYY-MM-DD)"
// @Param reverse query bool false "Reverse Order (default: false)"
// @Param limit query int false "Limit (default: 100)"
// @Param offset query string false "Offset (default: forward/reverse alphanumeric)"
// @Success 200 {array} event.Event "Events"
// @Failure 400 {object} APIEvent "Bad Request (invalid parameter)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/events [get]
func readEvents(c *gin.Context) {
	// Parse and validate query parameters, with defaults
	reverse, limit, offset, err := paginationParams(c, false, 100)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}
	entityID := c.Query("entity")
	if entityID != "" && !tuid.IsValid(tuid.TUID(entityID)) {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid TUID parameter, entity: %s", entityID))
		return
	}
	entityType := c.Query("type")
	logLevel := strings.ToUpper(c.Query("log_level"))
	if logLevel != "" && !event.LogLevel(logLevel).IsValid() {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid log level: %s", logLevel))
		return
	}
	date := strings.TrimSpace(c.Query("date"))
	if date != "" {
		_, err := time.Parse("2006-01-02", date)
		if err != nil {
			abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid date %s: %w", date, err))
			return
		}
	}
	// Read and return paginated Events
	if entityID != "" {
		// Read paginated Events for a specified entity
		es, err := api.EventService.ReadEventsByEntityIDAsJSON(c, entityID, reverse, limit, offset)
		if err != nil {
			e, _, _ := api.EventService.Create(c, event.Event{
				UserID:     contextUserID(c),
				EntityType: "Event",
				LogLevel:   event.ERROR,
				Message:    fmt.Errorf("read events by entityID %s: %w", entityID, err).Error(),
				URI:        c.Request.URL.String(),
				Err:        err,
			})
			abortWithError(c, http.StatusInternalServerError, e)
			return
		}
		c.Data(http.StatusOK, "application/json;charset=UTF-8", es)
	} else if entityType != "" {
		// Read paginated Events for a specified entity type
		es, err := api.EventService.ReadEventsByEntityTypeAsJSON(c, entityType, reverse, limit, offset)
		if err != nil {
			e, _, _ := api.EventService.Create(c, event.Event{
				UserID:     contextUserID(c),
				EntityType: "Event",
				LogLevel:   event.ERROR,
				Message:    fmt.Errorf("read events by entity type %s: %w", entityType, err).Error(),
				URI:        c.Request.URL.String(),
				Err:        err,
			})
			abortWithError(c, http.StatusInternalServerError, e)
			return
		}
		c.Data(http.StatusOK, "application/json;charset=UTF-8", es)
	} else if logLevel != "" {
		// Read paginated Events by Log Level
		es, err := api.EventService.ReadEventsByLogLevelAsJSON(c, logLevel, reverse, limit, offset)
		if err != nil {
			e, _, _ := api.EventService.Create(c, event.Event{
				UserID:     contextUserID(c),
				EntityType: "Event",
				LogLevel:   event.ERROR,
				Message:    fmt.Errorf("read events by logLevel %s: %w", logLevel, err).Error(),
				URI:        c.Request.URL.String(),
				Err:        err,
			})
			abortWithError(c, http.StatusInternalServerError, e)
			return
		}
		c.Data(http.StatusOK, "application/json;charset=UTF-8", es)
	} else if date != "" {
		// Read paginated Events by Date (YYYY-MM-DD)
		es, err := api.EventService.ReadEventsByDateAsJSON(c, date, reverse, limit, offset)
		if err != nil {
			e, _, _ := api.EventService.Create(c, event.Event{
				UserID:     contextUserID(c),
				EntityType: "Event",
				LogLevel:   event.ERROR,
				Message:    fmt.Errorf("read events by date %s: %w", date, err).Error(),
				URI:        c.Request.URL.String(),
				Err:        err,
			})
			abortWithError(c, http.StatusInternalServerError, e)
			return
		}
		c.Data(http.StatusOK, "application/json;charset=UTF-8", es)
	} else {
		// Read up to limit recent Events in reverse order
		es, err := api.EventService.ReadRecentEvents(c, limit)
		if err != nil {
			e, _, _ := api.EventService.Create(c, event.Event{
				UserID:     contextUserID(c),
				EntityType: "Event",
				LogLevel:   event.ERROR,
				Message:    fmt.Errorf("read %d recent events: %w", limit, err).Error(),
				URI:        c.Request.URL.String(),
				Err:        err,
			})
			abortWithError(c, http.StatusInternalServerError, e)
			return
		}
		c.JSON(http.StatusOK, es)
	}
}

// readEvent returns the current version of the specified Event.
//
// @Summary Get Event
// @Description Get Event by ID.
// @Tags Event
// @Produce json
// @Param id path string true "Event ID"
// @Success 200 {object} event.Event "Event"
// @Failure 400 {object} APIEvent "Bad Request (invalid path parameter ID)"
// @Failure 404 {object} APIEvent "Not Found"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/events/{id} [get]
func readEvent(c *gin.Context) {
	// Validate the path parameter ID
	id := c.Param("id")
	if !tuid.IsValid(tuid.TUID(id)) {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid path parameter ID: %s", id))
		return
	}
	// Read and return the specified Event
	e, err := api.EventService.ReadAsJSON(c, id)
	if err != nil && errors.Is(err, v.ErrNotFound) {
		abortWithError(c, http.StatusNotFound, fmt.Errorf("not found: event %s", id))
		return
	}
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}
	c.Data(http.StatusOK, "application/json;charset=UTF-8", e)
}

// existsEvent checks if the specified Event exists.
//
// @Summary Event Exists
// @Description Check if the specified Event exists.
// @Tags Event
// @Param id path string true "Event ID"
// @Success 204 "Event Exists"
// @Failure 400 "Bad Request (invalid path parameter ID)"
// @Failure 404 "Not Found"
// @Router /v1/events/{id} [head]
func existsEvent(c *gin.Context) {
	id := c.Param("id")
	if !tuid.IsValid(tuid.TUID(id)) {
		c.Status(http.StatusBadRequest)
	} else if !api.EventService.Exists(c, id) {
		c.Status(http.StatusNotFound)
	} else {
		c.Status(http.StatusNoContent)
	}
}

// deleteEvent deletes the specified Event.
//
// @Summary Delete Event
// @Description Delete and return the specified Event.
// @Tags Event
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param id path string true "Event ID"
// @Success 200 {object} event.Event "Deleted Event"
// @Failure 400 {object} APIEvent "Bad Request (invalid path parameter ID)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 404 {object} APIEvent "Not Found"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/events/{id} [delete]
func deleteEvent(c *gin.Context) {
	// Validate the path parameter ID
	id := c.Param("id")
	if !tuid.IsValid(tuid.TUID(id)) {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid path parameter ID: %s", id))
		return
	}
	// Delete the specified Event
	e, err := api.EventService.Delete(c, id)
	if err != nil && errors.Is(err, v.ErrNotFound) {
		abortWithError(c, http.StatusNotFound, fmt.Errorf("not found: event %s", id))
		return
	}
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, err)
		return
	}
	// Return the deleted event
	c.JSON(http.StatusOK, e)
}

// readEventEntityIDs returns a list of entity IDs for which events exist.
// It's useful for paging through events by entity ID.
//
// @Summary List Event Entity IDs
// @Description Get a paginated list of entity IDs for which events exist.
// @Tags Event
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param reverse query bool false "Reverse Order (default: false)"
// @Param limit query int false "Limit (default: 100)"
// @Param offset query string false "Offset (default: forward/reverse alphanumeric)"
// @Success 200 {array} string "Entity IDs"
// @Failure 400 {object} APIEvent "Bad Request (invalid pagination parameter)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/event_entity_ids [get]
func readEventEntityIDs(c *gin.Context) {
	// Parse pagination query parameters, with defaults
	reverse, limit, offset, err := paginationParams(c, false, 100)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}
	// Read and return the list of entity IDs for which events exist
	ids, err := api.EventService.ReadEntityIDs(c, reverse, limit, offset)
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: "Event",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read event entity ids: %w", err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.JSON(http.StatusOK, ids)
}

// readEventEntityTypes returns a list of entity types for which events exist.
// It's useful for paging through events by entity type.
//
// @Summary List Event Entity Types
// @Description Get a complete, sorted list of entity types for which events exist.
// @Tags Event
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Success 200 {array} string "Entity Types"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/event_entity_types [get]
func readEventEntityTypes(c *gin.Context) {
	// Read and return the list of entity types for which events exist
	types, err := api.EventService.ReadAllEntityTypes(c)
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: "Event",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read event entity types: %w", err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.JSON(http.StatusOK, types)
}

// readEventLogLevels returns a list of log levels for which events exist.
// It's useful for paging through events by log level.
//
// @Summary List Event Log Levels
// @Description Get a complete, sorted list of log levels for which events exist.
// @Tags Event
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Success 200 {array} string "Log Levels"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/event_log_levels [get]
func readEventLogLevels(c *gin.Context) {
	// Read and return the list of log levels for which events exist
	levels, err := api.EventService.ReadAllLogLevels(c)
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: "Event",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read event log levels: %w", err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.JSON(http.StatusOK, levels)
}

// readEventDates returns a list of ISO dates for which events exist.
// It's useful for paging through events by date.
//
// @Summary List Event Dates
// @Description Get a paginated list of ISO dates (e.g. yyyy-mm-dd) for which events exist.
// @Tags Event
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param reverse query bool false "Reverse Order (default: false)"
// @Param limit query int false "Limit (default: 100)"
// @Param offset query string false "Offset (default: forward/reverse alphanumeric)"
// @Success 200 {array} string "Dates"
// @Failure 400 {object} APIEvent "Bad Request (invalid pagination parameter)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/event_dates [get]
func readEventDates(c *gin.Context) {
	// Parse pagination query parameters, with defaults
	reverse, limit, offset, err := paginationParams(c, false, 100)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}
	// Read and return a list of dates for which events exist
	dates, err := api.EventService.ReadDates(c, reverse, limit, offset)
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: "Event",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read event dates: %w", err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.JSON(http.StatusOK, dates)
}
