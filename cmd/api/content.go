package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/voxtechnica/tuid-go"
	v "github.com/voxtechnica/versionary"

	"versionary-api/pkg/content"
	"versionary-api/pkg/event"
	"versionary-api/pkg/ref"
)

// registerContentRoutes initializes the Content routes.
func registerContentRoutes(r *gin.Engine) {
	r.POST("/v1/contents", roleAuthorizer("admin"), createContent)
	r.GET("/v1/contents", roleAuthorizer("admin"), readContents)
	r.GET("/v1/contents/:id", readContent)
	r.HEAD("/v1/contents/:id", existsContent)
	r.GET("/v1/contents/:id/versions", roleAuthorizer("admin"), readContentVersions)
	r.GET("/v1/contents/:id/versions/:versionid", readContentVersion)
	r.HEAD("/v1/contents/:id/versions/:versionid", existsContentVersion)
	r.PUT("/v1/contents/:id", roleAuthorizer("admin"), updateContent)
	r.DELETE("/v1/contents/:id", roleAuthorizer("admin"), deleteContent)
	r.DELETE("/v1/contents/:id/versions/:versionid", roleAuthorizer("admin"), deleteContentVersion)
	r.GET("/v1/content_types", roleAuthorizer("admin"), readContentTypes)
	r.GET("/v1/content_authors", roleAuthorizer("admin"), readContentAuthors)
	r.GET("/v1/content_editors", roleAuthorizer("admin"), readContentEditors)
	r.GET("/v1/content_tags", roleAuthorizer("admin"), readContentTags)
	r.GET("/v1/content_titles", roleAuthorizer("admin"), readContentTitles)
}

