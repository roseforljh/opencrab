package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	App  AppConfig
	DB   DBConfig
	HTTP HTTPConfig
}

type AppConfig struct {
	Name        string
	Environment string
}

type HTTPConfig struct {
	Address string
}

type DBConfig struct {
	Path string
}

func Load() Config {
	return Config{
		App: AppConfig{
			Name:        getEnv("OPENCRAB_APP_NAME", "OpenCrab"),
			Environment: getEnv("OPENCRAB_ENV", "development"),
		},
		DB: DBConfig{
			Path: getEnv("OPENCRAB_DB_PATH", "./data/opencrab.db"),
		},
		HTTP: HTTPConfig{
			Address: getEnv("OPENCRAB_HTTP_ADDR", ":8080"),
		},
	}
}

func Validate(cfg Config) error {
	if strings.TrimSpace(cfg.App.Name) == "" {
		return fmt.Errorf("应用名称不能为空")
	}

	environment := strings.TrimSpace(cfg.App.Environment)
	if environment == "" {
		return fmt.Errorf("运行环境不能为空")
	}

	switch environment {
	case "development", "test", "production":
	default:
		return fmt.Errorf("不支持的运行环境: %s", cfg.App.Environment)
	}

	address := strings.TrimSpace(cfg.HTTP.Address)
	if address == "" {
		return fmt.Errorf("HTTP 监听地址不能为空")
	}

	if !strings.Contains(address, ":") {
		return fmt.Errorf("HTTP 监听地址格式不正确: %s", cfg.HTTP.Address)
	}

	if strings.TrimSpace(cfg.DB.Path) == "" {
		return fmt.Errorf("数据库路径不能为空")
	}

	return nil
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
