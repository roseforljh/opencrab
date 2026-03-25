package router

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func hasRoute(routes gin.RoutesInfo, method, path string) bool {
	for _, route := range routes {
		if route.Method == method && route.Path == path {
			return true
		}
	}
	return false
}

func TestSetApiRouterDoesNotExposeLegacyAuthRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	SetApiRouter(engine)

	routes := engine.Routes()
	disallowedRoutes := []struct {
		method string
		path   string
	}{
		{method: "GET", path: "/api/verification"},
		{method: "GET", path: "/api/reset_password"},
		{method: "POST", path: "/api/user/reset"},
		{method: "GET", path: "/api/oauth/state"},
		{method: "GET", path: "/api/oauth/email/bind"},
		{method: "GET", path: "/api/oauth/wechat"},
		{method: "GET", path: "/api/oauth/wechat/bind"},
		{method: "GET", path: "/api/oauth/telegram/login"},
		{method: "GET", path: "/api/oauth/telegram/bind"},
		{method: "GET", path: "/api/oauth/:provider"},
		{method: "GET", path: "/api/user/oauth/bindings"},
		{method: "DELETE", path: "/api/user/oauth/bindings/:provider_id"},
		{method: "GET", path: "/api/user/:id/oauth/bindings"},
		{method: "DELETE", path: "/api/user/:id/oauth/bindings/:provider_id"},
		{method: "POST", path: "/api/custom-oauth-provider/discovery"},
		{method: "GET", path: "/api/custom-oauth-provider/"},
		{method: "GET", path: "/api/custom-oauth-provider/:id"},
		{method: "POST", path: "/api/custom-oauth-provider/"},
		{method: "PUT", path: "/api/custom-oauth-provider/:id"},
		{method: "DELETE", path: "/api/custom-oauth-provider/:id"},
		{method: "POST", path: "/api/user/register"},
		{method: "POST", path: "/api/user/login"},
		{method: "POST", path: "/api/user/passkey/login/begin"},
		{method: "POST", path: "/api/user/passkey/login/finish"},
		{method: "GET", path: "/api/user/passkey"},
		{method: "POST", path: "/api/user/passkey/register/begin"},
		{method: "POST", path: "/api/user/passkey/register/finish"},
		{method: "POST", path: "/api/user/passkey/verify/begin"},
		{method: "POST", path: "/api/user/passkey/verify/finish"},
		{method: "DELETE", path: "/api/user/passkey"},
		{method: "POST", path: "/api/verify"},
	}

	for _, route := range disallowedRoutes {
		if hasRoute(routes, route.method, route.path) {
			t.Fatalf("found legacy auth route still exposed: %s %s", route.method, route.path)
		}
	}
}
