package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/voxtechnica/tuid-go"
)

// initTuidRoutes initializes the TUID routes.
func initTuidRoutes(r *gin.Engine) {
	tuids := r.Group("/v1/tuids")
	{
		tuids.POST("", createTUID)
		tuids.GET("", readTUIDs)
		tuids.GET("/:id", readTUID)
	}
}

// createTUID creates a new TUID.
func createTUID(c *gin.Context) {
	t := tuid.NewID()
	info, err := t.Info()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":  http.StatusInternalServerError,
			"error": fmt.Errorf("create TUID error: %w", err).Error(),
		})
		return
	}
	c.JSON(http.StatusOK, info)
}

// readTUIDs reads/creates the specified number of TUIDs.
func readTUIDs(c *gin.Context) {
	limit := c.DefaultQuery("limit", "5")
	intLimit, err := strconv.Atoi(limit)
	if err != nil {
		intLimit = 5
	}
	ids := make([]tuid.TUIDInfo, intLimit)
	for i := 0; i < intLimit; i++ {
		ids[i], err = tuid.NewID().Info()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":  http.StatusInternalServerError,
				"error": fmt.Errorf("read TUIDs error: %w", err).Error(),
			})
			return
		}
	}
	c.JSON(http.StatusOK, ids)
}

// readTUID reads TUIDInfo for the specified TUID.
// It can be useful for extracting the timestamp from an ID.
func readTUID(c *gin.Context) {
	id := tuid.TUID(c.Param("id"))
	info, err := id.Info()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":  http.StatusBadRequest,
			"error": fmt.Errorf("read TUID %s error: %w", id, err).Error(),
		})
		return
	}
	c.JSON(http.StatusOK, info)
}
