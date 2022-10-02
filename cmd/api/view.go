package main

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/voxtechnica/tuid-go"
	v "github.com/voxtechnica/versionary"
	"net/http"
	"versionary-api/pkg/device"

	"versionary-api/pkg/event"
	"versionary-api/pkg/view"
)

// registerViewRoutes initializes the View routes.
func registerViewRoutes(r *gin.Engine) {
	r.POST("/v1/views", createView)
	r.DELETE("/v1/views/:id", roleAuthorizer("admin"), deleteView)
	r.GET("/v1/views", roleAuthorizer("admin"), readViews)
	r.GET("/v1/views/:id", readView)
	r.HEAD("/v1/views/:id", existsView)
	r.GET("/v1/view_dates", roleAuthorizer("admin"), readViewDates)
	r.GET("/v1/view_device_ids", roleAuthorizer("admin"), readViewDeviceIDs)
	r.GET("/v1/view_counts", roleAuthorizer("admin"), readViewCounts)
	r.GET("/v1/view_counts/:date", roleAuthorizer("admin"), readViewCount)
	r.HEAD("/v1/view_counts/:date", roleAuthorizer("admin"), existsViewCount)
	r.PUT("/v1/view_counts/:date", roleAuthorizer("admin"), updateViewCount)
}

// createView creates a new View.
//
// @Summary Create a new View
// @Description Create a new View with an associated new/updated Device.
// @Tags View
// @Accept json
// @Produce json
// @Param user-agent header string true "User-Agent Header"
// @Success 201 {object} view.View "Newly-created View"
// @Failure 422 {object} APIEvent "View validation errors"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Header 201 {string} Location "/v1/views/{id}"
// @Router /v1/views [post]
func createView(c *gin.Context) {
	// Parse the request body as a View and validate the body
	var body view.View
	if err := c.ShouldBindJSON(&body); err != nil {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid JSON body: %w", err))
		return
	}
	if body.Page.URI == "" {
		abortWithError(c, http.StatusUnprocessableEntity, fmt.Errorf("unprocessable entity: missing page URI"))
		return
	}

	// Create or update the associated Device
	var d device.Device
	var problems []string
	var err error
	if body.Client.DeviceID == "" || !tuid.IsValid(tuid.TUID(body.Client.DeviceID)) {
		// Create a Device if not supplied or invalid ID
		d, problems, err = api.DeviceService.Create(c, c.GetHeader("User-Agent"), contextUserID(c))
	} else {
		// Update the supplied Device
		d, problems, err = api.DeviceService.Update(c, body.Client.DeviceID, c.GetHeader("User-Agent"), contextUserID(c))
	}
	if len(problems) > 0 && err != nil {
		abortWithError(c, http.StatusUnprocessableEntity, fmt.Errorf("unprocessable entity: %w", err))
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   d.ID,
			EntityType: d.Type(),
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("create/update View Device %s: %w", d.ID, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}

	// Augment the View Client information
	body.Client.DeviceID = d.ID
	body.Client.UserAgent = d.UserAgent
	body.Client.IPAddress = c.ClientIP()
	body.Client.CountryCode = c.GetHeader("CloudFront-Viewer-Country")

	// Create the View
	body, problems, err = api.ViewService.Create(c, body)
	if len(problems) > 0 && err != nil {
		abortWithError(c, http.StatusUnprocessableEntity, fmt.Errorf("unprocessable entity: %w", err))
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   body.ID,
			EntityType: body.Type(),
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("create View %s: %w", body.ID, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}

	// Log the creation
	_, _, _ = api.EventService.Create(c, event.Event{
		UserID:     contextUserID(c),
		EntityID:   body.ID,
		EntityType: body.Type(),
		LogLevel:   event.INFO,
		Message:    fmt.Sprintf("created View %s", body.ID),
		URI:        c.Request.URL.String(),
	})

	// Return the new View
	c.Header("Location", c.Request.URL.String()+"/"+body.ID)
	c.JSON(http.StatusCreated, body)
}

// deleteView deletes the specified View.
//
// @Summary Delete View
// @Description Delete and return the specified View.
// @Tags View
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param id path string true "View ID"
// @Success 200 {object} view.View "View that was deleted"
// @Failure 400 {object} APIEvent "Bad Request (invalid path parameter ID)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 404 {object} APIEvent "Not Found"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/views/{id} [delete]
func deleteView(c *gin.Context) {
	// Validate the path parameter ID
	id := c.Param("id")
	if !tuid.IsValid(tuid.TUID(id)) {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid path parameter ID: %s", id))
		return
	}
	// Delete the specified View
	deleted, err := api.ViewService.Delete(c, id)
	if err != nil && errors.Is(err, v.ErrNotFound) {
		abortWithError(c, http.StatusNotFound, fmt.Errorf("not found: View %s", id))
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   id,
			EntityType: "View",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("delete View %s: %w", id, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	// Log the deletion
	_, _, _ = api.EventService.Create(c, event.Event{
		UserID:     contextUserID(c),
		EntityID:   deleted.ID,
		EntityType: deleted.Type(),
		LogLevel:   event.INFO,
		Message:    "deleted View " + deleted.ID,
		URI:        c.Request.URL.String(),
	})
	// Return the deleted view
	c.JSON(http.StatusOK, deleted)
}

// readViews returns a paginated list of Views.
//
// @Summary List Views
// @Description List Views, paging with reverse, limit, and offset. Optionally, filter by DeviceID or Date.
// @Tags View
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param device query string false "Device ID (optional, TUID)"
// @Param date query string false "Date (optional, YYYY-MM-DD)"
// @Param reverse query bool false "Reverse Order (default: false)"
// @Param limit query int false "Limit (default: 100)"
// @Param offset query string false "Offset (default: forward/reverse alphanumeric)"
// @Success 200 {array} view.View "Views"
// @Failure 400 {object} APIEvent "Bad Request (invalid parameter)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/views [get]
func readViews(c *gin.Context) {
	// Parse query parameters, with defaults
	reverse, limit, offset, err := paginationParams(c, false, 100)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}
	date := c.Query("date")
	if date != "" && !view.IsValidDate(date) {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid date: %s", date))
		return
	}
	deviceID := c.Query("device")
	if deviceID != "" && !tuid.IsValid(tuid.TUID(deviceID)) {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid Device ID: %s", deviceID))
		return
	}
	// Read and return paginated Views
	if date != "" {
		// Views by Date
		views, err := api.ViewService.ReadViewsByDateAsJSON(c, date, reverse, limit, offset)
		if err != nil {
			e, _, _ := api.EventService.Create(c, event.Event{
				UserID:     contextUserID(c),
				EntityType: "View",
				LogLevel:   event.ERROR,
				Message:    fmt.Errorf("read views by date %s: %w", date, err).Error(),
				URI:        c.Request.URL.String(),
				Err:        err,
			})
			abortWithError(c, http.StatusInternalServerError, e)
			return
		}
		c.Data(http.StatusOK, "application/json;charset=UTF-8", views)
	} else if deviceID != "" {
		// Views by Device ID
		views, err := api.ViewService.ReadViewsByDeviceIDAsJSON(c, deviceID, reverse, limit, offset)
		if err != nil {
			e, _, _ := api.EventService.Create(c, event.Event{
				UserID:     contextUserID(c),
				EntityType: "View",
				LogLevel:   event.ERROR,
				Message:    fmt.Errorf("read views by Device ID %s: %w", deviceID, err).Error(),
				URI:        c.Request.URL.String(),
				Err:        err,
			})
			abortWithError(c, http.StatusInternalServerError, e)
			return
		}
		c.Data(http.StatusOK, "application/json;charset=UTF-8", views)
	} else {
		// All Views, paginated, and read in parallel (slow)
		views := api.ViewService.ReadViews(c, reverse, limit, offset)
		c.JSON(http.StatusOK, views)
	}
}

// readView returns the specified View.
//
// @Summary Get View
// @Description Get View by ID.
// @Tags View
// @Produce json
// @Param id path string true "View ID"
// @Success 200 {object} view.View "View"
// @Failure 400 {object} APIEvent "Bad Request (invalid path parameter ID)"
// @Failure 404 {object} APIEvent "Not Found"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/views/{id} [get]
func readView(c *gin.Context) {
	// Validate the path parameter ID
	id := c.Param("id")
	if !tuid.IsValid(tuid.TUID(id)) {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid path parameter ID: %s", id))
		return
	}
	// Read and return the specified View
	json, err := api.ViewService.ReadAsJSON(c, id)
	if err != nil && errors.Is(err, v.ErrNotFound) {
		abortWithError(c, http.StatusNotFound, fmt.Errorf("not found: view %s", id))
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   id,
			EntityType: "View",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read View %s: %w", id, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.Data(http.StatusOK, "application/json;charset=UTF-8", json)
}

// existsView checks if the specified View exists.
//
// @Summary View Exists
// @Description Check if the specified View exists.
// @Tags View
// @Param id path string true "View ID"
// @Success 204 "View Exists"
// @Failure 400 "Bad Request (invalid path parameter ID)"
// @Failure 404 "Not Found"
// @Router /v1/views/{id} [head]
func existsView(c *gin.Context) {
	id := c.Param("id")
	if !tuid.IsValid(tuid.TUID(id)) {
		c.Status(http.StatusBadRequest)
	} else if !api.ViewService.Exists(c, id) {
		c.Status(http.StatusNotFound)
	} else {
		c.Status(http.StatusNoContent)
	}
}

// readViewDates returns a list of dates for which views exist.
// It's useful for paging through views by date.
//
// @Summary Get View Dates
// @Description Get a list of dates for which views exist.
// @Tags View
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Success 200 {array} string "View Dates"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/view_dates [get]
func readViewDates(c *gin.Context) {
	dates, err := api.ViewService.ReadAllDates(c)
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: "View",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read View dates: %w", err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.JSON(http.StatusOK, dates)
}

// readViewDeviceIDs returns a list of Device IDs for which views exist.
// It's useful for paging through views by Device ID.
//
// @Summary Get View Device IDs
// @Description Get a list of Device IDs for which views exist.
// @Tags View
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param reverse query bool false "Reverse Order (default: false)"
// @Param limit query int false "Limit (default: 100)"
// @Param offset query string false "Offset (default: forward/reverse alphanumeric)"
// @Success 200 {array} string "Device IDs"
// @Failure 400 {object} APIEvent "Bad Request (invalid parameter)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/view_device_ids [get]
func readViewDeviceIDs(c *gin.Context) {
	// Parse query parameters, with defaults
	reverse, limit, offset, err := paginationParams(c, false, 100)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}
	// Read and return the View Device IDs
	ids, err := api.ViewService.ReadDeviceIDs(c, reverse, limit, offset)
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: "View",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read View Device IDs: %w", err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.JSON(http.StatusOK, ids)
}

// readViewCounts returns a paginated list of view counts by date.
//
// @Summary Get View Counts
// @Description Get a paginated list of view counts by date.
// @Tags View
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param reverse query bool false "Reverse Order (default: false)"
// @Param limit query int false "Limit (default: 100)"
// @Param offset query string false "Offset (default: forward/reverse alphanumeric)"
// @Success 200 {array} view.Count "View Counts"
// @Failure 400 {object} APIEvent "Bad Request (invalid parameter)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/view_counts [get]
func readViewCounts(c *gin.Context) {
	// Parse query parameters, with defaults
	reverse, limit, offset, err := paginationParams(c, false, 100)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}
	// Read and return the View Counts by Date
	counts, err := api.ViewCountService.ReadCountsAsJSON(c, reverse, limit, offset)
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: "View",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read View Counts by Date: %w", err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.Data(http.StatusOK, "application/json", counts)
}

// readViewCount returns the number of views encountered on the specified date.
//
// @Summary Get View Count
// @Description Get the number of views encountered on the specified date.
// @Tags View
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param date path string true "Date (YYYY-MM-DD)"
// @Success 200 {object} view.Count "View Count"
// @Failure 400 {object} APIEvent "Bad Request (invalid path parameter date)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 404 {object} APIEvent "Not Found"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/view_counts/{date} [get]
func readViewCount(c *gin.Context) {
	// Validate the path parameter date (YYYY-MM-DD)
	date := c.Param("date")
	if !view.IsValidDate(date) {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid path parameter date: %s", date))
		return
	}
	// Read and return the View Count
	count, err := api.ViewCountService.ReadAsJSON(c, date)
	if err != nil && errors.Is(err, v.ErrNotFound) {
		abortWithError(c, http.StatusNotFound, fmt.Errorf("not found: ViewCount %s", date))
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: "ViewCount",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read ViewCount %s: %w", date, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.Data(http.StatusOK, "application/json;charset=UTF-8", count)
}

// existsViewCount checks if the specified View Count exists.
//
// @Summary View Count Exists
// @Description Check if the specified View Count exists.
// @Tags View
// @Param id path string true "Date (YYYY-MM-DD)"
// @Success 204 "View Exists"
// @Failure 400 "Bad Request (invalid path parameter date)"
// @Failure 404 "Not Found"
// @Router /v1/view_counts/{date} [head]
func existsViewCount(c *gin.Context) {
	date := c.Param("date")
	if !view.IsValidDate(date) {
		c.Status(http.StatusBadRequest)
	} else if !api.ViewCountService.Exists(c, date) {
		c.Status(http.StatusNotFound)
	} else {
		c.Status(http.StatusNoContent)
	}
}

// updateViewCount updates the number of views encountered on the specified date.
//
// @Summary Update View Count
// @Description Update the number of views encountered on the specified date.
// @Tags View
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param date path string true "Date (YYYY-MM-DD)"
// @Success 200 {object} view.Count "View Count"
// @Failure 400 {object} APIEvent "Bad Request (invalid path parameter date)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/view_counts/{date} [put]
func updateViewCount(c *gin.Context) {
	// Validate the path parameter date (YYYY-MM-DD)
	date := c.Param("date")
	if !view.IsValidDate(date) {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid path parameter date: %s", date))
		return
	}
	// Count views on the specified date
	var count view.Count
	count, err := api.ViewService.CountViewsByDate(c, date)
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: "View",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("count views on date %s: %w", date, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	// Update the View Count
	count, problems, err := api.ViewCountService.Write(c, count)
	if len(problems) > 0 && err != nil {
		abortWithError(c, http.StatusUnprocessableEntity, fmt.Errorf("unprocessable entity: %w", err))
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: "ViewCount",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("update ViewCount %s: %w", date, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	// Log the creation
	_, _, _ = api.EventService.Create(c, event.Event{
		UserID:     contextUserID(c),
		EntityType: count.Type(),
		LogLevel:   event.INFO,
		Message:    fmt.Sprintf("created %s %s", count.Type(), count.Date),
		URI:        c.Request.URL.String(),
	})
	// Return the View Count
	c.JSON(http.StatusOK, count)
}
