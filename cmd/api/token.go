package main

import (
	"fmt"
	"github.com/voxtechnica/tuid-go"
	"net/http"
	"strconv"
	"versionary-api/pkg/event"
	"versionary-api/pkg/user"

	"github.com/gin-gonic/gin"
)

// initTokenRoutes initializes the Token routes.
func initTokenRoutes(r *gin.Engine) {
	r.POST("/v1/tokens", createToken)
	r.GET("/v1/tokens", readTokens)
	r.GET("/v1/tokens/:id", readToken)
	r.DELETE("/v1/tokens/:id", deleteToken)
	r.GET("/v1/logout", logout)
	r.GET("/v1/token_user_ids", roleAuthorizer("admin"), readTokenUserIDs)
}

// createToken receives an OAuth TokenRequest, validates the User password,
// creates a new Token, and returns an OAuth TokenResponse.
func createToken(c *gin.Context) {
	// Parse the request body as an OAuth TokenRequest
	var req user.TokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":  http.StatusBadRequest,
			"error": fmt.Errorf("post token: JSON binding error: %w", err).Error(),
		})
		return
	}
	// Read the associated User
	u, err := api.UserService.Read(c, req.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":  http.StatusUnauthorized,
			"error": "invalid username or password",
		})
		return
	}
	// Validate the password
	if !u.ValidPassword(req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":  http.StatusUnauthorized,
			"error": "invalid username or password",
		})
		return
	}
	// Create a new token for the User
	t, err := api.TokenService.Create(c, user.Token{
		UserID: u.ID,
		Email:  u.Email,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":  http.StatusInternalServerError,
			"error": fmt.Errorf("post token for %s error: %w", u.ID, err).Error(),
		})
		return
	}
	// Log the token creation (best effort)
	_, _ = api.EventService.Create(c, event.Event{
		UserID:     t.UserID,
		EntityID:   t.ID,
		EntityType: t.Type(),
		LogLevel:   event.INFO,
		Message:    fmt.Sprintf("created Token %s for User %s", t.ID, u.ID),
		URI:        c.Request.URL.String(),
	})
	// Return an OAuth TokenResponse
	c.JSON(http.StatusOK, user.TokenResponse{
		AccessToken: t.ID,
		TokenType:   "Bearer",
		ExpiresAt:   t.ExpiresAt,
	})
}

// readTokens returns a paginated list of Tokens for the specified User.
// If the User is not specified, it's extracted from the Context.
// Administrators may read any user's tokens. Users may only read their own tokens.
func readTokens(c *gin.Context) {
	// Only authenticated users can read tokens
	cUser, ok := contextUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":  http.StatusUnauthorized,
			"error": "unauthorized",
		})
		return
	}
	// Only administrators can read tokens for other users
	idOrEmail := c.DefaultQuery("user", cUser.ID)
	if !(idOrEmail == cUser.ID || idOrEmail == cUser.Email) && !cUser.HasRole("admin") {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":  http.StatusUnauthorized,
			"error": "unauthorized",
		})
		return
	}
	// Read the associated User
	u, err := api.UserService.Read(c, idOrEmail)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":  http.StatusBadRequest,
			"error": fmt.Errorf("invalid User %s: %w", idOrEmail, err).Error(),
		})
		return
	}
	// Read the User's tokens
	tokens, err := api.TokenService.ReadAllTokensByUserIDAsJSON(c, u.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":  http.StatusInternalServerError,
			"error": fmt.Errorf("read tokens for User %s: %w", u.ID, err).Error(),
		})
		return
	}
	// Return the tokens
	c.Data(http.StatusOK, "application/json;charset=UTF-8", tokens)
}

// readToken returns a Token for the specified ID.
// This method is useful for verifying that a Bearer token is (still) active.
// Is the User still logged in? Use to their token to get the token.
func readToken(c *gin.Context) {
	// Only authenticated users can read tokens
	cUser, ok := contextUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":  http.StatusUnauthorized,
			"error": "unauthorized",
		})
		return
	}
	// Read the specified Token
	id := c.Param("id")
	t, err := api.TokenService.Read(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":  http.StatusNotFound,
			"error": fmt.Errorf("read token %s: %w", id, err).Error(),
		})
		return
	}
	// Only administrators can read tokens for other users
	if t.UserID != cUser.ID && !cUser.HasRole("admin") {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":  http.StatusUnauthorized,
			"error": "unauthorized",
		})
		return
	}
	// Return the token
	c.JSON(http.StatusOK, t)
}

// deleteToken deletes the specified Token.
// To "log out", the user should use the token to delete it.
// Administrators may delete any user's tokens. Users may only delete their own tokens.
func deleteToken(c *gin.Context) {
	// Only authenticated users can delete tokens
	cUser, ok := contextUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":  http.StatusUnauthorized,
			"error": "unauthorized",
		})
		return
	}
	// Read the specified Token
	id := c.Param("id")
	t, err := api.TokenService.Read(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":  http.StatusNotFound,
			"error": fmt.Errorf("read token %s: %w", id, err).Error(),
		})
		return
	}
	// Only administrators can delete tokens for other users
	if t.UserID != cUser.ID && !cUser.HasRole("admin") {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":  http.StatusUnauthorized,
			"error": "unauthorized",
		})
		return
	}
	// Delete the token
	t, err = api.TokenService.Delete(c, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":  http.StatusInternalServerError,
			"error": fmt.Errorf("delete token %s: %w", id, err).Error(),
		})
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
func logout(c *gin.Context) {
	cToken, ok := contextToken(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":  http.StatusUnauthorized,
			"error": "unauthorized",
		})
		return
	}
	// Delete the token
	t, err := api.TokenService.Delete(c, cToken.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":  http.StatusInternalServerError,
			"error": fmt.Errorf("logout token %s: %w", cToken.ID, err).Error(),
		})
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
func readTokenUserIDs(c *gin.Context) {
	// Parse query parameters, with defaults
	reverse, err := strconv.ParseBool(c.DefaultQuery("reverse", "false"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":  http.StatusBadRequest,
			"error": fmt.Errorf("read token user IDs: invalid reverse: %w", err).Error(),
		})
		return
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "1000"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":  http.StatusBadRequest,
			"error": fmt.Errorf("read token user IDs: invalid limit: %w", err).Error(),
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
	// Read paginated User IDs for which Tokens exist
	ids, err := api.TokenService.ReadUserIDs(c, reverse, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":  http.StatusInternalServerError,
			"error": fmt.Errorf("read token user IDs: %w", err).Error(),
		})
		return
	}
	// Return the IDs
	c.JSON(http.StatusOK, ids)
}
