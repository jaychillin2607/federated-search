package indexer

import (
	"fmt"

	"github.com/jaychillin2607/federated-search/internal/model"
)

// RegisteredSources returns the canonical list of indexer sources.
func RegisteredSources() []Source {
	return []Source{
		{
			Name:     "turfs",
			Category: model.CategoryTurf,
			SQL: `SELECT id, name, description, address, price_per_hour, updated_at, deleted_at
                    FROM turfs
                   WHERE updated_at >= $1
                   ORDER BY updated_at ASC
                   LIMIT $2`,
			Transform: func(row map[string]any) (model.Document, error) {
				id := toInt64(row["id"])
				name := toString(row["name"])
				return model.Document{
					ID:          fmt.Sprintf("turf_%d", id),
					Category:    model.CategoryTurf,
					Title:       name,
					Description: toString(row["description"]),
					Keywords:    []string{"turf", name},
					Deeplink:    fmt.Sprintf("/app/turfs/%d", id),
					Metadata: map[string]interface{}{
						"address":        toString(row["address"]),
						"price_per_hour": toFloat(row["price_per_hour"]),
					},
				}, nil
			},
		},
		{
			Name:     "gyms",
			Category: model.CategoryGym,
			SQL: `SELECT id, name, description, address, monthly_price, rating, updated_at, deleted_at
                    FROM gyms
                   WHERE updated_at >= $1
                   ORDER BY updated_at ASC
                   LIMIT $2`,
			Transform: func(row map[string]any) (model.Document, error) {
				id := toInt64(row["id"])
				name := toString(row["name"])
				return model.Document{
					ID:          fmt.Sprintf("gym_%d", id),
					Category:    model.CategoryGym,
					Title:       name,
					Description: toString(row["description"]),
					Keywords:    []string{"gym", name},
					Deeplink:    fmt.Sprintf("/app/gyms/%d", id),
					Metadata: map[string]interface{}{
						"address":       toString(row["address"]),
						"monthly_price": toFloat(row["monthly_price"]),
						"rating":        toFloat(row["rating"]),
					},
				}, nil
			},
		},
		{
			Name:     "coaches",
			Category: model.CategoryCoach,
			SQL: `SELECT id, name, bio, specialization, hourly_rate, experience_years, updated_at, deleted_at
                    FROM coaches
                   WHERE updated_at >= $1
                   ORDER BY updated_at ASC
                   LIMIT $2`,
			Transform: func(row map[string]any) (model.Document, error) {
				id := toInt64(row["id"])
				name := toString(row["name"])
				spec := toString(row["specialization"])
				return model.Document{
					ID:          fmt.Sprintf("coach_%d", id),
					Category:    model.CategoryCoach,
					Title:       name,
					Description: toString(row["bio"]),
					Keywords:    []string{"coach", spec},
					Deeplink:    fmt.Sprintf("/app/coaches/%d", id),
					Metadata: map[string]interface{}{
						"specialization":   spec,
						"hourly_rate":      toFloat(row["hourly_rate"]),
						"experience_years": toInt64(row["experience_years"]),
					},
				}, nil
			},
		},
		{
			Name:     "classes",
			Category: model.CategoryClass,
			SQL: `SELECT id, name, description, duration_min, level, updated_at, deleted_at
                    FROM classes
                   WHERE updated_at >= $1
                   ORDER BY updated_at ASC
                   LIMIT $2`,
			Transform: func(row map[string]any) (model.Document, error) {
				id := toInt64(row["id"])
				name := toString(row["name"])
				level := toString(row["level"])
				return model.Document{
					ID:          fmt.Sprintf("class_%d", id),
					Category:    model.CategoryClass,
					Title:       name,
					Description: toString(row["description"]),
					Keywords:    []string{"class", level},
					Deeplink:    fmt.Sprintf("/app/classes/%d", id),
					Metadata: map[string]interface{}{
						"duration_min": toInt64(row["duration_min"]),
						"level":        level,
					},
				}, nil
			},
		},
	}
}
