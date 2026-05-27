package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jaychillin2607/federated-search/internal/model"
	osq "github.com/jaychillin2607/federated-search/internal/opensearch"
)

type SearchResponse struct {
	Query   string                     `json:"query"`
	Total   int                        `json:"total"`
	TookMs  int64                      `json:"took_ms"`
	Results map[string][]osq.SearchHit `json:"results"`
}

type CategoryEntry struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
}

type CategoriesResponse struct {
	Categories []CategoryEntry `json:"categories"`
}

func Search(s osq.Searcher) gin.HandlerFunc {
	return func(c *gin.Context) {
		q := strings.TrimSpace(c.Query("q"))
		if q == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "bad_request",
				"message": "query parameter 'q' is required",
			})
			return
		}

		start := time.Now()
		results, err := s.Search(c.Request.Context(), q)
		took := time.Since(start).Milliseconds()
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "search_unavailable",
				"message": err.Error(),
			})
			return
		}

		total := 0
		for _, hits := range results {
			total += len(hits)
		}
		if results == nil {
			results = map[string][]osq.SearchHit{}
		}

		c.JSON(http.StatusOK, SearchResponse{
			Query:   q,
			Total:   total,
			TookMs:  took,
			Results: results,
		})
	}
}

func Categories(c *gin.Context) {
	out := CategoriesResponse{Categories: make([]CategoryEntry, 0, len(model.All()))}
	for _, name := range model.All() {
		out.Categories = append(out.Categories, CategoryEntry{
			Name:        name,
			DisplayName: model.DisplayNames[name],
		})
	}
	c.JSON(http.StatusOK, out)
}
