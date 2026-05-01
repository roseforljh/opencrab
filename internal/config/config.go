package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr string
	StatePath string
	Gateway  GatewayConfig
}

type GatewayConfig struct {
	OpenAI   OpenAIConfig
	Claude   ClaudeConfig
	Gemini   GeminiConfig
}

type OpenAIConfig struct {
	Timeout time.Duration
}

type ClaudeConfig struct {
	Version string
	Timeout time.Duration
}

type GeminiConfig struct {
	Timeout time.Duration
}

func Load() (Config, error) {
	timeout := 120 * time.Second
	if raw := strings.TrimSpace(os.Getenv("OPENCRAB_UPSTREAM_TIMEOUT")); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			return Config{}, fmt.Errorf("解析 OPENCRAB_UPSTREAM_TIMEOUT 失败: %w", err)
		}
		timeout = parsed
	}

	return Config{
		HTTPAddr: envOrDefault("OPENCRAB_HTTP_ADDR", ":8080"),
		StatePath: resolveStatePath(),
		Gateway: GatewayConfig{
			OpenAI: OpenAIConfig{
				Timeout: timeout,
			},
			Claude: ClaudeConfig{
				Version: envOrDefault("OPENCRAB_ANTHROPIC_VERSION", "2023-06-01"),
				Timeout: timeout,
			},
			Gemini: GeminiConfig{
				Timeout: timeout,
			},
		},
	}, nil
}

func resolveStatePath() string {
	if value := strings.TrimSpace(os.Getenv("OPENCRAB_STATE_PATH")); value != "" {
		return value
	}
	if value := strings.TrimSpace(os.Getenv("OPENCRAB_DB_PATH")); value != "" {
		return value
	}
	return "./data/opencrab-state.json"
}

func envOrDefault(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
