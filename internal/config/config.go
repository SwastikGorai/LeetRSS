package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	LeetCode LeetCodeConfig
	Cache    CacheConfig
}

type ServerConfig struct {
	Port int
}

type LeetCodeConfig struct {
	Username        string
	GraphQLEndpoint string
	Cookie          string
	CSRF            string
}

type CacheConfig struct {
	TTL time.Duration
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	username := os.Getenv("LEETCODE_USERNAME")
	if username == "" {
		return nil, fmt.Errorf("missing env LEETCODE_USERNAME")
	}

	cfg := &Config{
		Server: ServerConfig{
			Port: GetEnv("PORT", 8080).(int),
		},
		LeetCode: LeetCodeConfig{
			Username:        username,
			GraphQLEndpoint: GetEnv("LEETCODE_GRAPHQL_ENDPOINT", "https://leetcode.com/graphql/").(string),
			Cookie:          GetEnv("LEETCODE_COOKIE", "").(string),
			CSRF:            GetEnv("LEETCODE_CSRF", "").(string),
		},
		Cache: CacheConfig{
			TTL: GetEnv("CACHE_TTL", 5*time.Minute).(time.Duration),
		},
	}

	return cfg, nil
}

func GetEnv(key string, defaultValue any) any {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}

	switch def := defaultValue.(type) {
	case string:
		return value
	case int:
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
		return def
	case bool:
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
		return def
	case time.Duration:
		if durationValue, err := time.ParseDuration(value); err == nil {
			return durationValue
		}
		return def
	default:
		panic(fmt.Sprintf("unsupported type %T", defaultValue))
	}
}
