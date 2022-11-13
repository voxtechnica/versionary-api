package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/voxtechnica/tuid-go"
	v "github.com/voxtechnica/versionary"

	"versionary-api/pkg/device"
	"versionary-api/pkg/event"
)

// registerDeviceRoutes initializes the Device routes.
func registerDeviceRoutes(r *gin.Engine) {
	r.POST("/v1/devices", createDevice)
	r.PUT("/v1/devices/:id", updateDevice)
	r.DELETE("/v1/devices/:id", roleAuthorizer("admin"), deleteDevice)
	r.GET("/v1/devices", roleAuthorizer("admin"), readDevices)
	r.GET("/v1/devices/:id", readDevice)
	r.HEAD("/v1/devices/:id", existsDevice)
	r.GET("/v1/devices/:id/versions", roleAuthorizer("admin"), readDeviceVersions)
	r.GET("/v1/devices/:id/versions/:versionid", readDeviceVersion)
	r.HEAD("/v1/devices/:id/versions/:versionid", existsDeviceVersion)
	r.GET("/v1/device_dates", roleAuthorizer("admin"), readDeviceDates)
	r.GET("/v1/device_user_ids", roleAuthorizer("admin"), readDeviceUserIDs)
	r.GET("/v1/device_counts", roleAuthorizer("admin"), readDeviceCounts)
	r.GET("/v1/device_counts/:date", roleAuthorizer("admin"), readDeviceCount)
	r.HEAD("/v1/device_counts/:date", roleAuthorizer("admin"), existsDeviceCount)
	r.PUT("/v1/device_counts/:date", roleAuthorizer("admin"), updateDeviceCount)
}

