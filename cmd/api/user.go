package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/voxtechnica/tuid-go"
	v "github.com/voxtechnica/versionary"

	"versionary-api/pkg/event"
	"versionary-api/pkg/ref"
	"versionary-api/pkg/user"
)

// registerUserRoutes registers the User routes on the gin router.
func registerUserRoutes(r *gin.Engine) {
	r.POST("/v1/users", roleAuthorizer("admin"), createUser)
	r.GET("/v1/users", roleAuthorizer("admin"), readUsers)
	r.GET("/v1/users/:id", readUser)
	r.HEAD("/v1/users/:id", existsUser)
	r.GET("/v1/users/:id/versions", roleAuthorizer("admin"), readUserVersions)
	r.GET("/v1/users/:id/versions/:versionid", readUserVersion)
	r.HEAD("/v1/users/:id/versions/:versionid", existsUserVersion)
	r.PUT("/v1/users/:id", updateUser)
	r.DELETE("/v1/users/:id", deleteUser)
	r.DELETE("/v1/users/:id/versions/:versionid", roleAuthorizer("admin"), deleteUserVersion)
	r.GET("/v1/user_ids", readUserIDs)
	r.GET("/v1/user_names", roleAuthorizer("admin"), readUserNames)
	r.GET("/v1/user_emails", roleAuthorizer("admin"), readUserEmails)
	r.GET("/v1/user_orgs", roleAuthorizer("admin"), readUserOrgs)
	r.GET("/v1/user_roles", roleAuthorizer("admin"), readUserRoles)
	r.GET("/v1/user_statuses", roleAuthorizer("admin"), readUserStatuses)
}

