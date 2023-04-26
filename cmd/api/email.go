package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/voxtechnica/tuid-go"
	v "github.com/voxtechnica/versionary"

	"versionary-api/pkg/email"
	"versionary-api/pkg/event"
	"versionary-api/pkg/user"
)

// registerEmailRoutes initializes the Email routes.
func registerEmailRoutes(r *gin.Engine) {
	r.POST("/v1/emails", roleAuthorizer("admin"), createEmail)
	r.GET("/v1/emails", userAuthenticator(), readEmails)
	r.GET("/v1/emails/:id", userAuthenticator(), readEmail)
	r.HEAD("/v1/emails/:id", existsEmail)
	r.GET("/v1/emails/:id/versions", roleAuthorizer("admin"), readEmailVersions)
	r.GET("/v1/emails/:id/versions/:versionid", userAuthenticator(), readEmailVersion)
	r.HEAD("/v1/emails/:id/versions/:versionid", existsEmailVersion)
	r.PUT("/v1/emails/:id", roleAuthorizer("admin"), updateEmail)
	r.DELETE("/v1/emails/:id", roleAuthorizer("admin"), deleteEmail)
	r.GET("/v1/email_addresses", roleAuthorizer("admin"), readEmailAddresses)
	r.GET("/v1/email_statuses", roleAuthorizer("admin"), readEmailStatuses)
}

// createEmail creates and sends a new Email message.
//
// @Summary Create Email Message
// @Description Create/Send a new Email
// @Description Create and send a new Email message.
// @Tags Email
// @Accept json
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param email body email.Email true "Email"
// @Success 201 {object} email.Email "Newly-created Email"
// @Failure 400 {object} APIEvent "Bad Request (invalid JSON body)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 422 {object} APIEvent "Email validation errors"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Header 201 {string} Location "URL of the newly created Email"
// @Router /v1/emails [post]
func createEmail(c *gin.Context) {
	// Parse the request body as an Email
	var body email.Email
	if err := c.ShouldBindJSON(&body); err != nil {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid JSON body: %w", err))
		return
	}
	// Create/send a new Email
	e, problems, err := api.EmailService.Create(c, body)
	if len(problems) > 0 && err != nil {
		abortWithError(c, http.StatusUnprocessableEntity, fmt.Errorf("unprocessable entity: %w", err))
		return
	}
	if err != nil {
		evt, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   e.ID,
			EntityType: e.Type(),
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("create email %s: %w", e.ID, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, evt)
		return
	}
	// Log the creation
	_, _, _ = api.EventService.Create(c, event.Event{
		UserID:     contextUserID(c),
		EntityID:   e.ID,
		EntityType: e.Type(),
		LogLevel:   event.INFO,
		Message:    "created Email " + e.ID,
		URI:        c.Request.URL.String(),
	})
	// Return the new Email
	c.Header("Location", c.Request.URL.String()+"/"+e.ID)
	c.JSON(http.StatusCreated, e)
}

