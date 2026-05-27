package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Port                         int
	LogLevel                     string
	OpenSearchURL                string
	OpenSearchUsername           string
	OpenSearchPassword           string
	OpenSearchIndex              string
	OpenSearchInsecureSkipVerify bool
	PostgresDSN                  string
	InternalAPIKey               string
	IndexerIntervalSeconds       int
	IndexerBatchSize             int
}

func Load() (*Config, error) {
	v := viper.New()
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	v.SetDefault("PORT", 8080)
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("OPENSEARCH_URL", "http://localhost:9200")
	v.SetDefault("OPENSEARCH_USERNAME", "")
	v.SetDefault("OPENSEARCH_PASSWORD", "")
	v.SetDefault("OPENSEARCH_INDEX", "federated_search")
	v.SetDefault("OPENSEARCH_INSECURE_SKIP_VERIFY", false)
	v.SetDefault("POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/gym?sslmode=disable")
	v.SetDefault("INDEXER_INTERVAL_SECONDS", 60)
	v.SetDefault("INDEXER_BATCH_SIZE", 500)

	cfg := &Config{
		Port:                         v.GetInt("PORT"),
		LogLevel:                     v.GetString("LOG_LEVEL"),
		OpenSearchURL:                v.GetString("OPENSEARCH_URL"),
		OpenSearchUsername:           v.GetString("OPENSEARCH_USERNAME"),
		OpenSearchPassword:           v.GetString("OPENSEARCH_PASSWORD"),
		OpenSearchIndex:              v.GetString("OPENSEARCH_INDEX"),
		OpenSearchInsecureSkipVerify: v.GetBool("OPENSEARCH_INSECURE_SKIP_VERIFY"),
		PostgresDSN:                  v.GetString("POSTGRES_DSN"),
		InternalAPIKey:               v.GetString("INTERNAL_API_KEY"),
		IndexerIntervalSeconds:       v.GetInt("INDEXER_INTERVAL_SECONDS"),
		IndexerBatchSize:             v.GetInt("INDEXER_BATCH_SIZE"),
	}

	if cfg.InternalAPIKey == "" {
		return nil, fmt.Errorf("INTERNAL_API_KEY is required")
	}

	return cfg, nil
}
