package main

import (
	"log"

	"opencrab/internal/app"
)

// main 是整个后端服务的启动入口。
//
// 这里当前只做三件事情：
// 1. 创建应用实例。
// 2. 启动 HTTP 服务。
// 3. 在启动失败时直接输出错误并退出。
//
// 后续数据库、配置、日志、依赖注入都会先汇总到 app 包，再由这里统一启动。
func main() {
	application, err := app.New()
	if err != nil {
		log.Fatalf("创建应用失败: %v", err)
	}

	if err := application.Run(); err != nil {
		log.Fatalf("启动应用失败: %v", err)
	}
}
