# CUMT Nexus API

`cumt-nexus-api` 是一个社区内容平台后端，使用 Go、Gin、PostgreSQL 和 pgx 构建。项目采用模块化单体结构，按业务模块组织代码，重点覆盖用户身份、社区、帖子、评论、投票、搜索、通知、媒体上传和基础审核能力。

## 功能概览

- 用户注册、登录、Bearer JWT 认证和当前用户读取
- 社区列表、社区详情、社区创建申请、申请审核读取和平台 staff 审批
- 帖子发布、列表、详情、编辑、软删除
- 评论发布、列表、树状读取、编辑、软删除
- 帖子 upvote/downvote/cancel、保存、评论投票、社区关注和 `best` / `hot` / `new` / `top` / `rising` 排序
- 图片上传、帖子/评论图片附件绑定，支持本地存储和 Cloudflare R2
- 内容举报、平台 staff 移除内容、举报列表和举报处理
- PostgreSQL 基础搜索，支持匿名读取公开内容
- 站内通知读取、分类筛选、未读摘要和标记已读
- 统一错误响应、request id、请求日志、panic recovery、CORS 配置

## 技术栈

| 能力 | 选择 |
|---|---|
| Language | Go |
| HTTP | Gin |
| Database | PostgreSQL |
| Database driver | pgx |
| Auth | JWT access token |
| Migration | golang-migrate |
| Object storage | local filesystem / Cloudflare R2 |
| Logging | slog |

## 快速开始

### 1. 准备配置

```powershell
Copy-Item .env.example .env
```

macOS/Linux:

```bash
cp .env.example .env
```

本地默认使用 PostgreSQL 和 local object storage。生产或接近生产的图片上传应使用 Cloudflare R2；真实密钥只放在本地 `.env` 或部署平台 secret 中，不写入 README、提交信息或文档。

### 2. 启动 PostgreSQL

```bash
docker compose up -d postgres
```

### 3. 执行 migration

```bash
go run ./cmd/migrate up
```

查看 migration 版本：

```bash
go run ./cmd/migrate version
```

### 4. 启动 API

```bash
go run ./cmd/api
```

默认监听 `.env` 中的 `HTTP_ADDR`，示例配置为 `:8080`。

健康检查：

```bash
curl http://localhost:8080/healthz
```

预期响应：

```json
{"status":"ok"}
```

## 配置

项目启动时会自动加载当前目录下的 `.env`。完整配置样例见 [.env.example](.env.example)。

常用配置分组：

| 分组 | 变量前缀 | 说明 |
|---|---|---|
| App | `APP_` | 应用名、运行环境、启动超时 |
| PostgreSQL | `POSTGRES_` | 数据库连接和连接池参数 |
| HTTP | `HTTP_` | 监听地址、超时、CORS |
| Log | `LOG_` | 日志级别和格式 |
| Auth | `AUTH_` | JWT secret 和 access token TTL |
| Storage | `OBJECT_STORAGE_` | local/R2 对象存储配置 |
| Upload | `UPLOAD_` | 图片大小和数量限制 |

### Cloudflare R2

R2 使用 S3-compatible 配置。endpoint 使用账号级地址，bucket 单独配置：

```env
OBJECT_STORAGE_PROVIDER=r2
OBJECT_STORAGE_ENDPOINT=https://<account-id>.r2.cloudflarestorage.com
OBJECT_STORAGE_REGION=auto
OBJECT_STORAGE_BUCKET=<bucket-name>
OBJECT_STORAGE_ACCESS_KEY_ID=<access-key-id>
OBJECT_STORAGE_SECRET_ACCESS_KEY=<secret-access-key>
OBJECT_STORAGE_PUBLIC_BASE_URL=<public-media-base-url>
OBJECT_STORAGE_FORCE_PATH_STYLE=true
```

不要把 bucket 名拼进 `OBJECT_STORAGE_ENDPOINT`，也不要提交真实 access key 或 secret key。`OBJECT_STORAGE_PUBLIC_BASE_URL` 必须是浏览器可公开读取图片的 base URL，例如 R2 public development URL 或自定义域名；不要填写 `https://<account-id>.r2.cloudflarestorage.com` 这个 S3 API endpoint。

## API 概览

写操作、权限操作和用户态读取需要：

```http
Authorization: Bearer <access_token>
```

公开帖子流、公开帖子详情、公开评论、公开社区、公开用户主页和搜索支持匿名读取；如果请求携带 Bearer token，后端会返回当前用户视角的 `my_vote`、`is_saved`、`viewer_is_following` 和 `viewer_permissions`，无 token 时这些 viewer 字段为匿名态。格式错误、过期或签名错误的 token 仍返回 `unauthenticated`，不会静默降级为匿名。

### Auth

```text
POST /api/v1/auth/register
POST /api/v1/auth/login
GET  /api/v1/me
GET  /api/v1/me/saved-posts
GET  /api/v1/me/followed-communities
GET  /api/v1/users/:username            # public, optional Bearer
GET  /api/v1/users/:username/posts      # public, optional Bearer
GET  /api/v1/users/:username/comments   # public, optional Bearer
```

