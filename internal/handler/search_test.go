package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	osq "github.com/jaychillin2607/federated-search/internal/opensearch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSearcher struct {
	out map[string][]osq.SearchHit
	err error
}

func (f *fakeSearcher) Search(ctx context.Context, q string) (map[string][]osq.SearchHit, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.out, nil
}

func init() {
	gin.SetMode(gin.TestMode)
}

func TestSearchHappyPath(t *testing.T) {
	fake := &fakeSearcher{
		out: map[string][]osq.SearchHit{
			"gym": {
				{ID: "gym_1", Category: "gym", Title: "Test Gym", Deeplink: "/g/1", Score: 5.0},
			},
			"feature": {
				{ID: "feature_cart", Category: "feature", Title: "Cart", Deeplink: "/app/cart", Score: 3.1},
			},
		},
	}
	r := gin.New()
	r.GET("/api/v1/search", Search(fake))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp SearchResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "test", resp.Query)
	assert.Equal(t, 2, resp.Total)
	assert.Len(t, resp.Results, 2)
	assert.Equal(t, "gym_1", resp.Results["gym"][0].ID)
}

func TestSearchMissingQuery(t *testing.T) {
	r := gin.New()
	r.GET("/api/v1/search", Search(&fakeSearcher{}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSearchWhitespaceOnly(t *testing.T) {
	r := gin.New()
	r.GET("/api/v1/search", Search(&fakeSearcher{}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=%20%20", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
