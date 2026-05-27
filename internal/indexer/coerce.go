package indexer

import (
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"
)

func toString(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case []byte:
		return string(x)
	default:
		return ""
	}
}

func toInt64(v any) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case int32:
		return int64(x)
	case int16:
		return int64(x)
	case int:
		return int64(x)
	case float64:
		return int64(x)
	case string:
		if i, err := strconv.ParseInt(x, 10, 64); err == nil {
			return i
		}
	}
	return 0
}

func toFloat(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int64:
		return float64(x)
	case int32:
		return float64(x)
	case int:
		return float64(x)
	case pgtype.Numeric:
		if f, err := x.Float64Value(); err == nil {
			return f.Float64
		}
	case string:
		if f, err := strconv.ParseFloat(x, 64); err == nil {
			return f
		}
	}
	return 0
}
