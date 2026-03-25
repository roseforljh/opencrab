# OpenCrab

OpenCrab 是一个面向个人用户的 API 聚合与中转项目。

本项目当前定位为：
- 个人 API 聚合
- 自托管 API 网关
- 多上游统一接入与管理
- 面向个人用户与轻量场景的控制台

## 特性

- 个人版控制台 UI
- 渠道管理
- 模型管理
- Token 管理
- PIN 登录
- Docker 单容器部署
- SQLite 本地部署

## 适用场景

- 个人 API 中转
- 家庭实验室
- 自托管模型网关
- 多上游统一管理

## 快速开始

### Docker Compose

```bash
docker compose up -d --build
```

默认访问地址：

- http://localhost:5946

## 本地开发

前端：

```bash
cd web
npm install --legacy-peer-deps
npm run build
```

后端：

```bash
go run main.go
```

## 致谢

感谢相关开源社区与前置项目在 API 聚合、自托管网关、控制台交互和部署实践方面提供的启发与参考。

## 许可证说明

本项目仓库当前使用 [MIT License](./LICENSE)。

在对外发布和继续演进前，建议维护者自行确认：
- 当前保留下来的核心代码是否已完成来源梳理
- 已移除或替换不兼容许可证来源的代码与文件头
- 第三方依赖、素材、图标与文档生成物已按各自许可证要求保留说明

## 模块路径

```go
module github.com/roseforljh/opencrab
```