// readEmails returns a paginated list of Emails.
//
// @Summary List Email Messages
// @Description List Email Messages
// @Description List Emails, paging with reverse, limit, and offset. Optionally, filter by email address or status.
// @Description Regular users can only list their own Emails. Administrators can list all Emails.
// @Tags Email
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (User or Administrator)"
// @Param address query string false "Email Address" "(default: authenticated user's email address)"
// @Param status query string false "Status" Enums(PENDING, SENT, UNSENT, ERROR)
// @Param reverse query bool false "Reverse Order (default: false)"
// @Param limit query int false "Limit (default: 100)"
// @Param offset query string false "Offset (default: forward/reverse alphanumeric)"
// @Success 200 {array} email.Email "Emails"
// @Failure 400 {object} APIEvent "Bad Request (invalid parameter)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not Owner or Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/emails [get]
func readEmails(c *gin.Context) {
	// Parse query parameters, with defaults
	reverse, limit, offset, err := paginationParams(c, false, 100)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}
	// Read and return paginated Emails by Address (any user)
	address := c.Query("address")
	u, _ := contextUser(c) // the user has already been authenticated
	if !u.HasRole("admin") {
		address = u.Email
		if address == "" {
			abortWithError(c, http.StatusForbidden, fmt.Errorf("forbidden: user %s has no email address", u.ID))
			return
		}
	}
	if address != "" {
		// Standardize the email address
		i, err := email.NewIdentity("", address)
		if err != nil {
			abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: %w", err))
			return
		}
		address = i.Address
		es, err := api.EmailService.ReadEmailsByAddressAsJSON(c, address, reverse, limit, offset)
		if err != nil {
			evt, _, _ := api.EventService.Create(c, event.Event{
				UserID:     contextUserID(c),
				EntityID:   address,
				EntityType: "email",
				LogLevel:   event.ERROR,
				Message:    fmt.Errorf("read emails for %s: %w", address, err).Error(),
				URI:        c.Request.URL.String(),
				Err:        err,
			})
			abortWithError(c, http.StatusInternalServerError, evt)
			return
		}
		c.Data(http.StatusOK, "application/json", es)
		return
	}
	// Read and return paginated Emails by Status (admin only)
	status := strings.ToUpper(c.Query("status"))
	if status != "" && !user.Status(status).IsValid() {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid status: %s", status))
		return
	}
	if status != "" {
		es, err := api.EmailService.ReadEmailsByStatusAsJSON(c, status, reverse, limit, offset)
		if err != nil {
			evt, _, _ := api.EventService.Create(c, event.Event{
				UserID:     contextUserID(c),
				EntityType: "Email",
				LogLevel:   event.ERROR,
				Message:    fmt.Errorf("read emails by status %s: %w", status, err).Error(),
				URI:        c.Request.URL.String(),
				Err:        err,
			})
			abortWithError(c, http.StatusInternalServerError, evt)
			return
		}
		c.Data(http.StatusOK, "application/json;charset=UTF-8", es)
	} else {
		// Read and return paginated Emails (admin only)
		es := api.EmailService.ReadEmails(c, reverse, limit, offset)
		c.JSON(http.StatusOK, es)
	}
}

// readEmail returns the current version of the specified Email.
//
// @Summary Read Email Message
// @Description Get Email Message
// @Description Get Email Message by ID.
// @Tags Email
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (User or Administrator)"
// @Param id path string true "Email ID"
// @Success 200 {object} email.Email "Email"
// @Failure 400 {object} APIEvent "Bad Request (invalid path parameter ID)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Owner or Administrator)"
// @Failure 404 {object} APIEvent "Not Found"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/emails/{id} [get]
func readEmail(c *gin.Context) {
	// Validate the path parameter ID
	id := c.Param("id")
	if !tuid.IsValid(tuid.TUID(id)) {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid path parameter ID: %s", id))
		return
	}
	// Read the specified Email
	e, err := api.EmailService.Read(c, id)
	if err != nil && errors.Is(err, v.ErrNotFound) {
		abortWithError(c, http.StatusNotFound, fmt.Errorf("not found: email %s", id))
		return
	}
	if err != nil {
		evt, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   id,
			EntityType: "Email",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read email %s: %w", id, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, evt)
		return
	}
	// Verify that the user is authorized to read the Email
	u, _ := contextUser(c) // the user has already been authenticated
	if !u.HasRole("admin") && !e.IsParticipant(u.Email) {
		abortWithError(c, http.StatusForbidden, fmt.Errorf("forbidden: email %s", id))
		return
	}
	c.JSON(http.StatusOK, e)
}

// existsEmail checks if the specified Email exists.
//
// @Summary Email Message Exists
// @Description Email Message Exists
// @Description Check if the specified Email exists.
// @Tags Email
// @Param id path string true "Email ID"
// @Success 204 "Email Exists"
// @Failure 400 "Bad Request (invalid path parameter ID)"
// @Failure 404 "Not Found"
// @Router /v1/emails/{id} [head]
func existsEmail(c *gin.Context) {
	id := c.Param("id")
	if !tuid.IsValid(tuid.TUID(id)) {
		c.Status(http.StatusBadRequest)
	} else if !api.EmailService.Exists(c, id) {
		c.Status(http.StatusNotFound)
	} else {
		c.Status(http.StatusNoContent)
	}
}

