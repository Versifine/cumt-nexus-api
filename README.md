# CUMT Nexus API

`cumt-nexus-api` 是一个社区内容平台后端，使用 Go、Gin、PostgreSQL 和 pgx 构建。项目采用模块化单体结构，按业务模块组织代码，重点覆盖用户身份、社区、帖子、评论、投票、搜索、通知、媒体上传和基础审核能力。

## 功能概览

- 用户注册、登录、Bearer JWT 认证、当前用户读取和公开资料更新
- 社区列表、社区详情、社区创建申请、申请审核读取、平台 staff 审批、社区基础设置和社区规则管理
- 平台 staff 用户 / 社区 / 评论效果管理、运行开关和审计日志
- 帖子发布、列表、详情、编辑、软删除
- 评论发布、列表、树状读取、编辑、软删除
- 帖子 upvote/downvote/cancel、保存、评论投票、社区关注和 `best` / `hot` / `new` / `top` / `rising` 排序
- 图片上传、帖子/评论图片附件绑定、结构化 `content_refs` 引用持久化，支持本地存储和 Cloudflare R2
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
PATCH /api/v1/me/profile
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
GET  /api/v1/communities/:slug/manage
GET  /api/v1/communities/:slug/manage/posts
GET  /api/v1/communities/:slug/manage/comments
GET  /api/v1/communities/:slug/manage/reports
GET  /api/v1/communities/:slug/manage/members
GET  /api/v1/communities/:slug/manage/settings
PATCH /api/v1/communities/:slug/manage/settings
GET  /api/v1/communities/:slug/manage/rules
POST /api/v1/communities/:slug/manage/rules
PATCH /api/v1/communities/:slug/manage/rules/:rule_id
DELETE /api/v1/communities/:slug/manage/rules/:rule_id
POST /api/v1/communities/:slug/follow
DELETE /api/v1/communities/:slug/follow
POST /api/v1/community-applications
GET  /api/v1/community-applications
GET  /api/v1/community-applications/:id
POST /api/v1/community-applications/:id/approve
POST /api/v1/community-applications/:id/reject
```

社区管理设置读取允许 owner/moderator 进入管理上下文；设置更新仅允许 owner 修改 `name` 和 `description`。社区规则列表和 CRUD 允许 owner/moderator 使用，按 `position`、创建时间和 ID 稳定排序。

### Admin

```text
GET  /api/v1/admin/users
PATCH /api/v1/admin/users/:id
GET  /api/v1/admin/communities
PATCH /api/v1/admin/communities/:id
GET  /api/v1/admin/effects
PATCH /api/v1/admin/effects/:id
GET  /api/v1/admin/settings
PATCH /api/v1/admin/settings/:key
GET  /api/v1/admin/audit-logs
```

平台管理接口都需要 Bearer，并要求当前用户是 active 平台 staff。写操作会记录平台管理审计日志。运行开关包括 `registration_enabled`、`posting_enabled` 和 `upload_enabled`，分别控制注册、发帖 / 发评论和图片上传。

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
GET /api/v1/posts?source=recommended&sort=hot&t=day&limit=20&offset=0
```

`source=recommended` 当前是后端公开可解释推荐流，默认 `sort=hot`，匿名读取使用 `hot + new` 混排并做社区 rank 去重；携带有效 Bearer 时会给关注社区和互动过的社区加权。显式传 `sort=best|hot|new|top|rising` 时，该排序语义作为推荐基线；它不是机器学习推荐，也不是预计算时间线。`t` 支持 `hour|day|week|month|year|all`。

发帖和编辑帖子支持可选 `content_refs` 请求字段，元素结构为 `{ "kind": "image|link_preview|embed", "ref_id": "..." }`。`image` 引用必须指向同一次请求绑定或帖子当前已绑定的图片附件 ID；编辑时省略 `content_refs` 保留原引用，传空数组清空原引用。

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

发评论和编辑评论同样支持可选 `content_refs`，语义与帖子一致：按请求顺序返回，`image` 引用必须匹配评论已绑定图片附件，省略保留、空数组清空。

### Media

```text
POST /api/v1/uploads/images
POST /api/v1/link-previews/resolve
POST /api/v1/embeds/resolve
```

上传使用 `multipart/form-data`，字段为 `file` 和可选 `alt_text`；PNG/JPEG 上传成功后会返回解析出的图片 `width` 和 `height`。大于 512px 边长的 PNG/JPEG 会同步生成最大边 512px 的独立 JPEG 缩略图，附件响应中的 `thumbnail_url` 指向该缩略图；小图、WebP 或缩略图生成 / 上传失败时，`thumbnail_url` 回退为原图 `url`。
未绑定和异常附件通过后台清理命令回收：

```bash
go run ./cmd/media-cleanup -dry-run
go run ./cmd/media-cleanup -unbound-ttl=24h -failed-ttl=24h -limit=100
```

清理候选只包含 `owner_type=none` 的附件：`ready` 状态按未绑定 TTL 回收，`failed` / `blocked` 状态按异常 TTL 回收；TTL 从当前状态的 `updated_at` 计算。实际清理会先领取并删除附件元数据，再删除原图和缩略图对象。生产环境应由 cron、systemd timer 或 Windows Task Scheduler 周期性运行该命令。

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

搜索基于 PostgreSQL full-text search 和 `ts_rank_cd` 排序；帖子搜索字段权重为标题 > 社区名/slug > 正文，并叠加轻量时间衰减。

评论、回复、帖子点赞、评论点赞和正文 `@username` 提及会写入站内通知；点赞通知按收件人、通知类型、目标内容和小时窗口聚合未读计数。提及通知在帖子 / 评论发布时生成，编辑时只为新增提及生成，并进入 `mentions` 分类。

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
  admin/      平台管理、运行开关、审计日志
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
