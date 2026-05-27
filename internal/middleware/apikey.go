package middleware

import (
	"crypto/subtle"
	"net/http"

	"github.com/gin-gonic/gin"
)

const HeaderName = "X-Internal-Api-Key"

// APIKey returns a gin middleware that rejects requests whose X-Internal-Api-Key
// header does not match the expected value (constant-time comparison).
func APIKey(expected string) gin.HandlerFunc {
	expectedBytes := []byte(expected)
	return func(c *gin.Context) {
		got := c.GetHeader(HeaderName)
		if got == "" || subtle.ConstantTimeCompare([]byte(got), expectedBytes) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Next()
	}
}
