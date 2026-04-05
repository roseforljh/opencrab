package app

import (
	"fmt"
	"net/http"
	"time"

	"opencrab/internal/transport/httpserver"
)

// App 表示当前后端服务的应用实例。
//
// 这个结构体存在的目的是把“应用启动相关的信息”集中放在一起，
// 避免 main 函数后面不断膨胀，最终把配置、日志、数据库、路由全都写在入口文件里。
//
// 当前阶段只保留最小字段：
// 1. 服务监听地址。
// 2. HTTP Server。
//
// 后续会继续把配置对象、数据库连接、日志对象等逐步加进来。
type App struct {
	address string
	server  *http.Server
}

// New 负责创建应用实例并准备好 HTTP 服务。
//
// 当前版本先使用固定地址和基础路由，目标是尽快把骨架立起来，
// 等配置系统落地后，再把地址、超时等参数改成配置驱动。
func New() (*App, error) {
	address := ":8080"
	router := httpserver.NewRouter()

	server := &http.Server{
		Addr:              address,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return &App{
		address: address,
		server:  server,
	}, nil
}

// Run 负责真正启动 HTTP 服务。
//
// 这里单独拆成方法，而不是直接在 New 里启动，
// 是为了让“创建应用”和“运行应用”这两个阶段分开，
// 后续更方便补测试、补初始化逻辑、补优雅关闭。
func (a *App) Run() error {
	fmt.Printf("OpenCrab 后端服务启动中，监听地址: %s\n", a.address)
	return a.server.ListenAndServe()
}
