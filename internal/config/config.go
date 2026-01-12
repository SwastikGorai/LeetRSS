package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"leetcode-rss/internal/leetcode"
)

type Config struct {
	Server   ServerConfig
	LeetCode LeetCodeConfig
	Cache    CacheConfig
	Database DatabaseConfig
}

type DatabaseConfig struct {
	URL           string
	PublicBaseURL string
	RSSCacheTTL   time.Duration
}

type ServerConfig struct {
	Port           int
	HandlerTimeout time.Duration
}

type LeetCodeConfig struct {
	Usernames          []string
	MaxArticlesPerUser int
	GraphQLEndpoint    string
	Cookie             string
	CSRF               string
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

	usernames, err := parseUsernames(usernamesStr)
	if err != nil {
		return nil, err
	}

	maxArticlesPerUser := clampInt(GetEnv("LEETCODE_MAX_ARTICLES", 15).(int), 1, 50)

	cfg := &Config{
		Server: ServerConfig{
			Port:           GetEnv("PORT", 8080).(int),
			HandlerTimeout: GetEnv("HANDLER_TIMEOUT", 10*time.Second).(time.Duration),
		},
		LeetCode: LeetCodeConfig{
			Usernames:          usernames,
			MaxArticlesPerUser: maxArticlesPerUser,
			GraphQLEndpoint:    GetEnv("LEETCODE_GRAPHQL_ENDPOINT", "https://leetcode.com/graphql/").(string),
			Cookie:             GetEnv("LEETCODE_COOKIE", "").(string),
			CSRF:               GetEnv("LEETCODE_CSRF", "").(string),
		},
		Cache: CacheConfig{
			TTL: GetEnv("CACHE_TTL", 5*time.Minute).(time.Duration),
		},
		Database: DatabaseConfig{
			URL:           GetEnv("DATABASE_URL", "file:./data/leetrss.db?_journal=WAL").(string),
			PublicBaseURL: GetEnv("PUBLIC_BASE_URL", "http://localhost:8080").(string),
			RSSCacheTTL:   GetEnv("RSS_CACHE_TTL", 5*time.Minute).(time.Duration),
		},
	}

	return cfg, nil
}

func parseUsernames(s string) ([]string, error) {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	invalid := make([]string, 0)
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		if err := leetcode.ValidateUsername(trimmed); err != nil {
			invalid = append(invalid, trimmed)
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	if len(invalid) > 0 {
		return nil, fmt.Errorf("invalid username(s) in LEETCODE_USERNAMES: %s", strings.Join(invalid, ", "))
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no valid usernames found in LEETCODE_USERNAMES")
	}
	return result, nil
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

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
