package provider

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"opencrab/internal/domain"
)

func TestOpenRouterRealChannel(t *testing.T) {
	key := os.Getenv("OPENROUTER_TEST_KEY")
	if key == "" {
		t.Skip("OPENROUTER_TEST_KEY not set")
	}

	tester := NewChannelTester(testHTTPClient())
	_, err := tester.TestChannel(context.Background(), domain.UpstreamChannel{
		Name:     "openrouter-test",
		Provider: "OpenRouter",
		Endpoint: "https://openrouter.ai/api/v1",
		APIKey:   key,
	}, "qwen/qwen3.6-plus:free")

	if err == nil {
		t.Log("real openrouter request succeeded")
		return
	}

	t.Fatalf("real openrouter request failed: %v", err)
}

func TestKimiRealChannel(t *testing.T) {
	key := os.Getenv("KIMI_TEST_KEY")
	if key == "" {
		t.Skip("KIMI_TEST_KEY not set")
	}

	tester := NewChannelTester(testHTTPClient())
	_, err := tester.TestChannel(context.Background(), domain.UpstreamChannel{
		Name:     "kimi-test",
		Provider: "KIMI",
		Endpoint: "https://api.moonshot.cn/v1",
		APIKey:   key,
	}, "kimi-k2.5")

	if err == nil {
		t.Log("real kimi request succeeded")
		return
	}

	t.Fatalf("real kimi request failed: %v", err)
}

func testHTTPClient() *http.Client {
	return &http.Client{Timeout: 60 * time.Second}
}
