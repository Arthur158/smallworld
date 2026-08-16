package gateway

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Config struct {
	ListenAddr      string
	RedisAddr       string
	RedisURL        string
	DatabaseURL     string
	GatewayID       string
	RoomWorkerCount uint32
	AllowedOrigins  map[string]struct{}
	AllowAllOrigins bool
	PresenceTTL     time.Duration
	PresenceRefresh time.Duration
	WriteWait       time.Duration
	PongWait        time.Duration
	PingPeriod      time.Duration
	MaxMessageBytes int64
}

func LoadConfig() (Config, error) {
	cfg := Config{
		ListenAddr:      envOr("LISTEN_ADDR", ":8080"),
		RedisAddr:       envOr("REDIS_ADDR", "localhost:6379"),
		RedisURL:        os.Getenv("REDIS_URL"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		PresenceTTL:     60 * time.Second,
		PresenceRefresh: 20 * time.Second,
		WriteWait:       10 * time.Second,
		PongWait:        60 * time.Second,
		PingPeriod:      45 * time.Second,
		MaxMessageBytes: 1 << 20,
	}

	workerCount, err := strconv.ParseUint(envOr("ROOM_WORKER_COUNT", "1"), 10, 32)
	if err != nil || workerCount == 0 {
		return Config{}, fmt.Errorf("ROOM_WORKER_COUNT must be a positive integer")
	}
	cfg.RoomWorkerCount = uint32(workerCount)

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	cfg.GatewayID = os.Getenv("GATEWAY_ID")
	if cfg.GatewayID == "" {
		if hostname, err := os.Hostname(); err == nil && hostname != "" {
			cfg.GatewayID = hostname
		} else {
			cfg.GatewayID = "gateway-" + uuid.NewString()
		}
	}

	allowed := strings.TrimSpace(os.Getenv("WS_ALLOWED_ORIGINS"))
	if allowed == "" || allowed == "*" {
		cfg.AllowAllOrigins = true
	} else {
		cfg.AllowedOrigins = make(map[string]struct{})
		for _, origin := range strings.Split(allowed, ",") {
			origin = strings.TrimSpace(origin)
			if origin != "" {
				cfg.AllowedOrigins[origin] = struct{}{}
			}
		}
	}

	return cfg, nil
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
