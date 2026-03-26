package model

import (
	"bufio"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

//go:embed default_config.yaml
var defaultConfigFS embed.FS

type Config struct {
	Port              string
	Debug             bool
	SessionTTL        time.Duration
	CleanupInterval   time.Duration
	HTTPTimeout       time.Duration
	PollMaxRetries    int
	PollRetryInterval time.Duration
	UserAgent         string
	Referer           string
	BiliGenerateURL   string
	BiliPollURL       string
	RedisAddr         string
	RedisPassword     string
	RedisDB           int
}

func LoadConfig() Config {
	cfg := loadInternalDefaultConfig()

	if data, ok := loadTopLevelConfigData(); ok {
		applyMapConfig(&cfg, data)
	}

	applyEnvOverrides(&cfg)
	return cfg
}

func loadInternalDefaultConfig() Config {
	data, err := defaultConfigFS.ReadFile("default_config.yaml")
	if err != nil {
		panic(fmt.Errorf("read internal default config failed: %w", err))
	}

	cfg := Config{}
	applyMapConfig(&cfg, parseSimpleYAML(data))
	return cfg
}

func loadTopLevelConfigData() (map[string]string, bool) {
	for _, path := range topLevelConfigCandidates() {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		return parseSimpleYAML(data), true
	}
	return nil, false
}

func topLevelConfigCandidates() []string {
	candidates := []string{"config.yaml"}
	wd, err := os.Getwd()
	if err != nil {
		return candidates
	}

	for dir := wd; ; dir = filepath.Dir(dir) {
		candidates = append(candidates, filepath.Join(dir, "config.yaml"))
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return uniqueStrings(candidates)
}

func uniqueStrings(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

func parseSimpleYAML(data []byte) map[string]string {
	result := map[string]string{}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		val = strings.Trim(val, `"`)
		result[key] = val
	}
	return result
}

func applyMapConfig(cfg *Config, raw map[string]string) {
	if v, ok := raw["port"]; ok && strings.TrimSpace(v) != "" {
		cfg.Port = strings.TrimSpace(v)
	}
	if v, ok := raw["debug"]; ok {
		if b, err := strconv.ParseBool(strings.TrimSpace(v)); err == nil {
			cfg.Debug = b
		}
	}
	if v, ok := raw["session_ttl"]; ok {
		if d, err := time.ParseDuration(strings.TrimSpace(v)); err == nil && d > 0 {
			cfg.SessionTTL = d
		}
	}
	if v, ok := raw["cleanup_interval"]; ok {
		if d, err := time.ParseDuration(strings.TrimSpace(v)); err == nil && d > 0 {
			cfg.CleanupInterval = d
		}
	}
	if v, ok := raw["http_timeout"]; ok {
		if d, err := time.ParseDuration(strings.TrimSpace(v)); err == nil && d > 0 {
			cfg.HTTPTimeout = d
		}
	}
	if v, ok := raw["poll_max_retries"]; ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n >= 0 {
			cfg.PollMaxRetries = n
		}
	}
	if v, ok := raw["poll_retry_interval"]; ok {
		if d, err := time.ParseDuration(strings.TrimSpace(v)); err == nil && d > 0 {
			cfg.PollRetryInterval = d
		}
	}
	if v, ok := raw["user_agent"]; ok && strings.TrimSpace(v) != "" {
		cfg.UserAgent = strings.TrimSpace(v)
	}
	if v, ok := raw["referer"]; ok && strings.TrimSpace(v) != "" {
		cfg.Referer = strings.TrimSpace(v)
	}
	if v, ok := raw["bili_generate_url"]; ok && strings.TrimSpace(v) != "" {
		cfg.BiliGenerateURL = strings.TrimSpace(v)
	}
	if v, ok := raw["bili_poll_url"]; ok && strings.TrimSpace(v) != "" {
		cfg.BiliPollURL = strings.TrimSpace(v)
	}
	if v, ok := raw["redis_addr"]; ok {
		cfg.RedisAddr = strings.TrimSpace(v)
	}
	if v, ok := raw["redis_password"]; ok {
		cfg.RedisPassword = strings.TrimSpace(v)
	}
	if v, ok := raw["redis_db"]; ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n >= 0 {
			cfg.RedisDB = n
		}
	}
}

func applyEnvOverrides(cfg *Config) {
	cfg.Port = getEnv("PORT", cfg.Port)
	cfg.Debug = getEnv("DEBUG", strconv.FormatBool(cfg.Debug)) == "true"
	cfg.SessionTTL = getEnvDuration("SESSION_TTL", cfg.SessionTTL)
	cfg.CleanupInterval = getEnvDuration("CLEANUP_INTERVAL", cfg.CleanupInterval)
	cfg.HTTPTimeout = getEnvDuration("HTTP_TIMEOUT", cfg.HTTPTimeout)
	cfg.PollMaxRetries = getEnvInt("POLL_MAX_RETRIES", cfg.PollMaxRetries)
	cfg.PollRetryInterval = getEnvDuration("POLL_RETRY_INTERVAL", cfg.PollRetryInterval)
	cfg.UserAgent = getEnv("USER_AGENT", cfg.UserAgent)
	cfg.Referer = getEnv("REFERER", cfg.Referer)
	cfg.BiliGenerateURL = getEnv("BILI_GENERATE_URL", cfg.BiliGenerateURL)
	cfg.BiliPollURL = getEnv("BILI_POLL_URL", cfg.BiliPollURL)
	cfg.RedisAddr = getEnv("REDIS_ADDR", cfg.RedisAddr)
	cfg.RedisPassword = getEnv("REDIS_PASSWORD", cfg.RedisPassword)
	cfg.RedisDB = getEnvInt("REDIS_DB", cfg.RedisDB)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil && d > 0 {
			return d
		}
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil && n >= 0 {
			return n
		}
	}
	return fallback
}