### Communities

```text
GET  /api/v1/communities                # public, optional Bearer
GET  /api/v1/communities/:slug          # public, optional Bearer
POST /api/v1/communities/:slug/follow
DELETE /api/v1/communities/:slug/follow
POST /api/v1/community-applications
GET  /api/v1/community-applications
GET  /api/v1/community-applications/:id
POST /api/v1/community-applications/:id/approve
POST /api/v1/community-applications/:id/reject
```

### Posts

```text
GET    /api/v1/communities/:slug/posts  # public, optional Bearer
GET    /api/v1/posts                    # public, optional Bearer
GET    /api/v1/posts/:id                # public, optional Bearer
GET    /api/v1/users/:username/posts    # public, optional Bearer
POST   /api/v1/communities/:slug/posts
PATCH  /api/v1/posts/:id
DELETE /api/v1/posts/:id
POST   /api/v1/posts/:id/save
DELETE /api/v1/posts/:id/save
PUT    /api/v1/posts/:id/vote
DELETE /api/v1/posts/:id/vote
```

全站 feed 支持：

```text
GET /api/v1/posts?source=all&sort=new&limit=20&offset=0
GET /api/v1/posts?source=recommended&sort=best&t=day&limit=20&offset=0
```

`source=recommended` 当前是后端公开可解释排序流，默认 `sort=best`，不是个性化推荐。`t` 支持 `hour|day|week|month|year|all`。

### Comments

```text
POST   /api/v1/posts/:id/comments
GET    /api/v1/posts/:id/comments       # public, optional Bearer
GET    /api/v1/users/:username/comments # public, optional Bearer
PATCH  /api/v1/comments/:id
DELETE /api/v1/comments/:id
PUT    /api/v1/comments/:id/vote
DELETE /api/v1/comments/:id/vote
```

评论树读取通过查询参数启用：

```text
GET /api/v1/posts/:id/comments?view=tree&sort=new&limit=20&offset=0&max_depth=6
```

### Media

```text
POST /api/v1/uploads/images
POST /api/v1/link-previews/resolve
POST /api/v1/embeds/resolve
```

上传使用 `multipart/form-data`，字段为 `file` 和可选 `alt_text`。
链接预览和嵌入解析使用 JSON `{"url":"https://..."}`，会校验公开 HTTP(S) URL、拦截本机/私网地址，并只对首批白名单 provider 返回嵌入结果。

完整接口合同见 [docs/contracts/http-api-contract.md](docs/contracts/http-api-contract.md)，请求/响应字段合同见 [docs/contracts/http-api-schema.md](docs/contracts/http-api-schema.md)。

### Effects / Points

```text
GET  /api/v1/effects/catalog       # public, optional Bearer
GET  /api/v1/me/points
POST /api/v1/comments/:id/effects
```

评论效果应用使用 JSON `{"effect_id":"sparkle"}`。后端负责积分账户初始化、余额扣减、评论效果记录和积分流水审计。

### Moderation

```text
POST /api/v1/posts/:id/reports
POST /api/v1/comments/:id/reports
POST /api/v1/posts/:id/moderation/remove
POST /api/v1/comments/:id/moderation/remove
GET  /api/v1/moderation/reports
GET  /api/v1/moderation/reports/:id
POST /api/v1/moderation/reports/:id/dismiss
POST /api/v1/moderation/reports/:id/remove-target
```

### Search / Notifications

```text
GET  /api/v1/search                         # public, optional Bearer
GET  /api/v1/notifications/unread-summary
GET  /api/v1/notifications?category=likes&status=unread
POST /api/v1/notifications/:id/read
POST /api/v1/notifications/read-all
```

## 响应约定

错误响应统一为：

```json
{
  "error": {
    "code": "invalid_argument",
    "message": "..."
  }
}
```

当前错误码包括：

```text
invalid_argument
unauthenticated
forbidden
not_found
conflict
internal
```

`DELETE` 成功时返回 `204 No Content`。

错误响应、配置和 migration 的可校验合同文档集中放在 [docs/contracts/](docs/contracts/)。

## 本地开发

常用检查：

```bash
go test ./...
go build -buildvcs=false ./...
```

快速基线检查：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-current-baseline.ps1 -SkipHttpSmoke -R2Mode Skip
```

更细的本地 smoke 和契约校验脚本放在 `scripts/` 目录；README 只保留常用入口。

## 目录结构

```text
cmd/
  api/        HTTP API 启动入口
  migrate/    migration CLI
internal/
  auth/       认证、密码、token、HTTP auth
  user/       用户模型和当前用户
  community/  社区、申请、审批
  post/       帖子写入和读取
  comment/    评论写入和读取
  vote/       帖子投票
  moderation/ 举报和审核
  media/      图片附件
  storage/    local/R2 对象存储
  search/     搜索
  notification/ 通知
  platform/   配置、数据库、HTTP server、日志
migrations/   PostgreSQL migrations
scripts/      本地校验和 smoke 脚本
```

## License

[MIT License](./LICENSE)
