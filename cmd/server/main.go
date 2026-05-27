package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jaychillin2607/federated-search/internal/config"
	"github.com/jaychillin2607/federated-search/internal/handler"
	"github.com/jaychillin2607/federated-search/internal/indexer"
	osq "github.com/jaychillin2607/federated-search/internal/opensearch"
	"github.com/jaychillin2607/federated-search/internal/postgres"
	"github.com/jaychillin2607/federated-search/internal/server"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "err", err)
		os.Exit(1)
	}

	osClient, err := osq.New(osq.Options{
		URL:                cfg.OpenSearchURL,
		Username:           cfg.OpenSearchUsername,
		Password:           cfg.OpenSearchPassword,
		Index:              cfg.OpenSearchIndex,
		InsecureSkipVerify: cfg.OpenSearchInsecureSkipVerify,
	})
	if err != nil {
		slog.Error("opensearch client init failed", "err", err)
		os.Exit(1)
	}

	bootstrapCtx, cancelBootstrap := context.WithCancel(context.Background())
	if err := osClient.EnsureIndex(bootstrapCtx); err != nil {
		cancelBootstrap()
		slog.Error("ensure index failed", "err", err)
		os.Exit(1)
	}
	cancelBootstrap()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	deps := server.Deps{
		OpenSearch: osClient,
		APIKey:     cfg.InternalAPIKey,
	}

	// Postgres is optional in the server binary: admin endpoints only mount when reachable.
	pool, err := postgres.NewPool(ctx, cfg.PostgresDSN)
	if err != nil {
		slog.Warn("postgres unavailable, admin endpoints disabled", "err", err)
	} else {
		defer pool.Close()
		checkpoint := postgres.NewCheckpointStore(pool)
		deps.Checkpoint = checkpoint
		deps.Sources = func() []string {
			names := make([]string, 0, len(indexer.RegisteredSources()))
			for _, s := range indexer.RegisteredSources() {
				names = append(names, s.Name)
			}
			return names
		}
		_ = handler.SyncStatus // to ensure linker keeps it
	}

	r := server.NewRouter(deps)

	if err := server.Run(ctx, cfg.Port, r); err != nil {
		slog.Error("server exited with error", "err", err)
		os.Exit(1)
	}
	slog.Info("server stopped cleanly")
}
