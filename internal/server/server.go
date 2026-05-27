package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jaychillin2607/federated-search/internal/handler"
	"github.com/jaychillin2607/federated-search/internal/middleware"
	osq "github.com/jaychillin2607/federated-search/internal/opensearch"
)

type Deps struct {
	OpenSearch *osq.Client
	APIKey     string
	Checkpoint handler.CheckpointAPI
	Sources    handler.SourceLister
}

func NewRouter(deps Deps) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	r.GET("/healthz", handler.Healthz)
	r.GET("/readyz", handler.Readyz(deps.OpenSearch))

	api := r.Group("/api/v1")
	{
		api.GET("/search", handler.Search(deps.OpenSearch))
		api.GET("/search/categories", handler.Categories)
	}

	internal := r.Group("/internal/v1", middleware.APIKey(deps.APIKey))
	{
		internal.POST("/documents", handler.UpsertDocuments(deps.OpenSearch))
		internal.DELETE("/documents/:id", handler.DeleteDocument(deps.OpenSearch))
		internal.POST("/documents/bulk-delete", handler.BulkDeleteDocuments(deps.OpenSearch))
		if deps.Checkpoint != nil && deps.Sources != nil {
			internal.GET("/sync/status", handler.SyncStatus(deps.Checkpoint))
			internal.POST("/reindex", handler.Reindex(deps.Checkpoint, deps.Sources))
		}
	}

	return r
}

// Run starts the HTTP server and blocks until ctx is canceled.
func Run(ctx context.Context, port int, r *gin.Engine) error {
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: r,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("http server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server shutdown: %w", err)
		}
		return nil
	case err := <-errCh:
		return err
	}
}
