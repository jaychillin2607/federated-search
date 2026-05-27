package opensearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/jaychillin2607/federated-search/internal/model"
	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"
)

// BulkResult summarizes the outcome of a _bulk call.
type BulkResult struct {
	Indexed int
	Failed  int
	Errors  []BulkItemError
}

type BulkItemError struct {
	ID     string `json:"id"`
	Reason string `json:"reason"`
}

// BulkUpsert sends one _bulk request with index actions, upserting by _id.
func (c *Client) BulkUpsert(ctx context.Context, docs []model.Document) (BulkResult, error) {
	if len(docs) == 0 {
		return BulkResult{}, nil
	}
	var buf bytes.Buffer
	for _, d := range docs {
		meta := map[string]map[string]string{
			"index": {"_index": c.index, "_id": d.ID},
		}
		mb, err := json.Marshal(meta)
		if err != nil {
			return BulkResult{}, fmt.Errorf("marshal bulk meta: %w", err)
		}
		buf.Write(mb)
		buf.WriteByte('\n')
		db, err := json.Marshal(d)
		if err != nil {
			return BulkResult{}, fmt.Errorf("marshal doc %s: %w", d.ID, err)
		}
		buf.Write(db)
		buf.WriteByte('\n')
	}

	req := opensearchapi.BulkRequest{Body: bytes.NewReader(buf.Bytes())}
	resp, err := req.Do(ctx, c.raw)
	if err != nil {
		return BulkResult{}, fmt.Errorf("bulk: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return BulkResult{}, fmt.Errorf("read bulk response: %w", err)
	}
	if resp.IsError() {
		return BulkResult{}, fmt.Errorf("bulk http error %s: %s", resp.Status(), strings.TrimSpace(string(body)))
	}

	var parsed struct {
		Errors bool `json:"errors"`
		Items  []map[string]struct {
			ID     string `json:"_id"`
			Status int    `json:"status"`
			Result string `json:"result"`
			Error  *struct {
				Type   string `json:"type"`
				Reason string `json:"reason"`
			} `json:"error"`
		} `json:"items"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return BulkResult{}, fmt.Errorf("decode bulk response: %w", err)
	}

	out := BulkResult{}
	for _, it := range parsed.Items {
		for _, v := range it {
			if v.Error != nil {
				out.Failed++
				out.Errors = append(out.Errors, BulkItemError{ID: v.ID, Reason: v.Error.Reason})
			} else {
				out.Indexed++
			}
		}
	}
	return out, nil
}

// DeleteByID deletes a single document. Returns (found, error).
func (c *Client) DeleteByID(ctx context.Context, id string) (bool, error) {
	req := opensearchapi.DeleteRequest{Index: c.index, DocumentID: id}
	resp, err := req.Do(ctx, c.raw)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("read delete response: %w", err)
	}
	if resp.StatusCode == 404 {
		return false, nil
	}
	if resp.IsError() {
		return false, fmt.Errorf("delete http error %s: %s", resp.Status(), strings.TrimSpace(string(body)))
	}
	var parsed struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return false, fmt.Errorf("decode delete response: %w", err)
	}
	return parsed.Result != "not_found", nil
}

// BulkDelete deletes many ids in one _bulk request. Returns counts of deleted vs not_found.
func (c *Client) BulkDelete(ctx context.Context, ids []string) (deleted int, notFound int, err error) {
	if len(ids) == 0 {
		return 0, 0, nil
	}
	var buf bytes.Buffer
	for _, id := range ids {
		meta := map[string]map[string]string{
			"delete": {"_index": c.index, "_id": id},
		}
		mb, mErr := json.Marshal(meta)
		if mErr != nil {
			return 0, 0, fmt.Errorf("marshal delete meta: %w", mErr)
		}
		buf.Write(mb)
		buf.WriteByte('\n')
	}
	req := opensearchapi.BulkRequest{Body: bytes.NewReader(buf.Bytes())}
	resp, dErr := req.Do(ctx, c.raw)
	if dErr != nil {
		return 0, 0, fmt.Errorf("bulk delete: %w", dErr)
	}
	defer resp.Body.Close()
	body, rErr := io.ReadAll(resp.Body)
	if rErr != nil {
		return 0, 0, fmt.Errorf("read bulk delete response: %w", rErr)
	}
	if resp.IsError() {
		return 0, 0, fmt.Errorf("bulk delete http error %s: %s", resp.Status(), strings.TrimSpace(string(body)))
	}
	var parsed struct {
		Items []map[string]struct {
			Result string `json:"result"`
			Status int    `json:"status"`
		} `json:"items"`
	}
	if jErr := json.Unmarshal(body, &parsed); jErr != nil {
		return 0, 0, fmt.Errorf("decode bulk delete response: %w", jErr)
	}
	for _, it := range parsed.Items {
		for _, v := range it {
			if v.Result == "deleted" {
				deleted++
			} else if v.Result == "not_found" || v.Status == 404 {
				notFound++
			}
		}
	}
	return deleted, notFound, nil
}
