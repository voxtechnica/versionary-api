package main

import (
	"errors"
	"fmt"
	"github.com/voxtechnica/tuid-go"
	v "github.com/voxtechnica/versionary"
	"net/http"
	"versionary-api/pkg/event"
	"versionary-api/pkg/user"

	"github.com/gin-gonic/gin"
)

// registerTokenRoutes initializes the Token routes.
func registerTokenRoutes(r *gin.Engine) {
	r.POST("/v1/tokens", createToken)
	r.GET("/v1/tokens", readTokens)
	r.GET("/v1/tokens/:id", readToken)
	r.DELETE("/v1/tokens/:id", deleteToken)
	r.GET("/logout", logout)
	r.GET("/v1/token_user_ids", roleAuthorizer("admin"), readTokenUserIDs)
}

// createToken receives an OAuth TokenRequest, validates the User password,
// creates a new Token, and returns an OAuth TokenResponse.
//
// @Summary Create a new Token
// @Description Create a new OAuth Bearer Token.
// @Tags Token
// @Accept json
// @Produce json
// @Param TokenRequest body user.TokenRequest true "Token Request"
// @Success 201 {object} user.TokenResponse "Token Response"
// @Failure 400 {object} APIEvent "Bad Request (invalid JSON body)"
// @Failure 401 {object} APIEvent "Unauthorized (invalid username or password)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Header 201 {string} Location "URL of the newly created Token"
// @Router /v1/tokens [post]
func createToken(c *gin.Context) {
	// Parse the request body as an OAuth TokenRequest
	var req user.TokenRequest
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
		e, _ := api.EventService.Create(c, event.Event{
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
	t, err := api.TokenService.Create(c, user.Token{
		UserID: u.ID,
		Email:  u.Email,
	})
	if err != nil {
		e, _ := api.EventService.Create(c, event.Event{
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
	_, _ = api.EventService.Create(c, event.Event{
		UserID:     t.UserID,
		EntityID:   t.ID,
		EntityType: t.Type(),
		LogLevel:   event.INFO,
		Message:    fmt.Sprintf("created Token %s for User %s", t.ID, u.ID),
		URI:        c.Request.URL.String(),
	})
	// Return an OAuth TokenResponse
	c.Header("Location", c.Request.URL.String()+"/"+t.ID)
	c.JSON(http.StatusCreated, user.TokenResponse{
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
// @Description List OAuth Bearer Tokens by User ID (defaults to the Context User).
// @Tags Token
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (any role)"
// @Param user query string false "User ID or Email (defaults to the Context User)"
// @Success 200 {array} user.Token "Tokens"
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
		e, _ := api.EventService.Create(c, event.Event{
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
		e, _ := api.EventService.Create(c, event.Event{
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
// @Description Read OAuth Bearer Token by ID (e.g. to verify a token has not expired).
// @Tags Token
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (any role)"
// @Param id path string true "Token ID"
// @Success 200 {object} user.Token "Token"
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
		e, _ := api.EventService.Create(c, event.Event{
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

// deleteToken deletes the specified Token.
// To "log out", the user should use the token to delete it.
// Administrators may delete any user's tokens. Users may only delete their own tokens.
//
// @Summary Delete Token
// @Description Delete OAuth Bearer Token by ID (e.g. to log out).
// @Tags Token
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (any role)"
// @Param id path string true "Token ID"
// @Success 200 {object} user.Token "Token that was deleted"
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
		e, _ := api.EventService.Create(c, event.Event{
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
		e, _ := api.EventService.Create(c, event.Event{
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
	_, _ = api.EventService.Create(c, event.Event{
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
// @Description Delete the OAuth Bearer Token provided in the Authorization header (e.g. to log out).
// @Tags Token
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (any role)"
// @Success 200 {object} user.Token "Token that was deleted"
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
		e, _ := api.EventService.Create(c, event.Event{
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
	_, _ = api.EventService.Create(c, event.Event{
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

// readTokenUserIDs returns a paginated list of User IDs for which tokens exist. This endpoint is
// only available to administrators. It's useful for paging through tokens by user.
//
// @Summary List User IDs for which tokens exist
// @Description List User IDs for which tokens exist. This is useful for paging through tokens by user.
// @Tags Token
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param reverse query bool false "Reverse Order (default: false)"
// @Param limit query int false "Limit (default: 1000)"
// @Param offset query string false "Offset (default: forward/reverse alphanumeric)"
// @Success 200 {array} string "Paginated User IDs"
// @Failure 400 {object} APIEvent "Bad Request (invalid query parameter)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/token_user_ids [get]
func readTokenUserIDs(c *gin.Context) {
	// Parse query parameters, with defaults
	reverse, limit, offset, err := paginationParams(c, false, 1000)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}
	// Read paginated User IDs for which Tokens exist
	ids, err := api.TokenService.ReadUserIDs(c, reverse, limit, offset)
	if err != nil {
		e, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: "Token",
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read token user IDs: %w", err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	// Return the IDs
	c.JSON(http.StatusOK, ids)
}
