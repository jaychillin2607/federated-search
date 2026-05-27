package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newRouter(expected string) *gin.Engine {
	r := gin.New()
	r.Use(APIKey(expected))
	r.GET("/x", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

func TestAPIKey_Missing(t *testing.T) {
	r := newRouter("devkey")
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPIKey_Wrong(t *testing.T) {
	r := newRouter("devkey")
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set(HeaderName, "nope")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPIKey_Correct(t *testing.T) {
	r := newRouter("devkey")
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set(HeaderName, "devkey")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"ok":true`)
}
