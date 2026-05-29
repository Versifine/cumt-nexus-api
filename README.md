# CUMT Nexus API

`cumt-nexus-api` 是一个社区内容平台后端。当前采用 Go + Gin + PostgreSQL，架构形态是模块化单体。

## 当前状态

阶段：`阶段 1 认证与用户基础闭环`

已具备：

- 配置加载与校验
- PostgreSQL 连接池初始化
- migration 执行入口
- HTTP 服务启动、优雅关闭和 `/healthz`
- 统一错误响应、recovery、request id、请求日志
- 用户领域模型、密码哈希、用户仓储
- `POST /api/v1/auth/register`
- JWT access token 签发

下一步：

- `POST /api/v1/auth/login`
- 认证中间件
- `GET /api/v1/me`

## 接口

当前可用接口：

```text
GET  /healthz
POST /api/v1/auth/register
```

注册请求：

```bash
curl -i -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"alice\",\"password\":\"password123\"}"
```

成功响应返回 `access_token`、`token_type`、`expires_in` 和用户公开信息。响应不会包含 `password_hash`。

## 本地运行

### 1. 准备配置

```bash
cp .env.example .env
```

Windows PowerShell:

```powershell
Copy-Item .env.example .env
```

### 2. 启动 PostgreSQL

```bash
docker compose up -d postgres
```

### 3. 执行 migration

```bash
go run ./cmd/migrate up
```

查看版本：

```bash
go run ./cmd/migrate version
```

### 4. 启动 API

```bash
go run ./cmd/api
```

默认监听 `.env` 中的 `HTTP_ADDR`，当前示例配置是 `:8080`。

### 5. 健康检查

```bash
curl http://localhost:8080/healthz
```

预期：

```json
{"status":"ok"}
```

## 测试

```bash
go test ./...
go build -buildvcs=false ./...
```

普通 `go build ./...` 在部分本机环境可能受 Git safe.directory / VCS stamping 影响，可用 `-buildvcs=false` 验证代码构建。

## 文档

- `tasks.md`：当前阶段工单板
- `docs/internal/README.md`：内部文档索引
- `docs/internal/architecture/project-baseline.md`：架构基线
- `docs/internal/architecture/auth-user-v1.md`：阶段 1 认证设计
- `docs/internal/engineering/stage-1-auth-playbook.md`：阶段 1 认证实现手册

## 当前限制

- 阶段 1 暂不做 refresh token、多端会话、第三方登录、邮箱验证、找回密码和复杂 RBAC。
- `/healthz` 只表示进程存活，不做数据库 readiness 检查。

## License

[MIT License](./LICENSE)
