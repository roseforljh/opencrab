package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
			return domain.ChannelTestResult{Channel: "openai-main", Provider: "OpenAI", Model: model, StatusCode: 200, Message: "连接成功", Details: map[string]any{"request_url": "https://api.openai.com/v1/chat/completions", "upstream_status": 200}}, nil
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
	if payload.Details["request_url"] == nil || payload.Details["upstream_status"] != float64(200) {
		t.Fatalf("expected detailed channel test payload, got %#v", payload.Details)
	}
	if logged.Channel != "openai-main" || logged.Model != "gpt-4o-mini" || logged.StatusCode != http.StatusOK {
		t.Fatalf("unexpected request log: %#v", logged)
	}
	if !strings.Contains(logged.Details, `"upstream_status":200`) || !strings.Contains(logged.Details, `"request_url":"https://api.openai.com/v1/chat/completions"`) {
		t.Fatalf("expected detailed request log details, got %s", logged.Details)
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

func TestAdminAuthStatusReturnsInitializationFlags(t *testing.T) {
	router := NewRouter(Dependencies{
		GetAdminAuthState: func(ctx context.Context) (domain.AdminAuthState, error) {
			return domain.AdminAuthState{Initialized: false}, nil
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/admin/auth/status", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var payload domain.AdminAuthStatus
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Initialized || payload.Authenticated {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestProtectedAdminRouteRejectsAnonymousRequest(t *testing.T) {
	router := NewRouter(Dependencies{
		GetAdminAuthState: func(ctx context.Context) (domain.AdminAuthState, error) {
			return domain.AdminAuthState{Initialized: true, SessionSecret: strings.Repeat("ab", 32)}, nil
		},
		ListSettings: func(ctx context.Context) ([]domain.SystemSettingGroup, error) {
			return []domain.SystemSettingGroup{}, nil
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/admin/settings", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAdminSetupInitializesPasswordAndSession(t *testing.T) {
	router := NewRouter(Dependencies{
		SetupAdminPassword: func(ctx context.Context, password string) (domain.AdminAuthState, error) {
			return domain.AdminAuthState{Initialized: true, SessionSecret: strings.Repeat("ab", 32)}, nil
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/auth/setup", strings.NewReader(`{"password":"hunter2-password"}`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}
	if len(rec.Result().Cookies()) == 0 || rec.Result().Cookies()[0].Name != adminSessionCookieName {
		t.Fatalf("expected admin session cookie, got %#v", rec.Result().Cookies())
	}
}

func TestAdminLoginRejectsWrongPassword(t *testing.T) {
	router := NewRouter(Dependencies{
		VerifyAdminPassword: func(ctx context.Context, password string) (domain.AdminAuthState, error) {
			return domain.AdminAuthState{}, fmt.Errorf("密码错误")
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/auth/login", strings.NewReader(`{"password":"wrong-password"}`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAdminLogoutClearsSessionCookie(t *testing.T) {
	router := NewRouter(Dependencies{})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/auth/logout", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) == 0 || cookies[0].Name != adminSessionCookieName || cookies[0].MaxAge != -1 {
		t.Fatalf("expected cleared session cookie, got %#v", cookies)
	}
}

func TestAdminPasswordChangeReturnsAuthenticatedPayload(t *testing.T) {
	router := NewRouter(Dependencies{
		ChangeAdminPassword: func(ctx context.Context, input domain.AdminPasswordChangeInput) (domain.AdminAuthState, error) {
			return domain.AdminAuthState{Initialized: true, SessionSecret: strings.Repeat("ab", 32)}, nil
		},
	})
	req := httptest.NewRequest(http.MethodPut, "/api/admin/auth/password", strings.NewReader(`{"current_password":"old-password","new_password":"new-password-1","confirm_password":"new-password-1"}`))
	rec := httptest.NewRecorder()
	req.AddCookie(&http.Cookie{Name: adminSessionCookieName, Value: "1.signature"})

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestCreateAPIKeyRejectsMissingSecondaryPasswordWhenEnabled(t *testing.T) {
	router := NewRouter(Dependencies{
		GetAdminAuthState: func(ctx context.Context) (domain.AdminAuthState, error) {
			return domain.AdminAuthState{Initialized: true, SessionSecret: strings.Repeat("ab", 32)}, nil
		},
		VerifySecondaryPassword: func(ctx context.Context, password string) error {
			return fmt.Errorf("二级密码未通过校验")
		},
		CreateAPIKey: func(ctx context.Context, input domain.CreateAPIKeyInput) (domain.CreatedAPIKey, error) {
			return domain.CreatedAPIKey{}, nil
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/api-keys", strings.NewReader(`{"name":"console","enabled":true}`))
	req.AddCookie(&http.Cookie{Name: adminSessionCookieName, Value: signedSessionCookieValue(strings.Repeat("ab", 32))})
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestDeleteAPIKeyAcceptsSecondaryPasswordHeader(t *testing.T) {
	called := false
	router := NewRouter(Dependencies{
		GetAdminAuthState: func(ctx context.Context) (domain.AdminAuthState, error) {
			return domain.AdminAuthState{Initialized: true, SessionSecret: strings.Repeat("ab", 32)}, nil
		},
		VerifySecondaryPassword: func(ctx context.Context, password string) error {
			if password != "second-pass" {
				return fmt.Errorf("二级密码未通过校验")
			}
			return nil
		},
		DeleteAPIKey: func(ctx context.Context, id int64) error {
			called = true
			return nil
		},
	})
	req := httptest.NewRequest(http.MethodDelete, "/api/admin/api-keys/9", nil)
	req.Header.Set("X-OpenCrab-Secondary-Password", "second-pass")
	req.AddCookie(&http.Cookie{Name: adminSessionCookieName, Value: signedSessionCookieValue(strings.Repeat("ab", 32))})
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || !called {
		t.Fatalf("expected delete success, got code=%d called=%v", rec.Code, called)
	}
}

func TestSettingsRejectAdminSecurityKeys(t *testing.T) {
	router := NewRouter(Dependencies{
		GetAdminAuthState: func(ctx context.Context) (domain.AdminAuthState, error) {
			return domain.AdminAuthState{Initialized: true, SessionSecret: strings.Repeat("ab", 32)}, nil
		},
		UpdateSetting: func(ctx context.Context, input domain.UpdateSystemSettingInput) (domain.SystemSetting, error) {
			return domain.SystemSetting{}, nil
		},
	})
	req := httptest.NewRequest(http.MethodPut, "/api/admin/settings", strings.NewReader(`{"key":"admin.secondary_enabled","value":"false"}`))
	req.AddCookie(&http.Cookie{Name: adminSessionCookieName, Value: signedSessionCookieValue(strings.Repeat("ab", 32))})
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestCapabilityProfilesListReturnsCatalog(t *testing.T) {
	router := NewRouter(Dependencies{
		GetAdminAuthState: func(ctx context.Context) (domain.AdminAuthState, error) {
			return domain.AdminAuthState{Initialized: true, SessionSecret: strings.Repeat("ab", 32)}, nil
		},
		ListCapabilityProfiles: func(ctx context.Context) (domain.CapabilityProfileListResponse, error) {
			return domain.CapabilityProfileListResponse{
				Items:   []domain.CapabilityProfile{{ScopeType: "provider_default", ScopeKey: "openai", Operation: "responses", Capabilities: []string{"function_tools"}}},
				Catalog: domain.CapabilityCatalog{ScopeTypes: []string{"provider_default"}, Operations: []string{"responses"}, Items: []string{"function_tools"}},
			}, nil
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/admin/capability-profiles", nil)
	req.AddCookie(&http.Cookie{Name: adminSessionCookieName, Value: signedSessionCookieValue(strings.Repeat("ab", 32))})
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var payload domain.CapabilityProfileListResponse
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Items) != 1 || len(payload.Catalog.Items) != 1 {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestCapabilityProfilesUpdateAcceptsPayload(t *testing.T) {
	var captured domain.UpsertCapabilityProfileInput
	router := NewRouter(Dependencies{
		GetAdminAuthState: func(ctx context.Context) (domain.AdminAuthState, error) {
			return domain.AdminAuthState{Initialized: true, SessionSecret: strings.Repeat("ab", 32)}, nil
		},
		UpsertCapabilityProfile: func(ctx context.Context, input domain.UpsertCapabilityProfileInput) error {
			captured = input
			return nil
		},
	})
	req := httptest.NewRequest(http.MethodPut, "/api/admin/capability-profiles", strings.NewReader(`{"scope_type":"provider_default","scope_key":"openai","operation":"responses","enabled":true,"capabilities":["function_tools","builtin_shell"]}`))
	req.AddCookie(&http.Cookie{Name: adminSessionCookieName, Value: signedSessionCookieValue(strings.Repeat("ab", 32))})
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if captured.ScopeType != "provider_default" || captured.ScopeKey != "openai" || len(captured.Capabilities) != 2 {
		t.Fatalf("unexpected payload: %#v", captured)
	}
}

func TestCapabilityProfilesDeleteAcceptsPayload(t *testing.T) {
	var captured domain.DeleteCapabilityProfileInput
	router := NewRouter(Dependencies{
		GetAdminAuthState: func(ctx context.Context) (domain.AdminAuthState, error) {
			return domain.AdminAuthState{Initialized: true, SessionSecret: strings.Repeat("ab", 32)}, nil
		},
		DeleteCapabilityProfile: func(ctx context.Context, input domain.DeleteCapabilityProfileInput) error {
			captured = input
			return nil
		},
	})
	req := httptest.NewRequest(http.MethodDelete, "/api/admin/capability-profiles", strings.NewReader(`{"scope_type":"channel_override","scope_key":"openai-main","operation":"chat_completions"}`))
	req.AddCookie(&http.Cookie{Name: adminSessionCookieName, Value: signedSessionCookieValue(strings.Repeat("ab", 32))})
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if captured.ScopeType != "channel_override" || captured.ScopeKey != "openai-main" || captured.Operation != "chat_completions" {
		t.Fatalf("unexpected payload: %#v", captured)
	}
}

func signedSessionCookieValue(secret string) string {
	payload := fmt.Sprintf("%d", time.Now().Add(time.Hour).Unix())
	return payload + "." + signAdminSession(secret, payload)
}
