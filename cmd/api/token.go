package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/voxtechnica/tuid-go"
	v "github.com/voxtechnica/versionary"

	"versionary-api/pkg/event"
	"versionary-api/pkg/token"
)

// registerTokenRoutes initializes the Token routes.
func registerTokenRoutes(r *gin.Engine) {
	r.POST("/v1/tokens", createToken)
	r.GET("/v1/tokens", userAuthenticator(), readTokens)
	r.GET("/v1/tokens/:id", userAuthenticator(), readToken)
	r.HEAD("/v1/tokens/:id", userAuthenticator(), existsToken)
	r.DELETE("/v1/tokens/:id", userAuthenticator(), deleteToken)
	r.GET("/logout", userAuthenticator(), logout)
	r.GET("/v1/token_ids", roleAuthorizer("admin"), readTokenIDs)
	r.GET("/v1/token_users", roleAuthorizer("admin"), readTokenUsers)
}

// createToken receives an OAuth TokenRequest, validates the User password,
// creates a new Token, and returns an OAuth Response.
//
// @Summary Create Token
// @Description Create a new Token
// @Description Create a new OAuth Bearer Token.
// @Tags Token
// @Accept json
// @Produce json
// @Param TokenRequest body token.Request true "Token Request"
// @Success 201 {object} token.Response "Token Response"
// @Failure 400 {object} APIEvent "Bad Request (invalid JSON body)"
// @Failure 401 {object} APIEvent "Unauthorized (invalid username or password)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Header 201 {string} Location "URL of the newly created Token"
// @Router /v1/tokens [post]
func createToken(c *gin.Context) {
	// Parse the request body as an OAuth TokenRequest
	var req token.Request
	if err := c.ShouldBindJSON(&req); err != nil {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid JSON body: %w", err))
		return
	}
	// Read the associated User
	u, err := api.UserService.Read(c, req.Username)
	if err != nil && errors.Is(err, v.ErrNotFound) {
		abortWithError(c, http.StatusUnauthorized, errors.New("unauthenticated: invalid username or password"))
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: "Token",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("create token for %s: %w", req.Username, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	// Validate the password
	if !u.ValidPassword(req.Password) {
		abortWithError(c, http.StatusUnauthorized, errors.New("unauthenticated: invalid username or password"))
		return
	}
	// Create a new token for the User
	t, err := api.TokenService.Create(c, token.Token{
		UserID: u.ID,
		Email:  u.Email,
	})
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     u.ID,
			EntityID:   t.ID,
			EntityType: "Token",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("create token for %s: %w", u.ID, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	// Log the token creation
	_, _, _ = api.EventService.Create(c, event.Event{
		UserID:     t.UserID,
		EntityID:   t.ID,
		EntityType: t.Type(),
		LogLevel:   event.INFO,
		Message:    fmt.Sprintf("created Token %s for User %s", t.ID, u.ID),
		URI:        c.Request.URL.String(),
	})
	// Return an OAuth Response
	c.Header("Location", c.Request.URL.String()+"/"+t.ID)
	c.JSON(http.StatusCreated, token.Response{
		AccessToken: t.ID,
		TokenType:   "Bearer",
		ExpiresAt:   t.ExpiresAt,
	})
}

// readTokens returns a paginated list of Tokens for the specified User.
// If the User is not specified, it's extracted from the Context.
// Administrators may read any user's tokens. Users may only read their own tokens.
//
// @Summary List Tokens
// @Description List Tokens
// @Description List OAuth Bearer Tokens by User ID (defaults to the Context User).
// @Tags Token
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (any role)"
// @Param user query string false "User ID or Email (defaults to the Context User)"
// @Success 200 {array} token.Token "Tokens"
// @Failure 400 {object} APIEvent "Bad Request (invalid User ID or Email)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Forbidden (only administrators may read any user's tokens)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/tokens [get]
func readTokens(c *gin.Context) {
	// Only authenticated users can read tokens
	cUser, ok := contextUser(c)
	if !ok {
		abortWithError(c, http.StatusUnauthorized, errors.New("unauthenticated: read tokens"))
		return
	}
	// Only administrators can read tokens for other users
	idOrEmail := c.DefaultQuery("user", cUser.ID)
	if !(idOrEmail == cUser.ID || idOrEmail == cUser.Email) && !cUser.HasRole("admin") {
		abortWithError(c, http.StatusForbidden, errors.New("unauthorized: read tokens"))
		return
	}
	// Read the associated User
	u, err := api.UserService.Read(c, idOrEmail)
	if err != nil && errors.Is(err, v.ErrNotFound) {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("invalid User %s: %w", idOrEmail, err))
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: "Token",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read tokens for user %s: %w", idOrEmail, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	// Read the User's tokens
	tokens, err := api.TokenService.ReadAllTokensByUserIDAsJSON(c, u.ID)
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: "Token",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read tokens for user %s: %w", u.ID, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	// Return the tokens
	c.Data(http.StatusOK, "application/json;charset=UTF-8", tokens)
}

// readToken returns a Token for the specified ID.
// This method is useful for verifying that a Bearer token is (still) active.
// Is the User still logged in? Use to their token to get the token.
//
// @Summary Read Token
// @Description Read Token
// @Description Read OAuth Bearer Token by ID (e.g. to verify a token has not expired).
// @Tags Token
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (any role)"
// @Param id path string true "Token ID"
// @Success 200 {object} token.Token "Token"
// @Failure 400 {object} APIEvent "Bad Request (invalid path parameter ID)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Forbidden (only administrators may read any user's tokens)"
// @Failure 404 {object} APIEvent "Not Found (Token not found)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/tokens/{id} [get]
func readToken(c *gin.Context) {
	// Only authenticated users can read tokens
	cUser, ok := contextUser(c)
	if !ok {
		abortWithError(c, http.StatusUnauthorized, errors.New("unauthenticated: read token"))
		return
	}
	// Validate the token ID
	id := c.Param("id")
	if !tuid.IsValid(tuid.TUID(id)) {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid path parameter ID: %s", id))
		return
	}
	// Read the specified Token
	t, err := api.TokenService.Read(c, id)
	if err != nil && errors.Is(err, v.ErrNotFound) {
		abortWithError(c, http.StatusNotFound, fmt.Errorf("not found: Token %s", id))
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   id,
			EntityType: "Token",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read token %s: %w", id, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	// Only administrators can read tokens for other users
	if t.UserID != cUser.ID && !cUser.HasRole("admin") {
		abortWithError(c, http.StatusForbidden, errors.New("unauthorized: read token"))
		return
	}
	// Return the token
	c.JSON(http.StatusOK, t)
}

// existsToken checks if the specified Token exists.
//
// @Summary Token Exists
// @Description Token Exists
// @Description Check if the specified OAuth Bearer Token exists.
// @Tags Token
// @Param id path string true "Token ID"
// @Success 204 "Token Exists"
// @Failure 400 "Bad Request (invalid path parameter ID)"
// @Failure 404 "Not Found"
// @Router /v1/tokens/{id} [head]
func existsToken(c *gin.Context) {
	id := c.Param("id")
	if !tuid.IsValid(tuid.TUID(id)) {
		c.Status(http.StatusBadRequest)
	} else if !api.TokenService.Exists(c, id) {
		c.Status(http.StatusNotFound)
	} else {
		c.Status(http.StatusNoContent)
	}
}

// deleteToken deletes the specified Token.
// To "log out", the user should use the token to delete it.
// Administrators may delete any user's tokens. Users may only delete their own tokens.
//
// @Summary Delete Token
// @Description Delete Token
// @Description Delete OAuth Bearer Token by ID (e.g. to log out).
// @Tags Token
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (any role)"
// @Param id path string true "Token ID"
// @Success 200 {object} token.Token "Token that was deleted"
// @Failure 400 {object} APIEvent "Bad Request (invalid path parameter ID)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Forbidden (only administrators may delete any user's tokens)"
// @Failure 404 {object} APIEvent "Not Found (Token not found)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/tokens/{id} [delete]
func deleteToken(c *gin.Context) {
	// Only authenticated users can delete tokens
	cUser, ok := contextUser(c)
	if !ok {
		abortWithError(c, http.StatusUnauthorized, errors.New("unauthenticated: delete token"))
		return
	}
	// Validate the token ID
	id := c.Param("id")
	if !tuid.IsValid(tuid.TUID(id)) {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid path parameter ID: %s", id))
		return
	}
	// Read the specified Token
	t, err := api.TokenService.Read(c, id)
	if err != nil && errors.Is(err, v.ErrNotFound) {
		abortWithError(c, http.StatusNotFound, fmt.Errorf("not found: Token %s", id))
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     cUser.ID,
			EntityID:   id,
			EntityType: "Token",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("delete Token %s: %w", id, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	// Only administrators can delete tokens for other users
	if t.UserID != cUser.ID && !cUser.HasRole("admin") {
		abortWithError(c, http.StatusForbidden, errors.New("unauthorized: delete token"))
		return
	}
	// Delete the token
	t, err = api.TokenService.Delete(c, t.ID)
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     cUser.ID,
			EntityID:   t.ID,
			EntityType: t.Type(),
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("delete Token %s: %w", t.ID, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	// Log the token deletion (best effort)
	_, _, _ = api.EventService.Create(c, event.Event{
		UserID:     cUser.ID,
		EntityID:   t.ID,
		EntityType: t.Type(),
		LogLevel:   event.INFO,
		Message:    fmt.Sprintf("deleted Token %s for User %s", t.ID, t.UserID),
		URI:        c.Request.URL.String(),
	})
	// Return the deleted token
	c.JSON(http.StatusOK, t)
}

// logout deletes the Token provided in the Authorization header.
//
// @Summary Logout
// @Description Logout
// @Description Delete the OAuth Bearer Token provided in the Authorization header (e.g. to log out).
// @Tags Token
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (any role)"
// @Success 200 {object} token.Token "Token that was deleted"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /logout [get]
func logout(c *gin.Context) {
	cToken, ok := contextToken(c)
	if !ok {
		abortWithError(c, http.StatusUnauthorized, errors.New("unauthenticated: logout"))
		return
	}
	// Delete the token
	t, err := api.TokenService.Delete(c, cToken.ID)
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     cToken.UserID,
			EntityID:   cToken.ID,
			EntityType: t.Type(),
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("logout Token %s: %w", cToken.ID, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	// Log the token deletion (best effort)
	_, _, _ = api.EventService.Create(c, event.Event{
		UserID:     t.UserID,
		EntityID:   t.ID,
		EntityType: t.Type(),
		LogLevel:   event.INFO,
		Message:    fmt.Sprintf("deleted Token %s for User %s", t.ID, t.UserID),
		URI:        c.Request.URL.String(),
	})
	// Return the deleted token
	c.JSON(http.StatusOK, t)
}

// readTokenIDs returns a paginated list of Token/User ID pairs.
// This endpoint is only available to administrators. It's useful for paging through tokens.
//
// @Summary List Token/User ID Pairs
// @Description List Token/User ID pairs
// @Description List Token/User ID pairs. This is useful for paging through tokens.
// @Tags Token
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param sorted query bool false "Sort by User ID? (not paginated; default: false)"
// @Param reverse query bool false "Reverse Order (default: false)"
// @Param limit query int false "Limit (default: all)"
// @Param offset query string false "Offset (default: forward/reverse alphanumeric)"
// @Success 200 {array} v.TextValue "Token/User ID pairs"
// @Failure 400 {object} APIEvent "Bad Request (invalid query parameter)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/token_ids [get]
func readTokenIDs(c *gin.Context) {
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
	// Read and return the Token/User ID pairs
	var ids []v.TextValue
	var errMessage string
	if all {
		errMessage = "read all token/user ID pairs"
		ids, err = api.TokenService.ReadAllIDs(c, sortByValue)
	} else {
		errMessage = fmt.Sprintf("read %d token/user ID pairs", limit)
		ids, err = api.TokenService.ReadIDs(c, reverse, limit, offset)
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: api.TokenService.EntityType,
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("%s: %w", errMessage, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.JSON(http.StatusOK, ids)
}

// readTokenUsers returns a paginated list of User IDs and email addresses for which tokens exist.
// This endpoint is only available to administrators. It's useful for paging through tokens by user.
//
// @Summary List Token Users
// @Description List Users for which tokens exist
// @Description List User IDs and email addresses for which tokens exist.
// @Description This is useful for paging through tokens by user.
// @Tags Token
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param search query string false "Search Terms, separated by spaces"
// @Param any query bool false "Any Match? (default: false; all search terms must match)"
// @Param sorted query bool false "Sort by Email? (not paginated; default: false)"
// @Param reverse query bool false "Reverse Order (default: false)"
// @Param limit query int false "Limit (default: all)"
// @Param offset query string false "Offset (default: forward/reverse alphanumeric)"
// @Success 200 {array} v.TextValue "User IDs and Email Addresses"
// @Failure 400 {object} APIEvent "Bad Request (invalid query parameter)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/token_users [get]
func readTokenUsers(c *gin.Context) {
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
	// Read and return the User IDs and Email Addresses
	var users []v.TextValue
	var errMessage string
	if search != "" {
		errMessage = fmt.Sprintf("search (%s) token user email addresses", search)
		users, err = api.TokenService.FilterUsers(c, search, anyMatch)
	} else if all {
		errMessage = "read all token users"
		users, err = api.TokenService.ReadAllUsers(c, sortByValue)
	} else {
		errMessage = fmt.Sprintf("read %d token users", limit)
		users, err = api.TokenService.ReadUsers(c, reverse, limit, offset)
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: api.TokenService.EntityType,
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("%s: %w", errMessage, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.JSON(http.StatusOK, users)
}
