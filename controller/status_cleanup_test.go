package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/opencrab/common"
	"github.com/gin-gonic/gin"
)

type statusAPIResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data"`
}

func TestGetStatusDoesNotExposeLegacyAuthConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	context.Request = request

	GetStatus(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var response statusAPIResponse
	if err := common.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	legacyKeys := []string{
		"email_verification",
		"github_oauth",
		"github_client_id",
		"discord_oauth",
		"discord_client_id",
		"linuxdo_oauth",
		"linuxdo_client_id",
		"linuxdo_minimum_trust_level",
		"telegram_oauth",
		"telegram_bot_name",
		"wechat_login",
		"wechat_qrcode",
		"oidc_enabled",
		"oidc_client_id",
		"oidc_authorization_endpoint",
		"passkey_login",
		"passkey_display_name",
		"passkey_rp_id",
		"passkey_origins",
		"passkey_allow_insecure",
		"passkey_user_verification",
		"passkey_attachment",
		"custom_oauth_providers",
		"demo_site_enabled",
		"self_use_mode_enabled",
	}

	for _, key := range legacyKeys {
		if _, exists := response.Data[key]; exists {
			t.Fatalf("legacy auth key should not be exposed in status response: %s", key)
		}
	}
}
