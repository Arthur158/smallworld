package roomworker

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HealthAddr     string
	WorkerID       uint32
	WorkerCount    uint32
	RedisAddr      string
	RedisURL       string
	DatabaseURL    string
	CommandBlock   time.Duration
	SnapshotTTL    time.Duration
	DisplayRoomTTL time.Duration
	ConsumerGroup  string
	ConsumerName   string
}

func LoadConfig() (Config, error) {
	cfg := Config{
		HealthAddr:   envOr("HEALTH_ADDR", ":8081"),
		RedisAddr:    envOr("REDIS_ADDR", "localhost:6379"),
		RedisURL:     os.Getenv("REDIS_URL"),
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		CommandBlock: 5 * time.Second,
		// Normal rooms default to no Redis expiry. The room is explicitly
		// removed when the last player leaves; this avoids a quiet game
		// disappearing simply because nobody made a move for a day.
		SnapshotTTL:    0,
		DisplayRoomTTL: 2 * time.Hour,
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	count, err := strconv.ParseUint(envOr("ROOM_WORKER_COUNT", "1"), 10, 32)
	if err != nil || count == 0 {
		return Config{}, fmt.Errorf("ROOM_WORKER_COUNT must be a positive integer")
	}
	cfg.WorkerCount = uint32(count)

	workerIDRaw := os.Getenv("WORKER_ID")
	if workerIDRaw == "" {
		workerIDRaw = workerIDFromPodName(os.Getenv("POD_NAME"))
	}
	if workerIDRaw == "" {
		workerIDRaw = "0"
	}

	id, err := strconv.ParseUint(workerIDRaw, 10, 32)
	if err != nil {
		return Config{}, fmt.Errorf("WORKER_ID must be a non-negative integer: %w", err)
	}
	if id >= count {
		return Config{}, fmt.Errorf("WORKER_ID %d must be smaller than ROOM_WORKER_COUNT %d", id, count)
	}
	cfg.WorkerID = uint32(id)
	cfg.ConsumerGroup = fmt.Sprintf("room-worker-%d", cfg.WorkerID)
	cfg.ConsumerName = fmt.Sprintf("worker-%d", cfg.WorkerID)

	if value := os.Getenv("COMMAND_BLOCK"); value != "" {
		d, err := time.ParseDuration(value)
		if err != nil || d < 0 {
			return Config{}, fmt.Errorf("COMMAND_BLOCK must be a non-negative Go duration")
		}
		cfg.CommandBlock = d
	}
	if value := os.Getenv("ROOM_SNAPSHOT_TTL"); value != "" {
		d, err := time.ParseDuration(value)
		if err != nil || d < 0 {
			return Config{}, fmt.Errorf("ROOM_SNAPSHOT_TTL must be a non-negative Go duration")
		}
		cfg.SnapshotTTL = d
	}
	if value := os.Getenv("DISPLAY_ROOM_TTL"); value != "" {
		d, err := time.ParseDuration(value)
		if err != nil || d < 0 {
			return Config{}, fmt.Errorf("DISPLAY_ROOM_TTL must be a non-negative Go duration")
		}
		cfg.DisplayRoomTTL = d
	}

	return cfg, nil
}

func workerIDFromPodName(name string) string {
	if name == "" {
		return ""
	}
	idx := strings.LastIndex(name, "-")
	if idx < 0 || idx == len(name)-1 {
		return ""
	}
	suffix := name[idx+1:]
	if _, err := strconv.ParseUint(suffix, 10, 32); err != nil {
		return ""
	}
	return suffix
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