// createDevice creates a new Device.
//
// @Description Create a new Device
// @Description Create a new Device from a User-Agent header.
// @Tags Device
// @Accept json
// @Produce json
// @Param user-agent header string true "User-Agent Header"
// @Success 201 {object} device.Device "Newly-created Device"
// @Failure 422 {object} APIEvent "Device validation errors"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Header 201 {string} Location "/v1/devices/{id}"
// @Router /v1/devices [post]
func createDevice(c *gin.Context) {
	// Create a new Device
	d, problems, err := api.DeviceService.Create(c, c.GetHeader("User-Agent"), contextUserID(c))
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
			Message:    fmt.Errorf("create Device %s: %w", d.ID, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	// Log the creation
	_, _, _ = api.EventService.Create(c, event.Event{
		UserID:     contextUserID(c),
		EntityID:   d.ID,
		EntityType: d.Type(),
		LogLevel:   event.INFO,
		Message:    fmt.Sprintf("created Device %s", d.ID),
		URI:        c.Request.URL.String(),
	})
	// Return the new Device
	c.Header("Location", c.Request.URL.String()+"/"+d.ID)
	c.JSON(http.StatusCreated, d)
}

// updateDevice updates and returns the specified Device.
// Note that if the Device does not exist (e.g. TTL expired), a new Device will be created.
//
// @Description Update Device
// @Description Update the specified Device from a User-Agent header.
// @Tags Device
// @Accept json
// @Produce json
// @Param user-agent header string true "User-Agent Header"
// @Param id path string true "Device ID"
// @Success 200 {object} device.Device "Updated Device"
// @Success 201 {object} device.Device "Newly-created Device (old Device TTL expired)"
// @Failure 400 {object} APIEvent "Bad Request (invalid path parameter ID)"
// @Failure 422 {object} APIEvent "Device validation errors"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Header 201 {string} Location "/v1/devices/{id}"
// @Router /v1/devices/{id} [put]
func updateDevice(c *gin.Context) {
	// Validate the path parameter ID
	id := c.Param("id")
	if !tuid.IsValid(tuid.TUID(id)) {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid path parameter ID: %s", id))
		return
	}
	// Update the specified Device
	d, problems, err := api.DeviceService.Update(c, id, c.GetHeader("User-Agent"), contextUserID(c))
	if len(problems) > 0 && err != nil {
		abortWithError(c, http.StatusUnprocessableEntity, fmt.Errorf("unprocessable entity: %w", err))
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   id,
			EntityType: "Device",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("update Device %s: %w", id, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	// Log the update
	_, _, _ = api.EventService.Create(c, event.Event{
		UserID:     contextUserID(c),
		EntityID:   d.ID,
		EntityType: d.Type(),
		LogLevel:   event.INFO,
		Message:    fmt.Sprintf("updated Device %s", d.ID),
		URI:        c.Request.URL.String(),
	})
	// Return the updated Device
	if d.ID == id {
		c.JSON(http.StatusOK, d)
	} else {
		c.Header("Location", c.Request.URL.String()+"/"+d.ID)
		c.JSON(http.StatusCreated, d)
	}
}

// deleteDevice deletes the specified Device.
//
// @Description Delete Device
// @Description Delete and return the specified Device.
// @Tags Device
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param id path string true "Device ID"
// @Success 200 {object} device.Device "Device that was deleted"
// @Failure 400 {object} APIEvent "Bad Request (invalid path parameter ID)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 404 {object} APIEvent "Not Found"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/devices/{id} [delete]
func deleteDevice(c *gin.Context) {
	// Validate the path parameter ID
	id := c.Param("id")
	if !tuid.IsValid(tuid.TUID(id)) {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid path parameter ID: %s", id))
		return
	}
	// Delete the specified Device
	d, err := api.DeviceService.Delete(c, id)
	if err != nil && errors.Is(err, v.ErrNotFound) {
		abortWithError(c, http.StatusNotFound, fmt.Errorf("not found: device %s", id))
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   id,
			EntityType: "Device",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("delete Device %s: %w", id, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	// Log the deletion
	_, _, _ = api.EventService.Create(c, event.Event{
		UserID:     contextUserID(c),
		EntityID:   d.ID,
		EntityType: d.Type(),
		LogLevel:   event.INFO,
		Message:    "deleted Device " + d.ID,
		URI:        c.Request.URL.String(),
	})
	// Return the deleted device
	c.JSON(http.StatusOK, d)
}

// readDevices returns a paginated list of Devices.
//
// @Description List Devices
// @Description List Devices, paging with reverse, limit, and offset. Optionally, filter by UserID or Date.
// @Tags Device
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param user query string false "User ID (optional, TUID)"
// @Param date query string false "Date (optional, YYYY-MM-DD)"
// @Param reverse query bool false "Reverse Order (default: false)"
// @Param limit query int false "Limit (default: 100)"
// @Param offset query string false "Offset (default: forward/reverse alphanumeric)"
// @Success 200 {array} device.Device "Devices"
// @Failure 400 {object} APIEvent "Bad Request (invalid parameter)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/devices [get]
func readDevices(c *gin.Context) {
	// Parse query parameters, with defaults
	reverse, limit, offset, err := paginationParams(c, false, 100)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}
	date := c.Query("date")
	if date != "" && !device.IsValidDate(date) {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid date: %s", date))
		return
	}
	userID := c.Query("user")
	if userID != "" && !tuid.IsValid(tuid.TUID(userID)) {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid user id: %s", userID))
		return
	}
	// Read and return paginated Devices
	if date != "" {
		// Devices by Date
		devices, err := api.DeviceService.ReadDevicesByDateAsJSON(c, date, reverse, limit, offset)
		if err != nil {
			e, _, _ := api.EventService.Create(c, event.Event{
				UserID:     contextUserID(c),
				EntityType: "Device",
				LogLevel:   event.ERROR,
				Message:    fmt.Errorf("read devices by date %s: %w", date, err).Error(),
				URI:        c.Request.URL.String(),
				Err:        err,
			})
			abortWithError(c, http.StatusInternalServerError, e)
			return
		}
		c.Data(http.StatusOK, "application/json;charset=UTF-8", devices)
	} else if userID != "" {
		// Devices by User ID
		devices, err := api.DeviceService.ReadDevicesByUserIDAsJSON(c, userID, reverse, limit, offset)
		if err != nil {
			e, _, _ := api.EventService.Create(c, event.Event{
				UserID:     contextUserID(c),
				EntityType: "Device",
				LogLevel:   event.ERROR,
				Message:    fmt.Errorf("read devices by user id %s: %w", userID, err).Error(),
				URI:        c.Request.URL.String(),
				Err:        err,
			})
			abortWithError(c, http.StatusInternalServerError, e)
			return
		}
		c.Data(http.StatusOK, "application/json;charset=UTF-8", devices)
	} else {
		// All Devices, paginated, and read in parallel (slow)
		devices := api.DeviceService.ReadDevices(c, reverse, limit, offset)
		c.JSON(http.StatusOK, devices)
	}
}

// readDevice returns the current version of the specified Device.
//
// @Description Get Device
// @Description Get Device by ID.
// @Tags Device
// @Produce json
// @Param id path string true "Device ID"
// @Success 200 {object} device.Device "Device"
// @Failure 400 {object} APIEvent "Bad Request (invalid path parameter ID)"
// @Failure 404 {object} APIEvent "Not Found"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/devices/{id} [get]
func readDevice(c *gin.Context) {
	// Validate the path parameter ID
	id := c.Param("id")
	if !tuid.IsValid(tuid.TUID(id)) {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid path parameter ID: %s", id))
		return
	}
	// Read and return the specified Device
	d, err := api.DeviceService.ReadAsJSON(c, id)
	if err != nil && errors.Is(err, v.ErrNotFound) {
		abortWithError(c, http.StatusNotFound, fmt.Errorf("not found: device %s", id))
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   id,
			EntityType: "Device",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read device %s: %w", id, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.Data(http.StatusOK, "application/json;charset=UTF-8", d)
}

// existsDevice checks if the specified Device exists.
//
// @Description Device Exists
// @Description Check if the specified Device exists.
// @Tags Device
// @Param id path string true "Device ID"
// @Success 204 "Device Exists"
// @Failure 400 "Bad Request (invalid path parameter ID)"
// @Failure 404 "Not Found"
// @Router /v1/devices/{id} [head]
func existsDevice(c *gin.Context) {
	id := c.Param("id")
	if !tuid.IsValid(tuid.TUID(id)) {
		c.Status(http.StatusBadRequest)
	} else if !api.DeviceService.Exists(c, id) {
		c.Status(http.StatusNotFound)
	} else {
		c.Status(http.StatusNoContent)
	}
}

// readDeviceVersions returns a paginated list of versions of the specified Device.
//
// @Description Get Device Versions
// @Description Get Device Versions by ID, paging with reverse, limit, and offset.
// @Tags Device
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param id path string true "Device ID"
// @Param reverse query bool false "Reverse Order (default: false)"
// @Param limit query int false "Limit (default: 100)"
// @Param offset query string false "Offset (default: forward/reverse alphanumeric)"
// @Success 200 {array} device.Device "Device Versions"
// @Failure 400 {object} APIEvent "Bad Request (invalid parameter)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 404 {object} APIEvent "Not Found"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/devices/{id}/versions [get]
func readDeviceVersions(c *gin.Context) {
	// Validate parameters
	id := c.Param("id")
	if !tuid.IsValid(tuid.TUID(id)) {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid path parameter ID: %s", id))
		return
	}
	reverse, limit, offset, err := paginationParams(c, false, 100)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}
	// Verify that the Device exists
	if !api.DeviceService.Exists(c, id) {
		abortWithError(c, http.StatusNotFound, fmt.Errorf("not found: device %s", id))
		return
	}
	// Read and return the specified Device Versions
	versions, err := api.DeviceService.ReadVersionsAsJSON(c, id, reverse, limit, offset)
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   id,
			EntityType: "Device",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read device %s versions: %w", id, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.Data(http.StatusOK, "application/json;charset=UTF-8", versions)
}

// readDeviceVersion returns the specified version of the specified Device.
//
// @Description Get Device Version
// @Description Get Device Version by ID and VersionID.
// @Tags Device
// @Produce json
// @Param id path string true "Device ID"
// @Param versionid path string true "Device VersionID"
// @Success 200 {object} device.Device "Device Version"
// @Failure 400 {object} APIEvent "Bad Request (invalid path parameter)"
// @Failure 404 {object} APIEvent "Not Found"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/devices/{id}/versions/{versionid} [get]
func readDeviceVersion(c *gin.Context) {
	// Validate the path parameters
	id := c.Param("id")
	if !tuid.IsValid(tuid.TUID(id)) {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid path parameter ID: %s", id))
		return
	}
	versionid := c.Param("versionid")
	if !tuid.IsValid(tuid.TUID(versionid)) {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid path parameter VersionID: %s", versionid))
		return
	}
	// Read and return the Device Version
	version, err := api.DeviceService.ReadVersionAsJSON(c, id, versionid)
	if err != nil && errors.Is(err, v.ErrNotFound) {
		abortWithError(c, http.StatusNotFound, fmt.Errorf("not found: device %s version %s", id, versionid))
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   id,
			EntityType: "Device",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read device %s version %s: %w", id, versionid, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.Data(http.StatusOK, "application/json;charset=UTF-8", version)
}

// existsDeviceVersion checks if the specified Device version exists.
//
// @Description Device Version Exists
// @Description Check if the specified Device version exists.
// @Tags Device
// @Param id path string true "Device ID"
// @Param versionid path string true "Device VersionID"
// @Success 204 "Device Version Exists"
// @Failure 400 "Bad Request (invalid path parameter)"
// @Failure 404 "Not Found"
// @Router /v1/devices/{id}/versions/{versionid} [head]
func existsDeviceVersion(c *gin.Context) {
	id := c.Param("id")
	versionid := c.Param("versionid")
	if !tuid.IsValid(tuid.TUID(id)) || !tuid.IsValid(tuid.TUID(versionid)) {
		c.Status(http.StatusBadRequest)
	} else if !api.DeviceService.VersionExists(c, id, versionid) {
		c.Status(http.StatusNotFound)
	} else {
		c.Status(http.StatusNoContent)
	}
}

// readDeviceDates returns a list of dates for which devices exist.
// It's useful for paging through devices by date.
//
// @Description Get Device Dates
// @Description Get a list of dates for which devices exist.
// @Tags Device
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Success 200 {array} string "Device Dates"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/device_dates [get]
func readDeviceDates(c *gin.Context) {
	dates, err := api.DeviceService.ReadAllDates(c)
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: "Device",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read Device dates: %w", err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.JSON(http.StatusOK, dates)
}

// readDeviceUserIDs returns a list of User IDs for which devices exist.
// It's useful for paging through devices by User ID.
//
// @Description Get Device User IDs
// @Description Get a list of User IDs for which devices exist.
// @Tags Device
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param reverse query bool false "Reverse Order (default: false)"
// @Param limit query int false "Limit (default: 100)"
// @Param offset query string false "Offset (default: forward/reverse alphanumeric)"
// @Success 200 {array} string "User IDs"
// @Failure 400 {object} APIEvent "Bad Request (invalid parameter)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/device_user_ids [get]
func readDeviceUserIDs(c *gin.Context) {
	// Parse query parameters, with defaults
	reverse, limit, offset, err := paginationParams(c, false, 100)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}
	// Read and return the Device User IDs
	ids, err := api.DeviceService.ReadUserIDs(c, reverse, limit, offset)
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: "Device",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read Device User IDs: %w", err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.JSON(http.StatusOK, ids)
}

// readDeviceCounts returns a paginated list of device counts by date.
//
// @Description Get Device Counts
// @Description Get a paginated list of device counts by date.
// @Tags Device
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param reverse query bool false "Reverse Order (default: false)"
// @Param limit query int false "Limit (default: 100)"
// @Param offset query string false "Offset (default: forward/reverse alphanumeric)"
// @Success 200 {array} device.Count "Device Counts"
// @Failure 400 {object} APIEvent "Bad Request (invalid parameter)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/device_counts [get]
func readDeviceCounts(c *gin.Context) {
	// Parse query parameters, with defaults
	reverse, limit, offset, err := paginationParams(c, false, 100)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}
	// Read and return the Device Counts by Date
	counts, err := api.DeviceCountService.ReadCountsAsJSON(c, reverse, limit, offset)
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: "Device",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read Device Counts by Date: %w", err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.Data(http.StatusOK, "application/json", counts)
}

// readDeviceCount returns the number of devices encountered on the specified date.
//
// @Description Get Device Count
// @Description Get the number of devices encountered on the specified date.
// @Tags Device
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param date path string true "Date (YYYY-MM-DD)"
// @Success 200 {object} device.Count "Device Count"
// @Failure 400 {object} APIEvent "Bad Request (invalid path parameter date)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 404 {object} APIEvent "Not Found"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/device_counts/{date} [get]
func readDeviceCount(c *gin.Context) {
	// Validate the path parameter date (YYYY-MM-DD)
	date := c.Param("date")
	if !device.IsValidDate(date) {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid path parameter date: %s", date))
		return
	}
	// Read and return the Device Count
	count, err := api.DeviceCountService.ReadAsJSON(c, date)
	if err != nil && errors.Is(err, v.ErrNotFound) {
		abortWithError(c, http.StatusNotFound, fmt.Errorf("not found: DeviceCount %s", date))
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: "DeviceCount",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read DeviceCount %s: %w", date, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.Data(http.StatusOK, "application/json;charset=UTF-8", count)
}

// existsDeviceCount checks if the specified Device Count exists.
//
// @Description Device Count Exists
// @Description Check if the specified Device Count exists.
// @Tags Device
// @Param id path string true "Date (YYYY-MM-DD)"
// @Success 204 "Device Exists"
// @Failure 400 "Bad Request (invalid path parameter date)"
// @Failure 404 "Not Found"
// @Router /v1/device_counts/{date} [head]
func existsDeviceCount(c *gin.Context) {
	date := c.Param("date")
	if !device.IsValidDate(date) {
		c.Status(http.StatusBadRequest)
	} else if !api.DeviceCountService.Exists(c, date) {
		c.Status(http.StatusNotFound)
	} else {
		c.Status(http.StatusNoContent)
	}
}

// updateDeviceCount updates the number of devices encountered on the specified date.
//
// @Description Update Device Count
// @Description Update the number of devices encountered on the specified date.
// @Tags Device
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param date path string true "Date (YYYY-MM-DD)"
// @Success 200 {object} device.Count "Device Count"
// @Failure 400 {object} APIEvent "Bad Request (invalid path parameter date)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/device_counts/{date} [put]
func updateDeviceCount(c *gin.Context) {
	// Validate the path parameter date (YYYY-MM-DD)
	date := c.Param("date")
	if !device.IsValidDate(date) {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid path parameter date: %s", date))
		return
	}
	// Count devices on the specified date
	var count device.Count
	count, err := api.DeviceService.CountDevicesByDate(c, date)
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: "Device",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("count devices on date %s: %w", date, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	// Update the Device Count
	count, problems, err := api.DeviceCountService.Write(c, count)
	if len(problems) > 0 && err != nil {
		abortWithError(c, http.StatusUnprocessableEntity, fmt.Errorf("unprocessable entity: %w", err))
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: "DeviceCount",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("update DeviceCount %s: %w", date, err).Error(),
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
	// Return the Device Count
	c.JSON(http.StatusOK, count)
}
