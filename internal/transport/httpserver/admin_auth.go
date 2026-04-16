package httpserver

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"opencrab/internal/domain"
)

const adminSessionCookieName = "opencrab-admin-session"

func validateAdminPasswordInput(input domain.AdminPasswordInput) error {
	password := strings.TrimSpace(input.Password)
	if len(password) < 8 {
		return fmt.Errorf("密码至少需要 8 个字符")
	}
	return nil
}

func validateAdminPasswordChangeInput(input domain.AdminPasswordChangeInput) error {
	if strings.TrimSpace(input.CurrentPassword) == "" {
		return fmt.Errorf("当前密码不能为空")
	}
	if len(strings.TrimSpace(input.NewPassword)) < 8 {
		return fmt.Errorf("新密码至少需要 8 个字符")
	}
	if input.NewPassword != input.ConfirmPassword {
		return fmt.Errorf("两次输入的新密码不一致")
	}
	return nil
}

func validateSecondaryPasswordUpdateInput(input domain.AdminSecondaryPasswordUpdateInput) error {
	if strings.TrimSpace(input.CurrentAdminPassword) == "" {
		return fmt.Errorf("当前管理员密码不能为空")
	}
	if strings.TrimSpace(input.NewPassword) != "" || strings.TrimSpace(input.ConfirmPassword) != "" {
		if len(strings.TrimSpace(input.NewPassword)) < 8 {
			return fmt.Errorf("新密码至少需要 8 个字符")
		}
		if input.NewPassword != input.ConfirmPassword {
			return fmt.Errorf("两次输入的新密码不一致")
		}
	}
	return nil
}

func extractSecondaryPassword(req *http.Request) string {
	if req == nil {
		return ""
	}
	return strings.TrimSpace(req.Header.Get("X-OpenCrab-Secondary-Password"))
}

func buildAdminAuthStatus(req *http.Request, state domain.AdminAuthState) domain.AdminAuthStatus {
	status := domain.AdminAuthStatus{Initialized: state.Initialized}
	if !state.Initialized || strings.TrimSpace(state.SessionSecret) == "" || req == nil {
		return status
	}
	status.Authenticated = hasValidAdminSession(req, state.SessionSecret)
	return status
}

func buildAuthenticatedAdminStatus(state domain.AdminAuthState) domain.AdminAuthStatus {
	return domain.AdminAuthStatus{
		Initialized:   state.Initialized,
		Authenticated: state.Initialized,
	}
}

func requireAdminSession(deps Dependencies) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if deps.GetAdminAuthState == nil {
				next.ServeHTTP(w, req)
				return
			}
			state, err := deps.GetAdminAuthState(req.Context())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if !state.Initialized {
				http.Error(w, "管理员密码尚未初始化", http.StatusPreconditionRequired)
				return
			}
			if strings.TrimSpace(state.SessionSecret) == "" {
				http.Error(w, "管理员会话密钥缺失", http.StatusInternalServerError)
				return
			}
			if !hasValidAdminSession(req, state.SessionSecret) {
				http.Error(w, "未登录或登录已失效", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, req)
		})
	}
}

func writeAdminSessionCookie(w http.ResponseWriter, req *http.Request, sessionSecret string) {
	expiresAt := time.Now().Add(30 * 24 * time.Hour).Unix()
	payload := strconv.FormatInt(expiresAt, 10)
	signature := signAdminSession(sessionSecret, payload)
	http.SetCookie(w, &http.Cookie{
		Name:     adminSessionCookieName,
		Value:    payload + "." + signature,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   requestIsSecure(req),
		Expires:  time.Unix(expiresAt, 0),
		MaxAge:   30 * 24 * 60 * 60,
	})
}

func clearAdminSessionCookie(w http.ResponseWriter, req *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     adminSessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   requestIsSecure(req),
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}

func requestIsSecure(req *http.Request) bool {
	if req == nil {
		return false
	}
	if req.TLS != nil {
		return true
	}
	for _, value := range strings.Split(req.Header.Get("X-Forwarded-Proto"), ",") {
		if strings.EqualFold(strings.TrimSpace(value), "https") {
			return true
		}
	}
	return false
}

func adminAuthRateLimitKey(req *http.Request) string {
	if req == nil {
		return "unknown"
	}
	for _, headerName := range []string{"X-Forwarded-For", "X-Real-IP"} {
		for _, value := range strings.Split(req.Header.Get(headerName), ",") {
			trimmed := strings.TrimSpace(value)
			if trimmed != "" {
				return trimmed
			}
		}
	}
	if req.RemoteAddr != "" {
		return req.RemoteAddr
	}
	return "unknown"
}

func hasValidAdminSession(req *http.Request, sessionSecret string) bool {
	if req == nil || strings.TrimSpace(sessionSecret) == "" {
		return false
	}
	cookie, err := req.Cookie(adminSessionCookieName)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		return false
	}
	parts := strings.Split(cookie.Value, ".")
	if len(parts) != 2 {
		return false
	}
	expiresAt, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || expiresAt <= time.Now().Unix() {
		return false
	}
	expected := signAdminSession(sessionSecret, parts[0])
	return hmac.Equal([]byte(parts[1]), []byte(expected))
}

func signAdminSession(sessionSecret string, payload string) string {
	secretBytes, err := hex.DecodeString(sessionSecret)
	if err != nil {
		secretBytes = []byte(sessionSecret)
	}
	mac := hmac.New(sha256.New, secretBytes)
	_, _ = mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
