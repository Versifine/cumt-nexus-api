# CUMT Nexus API

`cumt-nexus-api` 是一个社区内容平台后端。当前采用 Go + Gin + PostgreSQL，架构形态是模块化单体。

## 当前状态

阶段：`阶段 2 社区/板块基础管理`

代码已完成阶段 1 认证与用户基础闭环，当前开始推进社区模块。

已具备：

- 配置加载与校验
- PostgreSQL 连接池初始化
- migration 执行入口
- HTTP 服务启动、优雅关闭和 `/healthz`
- 统一错误响应、recovery、request id、请求日志
- 用户领域模型、密码哈希、用户仓储
- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- JWT access token 签发
- Bearer JWT 认证中间件
- `GET /api/v1/me`
- 阶段 2 社区 schema migration
- 社区领域模型与 PostgreSQL repository

下一步：

- 执行 `T2-004`：公共总版与社区读取接口
- 初始化公共总版并提供社区列表、社区详情读取能力

## 接口

当前可用接口：

```text
GET  /healthz
POST /api/v1/auth/register
POST /api/v1/auth/login
GET  /api/v1/me
```

注册请求：

```bash
curl -i -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"alice\",\"password\":\"password123\"}"
```

成功响应返回 `access_token`、`token_type`、`expires_in` 和用户公开信息。响应不会包含 `password_hash`。

登录请求：

```bash
curl -i -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"alice\",\"password\":\"password123\"}"
```

成功响应结构与注册一致。

当前用户请求：

```bash
curl -i http://localhost:8080/api/v1/me \
  -H "Authorization: Bearer <access_token>"
```

成功响应返回当前用户公开信息：

```json
{
  "id": "uuid",
  "username": "alice",
  "status": "active",
  "created_at": "2026-05-30T00:00:00Z"
}
```

未带 token、token 格式错误、token 无效或 token 对应用户不存在时，统一返回 `401 unauthenticated`。

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
- `docs/internal/architecture/community-v1.md`：V1 社区业务架构与阶段 2 社区边界
- `docs/internal/engineering/stage-1-auth-playbook.md`：阶段 1 认证实现手册

## 当前限制

- 阶段 1 暂不做 refresh token、多端会话、第三方登录、邮箱验证、找回密码和复杂 RBAC。
- `/me` 只返回当前用户基础公开信息，不做资料编辑、头像、邮箱和权限列表。
- 认证中间件只验证 Bearer access token 并写入当前用户 ID，不查询数据库。
- 阶段 2 已落地社区 schema、domain 和 repository，但公共总版初始化、社区读取、申请和审批接口尚未实现。
- `/healthz` 只表示进程存活，不做数据库 readiness 检查。

## License

[MIT License](./LICENSE)
