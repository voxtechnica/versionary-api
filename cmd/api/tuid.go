package main

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/voxtechnica/tuid-go"
	"net/http"
	"strconv"
)

// registerTuidRoutes initializes the TUID routes.
func registerTuidRoutes(r *gin.Engine) {
	tuids := r.Group("/v1/tuids")
	{
		tuids.POST("", createTUID)
		tuids.GET("", readTUIDs)
		tuids.GET("/:id", readTUID)
	}
}

// createTUID generates a new TUID based on the current system time.
//
// @Summary Generate a new TUID
// @Description Generate a new TUID based on current system time and return the TUIDInfo.
// @Tags TUID
// @Produce json
// @Success 201 {object} tuid.TUIDInfo "TUIDInfo for the generated TUID"
// @Header 201 {string} Location "URL of the newly created TUID"
// @Router /v1/tuids [post]
func createTUID(c *gin.Context) {
	t := tuid.NewID()
	info, _ := t.Info() // generated IDs do not have parse errors
	c.Header("Location", c.Request.URL.String()+"/"+t.String())
	c.JSON(http.StatusCreated, info)
}

// readTUIDs generates the specified number of TUIDs, based on the current system time.
//
// @Summary Generate the specified number of TUIDs
// @Description Generate the specified number of TUIDs, based on the current system time.
// @Tags TUID
// @Produce json
// @Param limit query int false "Number of TUIDs (default: 5)"
// @Success 200 {array} tuid.TUIDInfo "TUIDInfo for the generated TUIDs"
// @Failure 400 {object} APIEvent "Invalid limit"
// @Router /v1/tuids [get]
func readTUIDs(c *gin.Context) {
	// Validate query parameters
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "5"))
	if err != nil || limit < 1 {
		abortWithError(c, http.StatusBadRequest, fmt.Errorf("bad request: invalid limit parameter: %s", c.Query("limit")))
		return
	}
	// Generate and return TUIDs
	ids := make([]tuid.TUIDInfo, limit)
	for i := 0; i < limit; i++ {
		ids[i], _ = tuid.NewID().Info()
	}
	c.JSON(http.StatusOK, ids)
}

// readTUID reads TUIDInfo for the specified TUID.
// It can be useful for extracting the timestamp from an ID.
//
// @Summary Read TUIDInfo for the provided TUID
// @Description Parse the provided TUID, returning the TUIDInfo. This can be useful for extracting the timestamp from an ID.
// @Tags TUID
// @Produce json
// @Param id path string true "TUID to parse (e.g. 9GEG9f25zjGI3ath)"
// @Success 200 {object} tuid.TUIDInfo "TUIDInfo for the provided TUID"
// @Failure 400 {object} APIEvent "Invalid TUID"
// @Router /v1/tuids/{id} [get]
func readTUID(c *gin.Context) {
	id := tuid.TUID(c.Param("id"))
	info, err := id.Info()
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err) // Parse error
		return
	}
	if !tuid.IsValid(id) {
		abortWithError(c, http.StatusBadRequest, errors.New("invalid TUID timestamp: "+info.Timestamp.String()))
		return
	}
	c.JSON(http.StatusOK, info)
}
