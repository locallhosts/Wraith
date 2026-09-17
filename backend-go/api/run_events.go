package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func listRunEvents(s *Server, c *gin.Context) {
	limit := 500
	if raw := c.Query("limit"); raw != "" {
		var parsed int
		if _, err := fmt.Sscanf(raw, "%d", &parsed); err == nil && parsed > 0 && parsed <= 2000 {
			limit = parsed
		}
	}
	events, err := s.Store.ListRunEvents(c.Request.Context(), c.Param("id"), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, events)
}
