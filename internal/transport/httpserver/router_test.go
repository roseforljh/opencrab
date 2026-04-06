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
