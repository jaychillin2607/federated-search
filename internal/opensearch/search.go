package opensearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/jaychillin2607/federated-search/internal/model"
	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"
)

// SearchHit is returned to API consumers.
type SearchHit struct {
	ID          string                 `json:"id"`
	Category    string                 `json:"category"`
	Title       string                 `json:"title"`
	Description string                 `json:"description,omitempty"`
	Deeplink    string                 `json:"deeplink"`
	Score       float64                `json:"score"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Searcher is the interface handlers depend on so they can be tested without
// a real OpenSearch backend.
type Searcher interface {
	Search(ctx context.Context, q string) (map[string][]SearchHit, error)
}

// BuildSearchQuery returns the OpenSearch query body for the given user input.
// Exported so tests can verify the produced JSON.
func BuildSearchQuery(q string) ([]byte, error) {
	body := map[string]interface{}{
		"size": 0,
		"query": map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":     q,
				"fields":    []string{"title^3", "keywords^2", "description"},
				"fuzziness": "AUTO",
				"operator":  "or",
			},
		},
		"aggs": map[string]interface{}{
			"by_category": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "category",
					"size":  20,
				},
				"aggs": map[string]interface{}{
					"top_hits": map[string]interface{}{
						"top_hits": map[string]interface{}{
							"size": 5,
							"sort": []map[string]interface{}{
								{"_score": map[string]interface{}{"order": "desc"}},
							},
						},
					},
				},
			},
		},
	}
	return json.Marshal(body)
}

// rawAggResponse mirrors the structure of an OpenSearch aggregation response.
type rawAggResponse struct {
	Aggregations struct {
		ByCategory struct {
			Buckets []struct {
				Key     string `json:"key"`
				TopHits struct {
					Hits struct {
						Hits []struct {
							ID     string  `json:"_id"`
							Score  float64 `json:"_score"`
							Source struct {
								ID          string                 `json:"id"`
								Category    string                 `json:"category"`
								Title       string                 `json:"title"`
								Description string                 `json:"description"`
								Deeplink    string                 `json:"deeplink"`
								Metadata    map[string]interface{} `json:"metadata,omitempty"`
							} `json:"_source"`
						} `json:"hits"`
					} `json:"hits"`
				} `json:"top_hits"`
			} `json:"buckets"`
		} `json:"by_category"`
	} `json:"aggregations"`
}

// ParseSearchResponse converts a canned aggregation response into the grouped
// map expected by the API.
func ParseSearchResponse(raw []byte) (map[string][]SearchHit, error) {
	var parsed rawAggResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("decode search response: %w", err)
	}

	results := make(map[string][]SearchHit)
	for _, bucket := range parsed.Aggregations.ByCategory.Buckets {
		hits := bucket.TopHits.Hits.Hits
		if len(hits) == 0 {
			continue
		}
		out := make([]SearchHit, 0, len(hits))
		for _, h := range hits {
			id := h.Source.ID
			if id == "" {
				id = h.ID
			}
			category := h.Source.Category
			if category == "" {
				category = bucket.Key
			}
			out = append(out, SearchHit{
				ID:          id,
				Category:    category,
				Title:       h.Source.Title,
				Description: h.Source.Description,
				Deeplink:    h.Source.Deeplink,
				Score:       h.Score,
				Metadata:    h.Source.Metadata,
			})
		}
		sort.SliceStable(out, func(i, j int) bool { return out[i].Score > out[j].Score })
		results[bucket.Key] = out
	}
	return results, nil
}

// Search runs a multi-match aggregated query against the configured index.
func (c *Client) Search(ctx context.Context, q string) (map[string][]SearchHit, error) {
	body, err := BuildSearchQuery(q)
	if err != nil {
		return nil, err
	}
	req := opensearchapi.SearchRequest{
		Index: []string{c.index},
		Body:  bytes.NewReader(body),
	}
	resp, err := req.Do(ctx, c.raw)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.IsError() {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search error: %s: %s", resp.Status(), strings.TrimSpace(string(b)))
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read search response: %w", err)
	}
	return ParseSearchResponse(raw)
}

// compile-time check we can satisfy the model dependency direction
var _ = model.Document{}
