package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// healthResponse 是健康检查接口的统一返回结构。
//
// 这里先保持字段极少，目的是让第一阶段先打通“服务活着”和“服务可对外响应”的验证链路。
// 后续会继续增加版本号、数据库状态、依赖状态等信息。
type healthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

// NewRouter 负责创建整个 HTTP 路由树。
//
// 当前阶段先只注册最基础的中间件与健康检查路由：
// 1. 统一 request id，方便后面串日志。
// 2. 统一恢复 panic，避免服务直接崩掉。
// 3. 提供 /healthz 和 /readyz 作为首批验证接口。
func NewRouter() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, healthResponse{
			Status:    "ok",
			Timestamp: time.Now().Format(time.RFC3339),
		})
	})

	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, healthResponse{
			Status:    "ready",
			Timestamp: time.Now().Format(time.RFC3339),
		})
	})

	return r
}

// writeJSON 负责把结构化数据写成 JSON 响应。
//
// 这里统一封装的目的是让后续接口都走同一套输出方式，
// 避免每个 handler 自己设置响应头、自己编码，导致风格不统一。
func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, "写入 JSON 响应失败", http.StatusInternalServerError)
	}
}
