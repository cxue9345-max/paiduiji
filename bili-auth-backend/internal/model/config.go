package model

import (
	"os"
	"strconv"
	"time"
)

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
	cfg := Config{
		Port:              getEnv("PORT", "8080"),
		Debug:             getEnv("DEBUG", "false") == "true",
		SessionTTL:        getEnvDuration("SESSION_TTL", 180*time.Second),
		CleanupInterval:   getEnvDuration("CLEANUP_INTERVAL", 30*time.Second),
		HTTPTimeout:       getEnvDuration("HTTP_TIMEOUT", 8*time.Second),
		PollMaxRetries:    getEnvInt("POLL_MAX_RETRIES", 2),
		PollRetryInterval: getEnvDuration("POLL_RETRY_INTERVAL", 500*time.Millisecond),
		UserAgent:         getEnv("USER_AGENT", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"),
		Referer:           getEnv("REFERER", "https://www.bilibili.com/"),
		BiliGenerateURL:   getEnv("BILI_GENERATE_URL", "https://passport.bilibili.com/x/passport-login/web/qrcode/generate?source=main-fe-header"),
		BiliPollURL:       getEnv("BILI_POLL_URL", "https://passport.bilibili.com/x/passport-login/web/qrcode/poll"),
		RedisAddr:         getEnv("REDIS_ADDR", ""),
		RedisPassword:     getEnv("REDIS_PASSWORD", ""),
		RedisDB:           getEnvInt("REDIS_DB", 0),
	}
	return cfg
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
		if err == nil {
			return d
		}
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			return n
		}
	}
	return fallback
}
