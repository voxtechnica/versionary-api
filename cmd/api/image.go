package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/voxtechnica/tuid-go"
	v "github.com/voxtechnica/versionary"

	"versionary-api/pkg/bucket"
	"versionary-api/pkg/event"
	"versionary-api/pkg/image"
	"versionary-api/pkg/ref"
	"versionary-api/pkg/user"
)

// registerImageRoutes initializes the Image routes.
func registerImageRoutes(r *gin.Engine) {
	r.POST("/v1/images", roleAuthorizer("admin"), createImage)
	r.GET("/v1/images", roleAuthorizer("admin"), readImages)
	r.GET("/v1/images/:id", readImage)
	r.HEAD("/v1/images/:id", existsImage)
	r.GET("/v1/images/:id/versions", roleAuthorizer("admin"), readImageVersions)
	r.GET("/v1/images/:id/versions/:versionid", readImageVersion)
	r.HEAD("/v1/images/:id/versions/:versionid", existsImageVersion)
	r.GET("/v1/images/:id/similar", roleAuthorizer("admin"), readSimilarImages)
	r.GET("/v1/images/:id/download_url", getImageDownloadURL)
	r.GET("/v1/images/:id/upload_url", roleAuthorizer("admin"), getImageUploadURL)
	r.PUT("/v1/images/:id", roleAuthorizer("admin"), updateImage)
	r.DELETE("/v1/images/:id", roleAuthorizer("admin"), deleteImage)
	r.DELETE("/v1/images/:id/versions/:versionid", roleAuthorizer("admin"), deleteImageVersion)
	r.GET("/v1/image_statuses", roleAuthorizer("admin"), readImageStatuses)
	r.GET("/v1/image_tags", roleAuthorizer("admin"), readImageTags)
	r.GET("/v1/image_labels", roleAuthorizer("admin"), readImageLabels)
}