// createUser creates a new User.
//
// @Summary Create User
// @Description Create a new User
// @Description Create a new User.
// @Tags User
// @Accept json
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param user body user.User true "User"
// @Success 201 {object} user.User "Newly-created User"
// @Failure 400 {object} APIEvent "Bad Request (invalid JSON body)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 422 {object} APIEvent "User validation errors"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Header 201 {string} Location "URL of the newly created User"
// @Router /v1/users [post]
func createUser(c *gin.Context) {
	// Parse the request body as a User
	var u user.User
	if err := c.ShouldBindJSON(&u); err != nil {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid JSON body: %w", err))
		return
	}
	// Create a new User
	u, problems, err := api.UserService.Create(c, u)
	if len(problems) > 0 && err != nil {
		abortWithError(c, http.StatusUnprocessableEntity, fmt.Errorf("unprocessable entity: %w", err))
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   u.ID,
			EntityType: u.Type(),
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("create user %s %s: %w", u.ID, u.Email, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	// Log the creation
	_, _, _ = api.EventService.Create(c, event.Event{
		UserID:     contextUserID(c),
		EntityID:   u.ID,
		EntityType: u.Type(),
		LogLevel:   event.INFO,
		Message:    fmt.Sprintf("created User %s %s", u.ID, u.Email),
		URI:        c.Request.URL.String(),
	})
	// Return the new User
	c.Header("Location", c.Request.URL.String()+"/"+u.ID)
	c.JSON(http.StatusCreated, u)
}

// readUsers returns a paginated list of Users.
//
// @Summary List Users
// @Description List Users
// @Description List Users, paging with reverse, limit, and offset.
// @Description Optionally, filter by email, organization, role, or status.
// @Tags User
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param email query string false "Email Address"
// @Param org query string false "Organization ID"
// @Param role query string false "Role (e.g. admin)"
// @Param status query string false "Status" Enums(PENDING, ENABLED, DISABLED)
// @Param reverse query bool false "Reverse Order (default: false)"
// @Param limit query int false "Limit (default: 100)"
// @Param offset query string false "Offset (default: forward/reverse alphanumeric)"
// @Success 200 {array} user.User "Users"
// @Failure 400 {object} APIEvent "Bad Request (invalid parameter)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/users [get]
func readUsers(c *gin.Context) {
	// Parse query parameters, with defaults
	reverse, limit, offset, err := paginationParams(c, false, 100)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}
	email := c.Query("email")
	orgID := c.Query("org")
	if orgID != "" && !tuid.IsValid(tuid.TUID(orgID)) {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid parameter, org : %s", orgID))
		return
	}
	role := c.Query("role")
	status := strings.ToUpper(c.Query("status"))
	if status != "" && !user.Status(status).IsValid() {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid status: %s", status))
		return
	}
	// Read and return paginated Users
	if email != "" {
		// Filter by email address (there should be only one user with this email address)
		u, err := api.UserService.ReadAllUsersByEmail(c, email)
		if err != nil {
			e, _, _ := api.EventService.Create(c, event.Event{
				UserID:     contextUserID(c),
				EntityType: "User",
				LogLevel:   event.ERROR,
				Message:    fmt.Errorf("read all users by email %s: %w", email, err).Error(),
				URI:        c.Request.URL.String(),
				Err:        err,
			})
			abortWithError(c, http.StatusInternalServerError, e)
			return
		}
		c.JSON(http.StatusOK, u)
	} else if orgID != "" {
		// Filter by Organization ID
		u, err := api.UserService.ReadUsersByOrgIDAsJSON(c, orgID, reverse, limit, offset)
		if err != nil {
			e, _, _ := api.EventService.Create(c, event.Event{
				UserID:     contextUserID(c),
				EntityType: "User",
				LogLevel:   event.ERROR,
				Message:    fmt.Errorf("read users by organization %s: %w", orgID, err).Error(),
				URI:        c.Request.URL.String(),
				Err:        err,
			})
			abortWithError(c, http.StatusInternalServerError, e)
			return
		}
		c.Data(http.StatusOK, "application/json;charset=UTF-8", u)
	} else if role != "" {
		// Filter by role (e.g. "admin")
		u, err := api.UserService.ReadUsersByRoleAsJSON(c, role, reverse, limit, offset)
		if err != nil {
			e, _, _ := api.EventService.Create(c, event.Event{
				UserID:     contextUserID(c),
				EntityType: "User",
				LogLevel:   event.ERROR,
				Message:    fmt.Errorf("read users by role %s: %w", role, err).Error(),
				URI:        c.Request.URL.String(),
				Err:        err,
			})
			abortWithError(c, http.StatusInternalServerError, e)
			return
		}
		c.Data(http.StatusOK, "application/json;charset=UTF-8", u)
	} else if status != "" {
		// Filter by status (e.g. "ENABLED")
		u, err := api.UserService.ReadUsersByStatusAsJSON(c, status, reverse, limit, offset)
		if err != nil {
			e, _, _ := api.EventService.Create(c, event.Event{
				UserID:     contextUserID(c),
				EntityType: "User",
				LogLevel:   event.ERROR,
				Message:    fmt.Errorf("read users by status %s: %w", status, err).Error(),
				URI:        c.Request.URL.String(),
				Err:        err,
			})
			abortWithError(c, http.StatusInternalServerError, e)
			return
		}
		c.Data(http.StatusOK, "application/json;charset=UTF-8", u)
	} else {
		// Unfiltered, fetched in parallel by ID
		u := api.UserService.ReadUsers(c, reverse, limit, offset)
		c.JSON(http.StatusOK, u)
	}
}

// readUser returns the current version of the specified User.
//
// @Summary Read User
// @Description Get User
// @Description Get User by ID or email, scrubbing sensitive information if the requester is not an administrator.
// @Tags User
// @Produce json
// @Param authorization header string true "OAuth Bearer Token"
// @Param id path string true "User ID or Email Address"
// @Success 200 {object} user.User "User"
// @Failure 400 {object} APIEvent "Bad Request (invalid path parameter ID)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Forbidden (only administrators may read any user)"
// @Failure 404 {object} APIEvent "Not Found"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/users/{id} [get]
func readUser(c *gin.Context) {
	// Only authenticated users can read a user
	cUser, ok := contextUser(c)
	if !ok {
		abortWithError(c, http.StatusUnauthorized, errors.New("unauthenticated: read user"))
		return
	}
	// Validate the path parameter ID (as either an email address or a TUID)
	idOrEmail := c.Param("id")
	var id string
	var email string
	if strings.Contains(idOrEmail, "@") {
		email = user.StandardizeEmail(idOrEmail)
		_, err := mail.ParseAddress(email)
		if err != nil {
			abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid path parameter %s: %w", email, err))
			return
		}
	} else {
		id = idOrEmail
		if !tuid.IsValid(tuid.TUID(id)) {
			abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid path parameter ID: %s", id))
			return
		}
	}
	// Read the specified User
	u, err := api.UserService.Read(c, idOrEmail)
	if err != nil && errors.Is(err, v.ErrNotFound) {
		abortWithError(c, http.StatusNotFound, fmt.Errorf("not found: user %s", idOrEmail))
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     cUser.ID,
			EntityID:   id,
			EntityType: "User",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read user %s: %w", idOrEmail, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	// Only administrators can read any User
	if u.ID != cUser.ID && !cUser.HasRole("admin") {
		abortWithError(c, http.StatusForbidden, errors.New("unauthorized: read user"))
		return
	}
	// Scrub sensitive information from the User
	if cUser.HasRole("admin") {
		c.JSON(http.StatusOK, u)
	} else {
		c.JSON(http.StatusOK, u.Scrub())
	}
}

// existsUser checks if the specified User exists.
//
// @Summary User Exists
// @Description User Exists
// @Description Check if the specified User exists.
// @Tags User
// @Param id path string true "User ID"
// @Success 204 "User Exists"
// @Failure 400 {object} APIEvent "Bad Request (invalid path parameter ID)"
// @Failure 404 {object} APIEvent "Not Found"
// @Router /v1/users/{id} [head]
func existsUser(c *gin.Context) {
	id := c.Param("id")
	if !tuid.IsValid(tuid.TUID(id)) {
		c.Status(http.StatusBadRequest)
	} else if !api.UserService.Exists(c, id) {
		c.Status(http.StatusNotFound)
	} else {
		c.Status(http.StatusNoContent)
	}
}

// readUserVersions returns a paginated list of versions of the specified User.
//
// @Summary List User Versions
// @Description List User Versions
// @Description Get User Versions by User ID, paging with reverse, limit, and offset.
// @Tags User
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param id path string true "User ID"
// @Param reverse query bool false "Reverse Order (default: false)"
// @Param limit query int false "Limit (default: 100)"
// @Param offset query string false "Offset (default: forward/reverse alphanumeric)"
// @Success 200 {array} user.User "User Versions"
// @Failure 400 {object} APIEvent "Bad Request (invalid path parameter ID)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 404 {object} APIEvent "Not Found"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/users/{id}/versions [get]
func readUserVersions(c *gin.Context) {
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
	// Verify that the User exists
	if !api.UserService.Exists(c, id) {
		abortWithError(c, http.StatusNotFound, fmt.Errorf("not found: user %s", id))
		return
	}
	// Read and return the specified User Versions
	versions, err := api.UserService.ReadVersionsAsJSON(c, id, reverse, limit, offset)
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   id,
			EntityType: "User",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read user %s versions: %w", id, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.Data(http.StatusOK, "application/json;charset=UTF-8", versions)
}

// readUserVersion returns the specified version of the specified User.
//
// @Summary Read User Version
// @Description Get User Version
// @Description Get User Version by ID and VersionID, scrubbing sensitive information if the requester is not an administrator.
// @Tags User
// @Produce json
// @Param authorization header string true "OAuth Bearer Token"
// @Param id path string true "User ID"
// @Param versionid path string true "User VersionID"
// @Success 200 {object} user.User "User Version"
// @Failure 400 {object} APIEvent "Bad Request (invalid path parameter)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 404 {object} APIEvent "Not Found"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/users/{id}/versions/{versionid} [get]
func readUserVersion(c *gin.Context) {
	// Only authenticated users can read a user
	cUser, ok := contextUser(c)
	if !ok {
		abortWithError(c, http.StatusUnauthorized, errors.New("unauthenticated: read user"))
		return
	}
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
	// Read the User Version
	u, err := api.UserService.ReadVersion(c, id, versionid)
	if err != nil && errors.Is(err, v.ErrNotFound) {
		abortWithError(c, http.StatusNotFound, fmt.Errorf("not found: user %s version %s", id, versionid))
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     cUser.ID,
			EntityID:   id,
			EntityType: "User",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read user %s version %s: %w", id, versionid, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	// Only administrators can read any User
	if u.ID != cUser.ID && !cUser.HasRole("admin") {
		abortWithError(c, http.StatusForbidden, errors.New("unauthorized: read user"))
		return
	}
	// Scrub sensitive information from the User version
	if cUser.HasRole("admin") {
		c.JSON(http.StatusOK, u)
	} else {
		c.JSON(http.StatusOK, u.Scrub())
	}
}

// existsUserVersion checks if the specified User version exists.
//
// @Summary User Version Exists
// @Description User Version Exists
// @Description Check if the specified User version exists.
// @Tags User
// @Param id path string true "User ID"
// @Param versionid path string true "User VersionID"
// @Success 204 "User Version Exists"
// @Failure 400 {object} APIEvent "Bad Request (invalid path parameter)"
// @Failure 404 {object} APIEvent "Not Found"
// @Router /v1/users/{id}/versions/{versionid} [head]
func existsUserVersion(c *gin.Context) {
	id := c.Param("id")
	versionid := c.Param("versionid")
	if !tuid.IsValid(tuid.TUID(id)) || !tuid.IsValid(tuid.TUID(versionid)) {
		c.Status(http.StatusBadRequest)
	} else if !api.UserService.VersionExists(c, id, versionid) {
		c.Status(http.StatusNotFound)
	} else {
		c.Status(http.StatusNoContent)
	}
}

// updateUser updates and returns the specified User.
// Note that the updated version needs to be complete; this is not a partial update (e.g. PATCH).
//
// @Summary Update User
// @Description Update User
// @Description Update the provided complete User, ensuring that sensitive information is retained.
// @Tags User
// @Accept json
// @Produce json
// @Param authorization header string true "OAuth Bearer Token"
// @Param user body user.User true "User"
// @Param id path string true "User ID"
// @Success 200 {object} user.User "User"
// @Failure 400 {object} APIEvent "Bad Request (invalid JSON or parameter)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 422 {object} APIEvent "User validation errors"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/users/{id} [put]
func updateUser(c *gin.Context) {
	// Only authenticated users can update a user
	cUser, ok := contextUser(c)
	if !ok {
		abortWithError(c, http.StatusUnauthorized, errors.New("unauthenticated: update user"))
		return
	}
	// Parse the request body as a User
	var u user.User
	if err := c.ShouldBindJSON(&u); err != nil {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid JSON body: %w", err))
		return
	}
	// Validate the path parameter ID
	id := c.Param("id")
	if !tuid.IsValid(tuid.TUID(id)) {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid path parameter ID: %s", id))
		return
	}
	// The path parameter ID must match the User ID
	if u.ID != id {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: path parameter ID %s does not match User ID %s", id, u.ID))
		return
	}
	// Only administrators can update any User
	if u.ID != cUser.ID && !cUser.HasRole("admin") {
		abortWithError(c, http.StatusForbidden, errors.New("unauthorized: update user"))
		return
	}
	// If the User is not an Administrator, restore sensitive information
	if !cUser.HasRole("admin") {
		// Read the prior version of the User
		prior, err := api.UserService.Read(c, id)
		if err != nil && errors.Is(err, v.ErrNotFound) {
			abortWithError(c, http.StatusNotFound, fmt.Errorf("not found: user %s", id))
			return
		}
		if err != nil {
			e, _, _ := api.EventService.Create(c, event.Event{
				UserID:     cUser.ID,
				EntityID:   id,
				EntityType: "User",
				LogLevel:   event.ERROR,
				Message:    fmt.Errorf("read user %s: %w", id, err).Error(),
				URI:        c.Request.URL.String(),
				Err:        err,
			})
			abortWithError(c, http.StatusInternalServerError, e)
			return
		}
		// Restore sensitive information from the prior version
		u = u.RestoreScrubbed(prior)
		// Avoid escalating privileges
		u.Roles = prior.Roles
	}
	// Update the provided User
	u, problems, err := api.UserService.Update(c, u)
	if len(problems) > 0 && err != nil {
		abortWithError(c, http.StatusUnprocessableEntity, fmt.Errorf("unprocessable entity: %w", err))
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     cUser.ID,
			EntityID:   u.ID,
			EntityType: u.Type(),
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("update user %s %s: %w", u.ID, u.Email, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	// Log the update
	_, _, _ = api.EventService.Create(c, event.Event{
		UserID:     cUser.ID,
		EntityID:   u.ID,
		EntityType: u.Type(),
		LogLevel:   event.INFO,
		Message:    fmt.Sprintf("updated User %s %s", u.ID, u.Email),
		URI:        c.Request.URL.String(),
	})
	// Scrub sensitive information from the User version
	if cUser.HasRole("admin") {
		c.JSON(http.StatusOK, u)
	} else {
		c.JSON(http.StatusOK, u.Scrub())
	}
}

// deleteUser deletes the specified User.
//
// @Summary Delete User
// @Description Delete User
// @Description Delete and return the specified User. Attempt to delete their associated Tokens as well, logging errors.
// @Tags User
// @Produce json
// @Param authorization header string true "OAuth Bearer Token"
// @Param id path string true "User ID"
// @Success 200 {object} user.User "User that was deleted"
// @Failure 400 {object} APIEvent "Bad Request (invalid path parameter ID)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 404 {object} APIEvent "Not Found"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/users/{id} [delete]
func deleteUser(c *gin.Context) {
	// Only authenticated users can read a user
	cUser, ok := contextUser(c)
	if !ok {
		abortWithError(c, http.StatusUnauthorized, errors.New("unauthenticated: delete user"))
		return
	}
	// Validate the path parameter ID
	id := c.Param("id")
	if !tuid.IsValid(tuid.TUID(id)) {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid path parameter ID: %s", id))
		return
	}
	// Only administrators can delete any User
	if id != cUser.ID && !cUser.HasRole("admin") {
		abortWithError(c, http.StatusForbidden, errors.New("unauthorized: delete user"))
		return
	}
	// Delete the specified User
	u, err := api.UserService.Delete(c, id)
	if err != nil && errors.Is(err, v.ErrNotFound) {
		abortWithError(c, http.StatusNotFound, fmt.Errorf("not found: user %s", id))
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     cUser.ID,
			EntityID:   id,
			EntityType: "User",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("delete user %s: %w", id, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	// Log the deletion
	_, _, _ = api.EventService.Create(c, event.Event{
		UserID:     cUser.ID,
		EntityID:   u.ID,
		EntityType: u.Type(),
		LogLevel:   event.INFO,
		Message:    fmt.Sprintf("deleted User %s %s", u.ID, u.Email),
		URI:        c.Request.URL.String(),
	})
	// Delete the user's tokens (make an attempt; they'll expire eventually)
	err = api.TokenService.DeleteAllTokensByUserID(c, id)
	if err != nil {
		_, _, _ = api.EventService.Create(c, event.Event{
			UserID:     cUser.ID,
			EntityID:   id,
			EntityType: "User",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("delete tokens for user %s: %w", id, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
	}
	// Return the deleted user
	c.JSON(http.StatusOK, u)
}

// deleteUserVersion deletes the specified User version.
//
// @Summary Delete User Version
// @Description Delete User Version
// @Description Delete and return the specified User Version.
// @Tags User
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param id path string true "User ID"
// @Param versionid path string true "User Version ID"
// @Success 200 {object} user.User "User version that was deleted"
// @Failure 400 {object} APIEvent "Bad Request (invalid path parameter ID)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 404 {object} APIEvent "Not Found"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/users/{id}/versions/{versionid} [delete]
func deleteUserVersion(c *gin.Context) {
	// Validate the path parameter ID
	id := c.Param("id")
	versionid := c.Param("versionid")
	refID, err := ref.NewRefID(api.UserService.EntityType, id, versionid)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid path parameter ID: %w", err))
		return
	}
	// Delete the specified User version
	d, err := api.UserService.DeleteVersion(c, id, versionid)
	if err != nil && errors.Is(err, v.ErrNotFound) {
		abortWithError(c, http.StatusNotFound, fmt.Errorf("not found: %s", refID))
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   id,
			EntityType: api.UserService.EntityType,
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("delete %s: %w", refID, err).Error(),
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
		Message:    "deleted " + d.RefID().String(),
		URI:        c.Request.URL.String(),
	})
	// Return the deleted User
	c.JSON(http.StatusOK, d)
}

// readUserIDs returns a list of User IDs for a given email address.
// It's primary function is to check for duplicate email addresses.
//
// @Summary List User IDs for Email Address
// @Description List User IDs for Email Address
// @Description List User IDs for a given email address.
// @Tags User
// @Produce json
// @Param email query string true "Email Address"
// @Success 200 {array} string "List of User IDs"
// @Failure 400 {object} APIEvent "Bad Request (invalid query parameter email)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/user_ids [get]
func readUserIDs(c *gin.Context) {
	// Validate the query parameter email
	email := c.Query("email")
	if email == "" {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: missing required query parameter: email"))
		return
	}
	_, err := mail.ParseAddress(email)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid query parameter email: %w", err))
		return
	}
	// Get the User IDs
	ids, err := api.UserService.ReadUserIDsByEmail(c, email)
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: api.UserService.EntityType,
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read User IDs for email %s: %w", email, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	// Return the User IDs
	c.JSON(http.StatusOK, ids)
}

// readUserNames returns a list of User IDs and Names.
// A "Name" is an RFC 5322 email address (e.g. "Given Family <given.family@example.com>").
//
// @Summary List User Names
// @Description List User IDs and Names
// @Description List User IDs and Names, paging with reverse, limit, and offset.
// @Description Optionally, filter results with search terms.
// @Tags User
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param search query string false "Search Terms, separated by spaces"
// @Param any query bool false "Any Match? (default: false; all search terms must match)"
// @Param sorted query bool false "Sort by Name/Email? (not paginated; default: false)"
// @Param reverse query bool false "Reverse Order (default: false)"
// @Param limit query int false "Limit (omit for all)"
// @Param offset query string false "Offset (default: forward/reverse alphanumeric)"
// @Success 200 {array} v.TextValue "User IDs and Names/Emails"
// @Failure 400 {object} APIEvent "Bad Request (invalid parameter)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/user_names [get]
func readUserNames(c *gin.Context) {
	// Parse query parameters, with defaults
	reverse, limit, offset, err := paginationParams(c, false, 1000)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}
	// Search query parameters
	search := c.Query("search")
	anyMatch, err := strconv.ParseBool(c.DefaultQuery("any", "false"))
	if err != nil {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid parameter, any: %w", err))
		return
	}
	// Sorting query parameters
	sortByValue, err := strconv.ParseBool(c.DefaultQuery("sorted", "false"))
	if err != nil {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid parameter, sorted: %w", err))
		return
	}
	all := sortByValue || c.Query("limit") == ""
	// Read and return the User IDs and Names/Emails
	var names []v.TextValue
	var errMessage string
	if search != "" {
		errMessage = fmt.Sprintf("search (%s) user names", search)
		names, err = api.UserService.FilterNames(c, search, anyMatch)
	} else if all {
		errMessage = "read all user names"
		names, err = api.UserService.ReadAllNames(c, sortByValue)
	} else {
		errMessage = fmt.Sprintf("read %d user names", limit)
		names, err = api.UserService.ReadNames(c, reverse, limit, offset)
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: api.UserService.EntityType,
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("%s: %w", errMessage, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.JSON(http.StatusOK, names)
}

// readUserEmails returns a list of email addresses for which users exist.
//
// @Summary List User Email Addresses
// @Description List User Email Addresses
// @Description Get a paginated list of email addresses for which users exist.
// @Tags User
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param reverse query bool false "Reverse Order (default: false)"
// @Param limit query int false "Limit (default: 100)"
// @Param offset query string false "Offset (default: forward/reverse alphanumeric)"
// @Success 200 {array} string "Email Addresses"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/user_emails [get]
func readUserEmails(c *gin.Context) {
	// Parse pagination query parameters, with defaults
	reverse, limit, offset, err := paginationParams(c, false, 100)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}
	// Read and return email addresses
	emails, err := api.UserService.ReadEmailAddresses(c, reverse, limit, offset)
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: "User",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read user email addresses: %w", err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.JSON(http.StatusOK, emails)
}

// readUserOrgs returns a list of Organization ID/Name pairs for which users exist.
// It's useful for paging through users by organization.
//
// @Summary List User Organizations
// @Description List User Organization ID/Name pairs
// @Description Get a list of Organization ID/Name pairs for which users exist.
// @Tags User
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param sorted query bool false "Sort by Organization Name? (not paginated; default: false)"
// @Param reverse query bool false "Reverse Order (default: false)"
// @Param limit query int false "Limit (omit for all)"
// @Param offset query string false "Offset (default: forward/reverse alphanumeric)"
// @Success 200 {array} v.TextValue "Organization ID/Name pairs"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/user_orgs [get]
func readUserOrgs(c *gin.Context) {
	// Parse query parameters, with defaults
	reverse, limit, offset, err := paginationParams(c, false, 1000)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}
	// Sorting query parameters
	sortByValue, err := strconv.ParseBool(c.DefaultQuery("sorted", "false"))
	if err != nil {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid parameter, sorted: %w", err))
		return
	}
	all := sortByValue || c.Query("limit") == ""
	// Read and return the Organization IDs and Names
	var orgs []v.TextValue
	var errMessage string
	if all {
		errMessage = "read all user organizations"
		orgs, err = api.UserService.ReadAllOrgs(c, sortByValue)
	} else {
		errMessage = fmt.Sprintf("read %d user organizations", limit)
		orgs, err = api.UserService.ReadOrgs(c, reverse, limit, offset)
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: api.UserService.EntityType,
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("%s: %w", errMessage, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.JSON(http.StatusOK, orgs)
}

// readUserRoles returns a list of roles for which users exist.
// It's useful for paging through users by role.
//
// @Summary List User Roles
// @Description List User Roles
// @Description Get a list of roles for which users exist.
// @Tags User
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Success 200 {array} string "User Roles"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/user_roles [get]
func readUserRoles(c *gin.Context) {
	roles, err := api.UserService.ReadAllRoles(c)
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: "User",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read user roles: %w", err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.JSON(http.StatusOK, roles)
}

// readUserStatuses returns a list of status codes for which users exist.
// It's useful for paging through users by status.
//
// @Summary List User Statuses
// @Description List User Statuses
// @Description Get a list of status codes for which users exist.
// @Tags User
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Success 200 {array} string "User Statuses"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/user_statuses [get]
func readUserStatuses(c *gin.Context) {
	statuses, err := api.UserService.ReadAllStatuses(c)
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: "User",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read user statuses: %w", err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.JSON(http.StatusOK, statuses)
}