// createContent creates a new unit of Content.
//
// @Summary Create Content
// @Description Create a new Content
// @Description Create a new unit of Content (Book, Chapter, Article, Category, etc.)
// @Tags Content
// @Accept json
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param content body content.Content true "Content"
// @Success 201 {object} content.Content "Newly-created Content"
// @Failure 400 {object} APIEvent "Bad Request (invalid JSON body)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 422 {object} APIEvent "Content validation errors"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Header 201 {string} Location "URL of the newly created Content"
// @Router /v1/contents [post]
func createContent(c *gin.Context) {
	// Parse the request body as a Content
	var body content.Content
	if err := c.ShouldBindJSON(&body); err != nil {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid JSON body: %w", err))
		return
	}
	// Identify the Editor
	editor, _ := contextUser(c)
	body.EditorID = editor.ID
	body.EditorName = editor.FullName()
	// Create a new Content
	created, problems, err := api.ContentService.Create(c, body)
	if len(problems) > 0 && err != nil {
		abortWithError(c, http.StatusUnprocessableEntity, fmt.Errorf("unprocessable entity: %w", err))
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   created.ID,
			EntityType: api.ContentService.EntityType,
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("create %s %s %s: %w", created.RefID(), created.Type, created.Title(), err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	// Log the creation
	_, _, _ = api.EventService.Create(c, event.Event{
		UserID:     contextUserID(c),
		EntityID:   created.ID,
		EntityType: api.ContentService.EntityType,
		LogLevel:   event.INFO,
		Message:    fmt.Sprintf("created %s %s %s", created.RefID(), created.Type, created.Title()),
		URI:        c.Request.URL.String(),
	})
	// Return the new Content
	c.Header("Location", c.Request.URL.String()+"/"+created.ID)
	c.JSON(http.StatusCreated, created)
}

// readContents returns a paginated list of Contents.
//
// @Summary List Contents
// @Description List Contents
// @Description List Contents, paging with reverse, limit, and offset.
// @Tags Content
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param reverse query bool false "Reverse Order (default: false)"
// @Param limit query int false "Limit (default: 10)"
// @Param offset query string false "Offset (default: forward/reverse alphanumeric)"
// @Success 200 {array} content.Content "Contents"
// @Failure 400 {object} APIEvent "Bad Request (invalid parameter)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/contents [get]
func readContents(c *gin.Context) {
	// Parse query parameters, with defaults
	reverse, limit, offset, err := paginationParams(c, false, 10)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}
	// Read and return paginated Contents
	contents := api.ContentService.ReadContents(c, reverse, limit, offset)
	c.JSON(http.StatusOK, contents)
}

// readContent returns the current version of the specified Content.
//
// @Summary Read Content
// @Description Get Content
// @Description Get Content by ID.
// @Tags Content
// @Produce json
// @Param id path string true "Content ID"
// @Success 200 {object} content.Content "Content"
// @Failure 400 {object} APIEvent "Bad Request (invalid path parameter ID)"
// @Failure 404 {object} APIEvent "Not Found"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/contents/{id} [get]
func readContent(c *gin.Context) {
	// Validate the path parameter ID
	id := c.Param("id")
	refID, err := ref.NewRefID(api.ContentService.EntityType, id, "")
	if err != nil {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid path parameter ID: %w", err))
		return
	}
	// Read and return the specified Content
	j, err := api.ContentService.ReadAsJSON(c, id)
	if err != nil && errors.Is(err, v.ErrNotFound) {
		abortWithError(c, http.StatusNotFound, fmt.Errorf("not found: %s", refID))
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   id,
			EntityType: api.ContentService.EntityType,
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read %s: %w", refID, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.Data(http.StatusOK, "application/json;charset=UTF-8", j)
}

// existsContent checks if the specified Content exists.
//
// @Summary Content Exists
// @Description Content Exists
// @Description Check if the specified Content exists.
// @Tags Content
// @Param id path string true "Content ID"
// @Success 204 "Content Exists"
// @Failure 400 "Bad Request (invalid path parameter ID)"
// @Failure 404 "Not Found"
// @Router /v1/contents/{id} [head]
func existsContent(c *gin.Context) {
	id := c.Param("id")
	if !tuid.IsValid(tuid.TUID(id)) {
		c.Status(http.StatusBadRequest)
	} else if !api.ContentService.Exists(c, id) {
		c.Status(http.StatusNotFound)
	} else {
		c.Status(http.StatusNoContent)
	}
}

// readContentVersions returns a paginated list of versions of the specified Content.
//
// @Summary List Content Versions
// @Description Get Content Versions
// @Description Get Content Versions by ID, paging with reverse, limit, and offset.
// @Tags Content
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param id path string true "Content ID"
// @Param reverse query bool false "Reverse Order (default: false)"
// @Param limit query int false "Limit (default: 10)"
// @Param offset query string false "Offset (default: forward/reverse alphanumeric)"
// @Success 200 {array} content.Content "Content Versions"
// @Failure 400 {object} APIEvent "Bad Request (invalid parameter)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 404 {object} APIEvent "Not Found"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/contents/{id}/versions [get]
func readContentVersions(c *gin.Context) {
	// Validate parameters
	id := c.Param("id")
	refID, err := ref.NewRefID(api.ContentService.EntityType, id, "")
	if err != nil {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid path parameter ID: %w", err))
		return
	}
	reverse, limit, offset, err := paginationParams(c, false, 10)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}
	// Verify that the Content exists
	if !api.ContentService.Exists(c, id) {
		abortWithError(c, http.StatusNotFound, fmt.Errorf("not found: %s", refID))
		return
	}
	// Read and return the specified Content Versions
	versions, err := api.ContentService.ReadVersionsAsJSON(c, id, reverse, limit, offset)
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   id,
			EntityType: api.ContentService.EntityType,
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read %s versions: %w", refID, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.Data(http.StatusOK, "application/json;charset=UTF-8", versions)
}

// readContentVersion returns the specified version of the specified Content.
//
// @Summary Read Content Version
// @Description Get Content Version
// @Description Get Content Version by ID and VersionID.
// @Tags Content
// @Produce json
// @Param id path string true "Content ID"
// @Param versionid path string true "Content VersionID"
// @Success 200 {object} content.Content "Content Version"
// @Failure 400 {object} APIEvent "Bad Request (invalid path parameter)"
// @Failure 404 {object} APIEvent "Not Found"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/contents/{id}/versions/{versionid} [get]
func readContentVersion(c *gin.Context) {
	// Validate the path parameters
	id := c.Param("id")
	versionid := c.Param("versionid")
	refID, err := ref.NewRefID(api.ContentService.EntityType, id, versionid)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid path parameter ID: %w", err))
		return
	}
	// Read and return the Content Version
	j, err := api.ContentService.ReadVersionAsJSON(c, id, versionid)
	if err != nil && errors.Is(err, v.ErrNotFound) {
		abortWithError(c, http.StatusNotFound, fmt.Errorf("not found: %s", refID))
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   id,
			EntityType: api.ContentService.EntityType,
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read %s: %w", refID, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.Data(http.StatusOK, "application/json;charset=UTF-8", j)
}

// existsContentVersion checks if the specified Content version exists.
//
// @Summary Content Version Exists
// @Description Content Version Exists
// @Description Check if the specified Content version exists.
// @Tags Content
// @Param id path string true "Content ID"
// @Param versionid path string true "Content VersionID"
// @Success 204 "Content Version Exists"
// @Failure 400 "Bad Request (invalid path parameter)"
// @Failure 404 "Not Found"
// @Router /v1/contents/{id}/versions/{versionid} [head]
func existsContentVersion(c *gin.Context) {
	id := c.Param("id")
	versionid := c.Param("versionid")
	if !tuid.IsValid(tuid.TUID(id)) || !tuid.IsValid(tuid.TUID(versionid)) {
		c.Status(http.StatusBadRequest)
	} else if !api.ContentService.VersionExists(c, id, versionid) {
		c.Status(http.StatusNotFound)
	} else {
		c.Status(http.StatusNoContent)
	}
}

// updateContent updates and returns the specified Content.
// Note that the updated version needs to be complete; this is not a partial update (e.g. PATCH).
//
// @Summary Update Content
// @Description Update Content
// @Description Update the provided, complete unit of Content.
// @Tags Content
// @Accept json
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param content body content.Content true "Content"
// @Param id path string true "Content ID"
// @Success 200 {object} content.Content "Content"
// @Failure 400 {object} APIEvent "Bad Request (invalid JSON or parameter)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 422 {object} APIEvent "Content validation errors"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/contents/{id} [put]
func updateContent(c *gin.Context) {
	// Parse the request body as a Content
	var body content.Content
	if err := c.ShouldBindJSON(&body); err != nil {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid JSON body: %w", err))
		return
	}
	// Validate the path parameter ID
	id := c.Param("id")
	refID, err := ref.NewRefID(api.ContentService.EntityType, id, "")
	if err != nil {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid path parameter ID: %w", err))
		return
	}
	// The path parameter ID must match the Content ID
	if body.ID != id {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: path parameter ID %s does not match Content ID %s", id, body.ID))
		return
	}
	// Identify the Editor
	editor, _ := contextUser(c)
	body.EditorID = editor.ID
	body.EditorName = editor.FullName()
	// Update the specified Content
	updated, problems, err := api.ContentService.Update(c, body)
	if len(problems) > 0 && err != nil {
		abortWithError(c, http.StatusUnprocessableEntity, fmt.Errorf("unprocessable entity %s: %w", refID, err))
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   updated.ID,
			EntityType: api.ContentService.EntityType,
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("update %s %s: %w", updated.RefID(), updated.Title(), err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	// Log the update
	_, _, _ = api.EventService.Create(c, event.Event{
		UserID:     contextUserID(c),
		EntityID:   updated.ID,
		EntityType: api.ContentService.EntityType,
		LogLevel:   event.INFO,
		Message:    fmt.Sprintf("updated %s %s", updated.RefID(), updated.Title()),
		URI:        c.Request.URL.String(),
	})
	// Return the updated Content
	c.JSON(http.StatusOK, updated)
}

// deleteContent deletes the specified Content.
//
// @Summary Delete Content
// @Description Delete Content
// @Description Delete and return the specified Content.
// @Tags Content
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param id path string true "Content ID"
// @Success 200 {object} content.Content "Content that was deleted"
// @Failure 400 {object} APIEvent "Bad Request (invalid path parameter ID)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 404 {object} APIEvent "Not Found"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/contents/{id} [delete]
func deleteContent(c *gin.Context) {
	// Validate the path parameter ID
	id := c.Param("id")
	refID, err := ref.NewRefID(api.ContentService.EntityType, id, "")
	if err != nil {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid path parameter ID: %w", err))
		return
	}
	// Delete the specified Content
	deleted, err := api.ContentService.Delete(c, id)
	if err != nil && errors.Is(err, v.ErrNotFound) {
		abortWithError(c, http.StatusNotFound, fmt.Errorf("not found: %s", refID))
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   id,
			EntityType: api.ContentService.EntityType,
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
		EntityID:   deleted.ID,
		EntityType: api.ContentService.EntityType,
		LogLevel:   event.INFO,
		Message:    fmt.Sprintf("deleted %s %s %s", refID, deleted.Type, deleted.Title()),
		URI:        c.Request.URL.String(),
	})
	// Return the deleted content
	c.JSON(http.StatusOK, deleted)
}

// deleteContentVersion deletes the specified Content version.
//
// @Summary Delete Content Version
// @Description Delete Content Version
// @Description Delete and return the specified Content version.
// @Tags Content
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param id path string true "Content ID"
// @Param versionid path string true "Content VersionID"
// @Success 200 {object} content.Content "Content version that was deleted"
// @Failure 400 {object} APIEvent "Bad Request (invalid path parameter ID)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 404 {object} APIEvent "Not Found"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/contents/{id}/versions/{versionid} [delete]
func deleteContentVersion(c *gin.Context) {
	// Validate the path parameter IDs
	id := c.Param("id")
	versionid := c.Param("versionid")
	refID, err := ref.NewRefID(api.ContentService.EntityType, id, versionid)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid path parameter ID: %w", err))
		return
	}
	// Delete the specified Content
	deleted, err := api.ContentService.DeleteVersion(c, id, versionid)
	if err != nil && errors.Is(err, v.ErrNotFound) {
		abortWithError(c, http.StatusNotFound, fmt.Errorf("not found: %s", refID))
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   id,
			EntityType: api.ContentService.EntityType,
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
		EntityID:   deleted.ID,
		EntityType: api.ContentService.EntityType,
		LogLevel:   event.INFO,
		Message:    fmt.Sprintf("deleted %s %s %s", deleted.RefID(), deleted.Type, deleted.Title()),
		URI:        c.Request.URL.String(),
	})
	// Return the deleted content
	c.JSON(http.StatusOK, deleted)
}

