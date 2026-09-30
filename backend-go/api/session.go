package api

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/locallhosts/Wraith/backend-go/auth"
)

func getSession(c *gin.Context) {
	id, ok := auth.GetIdentity(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthenticated"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"label":id.Label,"role":id.Role})
}
