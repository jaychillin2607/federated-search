package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jaychillin2607/federated-search/internal/model"
	osq "github.com/jaychillin2607/federated-search/internal/opensearch"
)

const (
	maxDocsPerRequest = 100
	maxIDLen          = 128
	maxTitleLen       = 256
	maxDescLen        = 2000
)

// DocumentsWriter is the subset of OpenSearch operations the documents handlers need.
// Defining an interface keeps handlers testable.
type DocumentsWriter interface {
	BulkUpsert(ctx context.Context, docs []model.Document) (osq.BulkResult, error)
	DeleteByID(ctx context.Context, id string) (bool, error)
	BulkDelete(ctx context.Context, ids []string) (deleted int, notFound int, err error)
}

type upsertRequest struct {
	Documents []model.Document `json:"documents"`
}

type upsertItemError struct {
	ID     string `json:"id"`
	Reason string `json:"reason"`
}

type upsertResponse struct {
	Indexed int               `json:"indexed"`
	Failed  int               `json:"failed"`
	Errors  []upsertItemError `json:"errors"`
}

type bulkDeleteRequest struct {
	IDs []string `json:"ids"`
}

type bulkDeleteResponse struct {
	Deleted  int `json:"deleted"`
	NotFound int `json:"not_found"`
}

// UpsertDocuments handles POST /internal/v1/documents.
func UpsertDocuments(w DocumentsWriter) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req upsertRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_json", "message": err.Error()})
			return
		}
		if len(req.Documents) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "documents must not be empty"})
			return
		}
		if len(req.Documents) > maxDocsPerRequest {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "bad_request",
				"message": "documents exceeds max of 100",
			})
			return
		}

		valid := make([]model.Document, 0, len(req.Documents))
		errors := make([]upsertItemError, 0)
		for i, d := range req.Documents {
			if reason := validateDocument(i, d); reason != "" {
				errors = append(errors, upsertItemError{ID: d.ID, Reason: reason})
				continue
			}
			valid = append(valid, d)
		}

		result := osq.BulkResult{}
		if len(valid) > 0 {
			r, err := w.BulkUpsert(c.Request.Context(), valid)
			if err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"error":   "search_unavailable",
					"message": err.Error(),
				})
				return
			}
			result = r
		}

		// merge bulk errors with validation errors
		for _, e := range result.Errors {
			errors = append(errors, upsertItemError{ID: e.ID, Reason: e.Reason})
		}

		resp := upsertResponse{
			Indexed: result.Indexed,
			Failed:  result.Failed + (len(req.Documents) - len(valid)),
			Errors:  errors,
		}

		status := http.StatusOK
		if resp.Failed > 0 {
			status = http.StatusMultiStatus
		}
		c.JSON(status, resp)
	}
}

// DeleteDocument handles DELETE /internal/v1/documents/:id.
func DeleteDocument(w DocumentsWriter) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := strings.TrimSpace(c.Param("id"))
		if id == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "id required"})
			return
		}
		found, err := w.DeleteByID(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "search_unavailable",
				"message": err.Error(),
			})
			return
		}
		if !found {
			c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
			return
		}
		c.Status(http.StatusNoContent)
	}
}

// BulkDeleteDocuments handles POST /internal/v1/documents/bulk-delete.
func BulkDeleteDocuments(w DocumentsWriter) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req bulkDeleteRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_json", "message": err.Error()})
			return
		}
		if len(req.IDs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "ids must not be empty"})
			return
		}
		if len(req.IDs) > maxDocsPerRequest {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "ids exceeds max of 100"})
			return
		}
		deleted, notFound, err := w.BulkDelete(c.Request.Context(), req.IDs)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "search_unavailable",
				"message": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, bulkDeleteResponse{Deleted: deleted, NotFound: notFound})
	}
}

func validateDocument(index int, d model.Document) string {
	if d.ID == "" {
		return "id is required"
	}
	if len(d.ID) > maxIDLen {
		return "id exceeds 128 chars"
	}
	if d.Category == "" {
		return "category is required"
	}
	if !model.IsValid(d.Category) {
		return "category is not in known set"
	}
	if d.Title == "" {
		return "title is required"
	}
	if len(d.Title) > maxTitleLen {
		return "title exceeds 256 chars"
	}
	if len(d.Description) > maxDescLen {
		return "description exceeds 2000 chars"
	}
	if d.Deeplink == "" {
		return "deeplink is required"
	}
	return ""
}
