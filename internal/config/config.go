package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
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
	Usernames      []string
	GraphQLEndpoint string
	Cookie          string
	CSRF            string
}

type CacheConfig struct {
	TTL time.Duration
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	usernamesStr := os.Getenv("LEETCODE_USERNAMES")
	if usernamesStr == "" {
		username := os.Getenv("LEETCODE_USERNAME")
		if username == "" {
			return nil, fmt.Errorf("missing env LEETCODE_USERNAMES or LEETCODE_USERNAME")
		}
		usernamesStr = username
	}

	usernames := parseUsernames(usernamesStr)
	if len(usernames) == 0 {
		return nil, fmt.Errorf("no valid usernames found in LEETCODE_USERNAMES")
	}

	cfg := &Config{
		Server: ServerConfig{
			Port: GetEnv("PORT", 8080).(int),
		},
		LeetCode: LeetCodeConfig{
			Usernames:      usernames,
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

func parseUsernames(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
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
