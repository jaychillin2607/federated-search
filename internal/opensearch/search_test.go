package opensearch

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildSearchQuery(t *testing.T) {
	body, err := BuildSearchQuery("crossfit")
	require.NoError(t, err)

	var got map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &got))

	assert.EqualValues(t, 0, got["size"])

	query := got["query"].(map[string]interface{})
	mm := query["multi_match"].(map[string]interface{})
	assert.Equal(t, "crossfit", mm["query"])
	assert.Equal(t, "AUTO", mm["fuzziness"])
	assert.Equal(t, "or", mm["operator"])

	fields := mm["fields"].([]interface{})
	assert.Equal(t, []interface{}{"title^3", "keywords^2", "description"}, fields)

	aggs := got["aggs"].(map[string]interface{})
	by := aggs["by_category"].(map[string]interface{})
	terms := by["terms"].(map[string]interface{})
	assert.Equal(t, "category", terms["field"])
	assert.EqualValues(t, 20, terms["size"])

	inner := by["aggs"].(map[string]interface{})
	top := inner["top_hits"].(map[string]interface{})
	th := top["top_hits"].(map[string]interface{})
	assert.EqualValues(t, 5, th["size"])
}

func TestParseSearchResponse(t *testing.T) {
	raw := []byte(`{
      "aggregations": {
        "by_category": {
          "buckets": [
            {
              "key": "gym",
              "top_hits": {
                "hits": {
                  "hits": [
                    {"_id":"gym_1","_score":1.2,"_source":{"id":"gym_1","category":"gym","title":"Low","description":"d","deeplink":"/g/1"}},
                    {"_id":"gym_2","_score":9.4,"_source":{"id":"gym_2","category":"gym","title":"High","description":"d","deeplink":"/g/2","metadata":{"rating":4.5}}}
                  ]
                }
              }
            },
            {
              "key": "empty",
              "top_hits": {"hits": {"hits": []}}
            },
            {
              "key": "feature",
              "top_hits": {
                "hits": {
                  "hits": [
                    {"_id":"feature_cart","_score":2.0,"_source":{"id":"feature_cart","category":"feature","title":"Cart","deeplink":"/app/cart"}}
                  ]
                }
              }
            }
          ]
        }
      }
    }`)

	results, err := ParseSearchResponse(raw)
	require.NoError(t, err)

	// empty bucket should be omitted
	_, hasEmpty := results["empty"]
	assert.False(t, hasEmpty, "categories with zero hits must be omitted")

	gyms, ok := results["gym"]
	require.True(t, ok)
	require.Len(t, gyms, 2)
	// sorted by score desc
	assert.Equal(t, "gym_2", gyms[0].ID)
	assert.Equal(t, "gym_1", gyms[1].ID)
	assert.Equal(t, 9.4, gyms[0].Score)
	assert.Equal(t, 4.5, gyms[0].Metadata["rating"])

	feats, ok := results["feature"]
	require.True(t, ok)
	require.Len(t, feats, 1)
	assert.Equal(t, "feature_cart", feats[0].ID)
	assert.Equal(t, "Cart", feats[0].Title)
}