// readEmailVersions returns a paginated list of versions of the specified Email.
//
// @Summary List Email Versions
// @Description Get Email Versions
// @Description Get Email Versions by ID, paging with reverse, limit, and offset.
// @Tags Email
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param id path string true "Email ID"
// @Param reverse query bool false "Reverse Order (default: false)"
// @Param limit query int false "Limit (default: 100)"
// @Param offset query string false "Offset (default: forward/reverse alphanumeric)"
// @Success 200 {array} email.Email "Email Versions"
// @Failure 400 {object} APIEvent "Bad Request (invalid parameter)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Owner or Administrator)"
// @Failure 404 {object} APIEvent "Not Found"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/emails/{id}/versions [get]
func readEmailVersions(c *gin.Context) {
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
	// Verify that the Email exists
	if !api.EmailService.Exists(c, id) {
		abortWithError(c, http.StatusNotFound, fmt.Errorf("not found: email %s", id))
		return
	}
	// Read and return the specified Email Versions
	versions, err := api.EmailService.ReadVersionsAsJSON(c, id, reverse, limit, offset)
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   id,
			EntityType: "Email",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read email %s versions: %w", id, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.Data(http.StatusOK, "application/json;charset=UTF-8", versions)
}

// readEmailVersion returns the specified version of the specified Email.
//
// @Summary Read Email Version
// @Description Get Email Version
// @Description Get Email Version by ID and VersionID.
// @Tags Email
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (User or Administrator)"
// @Param id path string true "Email ID"
// @Param versionid path string true "Email VersionID"
// @Success 200 {object} email.Email "Email Version"
// @Failure 400 {object} APIEvent "Bad Request (invalid path parameter)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Owner or Administrator)"
// @Failure 404 {object} APIEvent "Not Found"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/emails/{id}/versions/{versionid} [get]
func readEmailVersion(c *gin.Context) {
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
	// Read the Email Version
	version, err := api.EmailService.ReadVersion(c, id, versionid)
	if err != nil && errors.Is(err, v.ErrNotFound) {
		abortWithError(c, http.StatusNotFound, fmt.Errorf("not found: email %s version %s", id, versionid))
		return
	}
	if err != nil {
		evt, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   id,
			EntityType: "Email",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read email %s version %s: %w", id, versionid, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, evt)
		return
	}
	// Verify that the user is authorized to read the Email
	u, _ := contextUser(c) // the user has already been authenticated
	if !u.HasRole("admin") && !version.IsParticipant(u.Email) {
		abortWithError(c, http.StatusForbidden, fmt.Errorf("forbidden: email %s", id))
		return
	}
	c.JSON(http.StatusOK, version)
}

// existsEmailVersion checks if the specified Email version exists.
//
// @Summary Email Version Exists
// @Description Email Version Exists
// @Description Check if the specified Email version exists.
// @Tags Email
// @Param id path string true "Email ID"
// @Param versionid path string true "Email VersionID"
// @Success 204 "Email Version Exists"
// @Failure 400 "Bad Request (invalid path parameter)"
// @Failure 404 "Not Found"
// @Router /v1/emails/{id}/versions/{versionid} [head]
func existsEmailVersion(c *gin.Context) {
	id := c.Param("id")
	versionid := c.Param("versionid")
	if !tuid.IsValid(tuid.TUID(id)) || !tuid.IsValid(tuid.TUID(versionid)) {
		c.Status(http.StatusBadRequest)
	} else if !api.EmailService.VersionExists(c, id, versionid) {
		c.Status(http.StatusNotFound)
	} else {
		c.Status(http.StatusNoContent)
	}
}

