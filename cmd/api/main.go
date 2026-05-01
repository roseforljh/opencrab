package main

import (
	"log"
	"net/http"

	"opencrab/internal/anthropic"
	"opencrab/internal/config"
	"opencrab/internal/gemini"
	"opencrab/internal/gateway"
	"opencrab/internal/httpserver"
	"opencrab/internal/openai"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	provider := openai.NewClient(cfg.Gateway.OpenAI.Timeout)
	claudeProvider := anthropic.NewClient(
		cfg.Gateway.Claude.Version,
		cfg.Gateway.Claude.Timeout,
	)
	geminiProvider := gemini.NewClient(cfg.Gateway.Gemini.Timeout)
	if err := httpserver.InitCompatStorage(cfg.StatePath); err != nil {
		log.Fatalf("初始化兼容存储失败: %v", err)
	}
	service := gateway.NewService(gateway.NewCompositeProvider(provider, claudeProvider, geminiProvider))
	handler := httpserver.NewRouter(service)

	server := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: handler,
	}

	log.Printf("opencrab api listening on %s", cfg.HTTPAddr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("启动应用失败: %v", err)
	}
}