// readContentTypes returns a list of Content types for which contents exist.
// It's useful for paging through contents by type.
//
// @Summary List Content Types
// @Description List Content Types
// @Description List content types, for which contents exist.
// @Tags Content
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Success 200 {array} string "Content Types"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/content_types [get]
func readContentTypes(c *gin.Context) {
	types, err := api.ContentService.ReadAllTypes(c)
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: api.ContentService.EntityType,
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read content types: %w", err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.JSON(http.StatusOK, types)
}

// readContentAuthors returns a list of Content authors for which contents exist.
// It's useful for paging through contents by author.
//
// @Summary List Content Authors
// @Description List Content Authors
// @Description List content authors, for which contents exist.
// @Tags Content
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Success 200 {array} string "Content Authors"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/content_authors [get]
func readContentAuthors(c *gin.Context) {
	authors, err := api.ContentService.ReadAllAuthors(c)
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: api.ContentService.EntityType,
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read content authors: %w", err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.JSON(http.StatusOK, authors)
}

// readContentEditors returns a list of Content editors for which contents exist.
// It's useful for paging through contents by editor.
//
// @Summary List Content Editors
// @Description List Content Editors
// @Description List content editors (IDs and names), for which contents exist.
// @Tags Content
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param sorted query bool false "Sort by Name? (not paginated; default: false)"
// @Success 200 {array} v.TextValue "Content Editor IDs and Names"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/content_editors [get]
func readContentEditors(c *gin.Context) {
	sortByValue, err := strconv.ParseBool(c.DefaultQuery("sorted", "false"))
	if err != nil {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid parameter, sorted: %w", err))
		return
	}
	editors, err := api.ContentService.ReadAllEditorNames(c, sortByValue)
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: api.ContentService.EntityType,
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read content authors: %w", err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.JSON(http.StatusOK, editors)
}