// createImage creates a new Image.
//
// @Summary Create Image
// @Description Create a new Image
// @Description Create a new Image.
// @Tags Image
// @Accept json
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param image body image.Image true "Image"
// @Success 201 {object} image.Image "Newly-created Image"
// @Failure 400 {object} APIEvent "Bad Request (invalid JSON body)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 422 {object} APIEvent "Image validation errors"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Header 201 {string} Location "URL of the newly created Image"
// @Router /v1/images [post]
func createImage(c *gin.Context) {
	// Parse the request body as an Image
	var body image.Image
	if err := c.ShouldBindJSON(&body); err != nil {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid JSON body: %w", err))
		return
	}
	// Create a new Image
	i, problems, err := api.ImageService.Create(c, body)
	if len(problems) > 0 && err != nil {
		abortWithError(c, http.StatusUnprocessableEntity, fmt.Errorf("unprocessable entity: %w", err))
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   i.ID,
			EntityType: i.Type(),
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("create image %s: %w", i.ID, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	// Log the creation
	_, _, _ = api.EventService.Create(c, event.Event{
		UserID:     contextUserID(c),
		EntityID:   i.ID,
		EntityType: i.Type(),
		LogLevel:   event.INFO,
		Message:    fmt.Sprintf("created image %s %s", i.ID, i.Label()),
		URI:        c.Request.URL.String(),
	})
	// Return the new Image
	c.Header("Location", c.Request.URL.String()+"/"+i.ID)
	c.JSON(http.StatusCreated, i)
}

// readImages returns a paginated list of Images.
//
// @Summary List Images
// @Description List Images
// @Description List Images, paging with reverse, limit, and offset. Optionally, filter by status.
// @Tags Image
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param status query string false "Status" Enums(PENDING, UPLOADED, COMPLETE, ERROR)
// @Param tag query string false "Tag"
// @Param reverse query bool false "Reverse Order (default: false)"
// @Param limit query int false "Limit (default: 100)"
// @Param offset query string false "Offset (default: forward/reverse alphanumeric)"
// @Success 200 {array} image.Image "Images"
// @Failure 400 {object} APIEvent "Bad Request (invalid parameter)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/images [get]
func readImages(c *gin.Context) {
	// Parse query parameters, with defaults
	reverse, limit, offset, err := paginationParams(c, false, 100)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}
	status := strings.ToUpper(c.Query("status"))
	if status != "" && !user.Status(status).IsValid() {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid status: %s", status))
		return
	}
	tag := c.Query("tag")
	// Read and return paginated Images
	if status != "" {
		images, err := api.ImageService.ReadImagesByStatusAsJSON(c, status, reverse, limit, offset)
		if err != nil {
			e, _, _ := api.EventService.Create(c, event.Event{
				UserID:     contextUserID(c),
				EntityType: api.ImageService.EntityType,
				LogLevel:   event.ERROR,
				Message:    fmt.Errorf("read images by status %s: %w", status, err).Error(),
				URI:        c.Request.URL.String(),
				Err:        err,
			})
			abortWithError(c, http.StatusInternalServerError, e)
			return
		}
		c.Data(http.StatusOK, "application/json;charset=UTF-8", images)
	} else if tag != "" {
		images, err := api.ImageService.ReadImagesByTagAsJSON(c, tag, reverse, limit, offset)
		if err != nil {
			e, _, _ := api.EventService.Create(c, event.Event{
				UserID:     contextUserID(c),
				EntityType: api.ImageService.EntityType,
				LogLevel:   event.ERROR,
				Message:    fmt.Errorf("read images by tag %s: %w", tag, err).Error(),
				URI:        c.Request.URL.String(),
				Err:        err,
			})
			abortWithError(c, http.StatusInternalServerError, e)
			return
		}
		c.Data(http.StatusOK, "application/json;charset=UTF-8", images)
	} else {
		images := api.ImageService.ReadImages(c, reverse, limit, offset)
		c.JSON(http.StatusOK, images)
	}
}

// readImage returns the current version of the specified Image.
//
// @Summary Read Image Metadata
// @Description Get Image Metadata.
// @Description Get Image Metadata by ID.
// @Tags Image
// @Produce json
// @Param id path string true "Image ID"
// @Success 200 {object} image.Image "Image"
// @Failure 400 {object} APIEvent "Bad Request (invalid path parameter ID)"
// @Failure 404 {object} APIEvent "Not Found"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/images/{id} [get]
func readImage(c *gin.Context) {
	// Validate the path parameter ID
	id := c.Param("id")
	if !tuid.IsValid(tuid.TUID(id)) {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid path parameter ID: %s", id))
		return
	}
	// Read and return the specified Image
	i, err := api.ImageService.ReadAsJSON(c, id)
	if err != nil && errors.Is(err, v.ErrNotFound) {
		abortWithError(c, http.StatusNotFound, fmt.Errorf("not found: image %s", id))
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   id,
			EntityType: api.ImageService.EntityType,
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read image %s: %w", id, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.Data(http.StatusOK, "application/json;charset=UTF-8", i)
}

// existsImage checks if the specified Image exists.
//
// @Summary Image Exists
// @Description Image Exists
// @Description Check if the specified Image exists.
// @Tags Image
// @Param id path string true "Image ID"
// @Success 204 "Image Exists"
// @Failure 400 "Bad Request (invalid path parameter ID)"
// @Failure 404 "Not Found"
// @Router /v1/images/{id} [head]
func existsImage(c *gin.Context) {
	id := c.Param("id")
	if !tuid.IsValid(tuid.TUID(id)) {
		c.Status(http.StatusBadRequest)
	} else if !api.ImageService.Exists(c, id) {
		c.Status(http.StatusNotFound)
	} else {
		c.Status(http.StatusNoContent)
	}
}

// readImageVersions returns a paginated list of versions of the specified Image.
//
// @Summary List Image Versions
// @Description Get Image Versions
// @Description Get Image Versions by ID, paging with reverse, limit, and offset.
// @Tags Image
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param id path string true "Image ID"
// @Param reverse query bool false "Reverse Order (default: false)"
// @Param limit query int false "Limit (default: 100)"
// @Param offset query string false "Offset (default: forward/reverse alphanumeric)"
// @Success 200 {array} image.Image "Image Versions"
// @Failure 400 {object} APIEvent "Bad Request (invalid parameter)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 404 {object} APIEvent "Not Found"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/images/{id}/versions [get]
func readImageVersions(c *gin.Context) {
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
	// Verify that the Image exists
	if !api.ImageService.Exists(c, id) {
		abortWithError(c, http.StatusNotFound, fmt.Errorf("not found: image %s", id))
		return
	}
	// Read and return the specified Image Versions
	versions, err := api.ImageService.ReadVersionsAsJSON(c, id, reverse, limit, offset)
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   id,
			EntityType: api.ImageService.EntityType,
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read image %s versions: %w", id, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.Data(http.StatusOK, "application/json;charset=UTF-8", versions)
}

// readImageVersion returns the specified version of the specified Image.
//
// @Summary Read Image Version
// @Description Get Image Version
// @Description Get Image Version by ID and VersionID.
// @Tags Image
// @Produce json
// @Param id path string true "Image ID"
// @Param versionid path string true "Image VersionID"
// @Success 200 {object} image.Image "Image Version"
// @Failure 400 {object} APIEvent "Bad Request (invalid path parameter)"
// @Failure 404 {object} APIEvent "Not Found"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/images/{id}/versions/{versionid} [get]
func readImageVersion(c *gin.Context) {
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
	// Read and return the Image Version
	version, err := api.ImageService.ReadVersionAsJSON(c, id, versionid)
	if err != nil && errors.Is(err, v.ErrNotFound) {
		abortWithError(c, http.StatusNotFound, fmt.Errorf("not found: image %s version %s", id, versionid))
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   id,
			EntityType: api.ImageService.EntityType,
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read image %s version %s: %w", id, versionid, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.Data(http.StatusOK, "application/json;charset=UTF-8", version)
}

// existsImageVersion checks if the specified Image version exists.
//
// @Summary Image Version Exists
// @Description Image Version Exists
// @Description Check if the specified Image version exists.
// @Tags Image
// @Param id path string true "Image ID"
// @Param versionid path string true "Image VersionID"
// @Success 204 "Image Version Exists"
// @Failure 400 "Bad Request (invalid path parameter)"
// @Failure 404 "Not Found"
// @Router /v1/images/{id}/versions/{versionid} [head]
func existsImageVersion(c *gin.Context) {
	id := c.Param("id")
	versionid := c.Param("versionid")
	if !tuid.IsValid(tuid.TUID(id)) || !tuid.IsValid(tuid.TUID(versionid)) {
		c.Status(http.StatusBadRequest)
	} else if !api.ImageService.VersionExists(c, id, versionid) {
		c.Status(http.StatusNotFound)
	} else {
		c.Status(http.StatusNoContent)
	}
}

// readSimilarImages returns a list of similar Distance objects for the specified Image.
//
// @Summary Find Similar Images
// @Description Find Similar Images
// @Description Find similar Images, within the specified perceptual hash distance.
// @Tags Image
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param maxdist query int false "Maximum Distance (range: 0-64, default: 16)" default(16) minimum(0) maximum(64)
// @Param limit query int false "Limit (default: 20, max: 100)" default(20) maximum(100)
// @Success 200 {array} image.Distance "Image Distances"
// @Failure 400 {object} APIEvent "Bad Request (invalid path parameter ID)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 404 {object} APIEvent "Not Found"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/images/{id}/similar [get]
func readSimilarImages(c *gin.Context) {
	// Validate the path parameter ID
	id := c.Param("id")
	if !tuid.IsValid(tuid.TUID(id)) {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid path parameter ID: %s", id))
		return
	}
	// Validate the query parameters
	max, err := strconv.Atoi(c.DefaultQuery("maxdist", "16"))
	if err != nil || max < 0 || max > 64 {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid parameter, maxdist (expect 0-64): %w", err))
		return
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if err != nil || limit < 1 || limit > 100 {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid parameter, limit (expect 1-100): %w", err))
		return
	}
	// Read the specified Image
	i, err := api.ImageService.Read(c, id)
	if err != nil && errors.Is(err, v.ErrNotFound) {
		abortWithError(c, http.StatusNotFound, fmt.Errorf("not found: image %s", id))
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   id,
			EntityType: api.ImageService.EntityType,
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read similar images %s: %w", id, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	// Find similar Images
	similar, err := api.ImageService.FindSimilarImages(c, i.PHash, max, limit)
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   id,
			EntityType: api.ImageService.EntityType,
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read similar images %s: %w", id, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.JSON(http.StatusOK, similar)
}

// getImageDownloadURL returns the Image download URL.
//
// @Summary Get Image Download URL
// @Description Get Image Download URL
// @Description Get a pre-signed file download URL for the specified Image.
// @Tags Image
// @Produce json
// @Param id path string true "Image ID"
// @Success 200 {object} bucket.PreSignedURL "Image Download URL"
// @Failure 400 {object} APIEvent "Bad Request (invalid path parameter ID)"
// @Failure 404 {object} APIEvent "Not Found"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/images/{id}/download_url [get]
func getImageDownloadURL(c *gin.Context) {
	// Validate the path parameter ID
	id := c.Param("id")
	if !tuid.IsValid(tuid.TUID(id)) {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid path parameter ID: %s", id))
		return
	}
	// Get the Image download URL
	url, err := api.ImageService.DownloadURL(c, id, 10*time.Minute)
	if err != nil && errors.Is(err, v.ErrNotFound) {
		abortWithError(c, http.StatusNotFound, fmt.Errorf("not found: image %s", id))
		return
	}
	if err != nil && errors.Is(err, bucket.ErrFileNotFound) {
		abortWithError(c, http.StatusNotFound, fmt.Errorf("not found: image file %s", id))
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   id,
			EntityType: api.ImageService.EntityType,
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("get image %s download url: %w", id, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.JSON(http.StatusOK, url)
}

// getImageUploadURL returns the Image upload URL.
//
// @Summary Get Image Upload URL
// @Description Get Image Upload URL
// @Description Get a pre-signed file upload URL for the specified Image.
// @Tags Image
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param id path string true "Image ID"
// @Success 200 {object} bucket.PreSignedURL "Image Download URL"
// @Failure 400 {object} APIEvent "Bad Request (invalid path parameter ID)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 404 {object} APIEvent "Not Found"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/images/{id}/upload_url [get]
func getImageUploadURL(c *gin.Context) {
	// Validate the path parameter ID
	id := c.Param("id")
	if !tuid.IsValid(tuid.TUID(id)) {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid path parameter ID: %s", id))
		return
	}
	// Get the Image upload URL
	url, err := api.ImageService.UploadURL(c, id, 10*time.Minute)
	if err != nil && errors.Is(err, v.ErrNotFound) {
		abortWithError(c, http.StatusNotFound, fmt.Errorf("not found: image %s", id))
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   id,
			EntityType: api.ImageService.EntityType,
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("get image %s upload url: %w", id, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.JSON(http.StatusOK, url)
}

// updateImage updates and returns the specified Image.
// Note that the updated version needs to be complete; this is not a partial update (e.g. PATCH).
//
// @Summary Update Image
// @Description Update Image
// @Description Update the provided, complete Image.
// @Tags Image
// @Accept json
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param image body image.Image true "Image"
// @Param id path string true "Image ID"
// @Success 200 {object} image.Image "Image"
// @Failure 400 {object} APIEvent "Bad Request (invalid JSON or parameter)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 422 {object} APIEvent "Image validation errors"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/images/{id} [put]
func updateImage(c *gin.Context) {
	// Parse the request body as an Image
	var body image.Image
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
	// The path parameter ID must match the Image ID
	if body.ID != id {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: path parameter ID %s does not match image ID %s", id, body.ID))
		return
	}
	// Update the specified Image
	i, problems, err := api.ImageService.Update(c, body)
	if len(problems) > 0 && err != nil {
		abortWithError(c, http.StatusUnprocessableEntity, fmt.Errorf("unprocessable entity: %w", err))
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   i.ID,
			EntityType: i.Type(),
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("update image %s: %w", i.ID, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	// Log the update
	_, _, _ = api.EventService.Create(c, event.Event{
		UserID:     contextUserID(c),
		EntityID:   i.ID,
		EntityType: i.Type(),
		LogLevel:   event.INFO,
		Message:    fmt.Sprintf("updated image %s %s", i.ID, i.Label()),
		URI:        c.Request.URL.String(),
	})
	// Return the updated Image
	c.JSON(http.StatusOK, i)
}

// deleteImage deletes the specified Image.
//
// @Summary Delete Image
// @Description Delete Image
// @Description Delete and return the specified Image.
// @Description The associated Image file is also deleted.
// @Tags Image
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param id path string true "Image ID"
// @Success 200 {object} image.Image "Image that was deleted"
// @Failure 400 {object} APIEvent "Bad Request (invalid path parameter ID)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 404 {object} APIEvent "Not Found"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/images/{id} [delete]
func deleteImage(c *gin.Context) {
	// Validate the path parameter ID
	id := c.Param("id")
	if !tuid.IsValid(tuid.TUID(id)) {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid path parameter ID: %s", id))
		return
	}
	// Delete the specified Image
	i, err := api.ImageService.Delete(c, id)
	if err != nil && errors.Is(err, v.ErrNotFound) {
		abortWithError(c, http.StatusNotFound, fmt.Errorf("not found: image %s", id))
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   id,
			EntityType: api.ImageService.EntityType,
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("delete image %s: %w", id, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	// Log the deletion
	_, _, _ = api.EventService.Create(c, event.Event{
		UserID:     contextUserID(c),
		EntityID:   i.ID,
		EntityType: i.Type(),
		LogLevel:   event.INFO,
		Message:    fmt.Sprintf("deleted image %s %s", i.ID, i.Label()),
		URI:        c.Request.URL.String(),
	})
	// Return the deleted image
	c.JSON(http.StatusOK, i)
}

// deleteImageVersion returns the specified version of the specified Image.
//
// @Summary Delete Image Version
// @Description Delete Image Version
// @Description Delete Image Version by ID and VersionID.
// @Description Note that the associated Image file is not deleted.
// @Tags Image
// @Produce json
// @Param id path string true "Image ID"
// @Param versionid path string true "Image VersionID"
// @Success 200 {object} image.Image "Image Version"
// @Failure 400 {object} APIEvent "Bad Request (invalid path parameter)"
// @Failure 404 {object} APIEvent "Not Found"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/images/{id}/versions/{versionid} [delete]
func deleteImageVersion(c *gin.Context) {
	// Validate the path parameters
	id := c.Param("id")
	versionid := c.Param("versionid")
	refID, err := ref.NewRefID(api.ImageService.EntityType, id, versionid)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid path parameter ID: %w", err))
		return
	}
	// Delete and return the Image Version
	version, err := api.ImageService.DeleteVersion(c, id, versionid)
	if err != nil && errors.Is(err, v.ErrNotFound) {
		abortWithError(c, http.StatusNotFound, fmt.Errorf("not found: %s", refID))
		return
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityID:   id,
			EntityType: api.ImageService.EntityType,
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("delete %s: %w", refID, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.JSON(http.StatusOK, version)
}

// readImageStatuses returns a list of status codes for which images exist.
// It's useful for paging through images by status.
//
// @Summary List Image Statuses
// @Description Get Image Statuses
// @Description Get a complete list of status codes for which images exist.
// @Tags Image
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Success 200 {array} string "Image Statuses"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/image_statuses [get]
func readImageStatuses(c *gin.Context) {
	statuses, err := api.ImageService.ReadAllStatuses(c)
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: api.ImageService.EntityType,
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read image statuses: %w", err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.JSON(http.StatusOK, statuses)
}

// readImageTags returns a list of tags for which images exist.
// It's useful for paging through images by tag.
//
// @Summary List Image Tags
// @Description Get Image Tags
// @Description Get a complete list of tags for which images exist.
// @Tags Image
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Success 200 {array} string "Image Tags"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/image_tags [get]
func readImageTags(c *gin.Context) {
	tags, err := api.ImageService.ReadAllTags(c)
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: api.ImageService.EntityType,
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("read image tags: %w", err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.JSON(http.StatusOK, tags)
}

// readImageLabels returns a list of images labels, optionally filtered with search terms.
//
// @Summary List Image Labels
// @Description Get Image Labels
// @Description Get a list of Image Labels, optionally filtered with search terms.
// @Tags Image
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param search query string false "Search Terms, separated by spaces"
// @Param any query bool false "Any Match? (default: false; all search terms must match)"
// @Param sorted query bool false "Sort by Value? (not paginated; default: false)"
// @Param reverse query bool false "Reverse Order (default: false)"
// @Param limit query int false "Limit (default: 1000)"
// @Param offset query string false "Offset (default: forward/reverse alphanumeric)"
// @Success 200 {array} v.TextValue "Image Labels"
// @Failure 400 {object} APIEvent "Bad Request (invalid parameter)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Internal Server Error"
// @Router /v1/image_labels [get]
func readImageLabels(c *gin.Context) {
	// Pagination query parameters, with defaults
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
	// Read the Image Labels
	var labels []v.TextValue
	var errMessage string
	if search != "" {
		errMessage = fmt.Sprintf("search (%s) image labels", search)
		labels, err = api.ImageService.FilterImageLabels(c, search, anyMatch)
	} else if all {
		errMessage = "read all image labels"
		labels, err = api.ImageService.ReadAllImageLabels(c, sortByValue)
	} else {
		errMessage = fmt.Sprintf("read %d image labels", limit)
		labels, err = api.ImageService.ReadImageLabels(c, reverse, limit, offset)
	}
	if err != nil {
		e, _, _ := api.EventService.Create(c, event.Event{
			UserID:     contextUserID(c),
			EntityType: api.ImageService.EntityType,
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("%s: %w", errMessage, err).Error(),
			URI:        c.Request.URL.String(),
			Err:        err,
		})
		abortWithError(c, http.StatusInternalServerError, e)
		return
	}
	c.JSON(http.StatusOK, labels)
}
