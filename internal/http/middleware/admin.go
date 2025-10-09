package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"studyroom/internal/models"
)

func Admin() gin.HandlerFunc {
	return func(c *gin.Context) {
		v, ok := c.Get("user")
		if !ok { c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error":"not logged in"}); return }
		u := v.(*models.User)
		if !u.IsAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error":"admin only"}); return
		}
		c.Next()
	}
}