// readContentTags returns a list of Content tags for which contents exist.
// It's useful for paging through contents by tag.
//
// @Summary List Content Tags
// @Description List Content Tags
// @Description List content tags, for which contents exist.
// @Tags Content
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Success 200 {array} string "Content Tags"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/content_tags [get]
func readContentTags(c *gin.Context) {
	tags, err := api.ContentService.ReadAllTags(c)
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: api.ContentService.EntityType,
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read content tags: %w", err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.JSON(http.StatusOK, tags)
}

// readContentTitles returns a paginated list of Content titles.
//
// @Summary List Content Titles
// @Description List Content Titles
// @Description List Content Titles by type, author, or tag, paging with reverse, limit, and offset.
// @Description Optionally, filter results with search terms.
// @Tags Content
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param type query string false "Type" Enums(BOOK, CHAPTER, ARTICLE, CATEGORY)
// @Param author query string false "Author Name"
// @Param editor query string false "Editor ID"
// @Param tag query string false "Tag"
// @Param search query string false "Search Terms, separated by spaces"
// @Param any query bool false "Any Match? (default: false; all search terms must match)"
// @Param sorted query bool false "Sort by Title? (not paginated; default: false)"
// @Param reverse query bool false "Reverse Order (default: false)"
// @Param limit query int false "Limit (omit for all)"
// @Param offset query string false "Offset (default: forward/reverse alphanumeric)"
// @Success 200 {array} v.TextValue "Content IDs and Titles"
// @Failure 400 {object} APIEvent "Bad Request (invalid parameter)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/content_titles [get]
func readContentTitles(c *gin.Context) {
	// Parse query parameters, with defaults
	reverse, limit, offset, err := paginationParams(c, false, 1000)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}
	// Partition query parameters
	typ := c.Query("type")
	tag := c.Query("tag")
	author := c.Query("author")
	editorID := c.Query("editor")
	if editorID != "" && !tuid.IsValid(tuid.TUID(editorID)) {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("invalid editor ID: %s", editorID))
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
	// Read the Content Titles
	var titles []v.TextValue
	var errMessage string
	if typ != "" {
		if search != "" {
			errMessage = fmt.Sprintf("search (%s) content titles by type (%s)", search, typ)
			titles, err = api.ContentService.FilterTitlesByType(c, typ, search, anyMatch)
		} else if all {
			errMessage = fmt.Sprintf("read all content titles by type (%s)", typ)
			titles, err = api.ContentService.ReadAllTitlesByType(c, typ, sortByValue)
		} else {
			errMessage = fmt.Sprintf("read content titles by type (%s)", typ)
			titles, err = api.ContentService.ReadTitlesByType(c, typ, reverse, limit, offset)
		}
	} else if author != "" {
		if search != "" {
			errMessage = fmt.Sprintf("search (%s) content titles by author (%s)", search, author)
			titles, err = api.ContentService.FilterTitlesByAuthor(c, author, search, anyMatch)
		} else if all {
			errMessage = fmt.Sprintf("read all content titles by author (%s)", author)
			titles, err = api.ContentService.ReadAllTitlesByAuthor(c, author, sortByValue)
		} else {
			errMessage = fmt.Sprintf("read content titles by author (%s)", author)
			titles, err = api.ContentService.ReadTitlesByAuthor(c, author, reverse, limit, offset)
		}
	} else if editorID != "" {
		if search != "" {
			errMessage = fmt.Sprintf("search (%s) content titles by editor (%s)", search, editorID)
			titles, err = api.ContentService.FilterTitlesByEditorID(c, editorID, search, anyMatch)
		} else if all {
			errMessage = fmt.Sprintf("read all content titles by editor (%s)", editorID)
			titles, err = api.ContentService.ReadAllTitlesByEditorID(c, editorID, sortByValue)
		} else {
			errMessage = fmt.Sprintf("read content titles by editor (%s)", editorID)
			titles, err = api.ContentService.ReadTitlesByEditorID(c, editorID, reverse, limit, offset)
		}
	} else if tag != "" {
		if search != "" {
			errMessage = fmt.Sprintf("search (%s) content titles by tag (%s)", search, tag)
			titles, err = api.ContentService.FilterTitlesByTag(c, tag, search, anyMatch)
		} else if all {
			errMessage = fmt.Sprintf("read all content titles by tag (%s)", tag)
			titles, err = api.ContentService.ReadAllTitlesByTag(c, tag, sortByValue)
		} else {
			errMessage = fmt.Sprintf("read content titles by tag (%s)", tag)
			titles, err = api.ContentService.ReadTitlesByTag(c, tag, reverse, limit, offset)
		}
	} else {
		if search != "" {
			errMessage = fmt.Sprintf("search (%s) content titles", search)
			titles, err = api.ContentService.FilterTitles(c, search, anyMatch)
		} else if all {
			errMessage = "read all content titles"
			titles, err = api.ContentService.ReadAllTitles(c, sortByValue)
		} else {
			errMessage = "read content titles"
			titles, err = api.ContentService.ReadTitles(c, reverse, limit, offset)
		}
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: api.ContentService.EntityType,
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("%s: %w", errMessage, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.JSON(http.StatusOK, titles)
}