// updateEmail updates and returns the specified Email. If the status is PENDING, the Email is sent.
// Note that the updated version needs to be complete; this is not a partial update (e.g. PATCH).
//
// @Summary Update Email Message
// @Description Update/Send Email Message
// @Description Update the provided complete Email. If the status is PENDING, the Email is sent.
// @Tags Email
// @Accept json
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param email body email.Email true "Email"
// @Param id path string true "Email ID"
// @Success 200 {object} email.Email "Email"
// @Failure 400 {object} APIEvent "Bad Request (invalid JSON or parameter)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 422 {object} APIEvent "Email validation errors"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/emails/{id} [put]
func updateEmail(c *gin.Context) {
	// Parse the request body as an Email
	var body email.Email
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
	// The path parameter ID must match the Email ID
	if body.ID != id {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: path parameter ID %s does not match Email ID %s", id, body.ID))
		return
	}
	// Update the specified Email
	e, problems, err := api.EmailService.Update(c, body)
	if len(problems) > 0 && err != nil {
		abortWithError(c, http.StatusUnprocessableEntity, fmt.Errorf("unprocessable entity: %w", err))
		return
	}
	if err != nil {
		evt, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   e.ID,
			EntityType: e.Type(),
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("update email %s: %w", e.ID, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, evt)
		return
	}
	// Log the update
	_, _, _ = api.EventService.Create(c, event.Event{
		UserID:     contextUserID(c),
		EntityID:   e.ID,
		EntityType: e.Type(),
		LogLevel:   event.INFO,
		Message:    "updated Email " + e.ID,
		URI:        c.Request.URL.String(),
	})
	// Return the updated Email
	c.JSON(http.StatusOK, e)
}

// deleteEmail deletes the specified Email.
//
// @Summary Delete Email Message
// @Description Delete Email Message
// @Description Delete and return the specified Email.
// @Tags Email
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param id path string true "Email ID"
// @Success 200 {object} email.Email "Email that was deleted"
// @Failure 400 {object} APIEvent "Bad Request (invalid path parameter ID)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 404 {object} APIEvent "Not Found"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/emails/{id} [delete]
func deleteEmail(c *gin.Context) {
	// Validate the path parameter ID
	id := c.Param("id")
	if !tuid.IsValid(tuid.TUID(id)) {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid path parameter ID: %s", id))
		return
	}
	// Delete the specified Email
	e, err := api.EmailService.Delete(c, id)
	if err != nil && errors.Is(err, v.ErrNotFound) {
		abortWithError(c, http.StatusNotFound, fmt.Errorf("not found: email %s", id))
		return
	}
	if err != nil {
		evt, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   id,
			EntityType: "Email",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("delete email %s: %w", id, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, evt)
		return
	}
	// Log the deletion
	_, _, _ = api.EventService.Create(c, event.Event{
		UserID:     contextUserID(c),
		EntityID:   e.ID,
		EntityType: e.Type(),
		LogLevel:   event.INFO,
		Message:    "deleted Email " + e.ID,
		URI:        c.Request.URL.String(),
	})
	// Return the deleted email
	c.JSON(http.StatusOK, e)
}

// readEmailAddresses returns a list of email addresses for which emails exist.
// It's useful for paging through emails by email address.
//
// @Summary List Email Addresses
// @Description Get Email Addresses
// @Description Get a list of email addresses for which emails exist.
// @Tags Email
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param reverse query bool false "Reverse Order (default: false)"
// @Param limit query int false "Limit (default: 100)"
// @Param offset query string false "Offset (default: forward/reverse alphanumeric)"
// @Success 200 {array} string "Email Addresses"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/email_addresses [get]
func readEmailAddresses(c *gin.Context) {
	// Parse query parameters, with defaults
	reverse, limit, offset, err := paginationParams(c, false, 100)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}
	addresses, err := api.EmailService.ReadAddresses(c, reverse, limit, offset)
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: "Email",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read email addresses: %w", err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.JSON(http.StatusOK, addresses)
}

// readEmailStatuses returns a list of status codes for which emails exist.
// It's useful for paging through emails by status.
//
// @Summary List Email Statuses
// @Description Get Email Statuses
// @Description Get a list of status codes for which emails exist.
// @Tags Email
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Success 200 {array} string "Email Statuses"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/email_statuses [get]
func readEmailStatuses(c *gin.Context) {
	statuses, err := api.EmailService.ReadAllStatuses(c)
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: "Email",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read email statuses: %w", err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.JSON(http.StatusOK, statuses)
}
