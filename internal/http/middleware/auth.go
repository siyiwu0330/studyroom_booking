package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"studyroom/internal/service"
)

func Auth(svc service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		tok, _ := c.Cookie("session_token")
		u, err := svc.CurrentUser(tok)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "not logged in"})
			return
		}
		c.Set("user", u)
		c.Next()
	}
}
