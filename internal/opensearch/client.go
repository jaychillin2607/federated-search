package opensearch

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"
)

const indexMapping = `{
  "settings": {
    "number_of_shards": 1,
    "number_of_replicas": 0
  },
  "mappings": {
    "properties": {
      "id":          { "type": "keyword" },
      "category":    { "type": "keyword" },
      "title":       { "type": "text", "analyzer": "standard" },
      "description": { "type": "text", "analyzer": "standard" },
      "keywords":    { "type": "text", "analyzer": "standard" },
      "deeplink":    { "type": "keyword", "index": false },
      "metadata":    { "type": "object", "enabled": false }
    }
  }
}`

type Client struct {
	raw   *opensearch.Client
	index string
}

type Options struct {
	URL                string
	Username           string
	Password           string
	Index              string
	InsecureSkipVerify bool
}

func New(opts Options) (*Client, error) {
	cfg := opensearch.Config{
		Addresses: []string{opts.URL},
	}
	if opts.Username != "" {
		cfg.Username = opts.Username
		cfg.Password = opts.Password
	}
	if opts.InsecureSkipVerify {
		cfg.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	raw, err := opensearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("opensearch client: %w", err)
	}
	return &Client{raw: raw, index: opts.Index}, nil
}

func (c *Client) Index() string { return c.index }

func (c *Client) Raw() *opensearch.Client { return c.raw }

// EnsureIndex creates the configured index if it does not already exist.
func (c *Client) EnsureIndex(ctx context.Context) error {
	existsReq := opensearchapi.IndicesExistsRequest{Index: []string{c.index}}
	existsResp, err := existsReq.Do(ctx, c.raw)
	if err != nil {
		return fmt.Errorf("indices.exists: %w", err)
	}
	existsResp.Body.Close()
	if existsResp.StatusCode == 200 {
		slog.Info("opensearch index already exists", "index", c.index)
		return nil
	}
	if existsResp.StatusCode != 404 {
		return fmt.Errorf("indices.exists unexpected status: %d", existsResp.StatusCode)
	}

	createReq := opensearchapi.IndicesCreateRequest{
		Index: c.index,
		Body:  strings.NewReader(indexMapping),
	}
	createResp, err := createReq.Do(ctx, c.raw)
	if err != nil {
		return fmt.Errorf("indices.create: %w", err)
	}
	defer createResp.Body.Close()
	if createResp.IsError() {
		body, _ := io.ReadAll(createResp.Body)
		// Tolerate a concurrent creator (e.g. server and indexer racing).
		if strings.Contains(string(body), "resource_already_exists_exception") {
			slog.Info("opensearch index created by concurrent process", "index", c.index)
			return nil
		}
		return fmt.Errorf("indices.create error: %s: %s", createResp.Status(), string(body))
	}
	slog.Info("opensearch index created", "index", c.index)
	return nil
}

// Ping calls the cluster info endpoint and returns nil if reachable.
func (c *Client) Ping(ctx context.Context) error {
	req := opensearchapi.InfoRequest{}
	resp, err := req.Do(ctx, c.raw)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.IsError() {
		return fmt.Errorf("opensearch info: %s", resp.Status())
	}
	return nil
}
