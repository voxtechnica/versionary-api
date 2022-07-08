package main

import (
	"fmt"
	"github.com/voxtechnica/tuid-go"
	v "github.com/voxtechnica/versionary"
	"net/http"
	"strconv"
	"strings"
	"versionary-api/pkg/event"
	"versionary-api/pkg/user"

	"github.com/gin-gonic/gin"
)

// initOrganizationRoutes initializes the Organization routes.
func initOrganizationRoutes(r *gin.Engine) {
	r.POST("/v1/organizations", roleAuthorizer("admin"), createOrganization)
	r.GET("/v1/organizations", roleAuthorizer("admin"), readOrganizations)
	r.GET("/v1/organizations/:id", readOrganization)
	r.DELETE("/v1/organizations/:id", roleAuthorizer("admin"), deleteOrganization)
	r.GET("/v1/organization_statuses", roleAuthorizer("admin"), readOrganizationStatuses)
}

// createOrganization creates a new Organization.
func createOrganization(c *gin.Context) {
	// Parse the request body as an Organization
	var org user.Organization
	if err := c.ShouldBindJSON(&org); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":  http.StatusBadRequest,
			"error": fmt.Errorf("create organization: JSON binding error: %w", err).Error(),
		})
		return
	}
	// Create a new Organization
	o, problems, err := api.OrgService.Create(c, org)
	if len(problems) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":  http.StatusBadRequest,
			"error": err.Error(),
		})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":  http.StatusInternalServerError,
			"error": fmt.Errorf("create organization %s error: %w", org.Name, err).Error(),
		})
		return
	}
	// Log the creation (best effort)
	_, _ = api.EventService.Create(c, event.Event{
		UserID:     contextUserID(c),
		EntityID:   o.ID,
		EntityType: o.Type(),
		LogLevel:   event.INFO,
		Message:    fmt.Sprintf("created Organization %s %s", o.ID, o.Name),
		URI:        c.Request.URL.String(),
	})
	// Return the new Organization
	c.JSON(http.StatusOK, o)
}

// readOrganizations returns a paginated list of Organizations for the specified User.
func readOrganizations(c *gin.Context) {
	// Parse query parameters, with defaults
	reverse, err := strconv.ParseBool(c.DefaultQuery("reverse", "false"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":  http.StatusBadRequest,
			"error": fmt.Errorf("read organization user IDs: invalid reverse: %w", err).Error(),
		})
		return
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "1000"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":  http.StatusBadRequest,
			"error": fmt.Errorf("read organization user IDs: invalid limit: %w", err).Error(),
		})
		return
	}
	offset := c.Query("offset")
	if offset == "" {
		if reverse {
			offset = tuid.MaxID
		} else {
			offset = tuid.MinID
		}
	}
	status := strings.ToUpper(c.Query("status"))
	// Read paginated Organizations
	var orgs []user.Organization
	if status == "" {
		orgs = api.OrgService.ReadOrganizations(c, reverse, limit, offset)
	} else {
		orgs, err = api.OrgService.ReadOrganizationsByStatus(c, status, reverse, limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":  http.StatusInternalServerError,
				"error": fmt.Errorf("read organizations by status %s: %w", status, err).Error(),
			})
			return
		}
	}
	// Return the organizations
	c.JSON(http.StatusOK, orgs)
}

// readOrganization returns the current version of the specified Organization.
func readOrganization(c *gin.Context) {
	// Read the specified Organization
	id := c.Param("id")
	o, err := api.OrgService.ReadAsJSON(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":  http.StatusNotFound,
			"error": fmt.Errorf("read organization %s: %w", id, err).Error(),
		})
		return
	}
	// Return the organization
	c.Data(http.StatusOK, "application/json;charset=UTF-8", o)
}

// deleteOrganization deletes the specified Organization.
func deleteOrganization(c *gin.Context) {
	id := c.Param("id")
	o, err := api.OrgService.Delete(c, id)
	if err == v.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{
			"code":  http.StatusNotFound,
			"error": fmt.Errorf("delete organization %s: %w", id, err).Error(),
		})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":  http.StatusInternalServerError,
			"error": fmt.Errorf("delete organization %s: %w", id, err).Error(),
		})
		return
	}
	// Log the organization deletion (best effort)
	_, _ = api.EventService.Create(c, event.Event{
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
func readOrganizationStatuses(c *gin.Context) {
	statuses, err := api.OrgService.ReadAllStatuses(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":  http.StatusInternalServerError,
			"error": fmt.Errorf("read organization statuses: %w", err).Error(),
		})
		return
	}
	c.JSON(http.StatusOK, statuses)
}
