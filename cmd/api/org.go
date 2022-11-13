package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/voxtechnica/tuid-go"
	v "github.com/voxtechnica/versionary"

	"versionary-api/pkg/event"
	"versionary-api/pkg/org"
)

// registerOrganizationRoutes initializes the Organization routes.
func registerOrganizationRoutes(r *gin.Engine) {
	r.POST("/v1/organizations", roleAuthorizer("admin"), createOrganization)
	r.GET("/v1/organizations", roleAuthorizer("admin"), readOrganizations)
	r.GET("/v1/organizations/:id", readOrganization)
	r.HEAD("/v1/organizations/:id", existsOrganization)
	r.GET("/v1/organizations/:id/versions", roleAuthorizer("admin"), readOrganizationVersions)
	r.GET("/v1/organizations/:id/versions/:versionid", readOrganizationVersion)
	r.HEAD("/v1/organizations/:id/versions/:versionid", existsOrganizationVersion)
	r.PUT("/v1/organizations/:id", roleAuthorizer("admin"), updateOrganization)
	r.DELETE("/v1/organizations/:id", roleAuthorizer("admin"), deleteOrganization)
	r.GET("/v1/organization_statuses", roleAuthorizer("admin"), readOrganizationStatuses)
}

// createOrganization creates a new Organization.
//
// @Description Create a new Organization
// @Description Create a new Organization.
// @Tags Organization
// @Accept json
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param organization body org.Organization true "Organization"
// @Success 201 {object} org.Organization "Newly-created Organization"
// @Failure 400 {object} APIEvent "Bad Request (invalid JSON body)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 422 {object} APIEvent "Organization validation errors"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Header 201 {string} Location "URL of the newly created Organization"
// @Router /v1/organizations [post]
func createOrganization(c *gin.Context) {
	// Parse the request body as an Organization
	var body org.Organization
	if err := c.ShouldBindJSON(&body); err != nil {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid JSON body: %w", err))
		return
	}
	// Create a new Organization
	o, problems, err := api.OrgService.Create(c, body)
	if len(problems) > 0 && err != nil {
		abortWithError(c, http.StatusUnprocessableEntity, fmt.Errorf("unprocessable entity: %w", err))
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   o.ID,
			EntityType: o.Type(),
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("create organization %s %s: %w", o.ID, o.Name, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	// Log the creation
	_, _, _ = api.EventService.Create(c, event.Event{
		UserID:     contextUserID(c),
		EntityID:   o.ID,
		EntityType: o.Type(),
		LogLevel:   event.INFO,
		Message:    fmt.Sprintf("created Organization %s %s", o.ID, o.Name),
		URI:        c.Request.URL.String(),
	})
	// Return the new Organization
	c.Header("Location", c.Request.URL.String()+"/"+o.ID)
	c.JSON(http.StatusCreated, o)
}

// readOrganizations returns a paginated list of Organizations.
//
// @Description List Organizations
// @Description List Organizations, paging with reverse, limit, and offset. Optionally, filter by status.
// @Tags Organization
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param status query string false "Status" Enums(PENDING, ENABLED, DISABLED)
// @Param reverse query bool false "Reverse Order (default: false)"
// @Param limit query int false "Limit (default: 100)"
// @Param offset query string false "Offset (default: forward/reverse alphanumeric)"
// @Success 200 {array} org.Organization "Organizations"
// @Failure 400 {object} APIEvent "Bad Request (invalid parameter)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/organizations [get]
func readOrganizations(c *gin.Context) {
	// Parse query parameters, with defaults
	reverse, limit, offset, err := paginationParams(c, false, 100)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}
	status := strings.ToUpper(c.Query("status"))
	if status != "" && !org.Status(status).IsValid() {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid status: %s", status))
		return
	}
	// Read and return paginated Organizations
	if status != "" {
		orgs, err := api.OrgService.ReadOrganizationsByStatusAsJSON(c, status, reverse, limit, offset)
		if err != nil {
			e, _, _ := api.EventService.Create(c, event.Event{
				UserID:     contextUserID(c),
				EntityType: "Organization",
				LogLevel:   event.ERROR,
				Message:    fmt.Errorf("read organizations by status %s: %w", status, err).Error(),
				URI:        c.Request.URL.String(),
				Err:        err,
			})
			abortWithError(c, http.StatusInternalServerError, e)
			return
		}
		c.Data(http.StatusOK, "application/json;charset=UTF-8", orgs)
	} else {
		orgs := api.OrgService.ReadOrganizations(c, reverse, limit, offset)
		c.JSON(http.StatusOK, orgs)
	}
}

// readOrganization returns the current version of the specified Organization.
//
// @Description Get Organization
// @Description Get Organization by ID.
// @Tags Organization
// @Produce json
// @Param id path string true "Organization ID"
// @Success 200 {object} org.Organization "Organization"
// @Failure 400 {object} APIEvent "Bad Request (invalid path parameter ID)"
// @Failure 404 {object} APIEvent "Not Found"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/organizations/{id} [get]
func readOrganization(c *gin.Context) {
	// Validate the path parameter ID
	id := c.Param("id")
	if !tuid.IsValid(tuid.TUID(id)) {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid path parameter ID: %s", id))
		return
	}
	// Read and return the specified Organization
	o, err := api.OrgService.ReadAsJSON(c, id)
	if err != nil && errors.Is(err, v.ErrNotFound) {
		abortWithError(c, http.StatusNotFound, fmt.Errorf("not found: organization %s", id))
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   id,
			EntityType: "Organization",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read organization %s: %w", id, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.Data(http.StatusOK, "application/json;charset=UTF-8", o)
}

// existsOrganization checks if the specified Organization exists.
//
// @Description Organization Exists
// @Description Check if the specified Organization exists.
// @Tags Organization
// @Param id path string true "Organization ID"
// @Success 204 "Organization Exists"
// @Failure 400 "Bad Request (invalid path parameter ID)"
// @Failure 404 "Not Found"
// @Router /v1/organizations/{id} [head]
func existsOrganization(c *gin.Context) {
	id := c.Param("id")
	if !tuid.IsValid(tuid.TUID(id)) {
		c.Status(http.StatusBadRequest)
	} else if !api.OrgService.Exists(c, id) {
		c.Status(http.StatusNotFound)
	} else {
		c.Status(http.StatusNoContent)
	}
}

// readOrganizationVersions returns a paginated list of versions of the specified Organization.
//
// @Description Get Organization Versions
// @Description Get Organization Versions by ID, paging with reverse, limit, and offset.
// @Tags Organization
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param id path string true "Organization ID"
// @Param reverse query bool false "Reverse Order (default: false)"
// @Param limit query int false "Limit (default: 100)"
// @Param offset query string false "Offset (default: forward/reverse alphanumeric)"
// @Success 200 {array} org.Organization "Organization Versions"
// @Failure 400 {object} APIEvent "Bad Request (invalid parameter)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 404 {object} APIEvent "Not Found"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/organizations/{id}/versions [get]
func readOrganizationVersions(c *gin.Context) {
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
	// Verify that the Organization exists
	if !api.OrgService.Exists(c, id) {
		abortWithError(c, http.StatusNotFound, fmt.Errorf("not found: organization %s", id))
		return
	}
	// Read and return the specified Organization Versions
	versions, err := api.OrgService.ReadVersionsAsJSON(c, id, reverse, limit, offset)
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   id,
			EntityType: "Organization",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read organization %s versions: %w", id, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.Data(http.StatusOK, "application/json;charset=UTF-8", versions)
}

// readOrganizationVersion returns the specified version of the specified Organization.
//
// @Description Get Organization Version
// @Description Get Organization Version by ID and VersionID.
// @Tags Organization
// @Produce json
// @Param id path string true "Organization ID"
// @Param versionid path string true "Organization VersionID"
// @Success 200 {object} org.Organization "Organization Version"
// @Failure 400 {object} APIEvent "Bad Request (invalid path parameter)"
// @Failure 404 {object} APIEvent "Not Found"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/organizations/{id}/versions/{versionid} [get]
func readOrganizationVersion(c *gin.Context) {
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
	// Read and return the Organization Version
	version, err := api.OrgService.ReadVersionAsJSON(c, id, versionid)
	if err != nil && errors.Is(err, v.ErrNotFound) {
		abortWithError(c, http.StatusNotFound, fmt.Errorf("not found: organization %s version %s", id, versionid))
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   id,
			EntityType: "Organization",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read organization %s version %s: %w", id, versionid, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.Data(http.StatusOK, "application/json;charset=UTF-8", version)
}

// existsOrganizationVersion checks if the specified Organization version exists.
//
// @Description Organization Version Exists
// @Description Check if the specified Organization version exists.
// @Tags Organization
// @Param id path string true "Organization ID"
// @Param versionid path string true "Organization VersionID"
// @Success 204 "Organization Version Exists"
// @Failure 400 "Bad Request (invalid path parameter)"
// @Failure 404 "Not Found"
// @Router /v1/organizations/{id}/versions/{versionid} [head]
func existsOrganizationVersion(c *gin.Context) {
	id := c.Param("id")
	versionid := c.Param("versionid")
	if !tuid.IsValid(tuid.TUID(id)) || !tuid.IsValid(tuid.TUID(versionid)) {
		c.Status(http.StatusBadRequest)
	} else if !api.OrgService.VersionExists(c, id, versionid) {
		c.Status(http.StatusNotFound)
	} else {
		c.Status(http.StatusNoContent)
	}
}

// updateOrganization updates and returns the specified Organization.
// Note that the updated version needs to be complete; this is not a partial update (e.g. PATCH).
//
// @Description Update Organization
// @Description Update the provided, complete Organization.
// @Tags Organization
// @Accept json
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param organization body org.Organization true "Organization"
// @Param id path string true "Organization ID"
// @Success 200 {object} org.Organization "Organization"
// @Failure 400 {object} APIEvent "Bad Request (invalid JSON or parameter)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 422 {object} APIEvent "Organization validation errors"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/organizations/{id} [put]
func updateOrganization(c *gin.Context) {
	// Parse the request body as an Organization
	var body org.Organization
	if err := c.ShouldBindJSON(&body); err != nil {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid JSON body: %w", err))
		return
	}
	// Validate the path parameter ID
	id := c.Param("id")
	if !tuid.IsValid(tuid.TUID(id)) {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid path parameter ID: %s", id))
		return
	}
	// The path parameter ID must match the Organization ID
	if body.ID != id {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: path parameter ID %s does not match Organization ID %s", id, body.ID))
		return
	}
	// Update the specified Organization
	o, problems, err := api.OrgService.Update(c, body)
	if len(problems) > 0 && err != nil {
		abortWithError(c, http.StatusUnprocessableEntity, fmt.Errorf("unprocessable entity: %w", err))
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   o.ID,
			EntityType: o.Type(),
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("update organization %s %s: %w", o.ID, o.Name, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	// Log the update
	_, _, _ = api.EventService.Create(c, event.Event{
		UserID:     contextUserID(c),
		EntityID:   o.ID,
		EntityType: o.Type(),
		LogLevel:   event.INFO,
		Message:    fmt.Sprintf("updated Organization %s %s", o.ID, o.Name),
		URI:        c.Request.URL.String(),
	})
	// Return the updated Organization
	c.JSON(http.StatusOK, o)
}

// deleteOrganization deletes the specified Organization.
//
// @Description Delete Organization
// @Description Delete and return the specified Organization.
// @Tags Organization
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param id path string true "Organization ID"
// @Success 200 {object} org.Organization "Organization that was deleted"
// @Failure 400 {object} APIEvent "Bad Request (invalid path parameter ID)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 404 {object} APIEvent "Not Found"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/organizations/{id} [delete]
func deleteOrganization(c *gin.Context) {
	// Validate the path parameter ID
	id := c.Param("id")
	if !tuid.IsValid(tuid.TUID(id)) {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid path parameter ID: %s", id))
		return
	}
	// Delete the specified Organization
	o, err := api.OrgService.Delete(c, id)
	if err != nil && errors.Is(err, v.ErrNotFound) {
		abortWithError(c, http.StatusNotFound, fmt.Errorf("not found: organization %s", id))
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   id,
			EntityType: "Organization",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("delete organization %s: %w", id, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	// Log the deletion
	_, _, _ = api.EventService.Create(c, event.Event{
		UserID:     contextUserID(c),
		EntityID:   o.ID,
		EntityType: o.Type(),
		LogLevel:   event.INFO,
		Message:    fmt.Sprintf("deleted Organization %s %s", o.ID, o.Name),
		URI:        c.Request.URL.String(),
	})
	// Return the deleted organization
	c.JSON(http.StatusOK, o)
}

// readOrganizationStatuses returns a list of status codes for which organizations exist.
// It's useful for paging through organizations by status.
//
// @Description Get Organization Statuses
// @Description Get a list of status codes for which organizations exist.
// @Tags Organization
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Success 200 {array} string "Organization Statuses"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/organization_statuses [get]
func readOrganizationStatuses(c *gin.Context) {
	statuses, err := api.OrgService.ReadAllStatuses(c)
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: "Organization",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read organization statuses: %w", err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.JSON(http.StatusOK, statuses)
}
