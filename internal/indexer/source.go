package indexer

import "github.com/jaychillin2607/federated-search/internal/model"

// Source describes how to extract documents for one resource type from Postgres.
type Source struct {
	Name      string
	Category  string
	SQL       string
	Transform func(row map[string]any) (model.Document, error)
}
