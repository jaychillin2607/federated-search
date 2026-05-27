package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jaychillin2607/federated-search/internal/postgres"
)

// CheckpointAPI is the subset of CheckpointStore the admin handlers need.
type CheckpointAPI interface {
	List(ctx context.Context) ([]postgres.SyncState, error)
	ResetToEpoch(ctx context.Context, source string) error
	ResetAllToEpoch(ctx context.Context) error
}

type syncStatusItem struct {
	Name             string    `json:"name"`
	LastSyncedAt     time.Time `json:"last_synced_at"`
	LastSuccessCount int       `json:"last_success_count"`
	LastError        *string   `json:"last_error"`
}

type syncStatusResponse struct {
	Sources []syncStatusItem `json:"sources"`
}

type reindexRequest struct {
	Source string `json:"source"`
}

type reindexResponse struct {
	JobID  string `json:"job_id"`
	Source string `json:"source"`
	Status string `json:"status"`
}

// validSources is used to reject reindex requests for sources that aren't registered.
type SourceLister func() []string

func SyncStatus(store CheckpointAPI) gin.HandlerFunc {
	return func(c *gin.Context) {
		states, err := store.List(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "checkpoint_unavailable",
				"message": err.Error(),
			})
			return
		}
		out := syncStatusResponse{Sources: make([]syncStatusItem, 0, len(states))}
		for _, s := range states {
			out.Sources = append(out.Sources, syncStatusItem{
				Name:             s.Name,
				LastSyncedAt:     s.LastSyncedAt,
				LastSuccessCount: s.LastSuccessCount,
				LastError:        s.LastError,
			})
		}
		c.JSON(http.StatusOK, out)
	}
}

func Reindex(store CheckpointAPI, sources SourceLister) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req reindexRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_json", "message": err.Error()})
			return
		}
		if req.Source == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "source is required"})
			return
		}

		if req.Source == "all" {
			if err := store.ResetAllToEpoch(c.Request.Context()); err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"error":   "checkpoint_unavailable",
					"message": err.Error(),
				})
				return
			}
		} else {
			known := false
			for _, name := range sources() {
				if name == req.Source {
					known = true
					break
				}
			}
			if !known {
				c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "unknown source"})
				return
			}
			if err := store.ResetToEpoch(c.Request.Context(), req.Source); err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"error":   "checkpoint_unavailable",
					"message": err.Error(),
				})
				return
			}
		}

		c.JSON(http.StatusAccepted, reindexResponse{
			JobID:  uuid.NewString(),
			Source: req.Source,
			Status: "queued",
		})
	}
}
