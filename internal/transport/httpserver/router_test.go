package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"opencrab/internal/domain"
)

func TestCreateChannelRejectsInvalidPayload(t *testing.T) {
	router := NewRouter(Dependencies{
		CreateChannel: func(ctx context.Context, input domain.CreateChannelInput) (domain.Channel, error) {
			return domain.Channel{}, nil
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/channels", strings.NewReader(`{"name":""}`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateChannelAcceptsModelIDs(t *testing.T) {
	var captured domain.CreateChannelInput
	router := NewRouter(Dependencies{
		CreateChannel: func(ctx context.Context, input domain.CreateChannelInput) (domain.Channel, error) {
			captured = input
			return domain.Channel{ID: 1, Name: input.Name, Provider: input.Provider, Endpoint: input.Endpoint, Enabled: input.Enabled}, nil
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/channels", strings.NewReader(`{"name":"openai-main","provider":"OpenAI","endpoint":"https://api.openai.com/v1","api_key":"sk-test","enabled":true,"model_ids":["gpt-4.1","gpt-4.1-mini"]}`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}
	if len(captured.ModelIDs) != 2 {
		t.Fatalf("expected 2 model ids, got %d", len(captured.ModelIDs))
	}
	if captured.ModelIDs[0] != "gpt-4.1" || captured.ModelIDs[1] != "gpt-4.1-mini" {
		t.Fatalf("unexpected model ids: %#v", captured.ModelIDs)
	}
}

func TestCreateAPIKeyReturnsCreatedPayload(t *testing.T) {
	router := NewRouter(Dependencies{
		CreateAPIKey: func(ctx context.Context, input domain.CreateAPIKeyInput) (domain.CreatedAPIKey, error) {
			return domain.CreatedAPIKey{ID: 1, Name: input.Name, RawKey: "sk-opencrab-test", Enabled: input.Enabled}, nil
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/api-keys", strings.NewReader(`{"name":"web-console","enabled":true}`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	var payload domain.CreatedAPIKey
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if payload.RawKey == "" {
		t.Fatal("expected raw key in response")
	}
}

func TestChannelTestReturnsPayload(t *testing.T) {
	var logged domain.RequestLog
	router := NewRouter(Dependencies{
		TestChannel: func(ctx context.Context, id int64, model string) (domain.ChannelTestResult, error) {
			return domain.ChannelTestResult{Channel: "openai-main", Provider: "OpenAI", Model: model, StatusCode: 200, Message: "连接成功"}, nil
		},
		CreateRequestLog: func(ctx context.Context, item domain.RequestLog) error {
			logged = item
			return nil
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/channels/12/test", strings.NewReader(`{"model":"gpt-4o-mini"}`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var payload domain.ChannelTestResult
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if payload.Model != "gpt-4o-mini" {
		t.Fatalf("expected model gpt-4o-mini, got %s", payload.Model)
	}
	if payload.Message == "" {
		t.Fatal("expected success message in response")
	}
	if logged.Channel != "openai-main" || logged.Model != "gpt-4o-mini" || logged.StatusCode != http.StatusOK {
		t.Fatalf("unexpected request log: %#v", logged)
	}
}

func TestUpdateChannelAcceptsModelIDs(t *testing.T) {
	var captured domain.UpdateChannelInput
	router := NewRouter(Dependencies{
		UpdateChannel: func(ctx context.Context, id int64, input domain.UpdateChannelInput) error {
			captured = input
			return nil
		},
	})
	req := httptest.NewRequest(http.MethodPut, "/api/admin/channels/3", strings.NewReader(`{"name":"kimi","provider":"KIMI","endpoint":"https://api.moonshot.cn/v1","enabled":true,"model_ids":["kimi-k2.5"]}`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if len(captured.ModelIDs) != 1 || captured.ModelIDs[0] != "kimi-k2.5" {
		t.Fatalf("unexpected model ids: %#v", captured.ModelIDs)
	}
}

func TestCreateModelRouteAcceptsInvocationMode(t *testing.T) {
	var captured domain.CreateModelRouteInput
	router := NewRouter(Dependencies{
		CreateModelRoute: func(ctx context.Context, input domain.CreateModelRouteInput) (domain.ModelRoute, error) {
			captured = input
			return domain.ModelRoute{ID: 1, ModelAlias: input.ModelAlias, ChannelName: input.ChannelName, InvocationMode: input.InvocationMode, Priority: input.Priority, FallbackModel: input.FallbackModel}, nil
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/model-routes", strings.NewReader(`{"model_alias":"gpt-4o","channel_name":"claude-a","invocation_mode":"claude","priority":10,"fallback_model":""}`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}
	if captured.InvocationMode != "claude" {
		t.Fatalf("unexpected invocation mode: %#v", captured)
	}
}
