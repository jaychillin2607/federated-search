package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/jaychillin2607/federated-search/internal/config"
	"github.com/jaychillin2607/federated-search/internal/model"
	osq "github.com/jaychillin2607/federated-search/internal/opensearch"
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
		slog.Error("opensearch client failed", "err", err)
		os.Exit(1)
	}

	ctx := context.Background()
	if err := osClient.EnsureIndex(ctx); err != nil {
		slog.Error("ensure index failed", "err", err)
		os.Exit(1)
	}

	files := []string{
		filepath.Join("seeds", "features.json"),
		filepath.Join("seeds", "resources_sample.json"),
	}

	var all []model.Document
	for _, f := range files {
		docs, err := loadSeedFile(f)
		if err != nil {
			slog.Error("load seed file", "file", f, "err", err)
			os.Exit(1)
		}
		all = append(all, docs...)
		slog.Info("loaded seed file", "file", f, "count", len(docs))
	}

	if err := validateDocs(all); err != nil {
		slog.Error("seed validation failed", "err", err)
		os.Exit(1)
	}

	res, err := osClient.BulkUpsert(ctx, all)
	if err != nil {
		slog.Error("bulk index failed", "err", err)
		os.Exit(1)
	}
	slog.Info("seed complete", "indexed", res.Indexed, "failed", res.Failed)
	if res.Failed > 0 {
		os.Exit(1)
	}
}

func loadSeedFile(path string) ([]model.Document, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var docs []model.Document
	if err := json.Unmarshal(b, &docs); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	return docs, nil
}

func validateDocs(docs []model.Document) error {
	for i, d := range docs {
		if d.ID == "" {
			return fmt.Errorf("doc[%d]: id required", i)
		}
		if !model.IsValid(d.Category) {
			return fmt.Errorf("doc[%d] (%s): invalid category %q", i, d.ID, d.Category)
		}
		if d.Title == "" {
			return fmt.Errorf("doc[%d] (%s): title required", i, d.ID)
		}
		if d.Deeplink == "" {
			return fmt.Errorf("doc[%d] (%s): deeplink required", i, d.ID)
		}
	}
	return nil
}
